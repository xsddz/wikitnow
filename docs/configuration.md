# 配置文件说明

`wikitnow` 的所有配置均基于文件系统中的普通文本文件，无需学习专有格式。

---

## 凭证配置（身份认证）

工具需要飞书 `APP_ID` 和 `APP_SECRET` 才能调用 API。提供两种方式，**互为备选**：

| 优先级 | 来源 | 适用场景 |
|--------|------|---------|
| 高 | `FEISHU_APP_ID` / `FEISHU_APP_SECRET` 环境变量 | CI/CD、脚本自动化 |
| 低 | `~/.wikitnow/credentials.json` | 本地开发，避免反复输入 |

**`~/.wikitnow/credentials.json` 格式：**
```json
{
  "app_id": "cli_xxx",
  "app_secret": "xxx"
}
```

---

## 排除规则配置（`.wikitnow/ignore`）

控制哪些文件和目录**不**被同步到知识库。语法与 `.gitignore` 完全兼容。

### 查找优先级（自高向低，找到即停止）

```
<同步目录>/.wikitnow/ignore          ← 项目级，优先级最高
    │  未找到，向上查找父目录
    ↓
<父目录>/.wikitnow/ignore            ← 支持多级嵌套项目
    │  未找到，继续向上，直到根目录
    ↓
~/.wikitnow/ignore                   ← 用户全局个性化配置
    │  未找到
    ↓
/usr/local/etc/wikitnow/ignore       ← 系统级默认（随命令安装）
```

> **规则**：找到任何一个配置文件即停止，该文件**完全**接管排除逻辑，不与其他层叠加。
> 
> **隐藏文件**（以 `.` 开头，如 `.git`, `.DS_Store`）始终被跳过，不受配置文件影响。

### 示例配置

```bash
# <project>/.wikitnow/ignore

# 构建产物
dist/
build/
*.pyc

# 依赖（覆盖系统默认，按需保留或删除）
node_modules/

# 自定义：允许同步 PDF（在系统默认中 PDF 被过滤，这里去掉该规则即可让其同步）
# 提示：只要不写 *.pdf，PDF 文件就会被同步
```

### 自定义与系统默认的关系

| 场景 | 行为 |
|------|------|
| 无任何配置文件 | 打印警告，建议运行 init-ignore 初始化或手动创建 |
| 有项目级配置 | 仅使用该文件，系统默认**不生效** |
| 有用户全局配置 | 仅使用该文件，系统默认**不生效** |
| 仅有系统默认 | 使用 `/usr/local/etc/wikitnow/ignore`（安装时自动部署） |

> 如果您想在项目配置中**继续使用**系统默认的大多数规则，可以将 `/usr/local/etc/wikitnow/ignore` 的内容复制到项目的 `.wikitnow/ignore` 中，再按需修改。

### 查看当前生效规则

运行以下命令可直接查看**在当前目录下实际生效**的排除规则（按优先级链查找后找到即返回，并显示来源路径）：

```bash
wikitnow config show-ignore
```

### 初始化项目级规则

如果当前目录还没有配置文件，可一键生成一份（内容来自系统默认，可按需修改）：

```bash
wikitnow config init-ignore          # 在当前目录创建 .wikitnow/ignore
wikitnow config init-ignore --force  # 强制覆盖已有文件
```

---

## 配置文件位置一览

| 文件 | 作用 |
|------|------|
| `<project>/.wikitnow/ignore` | 项目级排除规则 |
| `~/.wikitnow/ignore` | 用户全局排除规则 |
| `~/.wikitnow/credentials.json` | 全局凭证配置 |
| `/usr/local/etc/wikitnow/ignore` | 系统级默认排除规则（随命令安装） |
