# MiniContainer

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org/)

> A minimal Linux container runtime written in Go for educational purposes.

MiniContainer implements the core primitives that power Docker and other container systems: **namespaces**, **cgroups**, **overlayfs**, and **networking** — all from scratch, with minimal dependencies.

---

## Why MiniContainer?

- **Learn by building** — Understand containers at the syscall level
- **Minimal dependencies** — Only Go stdlib + `golang.org/x/sys/unix`
- **Clean codebase** — Well-documented, easy to follow
- **Real isolation** — Not a toy; uses the same primitives as Docker

---

## Features

| Category | Features |
|----------|----------|
| **Namespaces** | UTS, PID, IPC, Mount, User, Network (all 6 Linux namespaces) |
| **Filesystem** | `pivot_root`, overlayfs (COW), volume mounts, `/proc`, `/sys`, `/dev` |
| **Networking** | Bridge (`minicontainer0`), veth pairs, IPAM, NAT for internet access |
| **Resource Limits** | Cgroups v2: memory (`--memory`), CPU (`--cpus`), pids (`--pids-limit`) |
| **Lifecycle** | Container IDs, state persistence, `ps`, `stop`, `rm` |
| **Terminal** | PTY allocation (`-it`), signal forwarding |
| **Modes** | Interactive, non-interactive, detached (`-d`) |

### CLI Commands

```
minicontainer run [flags] <command>   Run a container
minicontainer ps [-a]                 List containers
minicontainer stop <container>        Stop a running container
minicontainer rm <container|--all>    Remove stopped containers
minicontainer prune                   Clean stale overlay directories
minicontainer version                 Show version
```

### Run Flags

| Flag | Description |
|------|-------------|
| `--rootfs PATH` | Container root filesystem (required) |
| `--name NAME` | Container name |
| `--hostname NAME` | Container hostname |
| `-d` | Detached mode (background) |
| `-i` | Interactive (keep stdin open) |
| `-t` | Allocate pseudo-TTY |
| `-e KEY=VAL` | Set environment variable |
| `-v HOST:CONTAINER[:ro]` | Bind mount volume |
| `--memory SIZE` | Memory limit (e.g., `256m`, `1g`) |
| `--cpus N` | CPU limit (e.g., `0.5`, `2`) |
| `--pids-limit N` | Max number of processes |

---

## Quick Start

### 1. Build

```bash
make build
```

### 2. Get a rootfs

```bash
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz
mkdir -p /tmp/alpine-rootfs
tar -xzf alpine-minirootfs-3.19.0-x86_64.tar.gz -C /tmp/alpine-rootfs
```

### 3. Run a container

```bash
# Interactive shell
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs /bin/sh

# Run a command
sudo ./minicontainer run --rootfs /tmp/alpine-rootfs /bin/echo "Hello from container!"

# Detached mode
sudo ./minicontainer run -d --rootfs /tmp/alpine-rootfs /bin/sleep 60
sudo ./minicontainer ps
sudo ./minicontainer stop <id>

# With resource limits
sudo ./minicontainer run -it --memory 256m --cpus 0.5 --pids-limit 50 \
    --rootfs /tmp/alpine-rootfs /bin/sh
```

### Inside the container

```
/ # hostname
minicontainer

/ # ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 /bin/sh
    7 root      0:00 ps aux

/ # ls /dev
null  random  tty  urandom  zero

/ # ip addr show eth0
26: eth0@if27: <BROADCAST,MULTICAST,UP,LOWER_UP> ...
    inet 172.18.0.2/16 scope global eth0

/ # ping -c 2 8.8.8.8
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: seq=0 ttl=113 time=35.1 ms
64 bytes from 8.8.8.8: seq=1 ttl=113 time=34.2 ms

/ # exit
```

---

## Architecture

```mermaid
flowchart TB
    subgraph CLI["minicontainer CLI"]
        run[run]
        ps[ps]
        stop[stop]
        rm[rm]
        prune[prune]
        init[init]
    end

    subgraph Parent["Parent Process (host context)"]
        P1[Setup overlayfs]
        P2[Mount volumes]
        P3[Create bridge/veth]
        P4[Create cgroup]
        P5[Create state]
        P6[Move veth to netns]
        P7[Setup container IP]
        P8[Signal forwarding]
        P9[Wait / detach]
    end

    subgraph Child["Init Process (container context)"]
        C1[pivot_root]
        C2[Mount /proc /sys]
        C3[Setup /dev]
        C4[Set hostname]
        C5[exec user command]
    end

    run -->|"fork + namespaces"| Parent
    run -->|"re-exec /proc/self/exe init"| Child
    Parent --> P1 --> P2 --> P3 --> P4 --> P5 --> P6 --> P7 --> P8 --> P9
    Child --> C1 --> C2 --> C3 --> C4 --> C5
```

### Project Structure

```
minicontainer/
├── main.go                 # Entry point, CLI routing
├── cmd/
│   ├── config.go           # ContainerConfig, flag parsing
│   ├── init.go             # Init process (runs inside namespaces)
│   └── commands.go         # stop, rm, ps, prune commands
├── container/
│   ├── id.go               # Container ID generation (SHA256)
│   ├── runtime.go          # ContainerRuntime (shared lifecycle)
│   └── run.go              # Run modes (TTY, non-TTY, detached)
├── cgroup/
│   └── cgroup.go           # Cgroups v2 resource limits
├── network/
│   ├── bridge.go           # Bridge creation (minicontainer0)
│   ├── veth.go             # Veth pair creation and management
│   ├── ipam.go             # IP address allocation
│   ├── setup.go            # Container network configuration
│   └── nat.go              # NAT/masquerade for internet access
├── runtime/
│   └── pty.go              # PTY allocation, raw terminal mode
├── fs/
│   ├── cleanup.go          # Stale overlay cleanup
│   ├── dev.go              # /dev tmpfs and device nodes
│   ├── overlay.go          # Overlayfs mount/unmount
│   └── volume.go           # Volume bind mounts
├── state/
│   └── container.go        # State persistence (JSON)
└── Makefile
```

---

## Roadmap

- [x] **Phase 1**: Minimal isolation (namespaces, chroot)
- [x] **Phase 2**: Proper filesystem (pivot_root, overlayfs, volumes)
- [x] **Phase 3**: Container lifecycle (ps, stop, rm, detached mode)
- [x] **Phase 4**: Resource limits (cgroups v2: memory, CPU, pids)
- [x] **Phase 5**: Networking (veth, bridge, NAT) — port publishing pending
- [ ] **Phase 6**: OCI images (local tarball import)
- [ ] **Phase 7**: Registry pull (Docker Hub)
- [ ] **Phase 8**: Polish (logs, exec, inspect)

---

## Requirements

- **Linux** kernel 4.x+ (cgroups v2 recommended)
- **Go** 1.21+
- **Root access** (sudo) for container operations

---

## Development

```bash
make build      # Build binary
make check      # Run fmt, vet, build
make test       # Run tests (requires root)
make clean      # Clean build artifacts
```

---

## Author

**Junzhe Wang**

- junzhe.hwangfu@gmail.com — bug reports, contributions
- junzhe.wang2002@gmail.com — job opportunities, collaboration

---

## License

MIT License — see [LICENSE](LICENSE) for details.
