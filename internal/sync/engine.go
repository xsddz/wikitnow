package sync

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xsddz/wikitnow/internal/provider"
)

// TreeNode 用于在最终渲染前缓存渲染信息，以实现对齐填充
type treeNode struct {
	displayStr  string        // 前缀 + 文件名 (如 "├── main.go")
	displayLen  int           // 显示字符长度（考虑中文字符）
	statusStr   string        // 状态文本 (如 "✅ 将同步")
	isDir       bool          // 是否为目录
	hasError    bool          // 是否遇到了执行错误
	errorMsg    string        // 具体的错误信息
	originalCfg *uploadConfig // 携带给实际执行的闭包参数
}

type uploadConfig struct {
	filePath        string
	fileName        string
	spaceID         string
	parentNodeToken string
}

// Engine 负责执行主同步任务
type Engine struct {
	provider     provider.Provider
	ignorer      *Ignorer
	dryRun       bool
	useCodeBlock bool
}

// NewEngine 初始化一个新引擎
func NewEngine(p provider.Provider, baseDir string, dryRun bool, useCodeBlock bool) *Engine {
	return &Engine{
		provider:     p,
		ignorer:      NewIgnorer(baseDir),
		dryRun:       dryRun,
		useCodeBlock: useCodeBlock,
	}
}

// Sync 执行目录或文件同步（SyncAll 的单路径便捷包装）
func (e *Engine) Sync(localPath, spaceID, parentNodeToken string) error {
	return e.SyncAll([]string{localPath}, spaceID, parentNodeToken)
}

// SyncAll 将多个本地路径的节点统一收集、全局对齐渲染，并执行上传。
// 所有路径共享同一个 padWidth，保证跨路径列对齐。
func (e *Engine) SyncAll(localPaths []string, spaceID, parentNodeToken string) error {
	var allNodes []*treeNode
	for _, lp := range localPaths {
		nodes, err := e.collectNodes(lp, spaceID, parentNodeToken)
		if err != nil {
			return err
		}
		allNodes = append(allNodes, nodes...)
	}

	maxLen := 0
	for _, n := range allNodes {
		if n.displayLen > maxLen {
			maxLen = n.displayLen
		}
	}
	padWidth := maxLen + 4

	fmt.Print(renderNodes(allNodes, padWidth))

	for _, n := range allNodes {
		if n.originalCfg != nil && !n.isDir {
			if upErr := e.uploadSingleFile(n.originalCfg); upErr != nil {
				fmt.Printf("%s  ↳ ❌ 上传错误: %v\n", strings.Repeat(" ", n.displayLen), upErr)
			}
		}
	}

	return nil
}

// collectNodes 从单个本地路径收集所有渲染节点。
// 非 dryRun 模式下，目录节点会在此阶段调用 provider.CreateDir 建立远端节点。
func (e *Engine) collectNodes(localPath, spaceID, parentNodeToken string) ([]*treeNode, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}

	var nodes []*treeNode
	if info.IsDir() {
		nodes = append(nodes, &treeNode{
			displayStr: localPath,
			displayLen: displayWidth(localPath),
			statusStr:  "✅ 将同步",
			isDir:      true,
		})
		err = e.buildDirTree(localPath, spaceID, parentNodeToken, "", &nodes)
	} else {
		err = e.buildFileNode(localPath, spaceID, parentNodeToken, "", &nodes)
	}
	return nodes, err
}

// renderNodes 将节点列表渲染为对齐的树形文本并返回字符串。
// padWidth 为全局统一列宽；纯函数，无副作用，可直接单测。
func renderNodes(nodes []*treeNode, padWidth int) string {
	var sb strings.Builder
	for _, n := range nodes {
		paddingCount := padWidth - n.displayLen
		if paddingCount < 1 {
			paddingCount = 1
		}
		padding := strings.Repeat(" ", paddingCount)
		sb.WriteString(fmt.Sprintf("%s%s[%s]\n", n.displayStr, padding, n.statusStr))
	}
	return sb.String()
}

func (e *Engine) getFileStatus(filePath string) string {
	if e.ignorer.ShouldIgnore(filePath, false) {
		return "🚫 忽略"
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "❌ 读取错误"
	}
	if info.Size() > 4*1024*1024 {
		return "⚠️ 跳过 (超过 4MB)"
	}
	content, err := os.ReadFile(filePath)
	if err == nil && !utf8.Valid(content) {
		return "⚠️ 跳过 (含二进制/非文本)"
	}
	return "✅ 将同步"
}

func (e *Engine) getDirStatus(dirPath string) string {
	if e.ignorer.ShouldIgnore(dirPath, true) {
		return "🚫 忽略"
	}
	// 检查目录内是否有实际可同步的内容
	if !e.hasEffectiveContent(dirPath) {
		return "🚫 忽略"
	}
	return "✅ 将同步"
}

// hasEffectiveContent 递归检查一个目录是否有至少一个实际可同步的文件
func (e *Engine) hasEffectiveContent(dirPath string) bool {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		childPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			if !e.ignorer.ShouldIgnore(childPath, true) && e.hasEffectiveContent(childPath) {
				return true
			}
		} else {
			if e.getFileStatus(childPath) == "✅ 将同步" {
				return true
			}
		}
	}
	return false
}

func (e *Engine) buildFileNode(filePath, spaceID, parentNodeToken, prefix string, nodes *[]*treeNode) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	status := e.getFileStatus(filePath)
	display := prefix + info.Name()

	node := &treeNode{
		displayStr: display,
		displayLen: displayWidth(display),
		statusStr:  status,
		isDir:      false,
	}

	if status == "✅ 将同步" {
		node.originalCfg = &uploadConfig{
			filePath:        filePath,
			fileName:        info.Name(),
			spaceID:         spaceID,
			parentNodeToken: parentNodeToken,
		}
	}

	*nodes = append(*nodes, node)
	return nil
}

func (e *Engine) buildDirTree(dirPath, spaceID, parentNodeToken, prefix string, nodes *[]*treeNode) error {
	dirName := filepath.Base(dirPath)

	var dirNodeToken string
	if e.dryRun {
		dirNodeToken = "dry-run-token"
	} else {
		// 在遍历提取时，如果不是 dry-run，我们就必须得建立这个目录拿到父级 Token 才能让下面孩子节点传递
		resNode, err := e.provider.CreateDir(spaceID, parentNodeToken, dirName)
		if err != nil {
			return fmt.Errorf("建立远端目录节点失败 %s: %w", dirName, err)
		}
		dirNodeToken = resNode.ID
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		isLast := i == len(entries)-1
		var currentPrefix, nextPrefix string

		if isLast {
			currentPrefix = prefix + "└── "
			nextPrefix = prefix + "    "
		} else {
			currentPrefix = prefix + "├── "
			nextPrefix = prefix + "│   "
		}

		childPath := filepath.Join(dirPath, entry.Name())
		display := currentPrefix + entry.Name()

		if entry.IsDir() {
			status := e.getDirStatus(childPath)
			node := &treeNode{
				displayStr: display,
				displayLen: displayWidth(display),
				statusStr:  status,
				isDir:      true,
			}
			*nodes = append(*nodes, node)

			if status == "✅ 将同步" {
				if err := e.buildDirTree(childPath, spaceID, dirNodeToken, nextPrefix, nodes); err != nil {
					// 仅记录到终端防崩溃
					fmt.Printf("❌ 获取分支目录失败 %s: %v\n", childPath, err)
				}
			}
		} else {
			e.buildFileNode(childPath, spaceID, dirNodeToken, currentPrefix, nodes)
		}
	}

	return nil
}

func (e *Engine) uploadSingleFile(cfg *uploadConfig) error {
	if e.dryRun {
		return nil
	}

	_, err := e.provider.CreateDocument(cfg.spaceID, cfg.parentNodeToken, cfg.filePath, cfg.fileName, e.useCodeBlock)
	return err
}

// Stat 检查前缀
func Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

// displayWidth 计算字符串在终端中的显示宽度，CJK 字符计为 2 列
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r >= 0x1100 && (r <= 0x115F || // Hangul Jamo
			r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x303E) || // CJK Radicals
			(r >= 0x3040 && r <= 0x33FF) || // Japanese
			(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
			(r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified
			(r >= 0xA000 && r <= 0xA4CF) || // Yi
			(r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
			(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility
			(r >= 0xFE10 && r <= 0xFE19) ||
			(r >= 0xFE30 && r <= 0xFE6F) ||
			(r >= 0xFF00 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x1F300 && r <= 0x1F9FF)) { // Emoji
			width += 2
		} else {
			width += 1
		}
	}
	return width
}
