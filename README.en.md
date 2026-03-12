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
- ↔️ **Two-way Sync**: Local→cloud (sync) and cloud→local (pull)
- 🔌 **Extensible**: Provider interface — currently supports Feishu (Lark)

## 🚀 Quick Start

```bash
# 1. Install (one-line for macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/xsddz/wikitnow/main/scripts/install.sh | bash

# 2. Configure credentials (interactive, writes to ~/.wikitnow/config.json)
wikitnow auth setup

# 3. Safe Preview (omit --target to enter read-only preview mode)
wikitnow sync ./docs/guide.md

# 4. Recursively synchronize a directory and actually push to wiki
wikitnow sync ./src --target "https://my.feishu.cn/wiki/wikcnXyz123..."
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

### Uninstall

```bash
# Remove the binary and ~/.wikitnow config directory
curl -fsSL https://raw.githubusercontent.com/xsddz/wikitnow/main/scripts/install.sh | bash -s uninstall
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
| 3 | `~/.wikitnow/ignore` | User global, final fallback |

> Hidden files (starting with `.`) are always skipped, regardless of rules. If no config file exists at any level, no exclusion rules are applied.

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
# Safe Preview: show what would be synced (omit --target for read-only preview)
wikitnow sync <local-path>

# Multi-path preview
wikitnow sync <local-path1> <local-path2>

# Execute Sync: create nodes and upload to wiki (--target required)
wikitnow sync <local-path> --target <wiki-url>

# Multi-path push
wikitnow sync <local-path1> <local-path2> --target <wiki-url>

# Sync text files without code block wrapping
wikitnow sync <local-path> --target <wiki-url> --code-block=false

# -- Pull ---------------------------------------------------------
# Preview: fetch cloud document content and print to stdout (default)
wikitnow pull <wiki-url|docToken>

# Save to file: download cloud document as local Markdown
wikitnow pull <wiki-url|docToken> --output ./backup.md

# Overwrite existing file
wikitnow pull <wiki-url|docToken> --output ./backup.md --force

# Specify language for @mention elements (zh/en/ja)
wikitnow pull <wiki-url|docToken> --lang en

# Pipeline support
wikitnow pull <docToken> | grep -i "keyword"     # search content
wikitnow pull <docToken> | head -50              # preview first 50 lines
wikitnow pull <docToken> | wc -l                 # count lines

# -- Config -------------------------------------------------------
# Show currently active exclusion rules and their source path
wikitnow config show-ignore

# Generate .wikitnow/ignore in the current directory (content from built-in defaults)
wikitnow config init-ignore

# Force overwrite if file already exists
wikitnow config init-ignore --force

# Write to a custom path (e.g. user global config)
wikitnow config init-ignore --dest ~/.wikitnow/ignore
```

## 📝 License

[MIT License](LICENSE)
