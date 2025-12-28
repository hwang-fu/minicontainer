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

// Write implements io.Writer. Prefixes each line with timestamp and stream label.
func (tw *TimestampedLogWriter) Write(p []byte) (n int, err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	timestamp := time.Now().Format(time.RFC3339)

	// Split by newlines, keeping the newline with each segment
	lines := bytes.SplitAfter(p, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		_, err = fmt.Fprintf(tw.writer, "%s [%s] %s", timestamp, tw.stream, line)
		if err != nil {
			return 0, err
		}
	}

	return len(p), nil
}
