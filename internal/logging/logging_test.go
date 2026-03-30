package logging

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- Log File Creation ---

func TestNew_CreatesDateDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir, Command: "test"}

	l, err := mustLogger(t, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	today := time.Now().Format("2006-01-02")
	dateDir := filepath.Join(dir, today)
	info, err := os.Stat(dateDir)
	if err != nil {
		t.Fatalf("date directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("date directory is not a directory")
	}
}

func TestNew_CreatesLogFileWithCorrectNamePattern(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir, Command: "sessions"}

	l, err := mustLogger(t, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	logPath := l.LogPath()
	base := filepath.Base(logPath)

	// Expected pattern: HH-MM-SS-sessions.log
	if !strings.HasSuffix(base, "-sessions.log") {
		t.Errorf("log file name %q does not end with -sessions.log", base)
	}

	// Check the time prefix is 8 chars like "15-04-05"
	parts := strings.SplitN(base, "-", 4)
	if len(parts) < 4 {
		t.Fatalf("log file name %q doesn't have enough dash-separated parts", base)
	}
	// First three parts should be numeric (HH, MM, SS)
	for _, p := range parts[:3] {
		if len(p) != 2 {
			t.Errorf("time component %q is not 2 digits", p)
		}
	}
}

func TestLogPath_ReturnsCorrectPath(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir, Command: "info"}

	l, err := mustLogger(t, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	logPath := l.LogPath()
	if logPath == "" {
		t.Fatal("LogPath() returned empty string")
	}

	// File should exist on disk
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file does not exist at LogPath(): %v", err)
	}
}

func TestLogFile_InCorrectDateDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir, Command: "sync"}

	l, err := mustLogger(t, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	today := time.Now().Format("2006-01-02")
	expectedDir := filepath.Join(dir, today)
	actualDir := filepath.Dir(l.LogPath())

	if actualDir != expectedDir {
		t.Errorf("log file dir = %q, want %q", actualDir, expectedDir)
	}
}

// --- Structured Logging ---

func TestInfo_WritesValidJSON(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	l.Info("test message")
	entry := readLastEntry(t, l.LogPath())

	if entry.Level != "INFO" {
		t.Errorf("level = %q, want INFO", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("message = %q, want %q", entry.Message, "test message")
	}
}

func TestError_WritesValidJSON(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	l.Error("something broke")
	entry := readLastEntry(t, l.LogPath())

	if entry.Level != "ERROR" {
		t.Errorf("level = %q, want ERROR", entry.Level)
	}
	if entry.Message != "something broke" {
		t.Errorf("message = %q, want %q", entry.Message, "something broke")
	}
}

func TestDebug_WritesNothingWhenDebugOff(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test", Debug: false})
	defer cleanup()

	l.Debug("hidden message")

	entries := readAllEntries(t, l.LogPath())
	if len(entries) != 0 {
		t.Errorf("expected 0 entries with debug off, got %d", len(entries))
	}
}

func TestDebug_WritesJSONWhenDebugOn(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test", Debug: true})
	defer cleanup()

	l.Debug("visible debug")
	entry := readLastEntry(t, l.LogPath())

	if entry.Level != "DEBUG" {
		t.Errorf("level = %q, want DEBUG", entry.Level)
	}
	if entry.Message != "visible debug" {
		t.Errorf("message = %q, want %q", entry.Message, "visible debug")
	}
}

func TestWarn_WritesValidJSON(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	l.Warn("heads up")
	entry := readLastEntry(t, l.LogPath())

	if entry.Level != "WARN" {
		t.Errorf("level = %q, want WARN", entry.Level)
	}
	if entry.Message != "heads up" {
		t.Errorf("message = %q, want %q", entry.Message, "heads up")
	}
}

func TestLogEntry_HasRFC3339Timestamp(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	before := time.Now().UTC().Add(-1 * time.Second)
	l.Info("timestamp check")
	after := time.Now().UTC().Add(1 * time.Second)

	entry := readLastEntry(t, l.LogPath())
	ts, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		t.Fatalf("timestamp %q is not RFC3339: %v", entry.Timestamp, err)
	}
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}

func TestLogEntry_HasCorrectLevelField(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test", Debug: true})
	defer cleanup()

	l.Info("i")
	l.Error("e")
	l.Warn("w")
	l.Debug("d")

	entries := readAllEntries(t, l.LogPath())
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	expected := []string{"INFO", "ERROR", "WARN", "DEBUG"}
	for i, want := range expected {
		if entries[i].Level != want {
			t.Errorf("entry %d: level = %q, want %q", i, entries[i].Level, want)
		}
	}
}

func TestLogEntry_HasCorrectMessageField(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	l.Info("exact message content")
	entry := readLastEntry(t, l.LogPath())
	if entry.Message != "exact message content" {
		t.Errorf("message = %q, want %q", entry.Message, "exact message content")
	}
}

func TestLogEntry_IncludesFieldsWhenProvided(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	l.Info("with fields", Field{Key: "count", Value: 42}, Field{Key: "name", Value: "alice"})
	entry := readLastEntry(t, l.LogPath())

	if entry.Fields == nil {
		t.Fatal("fields is nil, expected non-nil map")
	}
	if v, ok := entry.Fields["count"]; !ok {
		t.Error("missing field 'count'")
	} else {
		// JSON numbers unmarshal as float64
		if fv, ok := v.(float64); !ok || fv != 42 {
			t.Errorf("field count = %v, want 42", v)
		}
	}
	if v, ok := entry.Fields["name"]; !ok || v != "alice" {
		t.Errorf("field name = %v, want alice", v)
	}
}

func TestLogEntry_OmitsFieldsWhenNoneProvided(t *testing.T) {
	l, cleanup := tempLogger(t, LogConfig{Command: "test"})
	defer cleanup()

	l.Info("no fields")

	// Read raw JSON to verify omitempty
	data, err := os.ReadFile(l.LogPath())
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if strings.Contains(string(data), `"fields"`) {
		t.Error("JSON contains 'fields' key when none were provided; expected omitempty")
	}
}

// --- Log Retention / Cleanup ---

func TestCleanupOldLogs_RemovesOldDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir}
	l := &Logger{config: cfg}

	// Create an old directory (30 days ago)
	oldDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	os.MkdirAll(filepath.Join(dir, oldDate), 0755)

	err := l.CleanupOldLogs(7)
	if err != nil {
		t.Fatalf("CleanupOldLogs error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, oldDate)); !os.IsNotExist(err) {
		t.Error("old directory should have been removed")
	}
}

func TestCleanupOldLogs_KeepsRecentDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir}
	l := &Logger{config: cfg}

	// Create a recent directory (2 days ago)
	recentDate := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	os.MkdirAll(filepath.Join(dir, recentDate), 0755)

	err := l.CleanupOldLogs(7)
	if err != nil {
		t.Fatalf("CleanupOldLogs error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, recentDate)); os.IsNotExist(err) {
		t.Error("recent directory should NOT have been removed")
	}
}

func TestCleanupOldLogs_IgnoresNonDateDirectories(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir}
	l := &Logger{config: cfg}

	// Create non-date directories
	os.MkdirAll(filepath.Join(dir, "not-a-date"), 0755)
	os.MkdirAll(filepath.Join(dir, "temp"), 0755)

	err := l.CleanupOldLogs(7)
	if err != nil {
		t.Fatalf("CleanupOldLogs error: %v", err)
	}

	// Both should still exist
	if _, err := os.Stat(filepath.Join(dir, "not-a-date")); os.IsNotExist(err) {
		t.Error("non-date directory 'not-a-date' should not be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "temp")); os.IsNotExist(err) {
		t.Error("non-date directory 'temp' should not be removed")
	}
}

func TestCleanupOldLogs_HandlesEmptyLogDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := LogConfig{LogDir: dir}
	l := &Logger{config: cfg}

	err := l.CleanupOldLogs(7)
	if err != nil {
		t.Fatalf("CleanupOldLogs on empty dir should not error: %v", err)
	}
}

// --- Utility Functions ---

func TestSessionLogFileName_CorrectFormat(t *testing.T) {
	name := SessionLogFileName("sessions")
	if !strings.HasSuffix(name, "-sessions.log") {
		t.Errorf("name %q doesn't end with -sessions.log", name)
	}
	if !strings.HasSuffix(name, ".log") {
		t.Errorf("name %q doesn't end with .log", name)
	}
}

func TestSessionLogFileName_SanitizesSpaces(t *testing.T) {
	name := SessionLogFileName("my command")
	if strings.Contains(name, " ") {
		t.Errorf("name %q still contains spaces", name)
	}
	if !strings.Contains(name, "my-command") {
		t.Errorf("name %q doesn't contain 'my-command'", name)
	}
}

func TestSessionLogFileName_SanitizesSlashes(t *testing.T) {
	name := SessionLogFileName("foo/bar")
	if strings.Contains(name, "/") {
		t.Errorf("name %q still contains slashes", name)
	}
	if !strings.Contains(name, "foo-bar") {
		t.Errorf("name %q doesn't contain 'foo-bar'", name)
	}
}

func TestSessionLogFileName_UsesUnknownForEmpty(t *testing.T) {
	name := SessionLogFileName("")
	if !strings.HasSuffix(name, "-unknown.log") {
		t.Errorf("name %q doesn't end with -unknown.log", name)
	}
}

func TestListLogDays_ReturnsSortedDateDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create date directories in non-sorted order
	dates := []string{"2025-03-15", "2025-03-10", "2025-03-20"}
	for _, d := range dates {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}

	days, err := ListLogDays(dir)
	if err != nil {
		t.Fatalf("ListLogDays error: %v", err)
	}

	if len(days) != 3 {
		t.Fatalf("expected 3 days, got %d", len(days))
	}

	expected := []string{"2025-03-10", "2025-03-15", "2025-03-20"}
	for i, want := range expected {
		if days[i] != want {
			t.Errorf("days[%d] = %q, want %q", i, days[i], want)
		}
	}
}

func TestListLogDays_IgnoresNonDateDirsAndFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a mix of date dirs, non-date dirs, and files
	os.MkdirAll(filepath.Join(dir, "2025-03-15"), 0755)
	os.MkdirAll(filepath.Join(dir, "not-a-date"), 0755)
	os.WriteFile(filepath.Join(dir, "2025-03-16"), []byte("file not dir"), 0644)
	os.WriteFile(filepath.Join(dir, "random.txt"), []byte("junk"), 0644)

	days, err := ListLogDays(dir)
	if err != nil {
		t.Fatalf("ListLogDays error: %v", err)
	}

	if len(days) != 1 {
		t.Fatalf("expected 1 day, got %d: %v", len(days), days)
	}
	if days[0] != "2025-03-15" {
		t.Errorf("days[0] = %q, want %q", days[0], "2025-03-15")
	}
}

func TestDayLogDir_ReturnsCorrectPath(t *testing.T) {
	base := "/tmp/logs"
	today := time.Now().Format("2006-01-02")
	expected := filepath.Join(base, today)

	got := DayLogDir(base)
	if got != expected {
		t.Errorf("DayLogDir = %q, want %q", got, expected)
	}
}

// --- No-op Logger ---

func TestNoopLogger_EmptyLogDir(t *testing.T) {
	l, err := New(LogConfig{LogDir: ""})
	if err != nil {
		t.Fatalf("New() with empty LogDir should not error: %v", err)
	}
	if l.LogPath() != "" {
		t.Errorf("no-op logger LogPath() = %q, want empty", l.LogPath())
	}
}

func TestNoopLogger_DoesNotPanicOnCalls(t *testing.T) {
	l, err := New(LogConfig{LogDir: ""})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// These should not panic
	l.Info("info msg")
	l.Error("error msg")
	l.Debug("debug msg")
	l.Warn("warn msg")
}

func TestNoopLogger_CloseReturnsNil(t *testing.T) {
	l, err := New(LogConfig{LogDir: ""})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Errorf("Close() on no-op logger returned error: %v", err)
	}
}

// --- Test helpers ---

// mustLogger creates a Logger and fails the test on error.
func mustLogger(t *testing.T, cfg LogConfig) (*Logger, error) {
	t.Helper()
	return New(cfg)
}

// tempLogger creates a Logger in a temp directory, returning a cleanup function.
func tempLogger(t *testing.T, cfg LogConfig) (*Logger, func()) {
	t.Helper()
	dir := t.TempDir()
	cfg.LogDir = dir
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = 30 // prevent background cleanup from removing today's directory
	}
	l, err := New(cfg)
	if err != nil {
		t.Fatalf("tempLogger: %v", err)
	}
	return l, func() { l.Close() }
}

// readAllEntries reads all JSON log entries from a file.
func readAllEntries(t *testing.T, path string) []LogEntry {
	t.Helper()

	// Ensure writes are flushed
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening log file: %v", err)
	}
	defer f.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid JSON line %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanning log file: %v", err)
	}
	return entries
}

// readLastEntry reads the last JSON log entry from a file.
func readLastEntry(t *testing.T, path string) LogEntry {
	t.Helper()
	entries := readAllEntries(t, path)
	if len(entries) == 0 {
		t.Fatal("log file has no entries")
	}
	return entries[len(entries)-1]
}
