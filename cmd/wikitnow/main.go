package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xsddz/wikitnow/internal/auth"
	"github.com/xsddz/wikitnow/internal/configs"
	"github.com/xsddz/wikitnow/internal/feishu"
	feishuprov "github.com/xsddz/wikitnow/internal/provider/feishu"
	"github.com/xsddz/wikitnow/internal/sync"
)

func main() {
	// 自定义 Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s <命令> [参数] [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "命令:\n")
		fmt.Fprintf(os.Stderr, "  sync    将本地文件或目录同步到知识库\n")
		fmt.Fprintf(os.Stderr, "  config  查看工具配置信息\n\n")
		fmt.Fprintf(os.Stderr, "示例:\n")
		fmt.Fprintf(os.Stderr, "  %s sync ./docs                                        # 仅安全预览将要同步的本地结构\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sync ./docs https://your-wiki-url                  # 安全预览即将要同步的节点树结构（含目标提取）\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sync ./docs https://your-wiki-url --apply          # 确认无误，执行真实的推送操作\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s config show-ignore                                 # 查看系统默认排除规则\n\n", os.Args[0])
	}

	flag.Parse()

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	switch args[0] {
	case "config":
		runConfig(args[1:])
		return
	case "sync":
		// 继续执行下面的 sync 逻辑
	default:
		fmt.Fprintf(os.Stderr, "❌ 未知命令: %s\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}

	// 自定义子命令参数解析器
	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
	var apply bool
	var useCodeBlock bool
	var debug bool
	syncCmd.BoolVar(&apply, "apply", false, "确认执行同步，向目标平台发起真实的创建和写入请求（默认行为是只读的安全预览）")
	syncCmd.BoolVar(&useCodeBlock, "code-block", true, "对于文本文件，是否将其内容使用代码块包裹插入（默认 true）。设为 false 则插入纯文本")
	syncCmd.BoolVar(&debug, "debug", false, "调试模式：将每次 HTTP 请求的方法和 URL 打印到 stderr")

	// Go 的 flag.FlagSet 遇到第一个非 flag 参数即停止解析，
	// 为支持 flags 写在位置参数之后（如 sync ./docs URL --apply），
	// 预先将所有 flag 参数（-- 或 - 开头）重排到位置参数前面。
	var flagArgs, posArgs []string
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
		} else {
			posArgs = append(posArgs, a)
		}
	}
	syncCmd.Parse(append(flagArgs, posArgs...))
	syncArgs := syncCmd.Args()

	if len(syncArgs) < 1 {
		fmt.Println("❌ 错误: 必须提供 <本地路径>")
		flag.Usage()
		os.Exit(1)
	}

	localPath := syncArgs[0]
	var wikiURL string
	if len(syncArgs) >= 2 {
		wikiURL = syncArgs[1]
	}

	// 真实上传时强制要求提供 wikiURL
	if apply && wikiURL == "" {
		fmt.Println("❌ 错误: 使用 --apply 进行真实同步时，必须提供目标 <Wiki URL>")
		os.Exit(1)
	}

	// 1. 初始化 Auth
	authManager, err := auth.NewTokenManager()
	if err != nil && apply {
		fmt.Printf("❌ 认证错误: %v\n", err)
		os.Exit(1)
	} else if err != nil && !apply {
		fmt.Printf("⚠️ 认证提醒: %v (仅做本地目录测试，不影响树状结构预览)\n", err)
	}

	// 2. 初始化 API 客户端
	client := feishu.NewClient(authManager, debug)

	// 实例化 Provider（当前默认只使用飞书）
	prov := feishuprov.NewProvider(client)

	var parentNodeToken string
	var spaceID string

	// 只有当提供了 wikiURL (包含 Apply 或单预览 URL 提取时) 才会进行解析
	if wikiURL != "" {
		// 3. 利用 Provider 提取根节点信息
		extractedSpace, extractedParent, err := prov.ExtractRoot(wikiURL)
		if err != nil {
			fmt.Printf("❌ URL 解析与提取失败: %v\n", err)
			os.Exit(1)
		}
		spaceID = extractedSpace
		parentNodeToken = extractedParent

		fmt.Printf("🔗 提取到父节点 Token: %s\n", parentNodeToken)
		fmt.Printf("📁 目标知识库 Space ID: %s\n", spaceID)
	}

	// 5. 初始化并启动同步引擎：在未携带 --apply 时，默认状态下处于 Dry-Run 安全模式
	engine := sync.NewEngine(prov, localPath, !apply, useCodeBlock)
	err = engine.Sync(localPath, spaceID, parentNodeToken)
	if err != nil {
		fmt.Printf("❌ 同步中断: %v\n", err)
		os.Exit(1)
	}

	if !apply {
		fmt.Println("\n✅ 预览结束。如需真实写入，请提供 Wiki URL 并追加 --apply 参数。")
	} else {
		fmt.Println("\n✅ 同步完成")
	}
}

// runConfig 处理 config 子命令
func runConfig(args []string) {
	const systemIgnorePath = "/usr/local/etc/wikitnow/ignore"

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "用法: wikitnow config <子命令>\n\n")
		fmt.Fprintf(os.Stderr, "可用子命令:\n")
		fmt.Fprintf(os.Stderr, "  show-ignore    查看当前目录下实际生效的排除规则\n")
		fmt.Fprintf(os.Stderr, "  init-ignore    在当前目录生成 .wikitnow/ignore（内容来自系统默认）\n")
		os.Exit(1)
	}

	switch args[0] {
	case "show-ignore":
		activePath := findActiveIgnoreFile(systemIgnorePath)
		if activePath == "" {
			fmt.Fprintf(os.Stderr, "⚠️  未找到任何 ignore 配置文件\n")
			fmt.Fprintf(os.Stderr, "   已查找: <当前目录及父目录>/.wikitnow/ignore、~/.wikitnow/ignore、%s\n", systemIgnorePath)
			fmt.Fprintf(os.Stderr, "   提示: 运行 wikitnow config init-ignore 可在当前目录生成一份。\n")
			os.Exit(1)
		}
		f, err := os.Open(activePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 无法读取规则文件 %s: %v\n", activePath, err)
			os.Exit(1)
		}
		defer f.Close()
		fmt.Printf("# 当前生效的排除规则 (%s)\n\n", activePath)
		io.Copy(os.Stdout, f)

	case "init-ignore":
		// 解析 --force 选项
		initCmd := flag.NewFlagSet("init-ignore", flag.ExitOnError)
		force := initCmd.Bool("force", false, "强制覆盖已存在的 .wikitnow/ignore")
		initCmd.Parse(args[1:])

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 无法获取当前目录: %v\n", err)
			os.Exit(1)
		}
		destDir := filepath.Join(cwd, ".wikitnow")
		destFile := filepath.Join(destDir, "ignore")

		// 检查目标文件是否已存在
		if _, err := os.Stat(destFile); err == nil && !*force {
			fmt.Fprintf(os.Stderr, "⚠️  文件已存在: %s\n", destFile)
			fmt.Fprintf(os.Stderr, "   若要覆盖，请追加 --force 参数。\n")
			os.Exit(1)
		}

		// 内容来源：优先系统默认文件，fallback 到二进制内嵌默认内容
		var content []byte
		if data, err := os.ReadFile(systemIgnorePath); err == nil {
			content = data
		} else {
			content = configs.IgnoreContent
		}

		// 创建目标目录和文件
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 无法创建目录 %s: %v\n", destDir, err)
			os.Exit(1)
		}
		if err := os.WriteFile(destFile, content, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 写入失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ 已生成: %s\n", destFile)
		fmt.Printf("   内容来自默认规则，可按需修改。\n")

	default:
		fmt.Fprintf(os.Stderr, "❌ 未知 config 子命令: %s\n", args[0])
		os.Exit(1)
	}
}

// findActiveIgnoreFile 按优先级链查找当前生效的 ignore 文件路径（找到即返回）。
// 查找顺序：CWD 向上逐级→用户全局→系统默认
func findActiveIgnoreFile(systemIgnorePath string) string {
	// 1. 从当前工作目录向上逐级查找
	cwd, err := os.Getwd()
	if err == nil {
		searchPath := cwd
		for {
			candidate := filepath.Join(searchPath, ".wikitnow", "ignore")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
			parent := filepath.Dir(searchPath)
			if parent == searchPath {
				break
			}
			searchPath = parent
		}
	}

	// 2. 用户全局
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".wikitnow", "ignore")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 3. 系统默认
	if _, err := os.Stat(systemIgnorePath); err == nil {
		return systemIgnorePath
	}

	return ""
}
