package runtime

// NamespaceType represents a Linux namespace type.
type NamespaceType struct {
	Name string // e.g., "pid", "mnt", "net"
	Flag int    // e.g., unix.CLONE_NEWPID
}
