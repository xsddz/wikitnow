<p align="center">
  <img src="docs/logo.svg" alt="wikitnow" width="120">
</p>

<h1 align="center">wikitnow</h1>

<p align="center">
  <strong>Wiki It Now 🚀</strong> — Publish your local Markdown notes to a cloud knowledge base
</p>

<p align="center">
  <a href="https://github.com/xsddz/wikitnow/releases"><img src="https://img.shields.io/github/v/release/xsddz/wikitnow?style=flat-square" alt="Release"></a>
  <a href="https://github.com/xsddz/wikitnow/blob/main/LICENSE"><img src="https://img.shields.io/github/license/xsddz/wikitnow?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/xsddz/wikitnow"><img src="https://goreportcard.com/badge/github.com/xsddz/wikitnow?style=flat-square" alt="Go Report Card"></a>
</p>

<p align="center">
  English | <a href="README.md">中文</a>
</p>

---

## ✨ Features

- 🏗️ **Wiki Builder**: Maps your local directory tree 1:1 into a cloud wiki hierarchy
- 🖥️ **Cross-platform**: Native binaries for macOS, Linux, and Windows
- 📂 **Smart Traversal**: Handles single files or full directory tree recursion automatically
- 🛡️ **Git-style Ignore**: `.wikitnow/ignore` with standard `.gitignore` glob syntax
- ⚡ **Zero-dependency**: Single standalone binary, no runtime required
- 🔒 **Safe By Default**: Dry-run preview by default; `--apply` to write
- 🔌 **Extensible**: Provider interface — currently supports Feishu (Lark)

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

| Priority | Source | Best For |
|----------|--------|----------|
| High | `FEISHU_APP_ID` / `FEISHU_APP_SECRET` env vars | CI/CD, automation |
| Low | `~/.wikitnow/credentials.json` | Local development |

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
