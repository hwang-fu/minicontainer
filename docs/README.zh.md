# MiniContainer

[English](../README.md) | [Deutsch](./README.de.md) | [Français](./README.fr.md) | **[繁體中文](./README.zh.md)** | [日本語](./README.jp.md)

> **注意：** 本 README 原文為英文撰寫。如有任何不清楚之處，請參閱[英文版本](../README.md)。

[![CI](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml/badge.svg)](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org/)

![Demo](./demo.gif)

> 一個以 Go 語言編寫的極簡 Linux 容器執行環境，專為教育目的設計。

MiniContainer 實現了驅動 Docker 和其他容器系統的核心原語：**命名空間（namespaces）**、**控制群組（cgroups）**、**疊加檔案系統（overlayfs）** 和 **網路** — 全部從零開始建構，僅使用最少的相依套件。

---

## 為何選擇 MiniContainer？

- **從建構中學習** — 在系統呼叫層級理解容器
- **最少相依套件** — 僅需 Go 標準函式庫 + `golang.org/x/sys/unix`
- **乾淨的程式碼** — 文件完善，易於追蹤
- **真正的隔離** — 不是玩具；使用與 Docker 相同的原語

---

## 容器技術：編排，而非發明（個人見解）

容器技術本質上是對多個既存 Linux 核心功能的巧妙整合：

| 能力 | 底層 Linux 技術 |
|------|----------------|
| 行程隔離 | **命名空間**（PID、網路、掛載、使用者等） |
| 資源限制 | **控制群組（cgroups）** |
| 分層檔案系統 | **OverlayFS / AUFS** |
| 根目錄隔離 | **chroot / pivot_root** |

這些技術在 Docker 或任何現代容器執行環境出現之前就已存在。您甚至可以手動組合這些原語來建立一個「窮人的容器」：

```bash
# 建立隔離的命名空間並在其中執行 bash
unshare --mount --uts --ipc --net --pid --fork bash
```

### 容器工具的真正價值

創新不在於技術本身 — 而在於 **抽象化和開發者體驗（DX）**：

- 將複雜的核心 API 封裝成簡單的命令，如 `docker run` 或 `podman run`
- 定義標準的映像格式和宣告式建構語法（Dockerfile、OCI 規範）
- 建立生態系統基礎設施，如登錄中心（Docker Hub、GitHub Container Registry）

這類似於 Git 並未發明「版本控制」這個概念，但優雅地將快照、有向無環圖（DAG）和內容定址儲存結合成一個*就是能用*的東西。

> **洞見：** 容器技術是編排，而非發明。真正繁重的工作由 Linux 核心完成。

這也是為什麼容器只能原生運行在 Linux 上 — 在 macOS 和 Windows 上，容器執行環境實際上是在幕後啟動一個隱藏的 Linux 虛擬機。

---

## 功能特色

| 類別 | 功能 |
|------|------|
| **命名空間** | UTS、PID、IPC、Mount、User、Network（全部 6 種 Linux 命名空間） |
| **檔案系統** | `pivot_root`、overlayfs（COW）、卷宗掛載、`/proc`、`/sys`、`/dev` |
| **網路** | 網橋（`minicontainer0`）、veth 配對、IPAM、NAT、連接埠發布（`-p`） |
| **資源限制** | Cgroups v2：記憶體（`--memory`）、CPU（`--cpus`）、行程數（`--pids-limit`） |
| **映像** | 從 Docker Hub 拉取、匯入 tarball、內容定址層 |
| **生命週期** | 容器 ID、狀態持久化、`ps`、`stop`、`rm` |
| **終端機** | PTY 分配（`-it`）、訊號轉發 |
| **模式** | 互動式、非互動式、分離式（`-d`） |

### CLI 命令

```
minicontainer run [flags] <image|--rootfs> <cmd>  執行容器
minicontainer pull <image>                        從登錄中心拉取映像
minicontainer ps [-a]                             列出容器
minicontainer stop <container>                    停止執行中的容器
minicontainer rm <container|--all>                移除已停止的容器
minicontainer import <tarball> <name[:tag]>       將 tarball 匯入為映像
minicontainer images                              列出本地映像
minicontainer rmi <image>                         移除映像
minicontainer prune                               清理過期的 overlay 目錄
minicontainer version                             顯示版本
```

### Run 選項

| 選項 | 說明 |
|------|------|
| `--rootfs PATH` | 容器根檔案系統（使用映像時為選用） |
| `--name NAME` | 容器名稱 |
| `--hostname NAME` | 容器主機名稱 |
| `-d` | 分離模式（背景執行） |
| `-i` | 互動式（保持 stdin 開啟） |
| `-t` | 分配虛擬終端機 |
| `-e KEY=VAL` | 設定環境變數 |
| `-v HOST:CONTAINER[:ro]` | 綁定掛載卷宗 |
| `--memory SIZE` | 記憶體限制（例如：`256m`、`1g`） |
| `--cpus N` | CPU 限制（例如：`0.5`、`2`） |
| `--pids-limit N` | 最大行程數 |
| `-p HOST:CONTAINER` | 將容器連接埠發布到主機 |

---

## 快速開始

### 1. 建構

```bash
make build
```

### 2. 取得 rootfs

```bash
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz
mkdir -p /tmp/alpine-rootfs
tar -xzf alpine-minirootfs-3.19.0-x86_64.tar.gz -C /tmp/alpine-rootfs
```

### 3. 執行容器

```bash
# 互動式 shell
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs /bin/sh

# 執行命令
sudo ./minicontainer run --rootfs /tmp/alpine-rootfs /bin/echo "來自容器的問候！"

# 分離模式
sudo ./minicontainer run -d --rootfs /tmp/alpine-rootfs /bin/sleep 60
sudo ./minicontainer ps
sudo ./minicontainer stop <id>

# 帶資源限制
sudo ./minicontainer run -it --memory 256m --cpus 0.5 --pids-limit 50 \
    --rootfs /tmp/alpine-rootfs /bin/sh
```

### 4. 從 Docker Hub 拉取（推薦）

```bash
# 拉取映像
sudo ./minicontainer pull alpine

# 列出映像
sudo ./minicontainer images

# 從拉取的映像執行
sudo ./minicontainer run -it alpine /bin/sh

# 移除映像
sudo ./minicontainer rmi alpine
```

### 5. 匯入本地 tarball（替代方案）

```bash
# 將 tarball 匯入為映像
sudo ./minicontainer import alpine-minirootfs-3.19.0-x86_64.tar.gz alpine:3.19

# 從匯入的映像執行
sudo ./minicontainer run -it alpine:3.19 /bin/sh
```

---

## 開發路線圖

- [x] **階段 1**：最小隔離（命名空間、chroot）
- [x] **階段 2**：完整檔案系統（pivot_root、overlayfs、卷宗）
- [x] **階段 3**：容器生命週期（ps、stop、rm、分離模式）
- [x] **階段 4**：資源限制（cgroups v2：記憶體、CPU、行程數）
- [x] **階段 5**：網路（veth、網橋、NAT、連接埠發布）
- [x] **階段 6**：OCI 映像（匯入、映像列表、移除、從映像執行）
- [x] **階段 7**：登錄中心拉取（Docker Hub、多架構支援）
- [ ] **階段 8**：完善（logs、exec、inspect）

---

## 系統需求

- **Linux** 核心 4.x+（建議使用 cgroups v2）
- **Go** 1.24+
- **Root 權限**（sudo）用於容器操作

---

## 開發

```bash
make build      # 建構二進位檔
make check      # 執行 fmt、vet、build
make test       # 執行測試（需要 root）
make clean      # 清理建構產物
```

---

## 作者

**王俊哲（Junzhe Wang）**

- junzhe.hwangfu@gmail.com — 錯誤回報、貢獻
- junzhe.wang2002@gmail.com — 工作機會、合作

---

## 授權條款

MIT 授權條款 — 詳見 [LICENSE](../LICENSE)。
