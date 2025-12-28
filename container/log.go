package container

import (
	"io"
	"sync"
)

// TimestampedLogWriter wraps an io.Writer and prefixes each line
// with a timestamp and stream label (stdout/stderr).
// Format: "2024-12-28T10:15:30Z [stdout] message"
type TimestampedLogWriter struct {
	writer io.Writer // Underlying writer (log file)
	stream string    // Stream label: "stdout" or "stderr"
	mu     sync.Mutex
}
