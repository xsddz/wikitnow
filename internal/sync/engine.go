package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xsddz/wikitnow/internal/provider"
)

// treeNode 用于在最终渲染前缓存渲染信息，以实现对齐填充
type treeNode struct {
	displayStr  string        // 前缀 + 文件名 (如 "├── main.go")
	displayLen  int           // 显示字符长度（考虑中文字符）
	statusStr   string        // 状态文本 (如 "✅ 将同步")
	isDir       bool          // 是否为目录
	originalCfg *uploadConfig // 携带给实际执行的闭包参数
}

// ancestorDir 记录一个目录的恢复信息，用于父节点失效时重建目录链
type ancestorDir struct {
	relPath     string // 相对 baseDir 的路径（用于更新 mapping）
	name        string // 目录名称（用于 CreateDir）
	parentToken string // 该目录父节点的 token（重建时作为 CreateDir 的父节点）
}

type uploadConfig struct {
	filePath         string
	fileName         string
	spaceID          string
	parentNodeToken  string
	existingObjToken string // 非空表示更新已有文档（复用远端节点），否则新建
	// 祖先目录链：[0] 是直接父目录，[1] 是祖父目录，以此类推
	// 用于当父目录节点失效时，递归向上重建整条目录链
	ancestors []ancestorDir
}

// Engine 负责执行主同步任务
type Engine struct {
	provider     provider.Provider
	ignorer      *Ignorer
	dryRun       bool
	useCodeBlock bool
	mapping      *MappingStore
	baseDir      string
	// per-SyncAll 的 hasEffectiveContent 结果缓存，避免对同一目录重复遍历
	contentCache map[string]bool
}

// NewEngine 初始化一个新引擎
func NewEngine(p provider.Provider, baseDir string, dryRun bool, useCodeBlock bool) *Engine {
	mapping, anchorDir := LoadMappingStore(baseDir)
	return &Engine{
		provider:     p,
		ignorer:      NewIgnorer(anchorDir),
		dryRun:       dryRun,
		useCodeBlock: useCodeBlock,
		mapping:      mapping,
		baseDir:      anchorDir,
	}
}

// Sync 执行目录或文件同步（SyncAll 的单路径便捷包装）
func (e *Engine) Sync(localPath, spaceID, parentNodeToken string) error {
	return e.SyncAll([]string{localPath}, spaceID, parentNodeToken)
}

// SyncAll 将多个本地路径的节点统一收集、全局对齐渲染，并执行上传。
// 所有路径共享同一个 padWidth，保证跨路径列对齐。
func (e *Engine) SyncAll(localPaths []string, spaceID, parentNodeToken string) error {
	// 初始化 per-call 缓存（避免 hasEffectiveContent 在同一 SyncAll 中重复遍历）
	e.contentCache = make(map[string]bool)

	// 如果指定了 target（非 dry-run），检查并处理 target 变更
	if !e.dryRun {
		if e.mapping.HasDifferentTarget(spaceID, parentNodeToken) {
			e.mapping.Clear()
		}
		e.mapping.UpdateMetadata(spaceID, parentNodeToken)
	}

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

	// 非 dry-run 模式下保存映射
	if !e.dryRun {
		e.mapping.Save(e.baseDir)
	}

	return nil
}

// collectNodes 从单个本地路径收集所有渲染节点。
// 非 dryRun 模式下，目录节点会在此阶段调用 provider.CreateDir 建立远端节点。
// 关键：将输入路径规范化为绝对路径后再操作，确保 filepath.Rel(e.baseDir, ...)
// 始终能正确计算（e.baseDir 也是绝对路径），无论用户传入相对路径还是绝对路径行为完全一致。
func (e *Engine) collectNodes(localPath, spaceID, parentNodeToken string) ([]*treeNode, error) {
	// 保留原始输入用于终端显示，规范化后的绝对路径用于所有 FS/映射操作
	displayPath := localPath
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return nil, fmt.Errorf("无法解析路径 %s: %w", localPath, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}

	var nodes []*treeNode
	if info.IsDir() {
		dirStatus := e.getDirStatus(absPath)
		nodes = append(nodes, &treeNode{
			displayStr: displayPath,
			displayLen: displayWidth(displayPath),
			statusStr:  dirStatus,
			isDir:      true,
		})
		// 无论目录状态如何，都显示其子文件列表，让用户能看到具体的文件状态
		err = e.buildDirTree(absPath, spaceID, parentNodeToken, "", &nodes, nil)
	} else {
		err = e.buildFileNode(absPath, spaceID, parentNodeToken, "", &nodes, nil)
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
		// 目录本身没被忽略，但内容都无需同步（被跳过/忽略）
		return "⏭️  无新文件"
	}
	return "📁 将同步"
}

// hasEffectiveContent 递归检查一个目录是否有至少一个实际可同步的文件。
// 结果会在 e.contentCache 中缓存，避免在同一 SyncAll 调用期间对同一目录重复遍历。
func (e *Engine) hasEffectiveContent(dirPath string) bool {
	if e.contentCache != nil {
		if cached, ok := e.contentCache[dirPath]; ok {
			return cached
		}
	}
	result := e.computeHasEffectiveContent(dirPath)
	if e.contentCache != nil {
		e.contentCache[dirPath] = result
	}
	return result
}

// computeHasEffectiveContent 是 hasEffectiveContent 的实际计算逻辑（不含缓存）
func (e *Engine) computeHasEffectiveContent(dirPath string) bool {
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
			// 检查文件是否真的需要同步
			status := e.getFileStatus(childPath)
			if status == "✅ 将同步" {
				// 检查映射关系：如果已上传且没改变，则不计入需同步的内容
				relPath, _ := filepath.Rel(e.baseDir, childPath)
				oldRecord := e.mapping.GetByLocalPath(relPath)
				if oldRecord != nil {
					// 文件有映射记录，检查是否改变
					changed, _ := oldRecord.HasChanged(childPath)
					if !changed {
						// 文件未改变，不计入有效内容
						continue
					}
				}
				// 要么无映射（新文件），要么已改变，都需要同步
				return true
			}
		}
	}
	return false
}

func (e *Engine) buildFileNode(filePath, spaceID, parentNodeToken, prefix string, nodes *[]*treeNode, ancestors []ancestorDir) error {
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

	// 检查映射关系和文件变更
	relPath, _ := filepath.Rel(e.baseDir, filePath)
	oldRecord := e.mapping.GetByLocalPath(relPath)
	if oldRecord != nil {
		// 文件已经上传过，检查是否改变
		changed, _ := oldRecord.HasChanged(filePath)
		if !changed {
			// 文件未改变，标记为已同步
			node.statusStr = "⏭️  已同步"
			*nodes = append(*nodes, node)
			return nil
		}
	}

	if status == "✅ 将同步" {
		cfg := &uploadConfig{
			filePath:        filePath,
			fileName:        info.Name(),
			spaceID:         spaceID,
			parentNodeToken: parentNodeToken,
			ancestors:       ancestors,
		}
		if oldRecord != nil {
			cfg.existingObjToken = oldRecord.ObjToken
		}
		node.originalCfg = cfg
	}

	*nodes = append(*nodes, node)
	return nil
}

func (e *Engine) buildDirTree(dirPath, spaceID, parentNodeToken, prefix string, nodes *[]*treeNode, ancestors []ancestorDir) error {
	dirName := filepath.Base(dirPath)
	relPath, _ := filepath.Rel(e.baseDir, dirPath)

	// 将当前目录信息前追到祖先链，供子文件/子目录的恢复逻辑使用
	ancestorsForChildren := append([]ancestorDir{{
		relPath:     relPath,
		name:        dirName,
		parentToken: parentNodeToken,
	}}, ancestors...)

	var dirNodeToken string

	if e.dryRun {
		dirNodeToken = "dry-run-token"
	} else {
		oldRecord := e.mapping.GetByLocalPath(relPath)

		if oldRecord != nil {
			// ① 目录已有映射（之前已创建），复用已有 token，不重复调用 CreateDir
			dirNodeToken = oldRecord.NodeToken
		} else if e.hasEffectiveContent(dirPath) {
			// ② 新目录且有内容需要同步，创建远端节点并记录映射
			resNode, err := e.provider.CreateDir(spaceID, parentNodeToken, dirName)
			if err != nil {
				return fmt.Errorf("建立远端目录节点失败 %s: %w", dirName, err)
			}
			dirNodeToken = resNode.ID
			e.mapping.AddOrUpdate(relPath, dirNodeToken, resNode.ObjToken, true, "")
		} else {
			// ③ 无映射且无有效内容（全部已同步或被忽略），跳过 CreateDir
			// 子文件同样不会上传，dirNodeToken 退化为父节点 token 仅供遍历展示
			dirNodeToken = parentNodeToken
		}
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

			// 无论子目录状态如何，都递归显示其内容，让用户看到完整的文件树
			if err := e.buildDirTree(childPath, spaceID, dirNodeToken, nextPrefix, nodes, ancestorsForChildren); err != nil {
				// 仅记录到终端防崩溃
				fmt.Printf("❌ 获取分支目录失败 %s: %v\n", childPath, err)
			}
		} else {
			e.buildFileNode(childPath, spaceID, dirNodeToken, currentPrefix, nodes, ancestorsForChildren)
		}
	}

	return nil
}

func (e *Engine) uploadSingleFile(cfg *uploadConfig) error {
	if e.dryRun {
		return nil
	}

	relPath, _ := filepath.Rel(e.baseDir, cfg.filePath)
	fileHash, _ := ComputeFileHash(cfg.filePath)

	if cfg.existingObjToken != "" {
		// 文件已存在映射且内容已变更：清空并重写远端文档，保持节点不变
		if err := e.provider.UpdateDocument(cfg.existingObjToken, cfg.filePath, cfg.fileName, e.useCodeBlock); err == nil {
			// 只更新哈希，NodeToken/ObjToken 不变
			oldRecord := e.mapping.GetByLocalPath(relPath)
			if oldRecord != nil {
				e.mapping.AddOrUpdate(relPath, oldRecord.NodeToken, oldRecord.ObjToken, false, fileHash)
			}
			return nil
		}
		// 更新失败（远端文档已被删除等），降级为新建
	}

	// 新建文档（首次同步，或远端已删除后降级重建）
	remoteNode, err := e.provider.CreateDocument(cfg.spaceID, cfg.parentNodeToken, cfg.filePath, cfg.fileName, e.useCodeBlock)
	if err != nil && len(cfg.ancestors) > 0 {
		// 父目录节点可能已失效，递归重建目录链后重试
		freshParentToken, rebuildErr := e.rebuildAncestors(cfg.spaceID, cfg.ancestors, 0)
		if rebuildErr == nil {
			remoteNode, err = e.provider.CreateDocument(cfg.spaceID, freshParentToken, cfg.filePath, cfg.fileName, e.useCodeBlock)
		}
	}
	if err != nil {
		return err
	}
	e.mapping.AddOrUpdate(relPath, remoteNode.ID, remoteNode.ObjToken, false, fileHash)
	return nil
}

// rebuildAncestors 递归重建目录链，返回 ancestors[idx] 目录的新 nodeToken。
// 当某一层的 parentToken 也失效时，会先递归重建上一层，再用新 token 创建当前层。
func (e *Engine) rebuildAncestors(spaceID string, ancestors []ancestorDir, idx int) (string, error) {
	if idx >= len(ancestors) {
		return "", fmt.Errorf("祖先链已耗尽，无法重建")
	}
	anc := ancestors[idx]
	newDir, err := e.provider.CreateDir(spaceID, anc.parentToken, anc.name)
	if err != nil {
		// 当前层的 parentToken 也失效，尝试向上重建
		freshParent, upErr := e.rebuildAncestors(spaceID, ancestors, idx+1)
		if upErr != nil {
			return "", fmt.Errorf("重建目录 %s 失败: %w", anc.name, err)
		}
		newDir, err = e.provider.CreateDir(spaceID, freshParent, anc.name)
		if err != nil {
			return "", fmt.Errorf("重建目录 %s 失败: %w", anc.name, err)
		}
	}
	e.mapping.AddOrUpdate(anc.relPath, newDir.ID, newDir.ObjToken, true, "")
	return newDir.ID, nil
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
