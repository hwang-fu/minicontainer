package cmd

// ContainerConfig holds the configuration options for a container.
// These are parsed from CLI flags in the run command and passed
// to the init process via environment variables.
type ContainerConfig struct {
	RootfsPath  string   // Path to container's root filesystem
	Hostname    string   // Custom hostname for the container
	Name        string   // Container name (for identification in ps, stop, etc.)
	Env         []string // User-specified environment variables (KEY=VALUE format)
	AutoRemove  bool     // If true, remove container filesystem on exit
	Interactive bool     // -i: Keep stdin open for interactive input
	AllocateTTY bool     // -t: Allocate pseudo-terminal for the container
	Volumes     []string // Volume mounts in "host:container" or "host:container:ro" format
	Detached    bool     // -d: Run container in background
	MemoryLimit string   // Memory limit (e.g., "256m", "1g")
}

// ParseRunFlags parses command-line flags for the run command.
// It returns the parsed config and the remaining arguments (the command to run).
// Example: ParseRunFlags(["--rootfs", "/tmp/alpine", "-e", "FOO=bar", "/bin/sh"])
// Returns: config{RootfsPath: "/tmp/alpine", Env: ["FOO=bar"]}, ["/bin/sh"]
func ParseRunFlags(args []string) (ContainerConfig, []string) {
	cfg := ContainerConfig{}

	i := 0
	for i < len(args) {
		switch args[i] {
		case "--rootfs":
			// Container root filesystem path
			if i+1 < len(args) {
				cfg.RootfsPath = args[i+1]
				i += 2
			}

		case "--hostname":
			// Custom hostname for UTS namespace
			if i+1 < len(args) {
				cfg.Hostname = args[i+1]
				i += 2
			}

		case "--name":
			// Container name for later reference (ps, stop, rm)
			if i+1 < len(args) {
				cfg.Name = args[i+1]
				i += 2
			}

		case "--memory":
			if i+1 < len(args) {
				cfg.MemoryLimit = args[i+1]
				i += 2
			}

		case "-e", "--env":
			// Environment variable in KEY=VALUE format
			// Can be specified multiple times
			if i+1 < len(args) {
				cfg.Env = append(cfg.Env, args[i+1])
				i += 2
			}

		case "-v", "--volume":
			// Volume mount in host:container or host:container:ro format
			if i+1 < len(args) {
				cfg.Volumes = append(cfg.Volumes, args[i+1])
				i += 2
			}

		case "-d", "--detach":
			cfg.Detached = true
			i++

		case "--rm":
			// Mark container for auto-removal on exit
			cfg.AutoRemove = true
			i++

		case "-i":
			// Interactive mode: keep stdin attached
			cfg.Interactive = true
			i++

		case "-t":
			// TTY mode: allocate pseudo-terminal
			cfg.AllocateTTY = true
			i++

		case "-it", "-ti":
			// Combined interactive + TTY (common shorthand)
			cfg.Interactive = true
			cfg.AllocateTTY = true
			i++

		default:
			// First non-flag argument is the command to run
			// Everything after is passed to that command
			return cfg, args[i:]
		}
	}
	return cfg, []string{}
}
