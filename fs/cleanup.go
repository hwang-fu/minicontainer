package fs

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// CleanupStaleOverlays removes orphaned overlay directories from previous runs.
// Only removes directories where the merged dir is NOT currently mounted.
// Returns the list of removed directories.
func CleanupStaleOverlays() []string {
	var removed []string
	pattern := "/tmp/minicontainer-overlay-*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return removed
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
		if err := os.RemoveAll(dir); err == nil {
			removed = append(removed, dir)
		}
	}
	return removed
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
