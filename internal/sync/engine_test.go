package sync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xsddz/wikitnow/internal/provider"
)

// ─── Mock Provider ────────────────────────────────────────────────────────────

type mockProvider struct {
	createDirFn      func(spaceID, parentID, name string) (*provider.Node, error)
	createDocumentFn func(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error)
	updateDocumentFn func(objToken, filePath, fileName string, useCodeBlock bool) error
}

func (m *mockProvider) PlatformName() string { return "mock" }
func (m *mockProvider) ExtractRoot(wikiURL string) (string, string, error) {
	return "", "", nil
}
func (m *mockProvider) CreateDir(spaceID, parentID, name string) (*provider.Node, error) {
	if m.createDirFn != nil {
		return m.createDirFn(spaceID, parentID, name)
	}
	return &provider.Node{ID: "dir-" + name, ObjToken: "obj-" + name}, nil
}
func (m *mockProvider) CreateDocument(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error) {
	if m.createDocumentFn != nil {
		return m.createDocumentFn(spaceID, parentID, filePath, fileName, useCodeBlock)
	}
	return &provider.Node{ID: "doc-" + fileName, ObjToken: "docobj-" + fileName}, nil
}
func (m *mockProvider) UpdateDocument(objToken, filePath, fileName string, useCodeBlock bool) error {
	if m.updateDocumentFn != nil {
		return m.updateDocumentFn(objToken, filePath, fileName, useCodeBlock)
	}
	return nil
}

// ─── 辅助函数 ──────────────────────────────────────────────────────────────────

// newTestEngine 创建一个基于临时目录的测试 Engine
func newTestEngine(t *testing.T, p provider.Provider, dryRun bool) (*Engine, string) {
	t.Helper()
	dir := t.TempDir()
	mapping := &MappingStore{Version: "1.0", Mappings: []MappingRecord{}}
	return &Engine{
		provider:     p,
		ignorer:      NewIgnorer(dir),
		dryRun:       dryRun,
		useCodeBlock: true,
		mapping:      mapping,
		baseDir:      dir,
		contentCache: make(map[string]bool),
	}, dir
}

// writeFile 在 dir 下创建文件
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// ─── TestDisplayWidth ─────────────────────────────────────────────────────────

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 5},
		{"中文", 4},      // CJK 字符各占 2 列
		{"hello中文", 9}, // 5 + 4
		{"📁", 2},       // emoji 占 2 列
		{"├── main.go", 11},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := displayWidth(tt.input)
			if got != tt.want {
				t.Errorf("displayWidth(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ─── TestGetFileStatus ────────────────────────────────────────────────────────

func TestGetFileStatus(t *testing.T) {
	e, dir := newTestEngine(t, &mockProvider{}, true)

	t.Run("普通文本文件", func(t *testing.T) {
		p := writeFile(t, dir, "readme.md", "# hello")
		got := e.getFileStatus(p)
		if got != "✅ 将同步" {
			t.Errorf("got %q, want '✅ 将同步'", got)
		}
	})

	t.Run("隐藏文件被忽略", func(t *testing.T) {
		p := writeFile(t, dir, ".hidden", "secret")
		got := e.getFileStatus(p)
		if got != "🚫 忽略" {
			t.Errorf("got %q, want '🚫 忽略'", got)
		}
	})

	t.Run("二进制文件被跳过", func(t *testing.T) {
		p := filepath.Join(dir, "binary.bin")
		// 写入包含非 UTF-8 字节的内容
		if err := os.WriteFile(p, []byte{0xFF, 0xFE, 0x00}, 0644); err != nil {
			t.Fatal(err)
		}
		got := e.getFileStatus(p)
		if got != "⚠️ 跳过 (含二进制/非文本)" {
			t.Errorf("got %q, want '⚠️ 跳过 (含二进制/非文本)'", got)
		}
	})

	t.Run("超大文件被跳过", func(t *testing.T) {
		p := filepath.Join(dir, "big.log")
		// 创建一个 4MB+1 字节的文件
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		f.Seek(4*1024*1024, 0)
		f.Write([]byte("x"))
		f.Close()
		got := e.getFileStatus(p)
		if got != "⚠️ 跳过 (超过 4MB)" {
			t.Errorf("got %q, want '⚠️ 跳过 (超过 4MB)'", got)
		}
	})
}

// ─── TestHasEffectiveContent ──────────────────────────────────────────────────

func TestHasEffectiveContent(t *testing.T) {
	t.Run("空目录返回false", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		got := e.hasEffectiveContent(dir)
		if got {
			t.Error("空目录应返回 false")
		}
	})

	t.Run("有普通文件返回true", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		writeFile(t, dir, "a.md", "content")
		got := e.hasEffectiveContent(dir)
		if !got {
			t.Error("有普通文件应返回 true")
		}
	})

	t.Run("只有隐藏文件返回false", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		writeFile(t, dir, ".gitignore", "*.log")
		got := e.hasEffectiveContent(dir)
		if got {
			t.Error("只有隐藏文件应返回 false")
		}
	})

	t.Run("文件已同步且未变更返回false", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, "doc.md", "content")
		hash, _ := ComputeFileHash(p)
		relPath, _ := filepath.Rel(dir, p)
		e.mapping.AddOrUpdate(relPath, "node-1", "obj-1", false, hash)
		got := e.hasEffectiveContent(dir)
		if got {
			t.Error("文件未变更应返回 false")
		}
	})

	t.Run("文件已同步但内容变更返回true", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, "doc.md", "content v1")
		relPath, _ := filepath.Rel(dir, p)
		// 记录旧哈希，然后改变文件内容
		e.mapping.AddOrUpdate(relPath, "node-1", "obj-1", false, "sha256:oldhash")
		got := e.hasEffectiveContent(dir)
		if !got {
			t.Error("文件已变更应返回 true")
		}
	})

	t.Run("嵌套子目录中有文件返回true", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		writeFile(t, dir, "sub/deep.md", "deep content")
		got := e.hasEffectiveContent(dir)
		if !got {
			t.Error("嵌套目录有文件应返回 true")
		}
	})

	t.Run("结果被缓存不重复遍历", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		writeFile(t, dir, "a.md", "content")
		// 第一次计算
		r1 := e.hasEffectiveContent(dir)
		// 手动修改缓存验证第二次走缓存
		e.contentCache[dir] = false
		r2 := e.hasEffectiveContent(dir)
		if r1 == r2 {
			t.Error("第二次应命中缓存，期望不同值")
		}
	})
}

// ─── TestBuildFileNode ────────────────────────────────────────────────────────

func TestBuildFileNode(t *testing.T) {
	t.Run("新文件生成待上传节点", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, "a.md", "hello")
		var nodes []*treeNode
		err := e.buildFileNode(p, "space1", "parent1", "", &nodes, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(nodes) != 1 {
			t.Fatalf("期望 1 个节点，实际 %d", len(nodes))
		}
		if nodes[0].statusStr != "✅ 将同步" {
			t.Errorf("new file status = %q, want '✅ 将同步'", nodes[0].statusStr)
		}
		if nodes[0].originalCfg == nil {
			t.Error("期望 originalCfg 不为 nil")
		}
		if nodes[0].originalCfg.parentNodeToken != "parent1" {
			t.Errorf("parentNodeToken = %q, want 'parent1'", nodes[0].originalCfg.parentNodeToken)
		}
	})

	t.Run("隐藏文件不生成上传配置", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, ".hidden", "secret")
		var nodes []*treeNode
		e.buildFileNode(p, "space1", "parent1", "", &nodes, nil, false)
		if len(nodes) != 1 {
			t.Fatalf("期望 1 个节点，实际 %d", len(nodes))
		}
		if nodes[0].originalCfg != nil {
			t.Error("隐藏文件不应生成 originalCfg")
		}
	})

	t.Run("已同步未变更文件标记为已同步", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, "b.md", "content")
		hash, _ := ComputeFileHash(p)
		relPath, _ := filepath.Rel(dir, p)
		e.mapping.AddOrUpdate(relPath, "node-b", "obj-b", false, hash)

		var nodes []*treeNode
		e.buildFileNode(p, "space1", "parent1", "", &nodes, nil, false)
		if nodes[0].statusStr != "⏭️  已同步" {
			t.Errorf("got %q, want '⏭️  已同步'", nodes[0].statusStr)
		}
		if nodes[0].originalCfg != nil {
			t.Error("已同步文件不应生成 originalCfg")
		}
	})

	t.Run("已同步但变更的文件传入existingObjToken", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, "c.md", "new content")
		relPath, _ := filepath.Rel(dir, p)
		e.mapping.AddOrUpdate(relPath, "node-c", "obj-c", false, "sha256:oldhash")

		var nodes []*treeNode
		e.buildFileNode(p, "space1", "parent1", "", &nodes, nil, false)
		if nodes[0].originalCfg == nil {
			t.Fatal("期望 originalCfg 不为 nil")
		}
		if nodes[0].originalCfg.existingObjToken != "obj-c" {
			t.Errorf("existingObjToken = %q, want 'obj-c'", nodes[0].originalCfg.existingObjToken)
		}
	})

	t.Run("ancestors链正确传入uploadConfig", func(t *testing.T) {
		e, dir := newTestEngine(t, &mockProvider{}, true)
		p := writeFile(t, dir, "d.md", "content")
		ancestors := []ancestorDir{
			{relPath: "sub", name: "sub", parentToken: "root-token"},
		}
		var nodes []*treeNode
		e.buildFileNode(p, "space1", "parent1", "", &nodes, ancestors, false)
		if nodes[0].originalCfg == nil {
			t.Fatal("期望 originalCfg 不为 nil")
		}
		if len(nodes[0].originalCfg.ancestors) != 1 {
			t.Errorf("ancestors len = %d, want 1", len(nodes[0].originalCfg.ancestors))
		}
		if nodes[0].originalCfg.ancestors[0].name != "sub" {
			t.Errorf("ancestors[0].name = %q, want 'sub'", nodes[0].originalCfg.ancestors[0].name)
		}
	})
}

// ─── TestRebuildAncestors ─────────────────────────────────────────────────────

func TestRebuildAncestors(t *testing.T) {
	t.Run("单层成功重建", func(t *testing.T) {
		e, _ := newTestEngine(t, &mockProvider{}, false)
		ancestors := []ancestorDir{
			{relPath: "a", name: "a", parentToken: "root-token"},
		}
		newToken, err := e.rebuildAncestors("space1", ancestors, 0)
		if err != nil {
			t.Fatal(err)
		}
		if newToken != "dir-a" {
			t.Errorf("newToken = %q, want 'dir-a'", newToken)
		}
		// 映射已更新
		rec := e.mapping.GetByLocalPath("a")
		if rec == nil || rec.NodeToken != "dir-a" {
			t.Errorf("mapping not updated: %+v", rec)
		}
	})

	t.Run("单层创建失败返回错误", func(t *testing.T) {
		p := &mockProvider{
			createDirFn: func(spaceID, parentID, name string) (*provider.Node, error) {
				return nil, errors.New("api error")
			},
		}
		e, _ := newTestEngine(t, p, false)
		ancestors := []ancestorDir{
			{relPath: "a", name: "a", parentToken: "root-token"},
		}
		_, err := e.rebuildAncestors("space1", ancestors, 0)
		if err == nil {
			t.Error("期望返回错误")
		}
	})

	t.Run("两层链：直接父节点失效，向上重建祖父后成功", func(t *testing.T) {
		callCount := 0
		p := &mockProvider{
			createDirFn: func(spaceID, parentID, name string) (*provider.Node, error) {
				callCount++
				// 第一次调用（重建 b，用旧 a_token）失败
				if callCount == 1 {
					return nil, fmt.Errorf("stale parent token")
				}
				// 第二次调用（重建 a，用 root-token）成功
				// 第三次调用（重建 b，用新 a_token）成功
				return &provider.Node{ID: "new-" + name, ObjToken: "obj-" + name}, nil
			},
		}
		e, _ := newTestEngine(t, p, false)
		ancestors := []ancestorDir{
			{relPath: "a/b", name: "b", parentToken: "stale-a-token"},
			{relPath: "a", name: "a", parentToken: "root-token"},
		}
		newToken, err := e.rebuildAncestors("space1", ancestors, 0)
		if err != nil {
			t.Fatal(err)
		}
		if newToken != "new-b" {
			t.Errorf("newToken = %q, want 'new-b'", newToken)
		}
		// 两个目录的映射都已更新
		recA := e.mapping.GetByLocalPath("a")
		recB := e.mapping.GetByLocalPath("a/b")
		if recA == nil || recA.NodeToken != "new-a" {
			t.Errorf("a mapping: %+v", recA)
		}
		if recB == nil || recB.NodeToken != "new-b" {
			t.Errorf("b mapping: %+v", recB)
		}
	})

	t.Run("祖先链exhausted返回错误", func(t *testing.T) {
		e, _ := newTestEngine(t, &mockProvider{}, false)
		_, err := e.rebuildAncestors("space1", []ancestorDir{}, 0)
		if err == nil {
			t.Error("空链应返回错误")
		}
	})
}

// ─── TestUploadSingleFile ─────────────────────────────────────────────────────

func TestUploadSingleFile(t *testing.T) {
	t.Run("新文件调用CreateDocument并更新映射", func(t *testing.T) {
		var gotFilePath string
		p := &mockProvider{
			createDocumentFn: func(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error) {
				gotFilePath = filePath
				return &provider.Node{ID: "node-doc", ObjToken: "obj-doc"}, nil
			},
		}
		e, dir := newTestEngine(t, p, false)
		filePath := writeFile(t, dir, "readme.md", "# Hello")
		cfg := &uploadConfig{
			filePath:        filePath,
			fileName:        "readme.md",
			spaceID:         "space1",
			parentNodeToken: "parent1",
		}
		err := e.uploadSingleFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if gotFilePath != filePath {
			t.Errorf("CreateDocument called with %q, want %q", gotFilePath, filePath)
		}
		relPath, _ := filepath.Rel(dir, filePath)
		rec := e.mapping.GetByLocalPath(relPath)
		if rec == nil || rec.NodeToken != "node-doc" {
			t.Errorf("mapping not updated: %+v", rec)
		}
	})

	t.Run("已变更文件调用UpdateDocument", func(t *testing.T) {
		updateCalled := false
		p := &mockProvider{
			updateDocumentFn: func(objToken, filePath, fileName string, useCodeBlock bool) error {
				updateCalled = true
				return nil
			},
		}
		e, dir := newTestEngine(t, p, false)
		filePath := writeFile(t, dir, "doc.md", "new content")
		relPath, _ := filepath.Rel(dir, filePath)
		e.mapping.AddOrUpdate(relPath, "node-old", "obj-old", false, "sha256:oldhash")

		cfg := &uploadConfig{
			filePath:         filePath,
			fileName:         "doc.md",
			spaceID:          "space1",
			parentNodeToken:  "parent1",
			existingObjToken: "obj-old",
		}
		err := e.uploadSingleFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !updateCalled {
			t.Error("期望调用 UpdateDocument")
		}
		// NodeToken 不变
		rec := e.mapping.GetByLocalPath(relPath)
		if rec == nil || rec.NodeToken != "node-old" {
			t.Errorf("NodeToken should be preserved: %+v", rec)
		}
	})

	t.Run("UpdateDocument失败降级为CreateDocument", func(t *testing.T) {
		createCalled := false
		p := &mockProvider{
			updateDocumentFn: func(objToken, filePath, fileName string, useCodeBlock bool) error {
				return errors.New("update failed")
			},
			createDocumentFn: func(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error) {
				createCalled = true
				return &provider.Node{ID: "new-node", ObjToken: "new-obj"}, nil
			},
		}
		e, dir := newTestEngine(t, p, false)
		filePath := writeFile(t, dir, "doc.md", "content")
		cfg := &uploadConfig{
			filePath:         filePath,
			fileName:         "doc.md",
			spaceID:          "space1",
			parentNodeToken:  "parent1",
			existingObjToken: "stale-obj",
		}
		err := e.uploadSingleFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !createCalled {
			t.Error("UpdateDocument 失败后应调用 CreateDocument")
		}
	})

	t.Run("CreateDocument失败且有ancestors则重建后重试", func(t *testing.T) {
		callCount := 0
		p := &mockProvider{
			createDocumentFn: func(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error) {
				callCount++
				if callCount == 1 {
					return nil, errors.New("stale parent token")
				}
				return &provider.Node{ID: "doc-new", ObjToken: "obj-new"}, nil
			},
			createDirFn: func(spaceID, parentID, name string) (*provider.Node, error) {
				return &provider.Node{ID: "rebuilt-dir", ObjToken: "obj-dir"}, nil
			},
		}
		e, dir := newTestEngine(t, p, false)
		filePath := writeFile(t, dir, "file.md", "content")
		cfg := &uploadConfig{
			filePath:        filePath,
			fileName:        "file.md",
			spaceID:         "space1",
			parentNodeToken: "stale-parent",
			ancestors: []ancestorDir{
				{relPath: "sub", name: "sub", parentToken: "root-token"},
			},
		}
		err := e.uploadSingleFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if callCount != 2 {
			t.Errorf("CreateDocument 调用次数 = %d, want 2", callCount)
		}
	})

	t.Run("dry-run模式不执行任何上传", func(t *testing.T) {
		createCalled := false
		p := &mockProvider{
			createDocumentFn: func(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error) {
				createCalled = true
				return &provider.Node{}, nil
			},
		}
		e, dir := newTestEngine(t, p, true) // dryRun=true
		filePath := writeFile(t, dir, "readme.md", "content")
		cfg := &uploadConfig{
			filePath:        filePath,
			fileName:        "readme.md",
			spaceID:         "space1",
			parentNodeToken: "parent1",
		}
		e.uploadSingleFile(cfg)
		if createCalled {
			t.Error("dry-run 模式不应调用 CreateDocument")
		}
	})
}

// ─── TestIgnoredDirCascade ────────────────────────────────────────────────────

func TestIgnoredDirCascade(t *testing.T) {
	// 被忽略目录下的文件也应显示为 "🚫 忽略"，不产生 originalCfg
	e, dir := newTestEngine(t, &mockProvider{}, true)
	// 创建隐藏目录（以 . 开头，硬性规则忽略）及其内部的普通文件
	writeFile(t, dir, ".hiddendir/readme.md", "should be ignored")
	writeFile(t, dir, ".hiddendir/sub/doc.md", "also ignored")

	hiddenDir := filepath.Join(dir, ".hiddendir")
	var nodes []*treeNode
	// ignored=true 模拟父目录已标记为忽略时的递归展示
	if err := e.buildDirTree(hiddenDir, "space1", "parent1", "", &nodes, nil, true); err != nil {
		t.Fatal(err)
	}

	for _, n := range nodes {
		if n.statusStr != "🚫 忽略" {
			t.Errorf("节点 %q 状态应为 '🚫 忽略'，实际 %q", n.displayStr, n.statusStr)
		}
		if n.originalCfg != nil {
			t.Errorf("节点 %q 不应生成 originalCfg（不应上传）", n.displayStr)
		}
	}
}

// ─── TestCollectNodes_PathNormalization ───────────────────────────────────────

func TestCollectNodes_PathNormalization(t *testing.T) {
	p := &mockProvider{}
	e, dir := newTestEngine(t, p, true)
	writeFile(t, dir, "sub/a.md", "content a")
	writeFile(t, dir, "sub/b.md", "content b")

	// 始终使用绝对路径；测试绝对路径与相对路径都能产生相同节点数
	absSubDir := filepath.Join(dir, "sub")
	// 构造一个等价的相对路径（相对于 dir）
	origWD, _ := os.Getwd()
	defer os.Chdir(origWD)
	os.Chdir(dir) // 切换到 dir，使 "sub" 是有效的相对路径

	var nodesRel, nodesAbs []*treeNode

	errRel := e.buildDirTree("sub", "space1", "parent1", "", &nodesRel, nil, false)
	errAbs := e.buildDirTree(absSubDir, "space1", "parent1", "", &nodesAbs, nil, false)

	if errRel != nil {
		t.Fatalf("relpath buildDirTree error: %v", errRel)
	}
	if errAbs != nil {
		t.Fatalf("abspath buildDirTree error: %v", errAbs)
	}

	// 两者产生的节点数应相同（relPath 键一致，不会重复创建）
	if len(nodesRel) != len(nodesAbs) {
		t.Errorf("节点数不同: rel=%d, abs=%d", len(nodesRel), len(nodesAbs))
	}
}
