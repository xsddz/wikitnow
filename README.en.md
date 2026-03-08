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
  <a href="README.md">中文</a> | English
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

# 2. Configure credentials (interactive, writes to ~/.wikitnow/config.json)
wikitnow auth setup

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

Credentials are stored in a single file `~/.wikitnow/config.json`, created automatically by `wikitnow auth setup` (permissions `600`):

```json
{
  "default_provider": "feishu",
  "feishu": {
    "app_id": "cli_a1b2c3d4e5f6",
    "app_secret": "your_app_secret_here"
  }
}
```

Environment variables take priority over the config file (ideal for CI/CD):

| Provider | Environment Variables |
|----------|-----------------------|
| Feishu | `WIKITNOW_FEISHU_APP_ID` / `WIKITNOW_FEISHU_APP_SECRET` |

**Priority (high → low):** CLI flags > env vars > `~/.wikitnow/config.json` > built-in defaults

### File Exclusion (`.wikitnow/ignore`)

Syntax is fully compatible with `.gitignore`. Lookup stops at the **first match found** — no merging between levels:

| Priority | Path | Description |
|----------|------|-------------|
| 1 | `<sync-dir>/.wikitnow/ignore` | Project-level, highest priority |
| 2 | `<parent dirs (ascending)>/.wikitnow/ignore` | Supports nested projects |
| 3 | `~/.wikitnow/ignore` | User global config |
| 4 | `/usr/local/etc/wikitnow/ignore` | System default (installed with binary) |

> Hidden files (starting with `.`) are always skipped, regardless of rules.

📖 See [docs/configuration.md](docs/configuration.md) for details.

## 📖 Usage

```bash
# Show help
wikitnow -h

# -- Credentials --------------------------------------------------
# Interactive credential setup (writes to ~/.wikitnow/config.json)
wikitnow auth setup

# Verify credentials are valid
wikitnow auth check

# -- Platform Info ------------------------------------------------
# List all supported platforms
wikitnow provider list

# -- Sync ---------------------------------------------------------
# Safe Preview: show what would be synced (Wiki URL optional)
wikitnow sync <local-path> [wiki-url]

# Execute Sync: create nodes and upload to wiki (Wiki URL required)
wikitnow sync <local-path> <wiki-url> --apply

# Sync text files without code block wrapping
wikitnow sync <local-path> <wiki-url> --apply --code-block=false

# -- Config -------------------------------------------------------
# Show currently active exclusion rules and their source path
wikitnow config show-ignore

# Generate .wikitnow/ignore in the current directory
wikitnow config init-ignore
```

## 📝 License

[MIT License](LICENSE)
