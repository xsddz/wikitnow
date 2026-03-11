# 我又写了一个小工具，让 OpenClaw 小龙虾做你的知识管理员


自从用上 OpenClaw，和 AI 聊天的频率高了很多。

聊技术方案，AI 帮你梳理思路；聊某个领域的问题，AI 帮你写一篇系统的总结；甚至随手聊几句，它就能整理出一篇条理清晰的文章。

内容质量挺高的，放着可惜。

但这些内容都在本地——一个个 Markdown 文件，散落在各个目录里。想发到飞书知识库和同事分享？打开浏览器，新建页面，手动复制粘贴，调格式，再一层层把目录结构在云端重新搭一遍。

每次都是这么折腾，折腾几次就懒得发了。

于是我又写了一个小工具，然后给它接入了 OpenClaw，让 OpenClaw 小龙虾真正做你的知识管理员——不只是帮你写，还帮你发出去。


## wikitnow：现在就把它 Wiki 化

wikitnow（Wiki it now）是一个跨平台的文档发布工具，支持 macOS、Linux 和 Windows。

它做的事情很简单：

- **极速建库** —— 一键将本地目录树 1:1 映射为云端知识库层级节点
- **智能分析** —— 自动识别单文件发布或目录树递归构建
- **Git 风格排除** —— 支持 `.wikitnow/ignore`，语法完全兼容 `.gitignore`
- **默认安全** —— 不加 `--target` 就是只读预览，防手滑

安装也很简单（以 Linux/macOS 为例）：

```bash
# 一键安装
curl -fsSL https://raw.githubusercontent.com/xsddz/wikitnow/main/scripts/install.sh | bash

# 配置飞书凭证（交互式引导，自动写入 ~/.wikitnow/config.json）
wikitnow auth setup
```

配置好之后，先来一次预览，看看会同步哪些文件：

```bash
wikitnow sync ./docs
```

```
📂 docs
├── 📄 README.md
├── 📂 guide
│   ├── 📄 quickstart.md
│   ├── 📄 configuration.md
│   └── 📄 faq.md
└── 📂 api
    ├── 📄 overview.md
    └── 📄 reference.md

[预览模式] 共发现 6 个文件，3 个目录节点。
加上 --target 参数以正式推送。
```

确认结构没问题，加上目标 Wiki 地址，正式推送：

```bash
wikitnow sync ./docs --target "https://my.feishu.cn/wiki/wikcnXyz123..."
```

本地目录有多深，云端就建多深。不用手动建节点，不用一张张复制粘贴。

有些文件不想发出去？在目录里放一个 `.wikitnow/ignore` 文件，语法和 `.gitignore` 完全一样：

```
# 草稿目录
drafts/

# 临时文件
*.tmp
*_draft.md
```

## my-skills/wikitnow：你的知识管理员

wikitnow 解决了发布的问题，但操作还是得你来。

我想要的是：和 OpenClaw 聊完，它整理好内容存到本地，然后直接告诉它"发出去"——剩下的它自己搞定。

于是我给 OpenClaw 写了一个技能：**my-skills/wikitnow**。

有了这个技能，OpenClaw 真正补上了知识管理的最后一环：

- **理解发布意图** —— 说"发布到飞书"、"同步 docs 目录"、"推送 Markdown"，它就知道该干什么
- **强制预览确认** —— 无论如何先跑预览，把文件树展示出来确认，再执行正式推送，杜绝误操作
- **全流程自主** —— 凭证检查、排除规则、多路径推送，它自己来处理

安装也不复杂：

```bash
# 安装技能
openclaw skills install github:xsddz/my-skills/wikitnow
```

装好之后，不需要记任何命令，直接跟它说：

```
我：把 ./docs 同步到飞书知识库
```

它会先问你目标地址，然后跑预览，把要发布的文件树展示出来：

```
OpenClaw：请提供目标知识库地址（飞书 Wiki 链接）。

我：https://my.feishu.cn/wiki/wikcnXyz123...

OpenClaw：好的，以下内容将同步到知识库，请确认：

📂 docs
├── 📄 README.md
├── 📂 guide
│   ├── 📄 quickstart.md
│   └── 📄 faq.md
└── 📂 api
    └── 📄 overview.md

共 5 个文件，2 个目录节点。
目标地址：https://my.feishu.cn/wiki/wikcnXyz123...

确认推送？

我：确认

OpenClaw：推送完成。✅
```

你说"确认"，它才真正执行写入。

凭证没配好？它帮你检查。
想把某个目录加入排除规则，不知道怎么写？它帮你生成。

从此，和 OpenClaw 的每次对话，都可以有一个完整的归宿：整理、存档、发布、分享。


## 最后

两个工具都开源了：

- **wikitnow**：https://github.com/xsddz/wikitnow
- **my-skills/wikitnow**：https://github.com/xsddz/my-skills/tree/main/skills/wikitnow

以前和 AI 聊完，内容就散了。现在，它帮你写，也帮你发出去。
