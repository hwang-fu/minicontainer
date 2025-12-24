package fs

// OverlayMount holds paths for an overlayfs mount.
// Used to track the mount for cleanup.
type OverlayMount struct {
	LowerDir  string // Base filesystem (read-only)
	UpperDir  string // Changes layer (writable)
	WorkDir   string // Overlayfs internal (must be empty)
	MergedDir string // Unified view (container sees this)
	BaseDir   string // Parent directory containing upper/work/merged
}
