package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// timeRangeRE validates HH:MM-HH:MM format.
var timeRangeRE = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$`)

// Session represents a parsed conference session.
type Session struct {
	HexID       string    `json:"hex_id"`
	ShortID     string    `json:"short_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location,omitempty"`
	Category    string    `json:"category,omitempty"`
	Speakers    []string  `json:"speakers,omitempty"`
	Materials   []Material `json:"materials,omitempty"`
	EventURL    string    `json:"event_url,omitempty"`
	FetchedAt   time.Time `json:"fetched_at"`
}

// Material represents a file attachment (slides, PDF) for a session.
type Material struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Friend represents a friend mapping (nickname -> Sched username).
type Friend struct {
	Nickname string `json:"nickname"`
	Username string `json:"username"`
}

// CacheMeta tracks when a resource was last fetched and its TTL.
type CacheMeta struct {
	Resource   string        `json:"resource"`
	FetchedAt  time.Time     `json:"fetched_at"`
	ETag       string        `json:"etag,omitempty"`
	TTLSeconds int           `json:"ttl_seconds"`
}

// SessionFilters holds filter criteria for listing sessions.
type SessionFilters struct {
	Track  string
	Day    string // YYYY-MM-DD
	Time   string // time range
	Search string
}

// APICall records a single API request for rate limiting.
type APICall struct {
	CalledAt time.Time `json:"called_at"`
	Endpoint string    `json:"endpoint"`
	Method   string    `json:"method"`
}

// Store wraps a SQLite database for caching and local state.
type Store struct {
	db *sql.DB
}

// New opens or creates a SQLite database at the given path.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	// Clean up old api_calls rows (>24h) to prevent unbounded growth
	s.CleanupAPICalls(24 * time.Hour)
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection (for testing).
func (s *Store) DB() *sql.DB {
	return s.db
}

// SchemaVersion is the current schema version. Increment this and add a new
// migration function to the migrations slice when the schema changes.
const SchemaVersion = 1

// migrations is an ordered list of migration functions. Each function migrates
// the database from version N-1 to version N (where N is the function's
// 1-based index in the slice). To add a new migration, append a function here
// and bump SchemaVersion.
var migrations = []func(db *sql.DB) error{
	migrateV1, // creates all initial tables and indexes
}

// migrateV1 creates the initial schema (all tables and indexes).
func migrateV1(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		hex_id      TEXT PRIMARY KEY,
		short_id    TEXT UNIQUE,
		title       TEXT NOT NULL,
		description TEXT,
		start_time  DATETIME,
		end_time    DATETIME,
		location    TEXT,
		category    TEXT,
		speakers    TEXT,
		materials   TEXT,
		event_url   TEXT,
		fetched_at  DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS schedule (
		hex_id   TEXT PRIMARY KEY REFERENCES sessions(hex_id),
		added_at DATETIME NOT NULL,
		source   TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS interests (
		hex_id     TEXT PRIMARY KEY REFERENCES sessions(hex_id),
		flagged_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS friends (
		nickname TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS friend_schedules (
		username TEXT NOT NULL,
		hex_id   TEXT NOT NULL,
		fetched_at DATETIME NOT NULL,
		PRIMARY KEY (username, hex_id)
	);

	CREATE TABLE IF NOT EXISTS api_calls (
		called_at DATETIME NOT NULL,
		endpoint  TEXT NOT NULL,
		method    TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS cache_meta (
		resource    TEXT PRIMARY KEY,
		fetched_at  DATETIME NOT NULL,
		etag        TEXT,
		ttl_seconds INTEGER NOT NULL DEFAULT 172800
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_short_id ON sessions(short_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_category ON sessions(category);
	CREATE INDEX IF NOT EXISTS idx_sessions_start ON sessions(start_time);
	CREATE INDEX IF NOT EXISTS idx_api_calls_time ON api_calls(called_at);
	`
	_, err := db.Exec(schema)
	return err
}

func (s *Store) migrate() error {
	// Ensure the schema_version table exists.
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		return fmt.Errorf("creating schema_version table: %w", err)
	}

	// Read current version (0 means fresh database).
	var current int
	row := s.db.QueryRow("SELECT version FROM schema_version LIMIT 1")
	if err := row.Scan(&current); err == sql.ErrNoRows {
		current = 0
	} else if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	// Guard against downgrades: database was created by a newer binary.
	if current > SchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d (downgrade not supported)", current, SchemaVersion)
	}

	// Already up to date.
	if current == SchemaVersion {
		return nil
	}

	// Run pending migrations in order.
	for v := current; v < SchemaVersion; v++ {
		if err := migrations[v](s.db); err != nil {
			return fmt.Errorf("migration to version %d failed: %w", v+1, err)
		}
	}

	// Record the new version.
	if current == 0 {
		_, err = s.db.Exec("INSERT INTO schema_version (version) VALUES (?)", SchemaVersion)
	} else {
		_, err = s.db.Exec("UPDATE schema_version SET version = ?", SchemaVersion)
	}
	if err != nil {
		return fmt.Errorf("updating schema version: %w", err)
	}

	return nil
}

// scanSession scans a session row, handling NULL short_id and JSON fields.
func scanSession(scanner interface{ Scan(...interface{}) error }) (*Session, error) {
	var sess Session
	var shortID sql.NullString
	var speakers, materials string
	err := scanner.Scan(&sess.HexID, &shortID, &sess.Title, &sess.Description,
		&sess.StartTime, &sess.EndTime, &sess.Location, &sess.Category,
		&speakers, &materials, &sess.EventURL, &sess.FetchedAt)
	if err != nil {
		return nil, err
	}
	sess.ShortID = shortID.String
	if speakers != "" {
		if err := json.Unmarshal([]byte(speakers), &sess.Speakers); err != nil {
			return nil, fmt.Errorf("unmarshaling speakers: %w", err)
		}
	}
	if materials != "" {
		if err := json.Unmarshal([]byte(materials), &sess.Materials); err != nil {
			return nil, fmt.Errorf("unmarshaling materials: %w", err)
		}
	}
	return &sess, nil
}

// UpsertSession inserts or updates a session.
func (s *Store) UpsertSession(sess Session) error {
	speakers := sess.Speakers
	if speakers == nil {
		speakers = []string{}
	}
	speakersBytes, err := json.Marshal(speakers)
	if err != nil {
		return fmt.Errorf("marshaling speakers: %w", err)
	}
	speakersJSON := string(speakersBytes)

	materials := sess.Materials
	if materials == nil {
		materials = []Material{}
	}
	materialsBytes, err := json.Marshal(materials)
	if err != nil {
		return fmt.Errorf("marshaling materials: %w", err)
	}
	materialsJSON := string(materialsBytes)
	// Store empty short_id as NULL so UNIQUE constraint doesn't conflict
	// (SQLite treats NULLs as distinct in UNIQUE constraints)
	var shortID interface{}
	if sess.ShortID != "" {
		shortID = sess.ShortID
	}
	_, err = s.db.Exec(`
		INSERT INTO sessions (hex_id, short_id, title, description, start_time, end_time, location, category, speakers, materials, event_url, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hex_id) DO UPDATE SET
			short_id=COALESCE(excluded.short_id, sessions.short_id), title=excluded.title, description=excluded.description,
			start_time=excluded.start_time, end_time=excluded.end_time, location=excluded.location,
			category=excluded.category, speakers=excluded.speakers, materials=excluded.materials,
			event_url=excluded.event_url, fetched_at=excluded.fetched_at`,
		sess.HexID, shortID, sess.Title, sess.Description,
		sess.StartTime, sess.EndTime, sess.Location, sess.Category,
		speakersJSON, materialsJSON, sess.EventURL, sess.FetchedAt,
	)
	return err
}

// GetSession retrieves a session by hex ID or short ID.
func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`
		SELECT hex_id, short_id, title, description, start_time, end_time, location, category, speakers, materials, event_url, fetched_at
		FROM sessions WHERE hex_id = ? OR short_id = ?`, id, id)
	sess, err := scanSession(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

// ListSessions returns sessions matching the given filters.
func (s *Store) ListSessions(filters SessionFilters) ([]Session, error) {
	query := "SELECT hex_id, short_id, title, description, start_time, end_time, location, category, speakers, materials, event_url, fetched_at FROM sessions WHERE 1=1"
	var args []interface{}

	if filters.Track != "" {
		query += " AND category = ?"
		args = append(args, filters.Track)
	}
	if filters.Day != "" {
		query += " AND substr(start_time, 1, 10) = ?"
		args = append(args, filters.Day)
	}
	if filters.Time != "" {
		if !timeRangeRE.MatchString(filters.Time) {
			return nil, fmt.Errorf("invalid time range format %q: expected HH:MM-HH:MM", filters.Time)
		}
		start := filters.Time[:5]
		end := filters.Time[6:]
		if start < end {
			// Normal range (e.g., 09:00-12:00)
			query += " AND substr(start_time, 12, 5) >= ? AND substr(start_time, 12, 5) <= ?"
			args = append(args, start, end)
		} else {
			// Cross-midnight range (e.g., 22:00-02:00)
			query += " AND (substr(start_time, 12, 5) >= ? OR substr(start_time, 12, 5) <= ?)"
			args = append(args, start, end)
		}
	}
	if filters.Search != "" {
		query += " AND (title LIKE ? OR description LIKE ?)"
		pattern := "%" + filters.Search + "%"
		args = append(args, pattern, pattern)
	}

	query += " ORDER BY start_time ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *sess)
	}
	return sessions, rows.Err()
}

// AddToSchedule marks a session as on the user's schedule.
func (s *Store) AddToSchedule(hexID, source string) error {
	_, err := s.db.Exec(`
		INSERT INTO schedule (hex_id, added_at, source) VALUES (?, ?, ?)
		ON CONFLICT(hex_id) DO UPDATE SET source=excluded.source`,
		hexID, time.Now().UTC(), source,
	)
	return err
}

// RemoveFromSchedule removes a session from the user's schedule.
func (s *Store) RemoveFromSchedule(hexID string) error {
	_, err := s.db.Exec("DELETE FROM schedule WHERE hex_id = ?", hexID)
	return err
}

// GetSchedule returns all sessions on the user's schedule.
func (s *Store) GetSchedule() ([]Session, error) {
	rows, err := s.db.Query(`
		SELECT s.hex_id, s.short_id, s.title, s.description, s.start_time, s.end_time, s.location, s.category, s.speakers, s.materials, s.event_url, s.fetched_at
		FROM sessions s INNER JOIN schedule sc ON s.hex_id = sc.hex_id
		ORDER BY s.start_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *sess)
	}
	return sessions, rows.Err()
}

// IsOnSchedule checks if a session is on the user's schedule.
func (s *Store) IsOnSchedule(hexID string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM schedule WHERE hex_id = ?", hexID).Scan(&count)
	return count > 0, err
}

// AddInterest flags a session as "interested" (local only).
func (s *Store) AddInterest(hexID string) error {
	_, err := s.db.Exec(`
		INSERT INTO interests (hex_id, flagged_at) VALUES (?, ?)
		ON CONFLICT(hex_id) DO NOTHING`,
		hexID, time.Now().UTC(),
	)
	return err
}

// RemoveInterest removes the "interested" flag from a session.
func (s *Store) RemoveInterest(hexID string) error {
	_, err := s.db.Exec("DELETE FROM interests WHERE hex_id = ?", hexID)
	return err
}

// ListInterests returns all sessions flagged as "interested".
func (s *Store) ListInterests() ([]Session, error) {
	rows, err := s.db.Query(`
		SELECT s.hex_id, s.short_id, s.title, s.description, s.start_time, s.end_time, s.location, s.category, s.speakers, s.materials, s.event_url, s.fetched_at
		FROM sessions s INNER JOIN interests i ON s.hex_id = i.hex_id
		ORDER BY s.start_time ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *sess)
	}
	return sessions, rows.Err()
}

// IsInterested checks if a session is flagged as interested.
func (s *Store) IsInterested(hexID string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM interests WHERE hex_id = ?", hexID).Scan(&count)
	return count > 0, err
}

// AddFriend adds a friend mapping.
func (s *Store) AddFriend(nickname, username string) error {
	_, err := s.db.Exec(`
		INSERT INTO friends (nickname, username) VALUES (?, ?)
		ON CONFLICT(nickname) DO UPDATE SET username=excluded.username`,
		nickname, username,
	)
	return err
}

// RemoveFriend removes a friend mapping.
func (s *Store) RemoveFriend(nickname string) error {
	result, err := s.db.Exec("DELETE FROM friends WHERE nickname = ?", nickname)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListFriends returns all friend mappings.
func (s *Store) ListFriends() ([]Friend, error) {
	rows, err := s.db.Query("SELECT nickname, username FROM friends ORDER BY nickname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []Friend
	for rows.Next() {
		var f Friend
		if err := rows.Scan(&f.Nickname, &f.Username); err != nil {
			return nil, err
		}
		friends = append(friends, f)
	}
	return friends, rows.Err()
}

// GetFriendByNickname returns a friend by nickname.
func (s *Store) GetFriendByNickname(nickname string) (*Friend, error) {
	var f Friend
	err := s.db.QueryRow("SELECT nickname, username FROM friends WHERE nickname = ?", nickname).Scan(&f.Nickname, &f.Username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// UpsertFriendSchedule replaces a friend's cached schedule.
func (s *Store) UpsertFriendSchedule(username string, hexIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM friend_schedules WHERE username = ?", username); err != nil {
		return err
	}
	for _, id := range hexIDs {
		if _, err := tx.Exec("INSERT INTO friend_schedules (username, hex_id, fetched_at) VALUES (?, ?, ?)",
			username, id, time.Now().UTC()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetFriendSchedule returns hex IDs on a friend's schedule.
func (s *Store) GetFriendSchedule(username string) ([]string, error) {
	rows, err := s.db.Query("SELECT hex_id FROM friend_schedules WHERE username = ?", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LogAPICall records an API call for rate limiting.
func (s *Store) LogAPICall(endpoint, method string) error {
	_, err := s.db.Exec("INSERT INTO api_calls (called_at, endpoint, method) VALUES (?, ?, ?)",
		time.Now().UTC(), endpoint, method)
	return err
}

// GetAPICallCount returns the number of API calls in the given window.
func (s *Store) GetAPICallCount(window time.Duration) (int, error) {
	var count int
	cutoff := time.Now().UTC().Add(-window)
	err := s.db.QueryRow("SELECT COUNT(*) FROM api_calls WHERE called_at >= ?", cutoff).Scan(&count)
	return count, err
}

// CleanupAPICalls removes API call records older than the given window.
func (s *Store) CleanupAPICalls(window time.Duration) error {
	cutoff := time.Now().UTC().Add(-window)
	_, err := s.db.Exec("DELETE FROM api_calls WHERE called_at < ?", cutoff)
	return err
}

// SetCacheMeta sets or updates cache metadata for a resource.
func (s *Store) SetCacheMeta(resource, etag string, ttlSeconds int) error {
	_, err := s.db.Exec(`
		INSERT INTO cache_meta (resource, fetched_at, etag, ttl_seconds) VALUES (?, ?, ?, ?)
		ON CONFLICT(resource) DO UPDATE SET fetched_at=excluded.fetched_at, etag=excluded.etag, ttl_seconds=excluded.ttl_seconds`,
		resource, time.Now().UTC(), etag, ttlSeconds,
	)
	return err
}

// GetCacheMeta returns cache metadata for a resource.
func (s *Store) GetCacheMeta(resource string) (*CacheMeta, error) {
	var cm CacheMeta
	err := s.db.QueryRow("SELECT resource, fetched_at, etag, ttl_seconds FROM cache_meta WHERE resource = ?", resource).
		Scan(&cm.Resource, &cm.FetchedAt, &cm.ETag, &cm.TTLSeconds)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cm, nil
}

// IsCacheStale checks if a cached resource has exceeded its TTL.
func (s *Store) IsCacheStale(resource string) (bool, error) {
	cm, err := s.GetCacheMeta(resource)
	if err != nil {
		return true, err
	}
	if cm == nil {
		return true, nil // no cache entry = stale
	}
	age := time.Since(cm.FetchedAt)
	return age > time.Duration(cm.TTLSeconds)*time.Second, nil
}

// SessionCount returns the total number of cached sessions.
func (s *Store) SessionCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	return count, err
}
