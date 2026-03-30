package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LogConfig configures the logging system.
type LogConfig struct {
	LogDir         string
	RetentionDays  int
	Debug          bool
	SyslogEnabled  bool
	Command        string // the CLI command being run (for log file naming)
}

// Field is a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// LogEntry represents a single structured log entry.
type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger writes structured JSON logs to date-stamped directories.
type Logger struct {
	config  LogConfig
	logFile *os.File
	logPath string
}

// New creates a Logger and opens the log file for the current invocation.
func New(config LogConfig) (*Logger, error) {
	l := &Logger{config: config}

	if config.LogDir == "" {
		return l, nil // no-op logger
	}

	logPath, err := l.createLogFile()
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}
	l.logPath = logPath

	// Run cleanup in background (non-blocking)
	go l.CleanupOldLogs(config.RetentionDays)

	return l, nil
}

// LogPath returns the path to the current session's log file.
func (l *Logger) LogPath() string {
	return l.logPath
}

// Close closes the log file.
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Info logs an informational message.
func (l *Logger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...Field) {
	l.log("ERROR", msg, fields...)
}

// Debug logs a debug message (only written if debug mode is enabled).
func (l *Logger) Debug(msg string, fields ...Field) {
	if !l.config.Debug {
		return
	}
	l.log("DEBUG", msg, fields...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log("WARN", msg, fields...)
}

func (l *Logger) log(level, msg string, fields ...Field) {
	if l.logFile == nil {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
	}

	if len(fields) > 0 {
		entry.Fields = make(map[string]interface{}, len(fields))
		for _, f := range fields {
			entry.Fields[f.Key] = f.Value
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	l.logFile.Write(append(data, '\n'))
}

func (l *Logger) createLogFile() (string, error) {
	now := time.Now()
	dateDir := now.Format("2006-01-02")
	dirPath := filepath.Join(l.config.LogDir, dateDir)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", err
	}

	command := l.config.Command
	if command == "" {
		command = "unknown"
	}

	fileName := fmt.Sprintf("%s-%s.log", now.Format("15-04-05"), command)
	logPath := filepath.Join(dirPath, fileName)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return "", err
	}
	l.logFile = f

	return logPath, nil
}

// CleanupOldLogs removes date directories older than retentionDays.
func (l *Logger) CleanupOldLogs(retentionDays int) error {
	if l.config.LogDir == "" {
		return nil
	}

	entries, err := os.ReadDir(l.config.LogDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirDate, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue // skip non-date directories
		}
		if dirDate.Before(cutoff) {
			dirPath := filepath.Join(l.config.LogDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				return fmt.Errorf("removing old log dir %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// DayLogDir returns the path to today's log directory.
func DayLogDir(baseDir string) string {
	return filepath.Join(baseDir, time.Now().Format("2006-01-02"))
}

// SessionLogFileName generates a log file name for the current invocation.
func SessionLogFileName(command string) string {
	now := time.Now()
	if command == "" {
		command = "unknown"
	}
	// Sanitize command (replace spaces and slashes)
	command = strings.ReplaceAll(command, " ", "-")
	command = strings.ReplaceAll(command, "/", "-")
	return fmt.Sprintf("%s-%s.log", now.Format("15-04-05"), command)
}

// ListLogDays returns all date directories in the log dir, sorted.
func ListLogDays(logDir string) ([]string, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}

	var days []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
			days = append(days, entry.Name())
		}
	}
	sort.Strings(days)
	return days, nil
}
