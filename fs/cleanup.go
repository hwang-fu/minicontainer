package fs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// CleanupStaleOverlays removes orphaned overlay directories from previous runs.
// Scans /tmp for minicontainer-overlay-* directories, unmounts if needed, and removes.
// Called at startup before creating new overlays.
func CleanupStaleOverlays() {
	pattern := "/tmp/minicontainer-overlay-*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return // Ignore glob errors
	}

	for _, dir := range matches {
		// Try to unmount merged directory (may or may not be mounted)
		mergedDir := filepath.Join(dir, "merged")
		syscall.Unmount(mergedDir, 0) // Ignore errors - may not be mounted

		// Remove the entire directory tree
		os.RemoveAll(dir)
	}
}

// getMountedPaths reads /proc/mounts and returns a set of mounted paths.
func getMountedPaths() map[string]bool {
	mounted := make(map[string]bool)

	file, err := os.Open("/proc/mounts")
	if err != nil {
		return mounted
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Format: device mountpoint fstype options...
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			mounted[fields[1]] = true
		}
	}

	return mounted
}
