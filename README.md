<p align="center">
  <img src="docs/logo.svg" alt="wikitnow" width="120">
</p>

<h1 align="center">wikitnow</h1>

<p align="center">
  <strong>现在就把它 Wiki 化！🚀</strong> — 本地 Markdown 一键发布到云端知识库
</p>

<p align="center">
  <a href="https://github.com/xsddz/wikitnow/releases"><img src="https://img.shields.io/github/v/release/xsddz/wikitnow?style=flat-square" alt="Release"></a>
  <a href="https://github.com/xsddz/wikitnow/blob/main/LICENSE"><img src="https://img.shields.io/github/license/xsddz/wikitnow?style=flat-square" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/xsddz/wikitnow"><img src="https://goreportcard.com/badge/github.com/xsddz/wikitnow?style=flat-square" alt="Go Report Card"></a>
</p>

<p align="center">
  中文 | <a href="README.en.md">English</a>
</p>

---

## ✨ 特性

- 🏗️ **极速建库**：一键将本地目录树 1:1 映射为云端知识库层级节点
- 🖥️ **跨平台**：macOS、Linux、Windows 原生二进制
- 📂 **智能分析**：自动识别单文件发布或目录树递归构建
- 🛡️ **Git 风格排除**：支持 `.wikitnow/ignore`，语法完全兼容 `.gitignore`
- ⚡ **开箱即用**：Go 编译，单一无依赖二进制
- 🔒 **默认安全**：默认只读预览，`--apply` 才触发实际写操作
- 🔌 **可扩展**：Provider 接口设计，当前支持飞书（Lark）

## 🚀 快速开始

```bash
# 1. 一键安装 (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/xsddz/wikitnow/main/scripts/install.sh | bash

# 2. 配置平台凭证（交互式，自动写入 ~/.wikitnow/config.json）
wikitnow auth setup

# 3. 安全预览（不指定 --target 时默认进入只读预览模式）
wikitnow sync ./docs/guide.md

# 4. 确认无误，递归将整个目录正式推送到知识库
wikitnow sync ./src --target "https://my.feishu.cn/wiki/wikcnXyz123..."
```

## 📋 运行要求

- Go 1.21+ (用于源码编译)
- 能够正常访问目标平台 API 的网络环境

## 📦 安装方式

### 源码编译

```bash
git clone https://github.com/xsddz/wikitnow.git
cd wikitnow
make build
sudo mv bin/wikitnow /usr/local/bin/
```

### 交叉编译

```bash
make build-all

# 或手动交叉编译:
# GOOS=linux GOARCH=amd64 go build -o bin/wikitnow-linux-amd64 ./cmd/wikitnow
```

## 🛠️ 本地开发

```bash
go test ./...        # 运行单元测试
make build           # 编译本地二进制文件
make build-all       # 一键跨平台交叉编译
```

## ⚙️ 配置说明

### 凭证配置

凭证统一存储于 `~/.wikitnow/config.json`，由 `wikitnow auth setup` 自动创建（权限 `600`）：

```json
{
  "default_provider": "feishu",
  "feishu": {
    "app_id": "cli_a1b2c3d4e5f6",
    "app_secret": "your_app_secret_here"
  }
}
```

环境变量优先级高于配置文件（适合 CI/CD 场景）：

| Provider | 环境变量 |
|----------|----------|
| 飞书 | `WIKITNOW_FEISHU_APP_ID` / `WIKITNOW_FEISHU_APP_SECRET` |

**读取优先级（高 → 低）**：命令行参数 > 环境变量 > `~/.wikitnow/config.json` > 内置默认值

### 排除规则配置（`.wikitnow/ignore`）

语法与 `.gitignore` 完全兼容。按以下优先级查找，**找到即停，不叠加**：

| 优先级 | 路径 | 说明 |
|--------|------|------|
| 1 | `<同步目录>/.wikitnow/ignore` | 项目级，优先级最高 |
| 2 | `<父目录（逐级向上）>/.wikitnow/ignore` | 支持多级嵌套项目 |
| 3 | `~/.wikitnow/ignore` | 用户全局配置 |
| 4 | `/usr/local/etc/wikitnow/ignore` | 系统默认（随命令安装） |

> 隐藏文件（以 `.` 开头）始终被跳过，不受规则文件影响。

📖 详细说明见 [docs/configuration.md](docs/configuration.md)

## 📖 使用方法

```bash
# 显示帮助信息
wikitnow -h

# ── 凭证管理 ────────────────────────────────────────────────
# 交互式配置凭证，写入 ~/.wikitnow/config.json
wikitnow auth setup

# 验证凭证是否有效
wikitnow auth check

# ── 平台查看 ────────────────────────────────────────────────
# 列出所有支持的知识库平台
wikitnow provider list

# ── 同步发布 ────────────────────────────────────────────────
# 安全预览：展示将要同步的文件树结构（不填 --target 则进入只读预览模式）
wikitnow sync <本地路径>

# 多路径预览
wikitnow sync <本地路径1> <本地路径2>

# 正式执行：建立节点架构并将本地数据发布到知识库（需指定 --target）
wikitnow sync <本地目录> --target <Wiki URL>

# 多路径推送
wikitnow sync <本地目录1> <本地目录2> --target <Wiki URL>

# 纯文本上传：对于文本文件，不使用代码块包裹内容直接排版
wikitnow sync <本地目录> --target <Wiki URL> --code-block=false

# ── 配置管理 ────────────────────────────────────────────────
# 查看当前生效的排除规则及来源路径
wikitnow config show-ignore

# 在当前目录生成 .wikitnow/ignore（内容来自系统默认）
wikitnow config init-ignore
```

## 📝 开源协议

[MIT License](LICENSE)
