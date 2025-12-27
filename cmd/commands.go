package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/hwang-fu/minicontainer/cgroup"
	"github.com/hwang-fu/minicontainer/fs"
	"github.com/hwang-fu/minicontainer/image"
	"github.com/hwang-fu/minicontainer/state"
)

// RunStop stops a running container.
func RunStop(idOrName string) {
	cs, err := state.FindContainer(idOrName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if cs.Status != state.StatusRunning {
		fmt.Fprintf(os.Stderr, "container %s is not running\n", cs.Name)
		os.Exit(1)
	}

	syscall.Kill(cs.PID, syscall.SIGTERM)
	time.Sleep(100 * time.Millisecond)
	syscall.Kill(cs.PID, syscall.SIGKILL)

	fmt.Println(state.ShortID(cs.ID))
}

// RunRm removes a stopped container.
func RunRm(idOrName string) {
	cs, err := state.FindContainer(idOrName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if cs.Status == state.StatusRunning {
		fmt.Fprintf(os.Stderr, "cannot remove running container %s, stop it first\n", cs.Name)
		os.Exit(1)
	}

	cgroup.RemoveContainerCgroup(cs.ID)
	if err := os.RemoveAll(state.ContainerDir(cs.ID)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove container: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(state.ShortID(cs.ID))
}

// RunRmAll removes all stopped containers.
func RunRmAll() {
	containers, err := state.ListContainers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, cs := range containers {
		if cs.Status == state.StatusRunning {
			continue
		}
		cgroup.RemoveContainerCgroup(cs.ID)
		os.RemoveAll(state.ContainerDir(cs.ID))
		fmt.Println(state.ShortID(cs.ID))
	}
}

// RunPs lists containers.
func RunPs(showAll bool) {
	containers, err := state.ListContainers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-12s  %-20s  %-10s  %s\n", "CONTAINER ID", "COMMAND", "STATUS", "NAME")
	for _, c := range containers {
		if !showAll && c.Status != state.StatusRunning {
			continue
		}
		cmdStr := strings.Join(c.Command, " ")
		if len(cmdStr) > 20 {
			cmdStr = cmdStr[:17] + "..."
		}
		fmt.Printf("%-12s  %-20s  %-10s  %s\n",
			state.ShortID(c.ID), cmdStr, c.Status, c.Name)
	}
}

// RunPrune removes stale overlay directories.
func RunPrune() {
	fmt.Println("Cleaning up stale overlay directories...")
	removed := fs.CleanupStaleOverlays()
	if len(removed) == 0 {
		fmt.Println("Nothing to clean.")
	} else {
		for _, dir := range removed {
			fmt.Printf("  Removed: %s\n", dir)
		}
		fmt.Printf("Removed %d directories.\n", len(removed))
	}
}

// RunImport imports a rootfs tarball as an image.
// Creates a single-layer image from the tarball that can be used with `run`.
//
// Parameters:
//   - tarballPath: path to the .tar or .tar.gz rootfs archive
//   - imageRef: image reference in "name" or "name:tag" format
func RunImport(tarballPath, imageRef string) {
	meta, err := image.ImportTarball(tarballPath, imageRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "import failed: %v\n", err)
		os.Exit(1)
	}

	// Print success with short ID (first 12 chars)
	fmt.Printf("Imported %s:%s (id: %s)\n", meta.Name, meta.Tag, meta.ID[:12])
}

// ResolveRootfs resolves the rootfs path from config or image reference.
// If --rootfs is provided, uses that directly.
// Otherwise, treats the first cmdArg as an image reference and looks it up.
//
// Parameters:
//   - cfg: container config (may have RootfsPath set)
//   - cmdArgs: command arguments (first may be image reference)
//
// Returns:
//   - updated config with RootfsPath set
//   - remaining command arguments (image reference removed if used)
//   - error if image lookup fails
func ResolveRootfs(cfg *ContainerConfig, cmdArgs []string) (*ContainerConfig, []string, error) {
	// If --rootfs provided, use it directly
	if cfg.RootfsPath != "" {
		return cfg, cmdArgs, nil
	}

	// Otherwise, first arg is image reference
	if len(cmdArgs) < 1 {
		return nil, nil, fmt.Errorf("no image or --rootfs specified")
	}

	imageRef := cmdArgs[0]
	cmdArgs = cmdArgs[1:] // Remaining args are the command

	// Look up image to get layer path
	rootfsPath, err := image.LookupImage(imageRef)
	if err != nil {
		return nil, nil, err
	}

	cfg.RootfsPath = rootfsPath
	return cfg, cmdArgs, nil
}

// RunImages lists all local images.
// Displays repository, tag, image ID (short), size, and creation time.
func RunImages() {
	images, err := image.ListImages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Print header
	fmt.Printf("%-15s  %-10s  %-12s  %-10s  %s\n",
		"REPOSITORY", "TAG", "IMAGE ID", "SIZE", "CREATED")

	// Print each image
	for _, img := range images {
		fmt.Printf("%-15s  %-10s  %-12s  %-10s  %s\n",
			img.Name,
			img.Tag,
			img.ID[:12],
			formatSize(img.Size),
			formatTimeAgo(img.CreatedAt),
		)
	}
}

// RunRmi removes an image by reference (name:tag or ID).
// Deletes image metadata and unreferenced layers.
//
// Parameters:
//   - ref: image reference ("name:tag") or image ID (full or short)
func RunRmi(ref string) {
	if err := image.RemoveImage(ref); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed: %s\n", ref)
}

// RunPull pulls an image from a registry.
func RunPull(ref string) {
	meta, err := image.Pull(ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pull failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Pulled: %s:%s (%s)\n", meta.Name, meta.Tag, meta.ID[:12])
}

// formatSize converts bytes to human-readable format (e.g., "3.2 MB").
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatTimeAgo converts a time to relative format (e.g., "2 minutes ago").
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "Just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
