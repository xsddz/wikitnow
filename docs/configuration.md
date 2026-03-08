# 配置说明

`wikitnow` 所有配置均基于普通文件，无需学习专有格式。

---

## 凭证配置

凭证统一存储于 `~/.wikitnow/config.json`，由 `wikitnow auth setup` 自动创建（文件权限 `600`）：

```json
{
  "default_provider": "feishu",
  "feishu": {
    "app_id": "cli_a1b2c3d4e5f6",
    "app_secret": "your_app_secret_here"
  }
}
```

也可通过环境变量提供凭证（适合 CI/CD），优先级高于配置文件：

| Provider | 环境变量 |
|----------|---------|
| 飞书 | `WIKITNOW_FEISHU_APP_ID` / `WIKITNOW_FEISHU_APP_SECRET` |

**读取优先级（高 → 低）**：命令行参数 > 环境变量 > `~/.wikitnow/config.json` > 内置默认值

### 管理凭证

```bash
# 交互式配置，自动写入 ~/.wikitnow/config.json
wikitnow auth setup

# 验证当前凭证是否有效
wikitnow auth check
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

> 找到任何一个配置文件即停止，该文件**完全**接管排除逻辑，不与其他层叠加。
>
> 隐藏文件（以 `.` 开头，如 `.git`、`.DS_Store`）始终被跳过，不受配置文件影响。

### 示例配置

```bash
# <project>/.wikitnow/ignore

# 构建产物
dist/
build/
*.pyc

# 依赖目录
node_modules/

# 自定义：只要不写 *.pdf，PDF 文件就会被同步
```

### 各层级关系说明

| 场景 | 行为 |
|------|------|
| 无任何配置文件 | 所有文件均同步（隐藏文件除外） |
| 有项目级配置 | 仅使用该文件，系统默认**不生效** |
| 有用户全局配置 | 仅使用该文件，系统默认**不生效** |
| 仅有系统默认 | 使用 `/usr/local/etc/wikitnow/ignore`（安装时自动部署） |

> 若想在项目配置中继续沿用系统默认的大多数规则，可将 `/usr/local/etc/wikitnow/ignore` 的内容复制到项目的 `.wikitnow/ignore` 中，再按需修改。

### 查看当前生效规则

```bash
# 显示当前目录实际生效的排除规则及其来源路径
wikitnow config show-ignore
```

### 初始化项目级规则

```bash
# 在当前目录生成 .wikitnow/ignore（内容来自系统默认，可按需修改）
wikitnow config init-ignore

# 强制覆盖已有文件
wikitnow config init-ignore --force
```

---

## 配置文件一览

| 文件 | 作用 | 管理方式 |
|------|------|----------|
| `~/.wikitnow/config.json` | 凭证与全局偏好 | `wikitnow auth setup` 或直接编辑 |
| `<project>/.wikitnow/ignore` | 项目级排除规则 | `wikitnow config init-ignore` 或直接编辑 |
| `~/.wikitnow/ignore` | 用户全局排除规则 | 直接编辑 |
| `/usr/local/etc/wikitnow/ignore` | 系统默认排除规则 | 随命令安装，可覆盖 |
