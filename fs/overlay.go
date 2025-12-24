package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// OverlayMount holds paths for an overlayfs mount.
// Used to track the mount for cleanup.
type OverlayMount struct {
	LowerDir  string // Base filesystem (read-only)
	UpperDir  string // Changes layer (writable)
	WorkDir   string // Overlayfs internal (must be empty)
	MergedDir string // Unified view (container sees this)
	BaseDir   string // Parent directory containing upper/work/merged
}

// SetupOverlayfs creates an overlayfs mount with the given lowerDir as the base.
// Returns an OverlayMount struct with all paths and a cleanup function.
// The cleanup function unmounts and removes temporary directories.
//
// Usage:
//
//	overlay, cleanup, err := SetupOverlayfs("/path/to/rootfs")
//	if err != nil { ... }
//	defer cleanup()
//	// Use overlay.MergedDir as the container's rootfs
func SetupOverlayfs(lowerDir string) (*OverlayMount, func() error, error) {
	baseDir, err := os.MkdirTemp("/tmp", "minicontainer-overlay-")
	if err != nil {
		return nil, nil, fmt.Errorf("create overlay base dir: %w", err)
	}

	// Create subdirectories
	upperDir := filepath.Join(baseDir, "upper")
	workDir := filepath.Join(baseDir, "work")
	mergedDir := filepath.Join(baseDir, "merged")

	for _, dir := range []string{upperDir, workDir, mergedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			os.RemoveAll(baseDir) // Cleanup on failure
			return nil, nil, fmt.Errorf("create overlay subdir %s: %w", dir, err)
		}
	}

	// Mount overlayfs
	if err := mountOverlay(lowerDir, upperDir, workDir, mergedDir); err != nil {
		os.RemoveAll(baseDir) // Cleanup on failure
		return nil, nil, err
	}

	overlay := &OverlayMount{
		LowerDir:  lowerDir,
		UpperDir:  upperDir,
		WorkDir:   workDir,
		MergedDir: mergedDir,
		BaseDir:   baseDir,
	}

	// Cleanup function: unmount and remove directories
	cleanup := func() error {
		// Unmount the overlay
		if err := syscall.Unmount(mergedDir, 0); err != nil {
			return fmt.Errorf("unmount overlay: %w", err)
		}
		// Remove all temporary directories
		if err := os.RemoveAll(baseDir); err != nil {
			return fmt.Errorf("remove overlay dirs: %w", err)
		}
		return nil
	}

	return overlay, cleanup, nil
}

// mountOverlay performs the actual overlayfs mount syscall.
func mountOverlay(lower, upper, work, merged string) error {
	// Build mount options string
	// Format: "lowerdir=<lower>,upperdir=<upper>,workdir=<work>"
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)

	// Mount overlayfs
	// - source: "overlay" (conventional name, not a real device)
	// - target: merged directory
	// - fstype: "overlay"
	// - flags: 0 (no special flags)
	// - data: mount options string
	if err := syscall.Mount("overlay", merged, "overlay", 0, opts); err != nil {
		return fmt.Errorf("mount overlay: %w", err)
	}
	return nil
}
