# MiniContainer

A Linux container runtime written in Go for educational purposes. Implements the core primitives used by Docker and other container systems: namespaces, cgroups, filesystem isolation, and networking.

## Features

**Currently Implemented:**

- **Namespace Isolation**
  - UTS (hostname)
  - PID (process IDs - container process is PID 1)
  - IPC (inter-process communication)
  - Mount (filesystem mounts)
  - User (UID/GID mapping when running as non-root)

- **Filesystem Isolation**
  - `pivot_root` for secure root filesystem switching
  - Overlayfs for copy-on-write (changes don't affect base rootfs)
  - Volume mounts (`-v host:container[:ro]`)
  - `/proc` mount (shows only container processes)
  - `/sys` mount (read-only)
  - `/dev` with essential devices (null, zero, random, urandom, tty)

- **Interactive Terminal**
  - PTY allocation (`-t` flag)
  - Interactive stdin (`-i` flag)
  - Full interactive mode (`-it`)

- **CLI Flags**
  - `--rootfs` - specify container root filesystem (required)
  - `--hostname` - custom container hostname
  - `-e, --env` - environment variables
  - `-v, --volume` - bind mount volumes (`host:container` or `host:container:ro`)
  - `-i` - keep stdin open
  - `-t` - allocate pseudo-TTY
  - `--name` - container name (placeholder)
  - `--rm` - auto-remove on exit (placeholder)

## Requirements

- Linux (kernel 4.x+ recommended)
- Go 1.21+
- Root access (sudo) for container operations

## Quick Start

### Build

```bash
make build
```

### Get a rootfs

```bash
# Download Alpine Linux minimal rootfs
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz

# Extract to test directory
mkdir -p /tmp/alpine-rootfs
tar -xzf alpine-minirootfs-3.19.0-x86_64.tar.gz -C /tmp/alpine-rootfs
```

### Run a container

```bash
# Interactive shell
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs /bin/sh

# Run a command
sudo ./minicontainer run --rootfs /tmp/alpine-rootfs /bin/echo "Hello from container!"

# With custom hostname and environment
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs --hostname mycontainer -e FOO=bar /bin/sh
```

### Inside the container

```bash
/ # hostname
mycontainer

/ # ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 /bin/sh
    7 root      0:00 ps aux

/ # ls /dev
null     random   tty      urandom  zero

/ # echo $FOO
bar

/ # exit
```

## Project Structure

```
minicontainer/
├── main.go              # Entry point, command routing, init handler
├── cmd/
│   └── config.go        # ContainerConfig, ParseRunFlags()
├── container/
│   └── run.go           # RunWithTTY(), RunWithoutTTY()
├── runtime/
│   └── pty.go           # OpenPTY(), SetRawMode()
├── fs/
│   ├── dev.go           # MountDevTmpfs(), CreateDeviceNodes()
│   ├── overlay.go       # SetupOverlayfs(), mountOverlay()
│   └── volume.go        # MountVolumes(), ParseVolumeSpec()
├── Makefile             # build, test, clean, fmt, vet, check
└── .claude/             # Project documentation
    ├── CLAUDE.md        # Development guide
    ├── project_progress.md
    └── project_requirements.md
```

## Development

```bash
# Build
make build

# Run all checks (fmt, vet, build)
make check

# Run tests (requires root)
make test

# Clean build artifacts
make clean
```

## Author

**Junzhe Wang**
- junzhe.hwangfu@gmail.com (for code contribution, bug report, and so on)
- junzhe.wang2002@gmail.com (for potential job offers or co-working opportunities)

## License

MIT License - see [LICENSE](LICENSE) for details.
