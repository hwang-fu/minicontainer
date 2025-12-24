package fs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// CleanupStaleOverlays removes orphaned overlay directories from previous runs.
// Only removes directories where the merged dir is NOT currently mounted.
// Called at startup before creating new overlays.
func CleanupStaleOverlays() {
	pattern := "/tmp/minicontainer-overlay-*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Get list of currently mounted paths
	mounted := getMountedPaths()

	for _, dir := range matches {
		mergedDir := filepath.Join(dir, "merged")

		// Skip if still mounted (in use by another container)
		if mounted[mergedDir] {
			continue
		}

		// Not mounted - safe to remove orphaned directory
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
