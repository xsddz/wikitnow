package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
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
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s <命令> [参数] [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "命令:\n")
		fmt.Fprintf(os.Stderr, "  sync      将本地文件或目录同步到知识库\n")
		fmt.Fprintf(os.Stderr, "  pull      从知识库获取文档为 Markdown 文件\n")
		fmt.Fprintf(os.Stderr, "  auth      管理平台凭证\n")
		fmt.Fprintf(os.Stderr, "  provider  查看支持的知识库平台\n")
		fmt.Fprintf(os.Stderr, "  config    配置管理\n\n")
		fmt.Fprintf(os.Stderr, "示例:\n")
		fmt.Fprintf(os.Stderr, "  %s sync ./docs                                               # 安全预览本地结构\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sync ./docs --target https://your-wiki-url                # 执行真实推送\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sync ./docs ./guides --target https://your-wiki-url       # 多路径推送\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s auth setup                                                # 交互式配置平台凭证\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s auth check                                                # 验证凭证是否有效\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s provider list                                             # 查看支持的平台\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s pull <URL|docToken>                                       # 预览文档内容\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s pull <URL|docToken> --output ./docs/guide.md              # 保存到文件\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s config show-ignore                                        # 查看当前生效的排除规则\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s config init-ignore                                        # 在当前目录生成 .wikitnow/ignore\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s config init-ignore --dest /usr/local/etc/wikitnow/ignore  # 指定输出路径\n\n", os.Args[0])
	}

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	switch args[0] {
	case "auth":
		runAuth(args[1:])
	case "provider":
		runProvider(args[1:])
	case "config":
		runConfig(args[1:])
	case "sync":
		runSync(args[1:])
	case "pull":
		runPull(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "❌ 未知命令: %s\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}
}

// syncOpts 存放 sync 子命令解析后的全部参数。
type syncOpts struct {
	target       string
	useCodeBlock bool
	debug        bool
	localPaths   []string
}

// parseSyncArgs 纯函数：将 sync 子命令的原始参数切片解析为 syncOpts。
// flags 与位置参数（本地路径）可以任意混排；flag 格式支持：
//
//	--key value
//	--key=value
func parseSyncArgs(args []string) (*syncOpts, error) {
	opts := &syncOpts{useCodeBlock: true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--target" || a == "-target":
			if i+1 >= len(args) {
				return nil, errors.New("--target 需要一个参数值")
			}
			i++
			opts.target = args[i]
		case strings.HasPrefix(a, "--target="):
			opts.target = strings.TrimPrefix(a, "--target=")
		case strings.HasPrefix(a, "-target="):
			opts.target = strings.TrimPrefix(a, "-target=")
		case a == "--code-block=false" || a == "-code-block=false":
			opts.useCodeBlock = false
		case a == "--code-block=true" || a == "--code-block" || a == "-code-block=true" || a == "-code-block":
			opts.useCodeBlock = true
		case a == "--debug" || a == "-debug":
			opts.debug = true
		case strings.HasPrefix(a, "-"):
			return nil, fmt.Errorf("未知参数: %s（用法: wikitnow sync <本地路径> [本地路径...] [--target <Wiki URL>] [--code-block=false]）", a)
		default:
			opts.localPaths = append(opts.localPaths, a)
		}
	}
	if len(opts.localPaths) == 0 {
		return nil, errors.New("必须提供至少一个本地路径")
	}
	return opts, nil
}

// runSync 处理 sync 子命令
func runSync(args []string) {
	opts, err := parseSyncArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 错误: %v\n", err)
		os.Exit(1)
	}

	apply := opts.target != ""
	if apply {
		if err := validateWikiURL(opts.target); err != nil {
			fmt.Fprintf(os.Stderr, "❌ --target 参数无效: %v\n", err)
			os.Exit(1)
		}
	}

	authManager, err := auth.NewTokenManager()
	if err != nil && apply {
		fmt.Printf("❌ 认证错误: %v\n", err)
		os.Exit(1)
	} else if err != nil {
		fmt.Printf("⚠️  %v\n   运行 'wikitnow auth setup' 可快速完成配置\n   (本次仅预览本地文件树)\n", err)
	}

	client := feishu.NewClient(authManager, opts.debug)
	prov := feishuprov.NewProvider(client)

	var parentNodeToken string
	var spaceID string

	if apply {
		extractedSpace, extractedParent, err := prov.ExtractRoot(opts.target)
		if err != nil {
			fmt.Printf("❌ 目标节点提取失败: %v\n", err)
			os.Exit(1)
		}
		spaceID = extractedSpace
		parentNodeToken = extractedParent

		fmt.Printf("🔗 提取到父节点 Token: %s\n", parentNodeToken)
		fmt.Printf("📁 目标知识库 Space ID: %s\n\n", spaceID)
	}

	// 使用 "." (CWD) 作为 ignorer 的基准目录，以统一处理多路径场景
	engine := sync.NewEngine(prov, ".", !apply, opts.useCodeBlock)
	if err := engine.SyncAll(opts.localPaths, spaceID, parentNodeToken); err != nil {
		fmt.Printf("❌ 同步中断: %v\n", err)
		os.Exit(1)
	}

	if !apply {
		fmt.Println("\n✅ 预览结束。如需真实写入，请追加 --target <Wiki URL>。")
	} else {
		fmt.Println("\n✅ 同步完成")
	}
}

// validateWikiURL 对 Wiki URL 做基础合法性校验，在调用远端 API 之前拦截明显错误。
func validateWikiURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL 不能为空")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return errors.New("URL 必须以 http:// 或 https:// 开头，请检查输入")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL 格式不合法: %w", err)
	}
	if u.Host == "" {
		return errors.New("URL 缺少域名部分，请检查输入")
	}
	return nil
}

// runAuth 处理 auth 子命令
func runAuth(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "用法: wikitnow auth <子命令>\n\n")
		fmt.Fprintf(os.Stderr, "可用子命令:\n")
		fmt.Fprintf(os.Stderr, "  setup [--provider feishu]   交互式配置凭证并写入 ~/.wikitnow/config.json\n")
		fmt.Fprintf(os.Stderr, "  check [--provider feishu]   验证当前凭证是否有效\n")
		os.Exit(1)
	}

	switch args[0] {
	case "setup":
		setupCmd := flag.NewFlagSet("setup", flag.ExitOnError)
		provider := setupCmd.String("provider", "feishu", "目标平台")
		setupCmd.Parse(args[1:])

		switch *provider {
		case "feishu":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("请输入飞书 App ID: ")
			appID, _ := reader.ReadString('\n')
			appID = strings.TrimSpace(appID)

			fmt.Print("请输入飞书 App Secret: ")
			appSecret, _ := reader.ReadString('\n')
			appSecret = strings.TrimSpace(appSecret)

			if appID == "" || appSecret == "" {
				fmt.Fprintln(os.Stderr, "❌ App ID 和 App Secret 不能为空")
				os.Exit(1)
			}

			cfg, err := auth.ReadGlobalConfig()
			if err != nil {
				cfg = &auth.GlobalConfig{}
			}
			if cfg.DefaultProvider == "" {
				cfg.DefaultProvider = "feishu"
			}
			cfg.Feishu = &auth.FeishuConfig{AppID: appID, AppSecret: appSecret}

			if err := auth.WriteGlobalConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "❌ 写入配置失败: %v\n", err)
				os.Exit(1)
			}
			cfgPath, _ := auth.ConfigPath()
			fmt.Printf("✅ 飞书凭证已保存至 %s\n", cfgPath)

		default:
			fmt.Fprintf(os.Stderr, "❌ 未知 provider: %s\n", *provider)
			os.Exit(1)
		}

	case "check":
		checkCmd := flag.NewFlagSet("check", flag.ExitOnError)
		checkCmd.String("provider", "feishu", "目标平台")
		checkCmd.Parse(args[1:])

		mgr, err := auth.NewTokenManager()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 凭证无效: %v\n", err)
			os.Exit(1)
		}
		_, err = mgr.GetToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 凭证验证失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 凭证有效")

	default:
		fmt.Fprintf(os.Stderr, "❌ 未知 auth 子命令: %s\n", args[0])
		os.Exit(1)
	}
}

// runProvider 处理 provider 子命令
func runProvider(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "用法: wikitnow provider <子命令>\n\n")
		fmt.Fprintf(os.Stderr, "可用子命令:\n")
		fmt.Fprintf(os.Stderr, "  list   列出所有支持的知识库平台\n")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		fmt.Println("支持的知识库平台:")
		fmt.Println()
		fmt.Println("  feishu   飞书（Lark）知识库")
		fmt.Println("           凭证: WIKITNOW_FEISHU_APP_ID / WIKITNOW_FEISHU_APP_SECRET")
		fmt.Println("           配置: ~/.wikitnow/config.json -> feishu.app_id / feishu.app_secret")
	default:
		fmt.Fprintf(os.Stderr, "❌ 未知 provider 子命令: %s\n", args[0])
		os.Exit(1)
	}
}

// runConfig 处理 config 子命令
func runConfig(args []string) {
	const systemIgnorePath = "/usr/local/etc/wikitnow/ignore"

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "用法: wikitnow config <子命令>\n\n")
		fmt.Fprintf(os.Stderr, "可用子命令:\n")
		fmt.Fprintf(os.Stderr, "  show-ignore            查看当前目录下实际生效的排除规则\n")
		fmt.Fprintf(os.Stderr, "  init-ignore [--force] [--dest <path>]  在当前目录（或指定路径）生成 .wikitnow/ignore\n")
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
		initCmd := flag.NewFlagSet("init-ignore", flag.ExitOnError)
		force := initCmd.Bool("force", false, "强制覆盖已存在的文件")
		dest := initCmd.String("dest", "", "直接指定输出文件的完整路径（默认：<当前目录>/.wikitnow/ignore）")
		initCmd.Parse(args[1:])

		var destDir, destFile string
		if *dest != "" {
			destFile = *dest
			destDir = filepath.Dir(destFile)
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ 无法获取当前目录: %v\n", err)
				os.Exit(1)
			}
			destDir = filepath.Join(cwd, ".wikitnow")
			destFile = filepath.Join(destDir, "ignore")
		}

		if _, err := os.Stat(destFile); err == nil && !*force {
			fmt.Fprintf(os.Stderr, "⚠️  文件已存在: %s\n", destFile)
			fmt.Fprintf(os.Stderr, "   若要覆盖，请追加 --force 参数。\n")
			os.Exit(1)
		}

		var content []byte
		if data, err := os.ReadFile(systemIgnorePath); err == nil {
			content = data
		} else {
			content = configs.IgnoreContent
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 无法创建目录 %s: %v\n", destDir, err)
			os.Exit(1)
		}
		if err := os.WriteFile(destFile, content, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 写入失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ 已生成: %s\n", destFile)
		fmt.Println("   内容来自默认规则，可按需修改。")

	default:
		fmt.Fprintf(os.Stderr, "❌ 未知 config 子命令: %s\n", args[0])
		os.Exit(1)
	}
}

// findActiveIgnoreFile 按优先级链查找当前生效的 ignore 文件路径（找到即返回）。
// 查找顺序：CWD 向上逐级 → 用户全局 → 系统默认
func findActiveIgnoreFile(systemIgnorePath string) string {
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

	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".wikitnow", "ignore")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	if _, err := os.Stat(systemIgnorePath); err == nil {
		return systemIgnorePath
	}

	return ""
}

// pullOpts 存放 pull 子命令解析后的参数
type pullOpts struct {
	output string // 输出文件路径（不指定则预览）
	force  bool   // 是否覆盖已有文件
	lang   string // 语言（zh / en / ja 等）
	docRef string // 文档引用（URL 或 token）
}

// parsePullArgs 解析 pull 子命令参数
func parsePullArgs(args []string) (*pullOpts, error) {
	if len(args) == 0 {
		return nil, errors.New("必须提供文档 URL 或 doc_token")
	}

	opts := &pullOpts{
		lang:   "zh", // 默认中文
		docRef: args[0],
	}

	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--output" || a == "-output":
			if i+1 >= len(args) {
				return nil, errors.New("--output 需要一个参数值")
			}
			i++
			opts.output = args[i]
		case strings.HasPrefix(a, "--output="):
			opts.output = strings.TrimPrefix(a, "--output=")
		case strings.HasPrefix(a, "-output="):
			opts.output = strings.TrimPrefix(a, "-output=")
		case a == "--force" || a == "-force":
			opts.force = true
		case a == "--lang" || a == "-lang":
			if i+1 >= len(args) {
				return nil, errors.New("--lang 需要一个参数值")
			}
			i++
			opts.lang = args[i]
		case strings.HasPrefix(a, "--lang="):
			opts.lang = strings.TrimPrefix(a, "--lang=")
		case strings.HasPrefix(a, "-lang="):
			opts.lang = strings.TrimPrefix(a, "-lang=")
		case strings.HasPrefix(a, "-"):
			return nil, fmt.Errorf("未知参数: %s（用法: wikitnow pull <URL|token> [--output <path>] [--force] [--lang <lang>]）", a)
		}
	}

	return opts, nil
}

// extractTokenFromURL 从 URL 或 token 字符串中提取 doc_token
func extractTokenFromURL(input string) (string, error) {
	// 直接 token：22-27 个字符，由字母数字组成
	if len(input) >= 22 && len(input) <= 27 && isValidToken(input) {
		return input, nil
	}

	// 尝试从 URL 中提取：支持多种格式
	// https://my.feishu.cn/wiki/.../docxABC123...
	// https://my.feishu.cn/docx/ABC123...
	// https://feishu.cn/docx/ABC123...

	// 使用简单的方式：从右往左找最长的有效 token 序列
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})

	// 倒序查找，以获取 URL 末尾最可能是 token 的部分
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if len(part) >= 22 && len(part) <= 27 && isValidToken(part) {
			return part, nil
		}
	}

	return "", fmt.Errorf("无法从输入中提取有效的 doc_token: %s", input)
}

// isValidToken 检查字符串是否是有效的 token 格式
func isValidToken(s string) bool {
	if len(s) < 22 || len(s) > 27 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// runPull 处理 pull 子命令：从云文档获取 Markdown 内容
func runPull(args []string) {
	opts, err := parsePullArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 错误: %v\n", err)
		os.Exit(1)
	}

	// 提取 doc_token
	docToken, err := extractTokenFromURL(opts.docRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 获取认证凭证
	authManager, err := auth.NewTokenManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 认证错误: %v\n", err)
		fmt.Fprintf(os.Stderr, "   请运行 'wikitnow auth setup' 完成凭证配置\n")
		os.Exit(1)
	}

	// 创建客户端
	client := feishu.NewClient(authManager, false)

	// 获取文档内容
	content, err := client.GetDocumentContent(docToken, opts.lang)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 获取文档失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "   请检查:\n")
		fmt.Fprintf(os.Stderr, "   1. doc_token 是否正确\n")
		fmt.Fprintf(os.Stderr, "   2. 应用是否有文档读权限\n")
		fmt.Fprintf(os.Stderr, "   3. 文档是否已被删除\n")
		os.Exit(1)
	}

	// 预览模式：输出到 stdout
	if opts.output == "" {
		fmt.Print(content)
		return
	}

	// 保存模式：写入文件
	if _, err := os.Stat(opts.output); err == nil && !opts.force {
		fmt.Fprintf(os.Stderr, "⚠️  文件已存在: %s\n", opts.output)
		fmt.Fprintf(os.Stderr, "   若要覆盖，请追加 --force 参数。\n")
		os.Exit(1)
	}

	// 创建目录
	outputDir := filepath.Dir(opts.output)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "❌ 创建目录失败: %v\n", err)
			os.Exit(1)
		}
	}

	// 写入文件
	if err := os.WriteFile(opts.output, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 写入文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 文档已保存到: %s\n", opts.output)
}
