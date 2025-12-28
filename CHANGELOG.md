# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-12-28

### Added

#### Container Runtime
- Linux namespace isolation (PID, Mount, Network, UTS, IPC, User)
- Cgroups v2 resource limits (memory, CPU, pids)
- Overlayfs filesystem with copy-on-write
- Volume mounts (`-v host:container[:ro]`)
- pivot_root for secure root filesystem isolation

#### Networking
- Bridge network (`minicontainer0`)
- Veth pair creation and management
- IP address allocation (IPAM)
- NAT for outbound connectivity
- Port publishing (`-p host:container`)

#### Image Management
- Pull images from Docker Hub
- Import local tarballs as images
- Content-addressable layer storage
- Layer caching and deduplication

#### CLI Commands
- `run` - Create and run containers
- `exec` - Execute commands in running containers
- `stop` - Stop running containers
- `rm` - Remove stopped containers
- `ps` - List containers
- `logs` - Fetch container logs (with timestamps)
- `inspect` - Display container details as JSON
- `pull` - Pull images from registry
- `images` - List local images
- `rmi` - Remove images
- `import` - Import tarballs as images
- `prune` - Clean up stale overlays
- `--help` / `-h` - Global and per-command help
- `--version` / `-v` - Version information
