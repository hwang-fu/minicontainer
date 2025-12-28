# MiniContainer

[English](../README.md) | [Deutsch](./README.de.md) | **[Français](./README.fr.md)** | [繁體中文](./README.zh.md) | [日本語](./README.jp.md)

> **Note :** Ce README a été rédigé à l'origine en anglais. En cas de doute, veuillez consulter la [version anglaise](../README.md).

[![CI](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml/badge.svg)](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org/)

![Demo](./demo.gif)

> Un environnement d'exécution de conteneurs Linux minimal écrit en Go à des fins éducatives.

MiniContainer implémente les primitives fondamentales qui alimentent Docker et autres systèmes de conteneurs : **namespaces**, **cgroups**, **overlayfs** et **réseau** — tout cela depuis zéro, avec un minimum de dépendances.

---

## Pourquoi MiniContainer ?

- **Apprendre en construisant** — Comprendre les conteneurs au niveau des appels système
- **Dépendances minimales** — Seulement la stdlib Go + `golang.org/x/sys/unix`
- **Code source propre** — Bien documenté, facile à suivre
- **Isolation réelle** — Pas un jouet ; utilise les mêmes primitives que Docker

---

## Technologie des conteneurs : Orchestration, pas invention (Quelques opinions personnelles)

La technologie des conteneurs est essentiellement un assemblage astucieux de plusieurs fonctionnalités préexistantes du noyau Linux :

| Capacité | Technologie Linux sous-jacente |
|----------|-------------------------------|
| Isolation des processus | **namespaces** (PID, réseau, mount, user, etc.) |
| Limitation des ressources | **cgroups** (control groups) |
| Système de fichiers en couches | **OverlayFS / AUFS** |
| Isolation du répertoire racine | **chroot / pivot_root** |

Toutes ces technologies existaient bien avant l'arrivée de Docker ou de tout autre environnement d'exécution de conteneurs moderne. Vous pouvez même assembler manuellement ces primitives pour créer un "conteneur du pauvre" :

```bash
# Créer un namespace isolé et exécuter bash dedans
unshare --mount --uts --ipc --net --pid --fork bash
```

### La vraie valeur des outils de conteneurisation

L'innovation ne réside pas dans la technologie elle-même — elle réside dans **l'abstraction et l'expérience développeur (DX)** :

- Encapsuler des APIs noyau complexes dans des commandes simples comme `docker run` ou `podman run`
- Définir un format d'image standard et une syntaxe de build déclarative (Dockerfile, spécification OCI)
- Construire une infrastructure d'écosystème comme les registres (Docker Hub, GitHub Container Registry)

C'est similaire à la façon dont Git n'a pas inventé le "contrôle de version" en tant que concept, mais a élégamment combiné les snapshots, les DAGs et le stockage adressable par contenu en quelque chose qui *fonctionne* tout simplement.

> **L'insight :** La technologie des conteneurs est de l'orchestration, pas une invention. Le gros du travail est fait par le noyau Linux.

C'est aussi pourquoi les conteneurs ne fonctionnent nativement que sur Linux — sur macOS et Windows, les environnements d'exécution de conteneurs lancent en fait une VM Linux cachée en coulisses.

---

## Fonctionnalités

| Catégorie | Fonctionnalités |
|-----------|-----------------|
| **Namespaces** | UTS, PID, IPC, Mount, User, Network (les 6 namespaces Linux) |
| **Système de fichiers** | `pivot_root`, overlayfs (COW), montages de volumes, `/proc`, `/sys`, `/dev` |
| **Réseau** | Bridge (`minicontainer0`), paires veth, IPAM, NAT, publication de ports (`-p`) |
| **Limites de ressources** | Cgroups v2 : mémoire (`--memory`), CPU (`--cpus`), pids (`--pids-limit`) |
| **Images** | Pull depuis Docker Hub, import de tarballs, couches adressables par contenu |
| **Cycle de vie** | IDs de conteneurs, persistance d'état, `ps`, `stop`, `rm`, `logs`, `exec`, `inspect` |
| **Terminal** | Allocation PTY (`-it`), transfert de signaux |
| **Modes** | Interactif, non-interactif, détaché (`-d`) |

### Commandes CLI

```
Commandes conteneur :
  run [flags] <image|--rootfs> <cmd>    Créer et exécuter un conteneur
  exec <container> <command>            Exécuter une commande dans un conteneur
  stop <container>                      Arrêter un conteneur en cours
  rm <container|--all>                  Supprimer un conteneur arrêté
  ps [-a]                               Lister les conteneurs
  logs <container>                      Récupérer les logs d'un conteneur
  inspect <container>                   Afficher les informations détaillées

Commandes image :
  images                                Lister les images locales
  pull <image>                          Récupérer une image depuis un registre
  import <tarball> <name[:tag]>         Importer un tarball comme image
  rmi <image>                           Supprimer une image

Autres commandes :
  prune                                 Nettoyer les répertoires overlay obsolètes
  version                               Afficher les informations de version

Exécutez 'minicontainer help <commande>' pour plus d'informations sur une commande.
```

### Options de run

| Option | Description |
|--------|-------------|
| `--rootfs PATH` | Système de fichiers racine du conteneur (optionnel si utilisation d'une image) |
| `--name NAME` | Nom du conteneur |
| `--hostname NAME` | Nom d'hôte du conteneur |
| `-d` | Mode détaché (arrière-plan) |
| `-i` | Interactif (garder stdin ouvert) |
| `-t` | Allouer un pseudo-TTY |
| `-e KEY=VAL` | Définir une variable d'environnement |
| `-v HOST:CONTAINER[:ro]` | Montage bind de volume |
| `--memory SIZE` | Limite mémoire (ex: `256m`, `1g`) |
| `--cpus N` | Limite CPU (ex: `0.5`, `2`) |
| `--pids-limit N` | Nombre maximum de processus |
| `-p HOST:CONTAINER` | Publier un port du conteneur sur l'hôte |

---

## Démarrage rapide

### 1. Compilation

```bash
make build
```

### 2. Obtenir un rootfs

```bash
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz
mkdir -p /tmp/alpine-rootfs
tar -xzf alpine-minirootfs-3.19.0-x86_64.tar.gz -C /tmp/alpine-rootfs
```

### 3. Exécuter un conteneur

```bash
# Shell interactif
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs /bin/sh

# Exécuter une commande
sudo ./minicontainer run --rootfs /tmp/alpine-rootfs /bin/echo "Bonjour depuis le conteneur !"

# Mode détaché
sudo ./minicontainer run -d --rootfs /tmp/alpine-rootfs /bin/sleep 60
sudo ./minicontainer ps
sudo ./minicontainer stop <id>

# Avec limites de ressources
sudo ./minicontainer run -it --memory 256m --cpus 0.5 --pids-limit 50 \
    --rootfs /tmp/alpine-rootfs /bin/sh
```

### 4. Récupérer depuis Docker Hub (recommandé)

```bash
# Récupérer une image
sudo ./minicontainer pull alpine

# Lister les images
sudo ./minicontainer images

# Exécuter depuis l'image récupérée
sudo ./minicontainer run -it alpine /bin/sh

# Supprimer l'image
sudo ./minicontainer rmi alpine
```

### 5. Importer un tarball local (alternative)

```bash
# Importer un tarball comme image
sudo ./minicontainer import alpine-minirootfs-3.19.0-x86_64.tar.gz alpine:3.19

# Exécuter depuis l'image importée
sudo ./minicontainer run -it alpine:3.19 /bin/sh
```

---

## Prérequis

- Noyau **Linux** 4.x+ (cgroups v2 recommandé)
- **Go** 1.24+
- **Accès root** (sudo) pour les opérations de conteneurs

---

## Développement

```bash
make build      # Compiler le binaire
make check      # Exécuter fmt, vet, build
make test       # Exécuter les tests (nécessite root)
make clean      # Nettoyer les artefacts de compilation
```

---

## Auteur

**Junzhe Wang**

- junzhe.hwangfu@gmail.com — rapports de bugs, contributions
- junzhe.wang2002@gmail.com — opportunités d'emploi, collaboration

---

## Licence

Licence MIT — voir [LICENSE](../LICENSE) pour les détails.
