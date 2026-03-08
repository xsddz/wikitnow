<p align="center">
  <img src="docs/logo.svg" alt="wikitnow" width="120">
</p>

<h1 align="center">wikitnow</h1>

<p align="center">
  <strong>Wiki It Now 🚀</strong> — A lightweight, high-performance CLI tool that instantly transforms your local Markdown notes and directory structure into a structured cloud knowledge base.
</p>

<p align="center">
  <a href="https://github.com/xsddz/wikitnow/releases"><img src="https://img.shields.io/github/v/release/xsddz/wikitnow?style=flat-square" alt="Release"></a>
  <a href="https://github.com/xsddz/wikitnow/blob/main/LICENSE"><img src="https://img.shields.io/github/license/xsddz/wikitnow?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/xsddz/wikitnow"><img src="https://goreportcard.com/badge/github.com/xsddz/wikitnow?style=flat-square" alt="Go Report Card"></a>
</p>

<p align="center">
  English | <a href="README.zh-CN.md">中文</a>
</p>

---

## ✨ Core Philosophy

Build your local repository of knowledge and let `wikitnow` effortlessly transform it into a structured, shareable cloud Knowledge Base.

- 🏗️ **Wiki Builder**: Recursively maps your entire local directory tree into a perfectly mirroring wiki hierarchy in seconds.
- 🖥️ **Cross-platform**: Native binaries for macOS, Linux, and Windows.
- 📂 **Smart Traversal**: Automatically detects whether you are syncing a single file or recursively building an entire directory tree.
- 🛡️ **Git-style Ignore**: Supports `.wikitnow/ignore` with standard `.gitignore` glob matching.
- ⚡ **Zero-dependency**: Written in Go, compiles to a single, standalone binary.
- 🔒 **Safe By Default**: By default, `sync` acts as a read-only preview. Use `--apply` to actually execute changes.
- 🔌 **Extensible**: Built on a Provider interface — currently supports Feishu (Lark), with architecture ready for other platforms.

## 🚀 Quick Start

```bash
# 1. Install (one-line for macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/xsddz/wikitnow/main/scripts/install.sh | bash

# 2. Configure Credentials (using Feishu as example)
export FEISHU_APP_ID="cli_a1b2..."
export FEISHU_APP_SECRET="your_app_secret_here"

# 3. Safe Preview (Wiki URL is optional; when provided, also shows target node info)
wikitnow sync ./docs/guide.md "https://my.feishu.cn/wiki/wikcnXyz123..."

# 4. Recursively synchronize a directory and actually apply changes
wikitnow sync ./src "https://my.feishu.cn/wiki/wikcnXyz123..." --apply
```

## 📋 Requirements

- Go 1.21+ (for building from source)
- Network access to the target platform APIs

## 📦 Installation

### From Source

```bash
git clone https://github.com/xsddz/wikitnow.git
cd wikitnow
make build
sudo mv bin/wikitnow /usr/local/bin/
```

### Cross-compilation

```bash
make build-all

# Alternatively, manual cross-compilation:
# GOOS=linux GOARCH=amd64 go build -o bin/wikitnow-linux-amd64 ./cmd/wikitnow
```

## 🛠️ Development

```bash
go test ./...        # Run tests
make build           # Build binary
make build-all       # Cross-platform build
```

## ⚙️ Configuration

### Authentication

The CLI requires your Feishu `APP_ID` and `APP_SECRET` to interact with Feishu APIs.

**Option A: Environment Variables (Recommended for CI/CD)**

```bash
export FEISHU_APP_ID="cli_xxx"
export FEISHU_APP_SECRET="xxx"
```

**Option B: Global Credentials File (Recommended for Local Dev)**

Create `~/.wikitnow/credentials.json`:
```json
{
  "app_id": "cli_xxx",
  "app_secret": "xxx"
}
```

### File Exclusion (`.wikitnow/ignore`)

Create a `.wikitnow/ignore` file to specify files and directories to skip. The syntax is identical to `.gitignore`.

**Lookup priority (first match wins):**

```
<sync-dir>/.wikitnow/ignore    ← project-level, highest priority
parent dirs (ascending)/.wikitnow/ignore
~/.wikitnow/ignore               ← user global config
/usr/local/etc/wikitnow/ignore  ← system default (installed with binary)
```

> Hidden files (starting with `.`) are always skipped regardless of config.

📖 See [docs/configuration.md](docs/configuration.md) for details.

## 📖 Usage

```bash
# Show help
wikitnow -h

# Safe Preview: show what would be synced (Wiki URL optional)
wikitnow sync <local-path> [wiki-url]

# Execute Sync: create nodes and upload to wiki (Wiki URL required)
wikitnow sync <local-path> <wiki-url> --apply

# Sync text files without code block wrapping
wikitnow sync <local-path> <wiki-url> --apply --code-block=false
```

## 📝 License

[MIT License](LICENSE)
