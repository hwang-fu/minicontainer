# MiniContainer

[English](../README.md) | [Deutsch](./README.de.md) | [Français](./README.fr.md) | [繁體中文](./README.zh.md) | **[日本語](./README.jp.md)**

> **注意：** この README は元々英語で書かれています。内容が分かりにくい場合は、[英語版](../README.md)をご参照ください。

[![CI](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml/badge.svg)](https://github.com/hwang-fu/minicontainer/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org/)

![Demo](./demo.gif)

> 教育目的で Go 言語で書かれた最小限の Linux コンテナランタイム。

MiniContainer は Docker やその他のコンテナシステムを支える基本的なプリミティブを実装しています：**名前空間（namespaces）**、**cgroups**、**overlayfs**、**ネットワーク** — すべてゼロから、最小限の依存関係で構築されています。

---

## なぜ MiniContainer なのか？

- **作りながら学ぶ** — システムコールレベルでコンテナを理解する
- **最小限の依存関係** — Go 標準ライブラリ + `golang.org/x/sys/unix` のみ
- **クリーンなコードベース** — ドキュメントが充実し、追いやすい
- **本物の分離** — おもちゃではない; Docker と同じプリミティブを使用

---

## コンテナ技術：オーケストレーションであり、発明ではない（個人的な見解）

コンテナ技術は本質的に、既存の Linux カーネル機能を巧みに組み合わせたものです：

| 機能 | 基盤となる Linux 技術 |
|------|----------------------|
| プロセス分離 | **名前空間**（PID、ネットワーク、マウント、ユーザーなど） |
| リソース制限 | **cgroups**（コントロールグループ） |
| 階層化ファイルシステム | **OverlayFS / AUFS** |
| ルートディレクトリ分離 | **chroot / pivot_root** |

これらの技術はすべて Docker や現代のコンテナランタイムが登場するずっと前から存在していました。これらのプリミティブを手動で組み合わせて「貧者のコンテナ」を作ることもできます：

```bash
# 分離された名前空間を作成し、その中で bash を実行
unshare --mount --uts --ipc --net --pid --fork bash
```

### コンテナツールの真の価値

革新は技術そのものにあるのではなく — **抽象化と開発者体験（DX）** にあります：

- 複雑なカーネル API を `docker run` や `podman run` のようなシンプルなコマンドにラップする
- 標準的なイメージフォーマットと宣言的なビルド構文を定義する（Dockerfile、OCI 仕様）
- レジストリのようなエコシステムインフラを構築する（Docker Hub、GitHub Container Registry）

これは Git が「バージョン管理」というコンセプトを発明したわけではなく、スナップショット、DAG、コンテンツアドレッサブルストレージをエレガントに組み合わせて、*ただ動く*ものにしたのと似ています。

> **洞察：** コンテナ技術はオーケストレーションであり、発明ではない。重い仕事は Linux カーネルが行っています。

これがコンテナが Linux でしかネイティブに動作しない理由でもあります — macOS や Windows では、コンテナランタイムは実際には裏で隠れた Linux VM を起動しています。

---

## 機能

| カテゴリ | 機能 |
|---------|------|
| **名前空間** | UTS、PID、IPC、Mount、User、Network（Linux の 6 つの名前空間すべて） |
| **ファイルシステム** | `pivot_root`、overlayfs（COW）、ボリュームマウント、`/proc`、`/sys`、`/dev` |
| **ネットワーク** | ブリッジ（`minicontainer0`）、veth ペア、IPAM、NAT、ポート公開（`-p`） |
| **リソース制限** | Cgroups v2：メモリ（`--memory`）、CPU（`--cpus`）、プロセス数（`--pids-limit`） |
| **イメージ** | Docker Hub からのプル、tarball インポート、コンテンツアドレッサブルレイヤー |
| **ライフサイクル** | コンテナ ID、状態の永続化、`ps`、`stop`、`rm`、`logs`、`exec`、`inspect` |
| **ターミナル** | PTY 割り当て（`-it`）、シグナル転送 |
| **モード** | 対話型、非対話型、デタッチ（`-d`） |

### CLI コマンド

```
コンテナコマンド：
  run [flags] <image|--rootfs> <cmd>    コンテナを作成して実行
  exec <container> <command>            実行中のコンテナでコマンドを実行
  stop <container>                      実行中のコンテナを停止
  rm <container|--all>                  停止したコンテナを削除
  ps [-a]                               コンテナ一覧
  logs <container>                      コンテナのログを取得
  inspect <container>                   コンテナの詳細情報を表示

イメージコマンド：
  images                                ローカルイメージ一覧
  pull <image>                          レジストリからイメージをプル
  import <tarball> <name[:tag]>         tarball をイメージとしてインポート
  rmi <image>                           イメージを削除

その他のコマンド：
  prune                                 古い overlay ディレクトリをクリーンアップ
  version                               バージョン情報を表示

'minicontainer help <コマンド>' でコマンドの詳細を表示。
```

### Run オプション

| オプション | 説明 |
|-----------|------|
| `--rootfs PATH` | コンテナのルートファイルシステム（イメージ使用時はオプション） |
| `--name NAME` | コンテナ名 |
| `--hostname NAME` | コンテナのホスト名 |
| `-d` | デタッチモード（バックグラウンド） |
| `-i` | 対話型（stdin を開いたまま） |
| `-t` | 疑似 TTY を割り当て |
| `-e KEY=VAL` | 環境変数を設定 |
| `-v HOST:CONTAINER[:ro]` | ボリュームのバインドマウント |
| `--memory SIZE` | メモリ制限（例：`256m`、`1g`） |
| `--cpus N` | CPU 制限（例：`0.5`、`2`） |
| `--pids-limit N` | 最大プロセス数 |
| `-p HOST:CONTAINER` | コンテナポートをホストに公開 |

---

## クイックスタート

### 1. ビルド

```bash
make build
```

### 2. rootfs を取得

```bash
wget https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.0-x86_64.tar.gz
mkdir -p /tmp/alpine-rootfs
tar -xzf alpine-minirootfs-3.19.0-x86_64.tar.gz -C /tmp/alpine-rootfs
```

### 3. コンテナを実行

```bash
# 対話型シェル
sudo ./minicontainer run -it --rootfs /tmp/alpine-rootfs /bin/sh

# コマンドを実行
sudo ./minicontainer run --rootfs /tmp/alpine-rootfs /bin/echo "コンテナからこんにちは！"

# デタッチモード
sudo ./minicontainer run -d --rootfs /tmp/alpine-rootfs /bin/sleep 60
sudo ./minicontainer ps
sudo ./minicontainer stop <id>

# リソース制限付き
sudo ./minicontainer run -it --memory 256m --cpus 0.5 --pids-limit 50 \
    --rootfs /tmp/alpine-rootfs /bin/sh
```

### 4. Docker Hub からプル（推奨）

```bash
# イメージをプル
sudo ./minicontainer pull alpine

# イメージ一覧
sudo ./minicontainer images

# プルしたイメージから実行
sudo ./minicontainer run -it alpine /bin/sh

# イメージを削除
sudo ./minicontainer rmi alpine
```

### 5. ローカル tarball をインポート（代替）

```bash
# tarball をイメージとしてインポート
sudo ./minicontainer import alpine-minirootfs-3.19.0-x86_64.tar.gz alpine:3.19

# インポートしたイメージから実行
sudo ./minicontainer run -it alpine:3.19 /bin/sh
```

---

## 要件

- **Linux** カーネル 4.x+（cgroups v2 推奨）
- **Go** 1.24+
- **root アクセス**（sudo）コンテナ操作に必要

---

## 開発

```bash
make build      # バイナリをビルド
make check      # fmt、vet、build を実行
make test       # テストを実行（root 必要）
make clean      # ビルド成果物をクリーンアップ
```

---

## 作者

**王浚哲（Junzhe Wang）**

- junzhe.hwangfu@gmail.com — バグ報告、貢献
- junzhe.wang2002@gmail.com — 仕事の機会、コラボレーション

---

## ライセンス

MIT ライセンス — 詳細は [LICENSE](../LICENSE) を参照。
