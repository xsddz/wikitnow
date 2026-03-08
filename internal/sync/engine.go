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

// Sync 执行目录或文件同步
// localPath: 待同步的本地绝对/相对路径
// spaceID: 知识库空间 ID
// parentNodeToken: 目标的父节点
func (e *Engine) Sync(localPath, spaceID, parentNodeToken string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	// 准备收集渲染树
	var nodes []*treeNode

	// 根节点
	nodes = append(nodes, &treeNode{
		displayStr: localPath,
		displayLen: utf8.RuneCountInString(localPath),
		statusStr:  "📦 根目录",
		isDir:      info.IsDir(),
	})

	if info.IsDir() {
		err = e.buildDirTree(localPath, spaceID, parentNodeToken, "", &nodes)
	} else {
		err = e.buildFileNode(localPath, spaceID, parentNodeToken, "", &nodes)
	}

	if err != nil {
		return err
	}

	// 计算最长的前缀名，用于排版对齐
	maxLen := 0
	for _, n := range nodes {
		if n.displayLen > maxLen {
			maxLen = n.displayLen
		}
	}

	// 在最长长度基础上再加几个空格作为缓冲间距
	padWidth := maxLen + 4

	// 渲染与执行
	for _, n := range nodes {
		// 中文在控制台通常等效于 2 个英文字符的宽度，这里采取一个更精准的 Terminal 渲染填充算法
		// 但最简单的实现是使用 strings.Repeat 配合 Rune 宽度
		paddingCount := padWidth - n.displayLen
		if paddingCount < 1 {
			paddingCount = 1
		}
		padding := strings.Repeat(" ", paddingCount)

		// 打印对齐的树
		fmt.Printf("%s%s[%s]\n", n.displayStr, padding, n.statusStr)

		// 执行真正的上传动作（收集好后顺带执行）
		if n.originalCfg != nil {
			if n.isDir {
				// 目录创建已经在 buildDirTree 时针对非 dry-run 处理完毕，这里如果有别的逻辑可扩展
			} else {
				// 尝试上传文件
				if upErr := e.uploadSingleFile(n.originalCfg); upErr != nil {
					fmt.Printf("%s  ↳ ❌ 上传错误: %v\n", strings.Repeat(" ", n.displayLen), upErr)
				}
			}
		}
	}

	return nil
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
		return "⚠️ 跳过 (过大量>4MB)"
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
	// 检查目录内是否有实际可序内容
	if !e.hasEffectiveContent(dirPath) {
		return "🚫 空目录"
	}
	return "📁 将同步"
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
		displayLen: utf8.RuneCountInString(display),
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
				displayLen: utf8.RuneCountInString(display),
				statusStr:  status,
				isDir:      true,
			}
			*nodes = append(*nodes, node)

			if status == "📁 将同步" {
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
