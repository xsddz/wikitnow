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
  <a href="README.en.md">English</a> | 中文
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

# 2. 配置平台凭证（以飞书为例）
export FEISHU_APP_ID="cli_a1b2c3d4e5f6"
export FEISHU_APP_SECRET="your_app_secret_here"

# 3. 安全预览（Wiki URL 可省略；提供时额外显示目标节点信息）
wikitnow sync ./docs/guide.md "https://my.feishu.cn/wiki/wikcnXyz123..."

# 4. 确认无误，递归将整个目录正式推送到知识库
wikitnow sync ./src "https://my.feishu.cn/wiki/wikcnXyz123..." --apply
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

| 优先级 | 来源 | 适用场景 |
|--------|------|---------|
| 高 | `FEISHU_APP_ID` / `FEISHU_APP_SECRET` 环境变量 | CI/CD、脚本自动化 |
| 低 | `~/.wikitnow/credentials.json` | 本地开发 |

### 排除规则配置（`.wikitnow/ignore`）

语法与 `.gitignore` 完全兼容。**查找优先级（自高向低，找到即停）**：

```
<同步目录>/.wikitnow/ignore    ← 项目级，优先级最高
父目录（逐级向上）/.wikitnow/ignore
~/.wikitnow/ignore               ← 用户全局配置
/usr/local/etc/wikitnow/ignore  ← 系统默认（随命令安装）
```

> 找到即停，该文件完全接管排除逻辑，不与其他层叠加。隐藏文件（以 `.` 开头）始终被跳过。

📖 详细说明见 [docs/configuration.md](docs/configuration.md)

## 📖 使用方法

```bash
# 显示帮助信息
wikitnow -h

# 安全预览：展示将要同步的文件树结构（Wiki URL 可省略；提供时额外显示目标节点信息）
wikitnow sync <本地路径> [Wiki URL]

# 正式执行：建立节点架构并将本地数据覆盖性发布到知识库（Wiki URL 必填）
wikitnow sync <本地目录> <Wiki URL> --apply

# 纯文本上传：对于文本文件，不使用代码块包裹内容直接排版
wikitnow sync <本地目录> <Wiki URL> --apply --code-block=false
```

## 📝 开源协议

[MIT License](LICENSE)
