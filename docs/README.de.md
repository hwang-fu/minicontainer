# MiniContainer

[English](../README.md) | **[Deutsch](./README.de.md)** | [Français](./README.fr.md) | [繁體中文](./README.zh.md) | [日本語](./README.jp.md)

> **Hinweis:** Diese README wurde ursprünglich auf Englisch verfasst. Bei Unklarheiten konsultieren Sie bitte die [englische Version](../README.md).

[![CI](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml/badge.svg)](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org/)

![Demo](./demo.gif)

> Eine minimale Linux-Container-Laufzeitumgebung in Go geschrieben, für Bildungszwecke.

MiniContainer implementiert die grundlegenden Primitive, die Docker und andere Container-Systeme antreiben: **Namespaces**, **Cgroups**, **Overlayfs** und **Netzwerk** — alles von Grund auf, mit minimalen Abhängigkeiten.

---

## Warum MiniContainer?

- **Lernen durch Bauen** — Container auf Syscall-Ebene verstehen
- **Minimale Abhängigkeiten** — Nur Go stdlib + `golang.org/x/sys/unix`
- **Saubere Codebasis** — Gut dokumentiert, leicht nachvollziehbar
- **Echte Isolation** — Kein Spielzeug; verwendet dieselben Primitive wie Docker

---

## Container-Technologie: Orchestrierung, keine Erfindung (Einige persönliche Meinungen)

Container-Technologie ist im Wesentlichen eine clevere Kombination mehrerer bereits existierender Linux-Kernel-Funktionen:

| Fähigkeit | Zugrundeliegende Linux-Technologie |
|-----------|-----------------------------------|
| Prozess-Isolation | **Namespaces** (PID, Netzwerk, Mount, User, etc.) |
| Ressourcenbegrenzung | **Cgroups** (Control Groups) |
| Geschichtetes Dateisystem | **OverlayFS / AUFS** |
| Root-Verzeichnis-Isolation | **chroot / pivot_root** |

All diese Technologien existierten lange bevor Docker oder eine moderne Container-Laufzeit erschien. Sie können sogar manuell diese Primitive zusammenfügen, um einen "armen Manns Container" zu erstellen:

```bash
# Erstelle einen isolierten Namespace und führe bash darin aus
unshare --mount --uts --ipc --net --pid --fork bash
```

### Der wahre Wert von Container-Werkzeugen

Die Innovation liegt nicht in der Technologie selbst — sie liegt in der **Abstraktion und Entwicklererfahrung (DX)**:

- Komplexe Kernel-APIs in einfache Befehle wie `docker run` oder `podman run` verpacken
- Ein standardisiertes Image-Format und deklarative Build-Syntax definieren (Dockerfile, OCI-Spezifikation)
- Ökosystem-Infrastruktur wie Registries aufbauen (Docker Hub, GitHub Container Registry)

Es ist ähnlich wie Git nicht "Versionskontrolle" als Konzept erfunden hat, aber elegant Snapshots, DAGs und inhaltsadressierbaren Speicher zu etwas kombiniert hat, das einfach *funktioniert*.

> **Die Erkenntnis:** Container-Technologie ist Orchestrierung, keine Erfindung. Die eigentliche schwere Arbeit wird vom Linux-Kernel erledigt.

Dies ist auch der Grund, warum Container nur nativ auf Linux laufen — auf macOS und Windows starten Container-Laufzeiten tatsächlich eine versteckte Linux-VM im Hintergrund.

---

## Funktionen

| Kategorie | Funktionen |
|-----------|------------|
| **Namespaces** | UTS, PID, IPC, Mount, User, Network (alle 6 Linux-Namespaces) |
| **Dateisystem** | `pivot_root`, Overlayfs (COW), Volume-Mounts, `/proc`, `/sys`, `/dev` |
| **Netzwerk** | Bridge (`minicontainer0`), Veth-Paare, IPAM, NAT, Port-Publishing (`-p`) |
| **Ressourcenlimits** | Cgroups v2: Speicher (`--memory`), CPU (`--cpus`), Pids (`--pids-limit`) |
| **Images** | Pull von Docker Hub, Tarball-Import, inhaltsadressierbare Layer |
| **Lebenszyklus** | Container-IDs, Zustandspersistenz, `ps`, `stop`, `rm` |
| **Terminal** | PTY-Allokation (`-it`), Signalweiterleitung |
| **Modi** | Interaktiv, nicht-interaktiv, abgetrennt (`-d`) |

### CLI-Befehle

```
minicontainer run [flags] <image|--rootfs> <cmd>  Container ausführen
minicontainer pull <image>                        Image von Registry ziehen
minicontainer ps [-a]                             Container auflisten
minicontainer stop <container>                    Laufenden Container stoppen
minicontainer rm <container|--all>                Gestoppte Container entfernen
minicontainer import <tarball> <name[:tag]>       Tarball als Image importieren
minicontainer images                              Lokale Images auflisten
minicontainer rmi <image>                         Image entfernen
minicontainer prune                               Veraltete Overlay-Verzeichnisse bereinigen
minicontainer version                             Version anzeigen
```

### Run-Flags

| Flag | Beschreibung |
|------|--------------|
| `--rootfs PATH` | Container-Root-Dateisystem (optional bei Image-Nutzung) |
| `--name NAME` | Container-Name |
| `--hostname NAME` | Container-Hostname |
| `-d` | Abgetrennter Modus (Hintergrund) |
| `-i` | Interaktiv (stdin offen halten) |
| `-t` | Pseudo-TTY allokieren |
| `-e KEY=VAL` | Umgebungsvariable setzen |
| `-v HOST:CONTAINER[:ro]` | Volume-Bind-Mount |
| `--memory SIZE` | Speicherlimit (z.B. `256m`, `1g`) |
| `--cpus N` | CPU-Limit (z.B. `0.5`, `2`) |
| `--pids-limit N` | Maximale Prozessanzahl |
| `-p HOST:CONTAINER` | Container-Port auf Host veröffentlichen |

---

## Schnellstart

### 1. Bauen

```bash
make build
```

### 2. Rootfs besorgen

```bash
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz
mkdir -p /tmp/alpine-rootfs
tar -xzf alpine-minirootfs-3.19.0-x86_64.tar.gz -C /tmp/alpine-rootfs
```

### 3. Container ausführen

```bash
# Interaktive Shell
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs /bin/sh

# Befehl ausführen
sudo ./minicontainer run --rootfs /tmp/alpine-rootfs /bin/echo "Hallo aus dem Container!"

# Abgetrennter Modus
sudo ./minicontainer run -d --rootfs /tmp/alpine-rootfs /bin/sleep 60
sudo ./minicontainer ps
sudo ./minicontainer stop <id>

# Mit Ressourcenlimits
sudo ./minicontainer run -it --memory 256m --cpus 0.5 --pids-limit 50 \
    --rootfs /tmp/alpine-rootfs /bin/sh
```

### 4. Von Docker Hub ziehen (empfohlen)

```bash
# Image ziehen
sudo ./minicontainer pull alpine

# Images auflisten
sudo ./minicontainer images

# Vom gezogenen Image ausführen
sudo ./minicontainer run -it alpine /bin/sh

# Image entfernen
sudo ./minicontainer rmi alpine
```

### 5. Lokales Tarball importieren (Alternative)

```bash
# Tarball als Image importieren
sudo ./minicontainer import alpine-minirootfs-3.19.0-x86_64.tar.gz alpine:3.19

# Vom importierten Image ausführen
sudo ./minicontainer run -it alpine:3.19 /bin/sh
```

---

## Roadmap

- [x] **Phase 1**: Minimale Isolation (Namespaces, Chroot)
- [x] **Phase 2**: Korrektes Dateisystem (pivot_root, Overlayfs, Volumes)
- [x] **Phase 3**: Container-Lebenszyklus (ps, stop, rm, abgetrennter Modus)
- [x] **Phase 4**: Ressourcenlimits (Cgroups v2: Speicher, CPU, Pids)
- [x] **Phase 5**: Netzwerk (Veth, Bridge, NAT, Port-Publishing)
- [x] **Phase 6**: OCI-Images (Import, Images, Rmi, Ausführung von Image)
- [x] **Phase 7**: Registry-Pull (Docker Hub, Multi-Arch-Unterstützung)
- [ ] **Phase 8**: Feinschliff (Logs, Exec, Inspect)

---

## Anforderungen

- **Linux** Kernel 4.x+ (Cgroups v2 empfohlen)
- **Go** 1.24+
- **Root-Zugang** (sudo) für Container-Operationen

---

## Entwicklung

```bash
make build      # Binary bauen
make check      # fmt, vet, build ausführen
make test       # Tests ausführen (erfordert Root)
make clean      # Build-Artefakte bereinigen
```

---

## Autor

**Junzhe Wang**

- junzhe.hwangfu@gmail.com — Fehlerberichte, Beiträge
- junzhe.wang2002@gmail.com — Jobangebote, Zusammenarbeit

---

## Lizenz

MIT-Lizenz — siehe [LICENSE](../LICENSE) für Details.
