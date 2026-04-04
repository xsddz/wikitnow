package sync

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── TestMappingStore_AddOrUpdateAndGet ───────────────────────────────────────

func TestMappingStore_AddOrUpdateAndGet(t *testing.T) {
	s := &MappingStore{Version: "1.0", Mappings: []MappingRecord{}}

	t.Run("新增记录并查询", func(t *testing.T) {
		s.AddOrUpdate("a/b.md", "node-1", "obj-1", false, "sha256:abc")
		rec := s.GetByLocalPath("a/b.md")
		if rec == nil {
			t.Fatal("期望找到记录")
		}
		if rec.NodeToken != "node-1" || rec.ObjToken != "obj-1" || rec.FileHash != "sha256:abc" {
			t.Errorf("记录内容不符: %+v", rec)
		}
	})

	t.Run("更新已有记录", func(t *testing.T) {
		s.AddOrUpdate("a/b.md", "node-2", "obj-2", false, "sha256:xyz")
		rec := s.GetByLocalPath("a/b.md")
		if rec.NodeToken != "node-2" || rec.FileHash != "sha256:xyz" {
			t.Errorf("更新后记录不符: %+v", rec)
		}
		// Mappings 长度不应增加
		if len(s.Mappings) != 1 {
			t.Errorf("应仍为 1 条记录，实际 %d", len(s.Mappings))
		}
	})

	t.Run("查询不存在的键返回nil", func(t *testing.T) {
		rec := s.GetByLocalPath("not/exist.md")
		if rec != nil {
			t.Errorf("期望 nil，实际 %+v", rec)
		}
	})

	t.Run("多条记录独立存储", func(t *testing.T) {
		s2 := &MappingStore{Version: "1.0", Mappings: []MappingRecord{}}
		s2.AddOrUpdate("x.md", "n1", "o1", false, "h1")
		s2.AddOrUpdate("y.md", "n2", "o2", false, "h2")
		s2.AddOrUpdate("z/a.md", "n3", "o3", false, "h3")

		if s2.GetByLocalPath("x.md").NodeToken != "n1" {
			t.Error("x.md 查询失败")
		}
		if s2.GetByLocalPath("y.md").NodeToken != "n2" {
			t.Error("y.md 查询失败")
		}
		if s2.GetByLocalPath("z/a.md").NodeToken != "n3" {
			t.Error("z/a.md 查询失败")
		}
	})
}

// ─── TestMappingStore_Clear ───────────────────────────────────────────────────

func TestMappingStore_Clear(t *testing.T) {
	s := &MappingStore{Version: "1.0", Mappings: []MappingRecord{}}
	s.AddOrUpdate("a.md", "n1", "o1", false, "h1")
	s.AddOrUpdate("b.md", "n2", "o2", false, "h2")

	s.Clear()

	if len(s.Mappings) != 0 {
		t.Errorf("Clear 后 Mappings 应为空，实际 %d", len(s.Mappings))
	}
	if s.GetByLocalPath("a.md") != nil {
		t.Error("Clear 后查询应返回 nil")
	}
	// Clear 后仍可正常 AddOrUpdate
	s.AddOrUpdate("c.md", "n3", "o3", false, "h3")
	rec := s.GetByLocalPath("c.md")
	if rec == nil || rec.NodeToken != "n3" {
		t.Errorf("Clear 后 AddOrUpdate 异常: %+v", rec)
	}
}

// ─── TestMappingStore_HasDifferentTarget ──────────────────────────────────────

func TestMappingStore_HasDifferentTarget(t *testing.T) {
	tests := []struct {
		name          string
		storeSpace    string
		storeRoot     string
		querySpace    string
		queryRoot     string
		wantDifferent bool
	}{
		{"空store不认为是不同目标", "", "", "sp1", "root1", false},
		{"相同target", "sp1", "root1", "sp1", "root1", false},
		{"spaceID不同", "sp1", "root1", "sp2", "root1", true},
		{"rootNodeToken不同", "sp1", "root1", "sp1", "root2", true},
		{"两者均不同", "sp1", "root1", "sp2", "root2", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MappingStore{SpaceID: tt.storeSpace, RootNodeToken: tt.storeRoot}
			got := s.HasDifferentTarget(tt.querySpace, tt.queryRoot)
			if got != tt.wantDifferent {
				t.Errorf("HasDifferentTarget = %v, want %v", got, tt.wantDifferent)
			}
		})
	}
}

// ─── TestComputeFileHash ──────────────────────────────────────────────────────

func TestComputeFileHash(t *testing.T) {
	dir := t.TempDir()

	t.Run("相同内容产生相同哈希", func(t *testing.T) {
		p1 := filepath.Join(dir, "f1.txt")
		p2 := filepath.Join(dir, "f2.txt")
		os.WriteFile(p1, []byte("hello world"), 0644)
		os.WriteFile(p2, []byte("hello world"), 0644)
		h1, err1 := ComputeFileHash(p1)
		h2, err2 := ComputeFileHash(p2)
		if err1 != nil || err2 != nil {
			t.Fatalf("hash error: %v %v", err1, err2)
		}
		if h1 != h2 {
			t.Errorf("相同内容哈希不同: %q vs %q", h1, h2)
		}
	})

	t.Run("不同内容产生不同哈希", func(t *testing.T) {
		p1 := filepath.Join(dir, "g1.txt")
		p2 := filepath.Join(dir, "g2.txt")
		os.WriteFile(p1, []byte("content A"), 0644)
		os.WriteFile(p2, []byte("content B"), 0644)
		h1, _ := ComputeFileHash(p1)
		h2, _ := ComputeFileHash(p2)
		if h1 == h2 {
			t.Error("不同内容哈希应不同")
		}
	})

	t.Run("哈希以sha256:开头", func(t *testing.T) {
		p := filepath.Join(dir, "h.txt")
		os.WriteFile(p, []byte("test"), 0644)
		h, err := ComputeFileHash(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(h) < 7 || h[:7] != "sha256:" {
			t.Errorf("哈希格式不符: %q", h)
		}
	})

	t.Run("文件不存在返回错误", func(t *testing.T) {
		_, err := ComputeFileHash(filepath.Join(dir, "nonexist.txt"))
		if err == nil {
			t.Error("期望返回错误")
		}
	})
}

// ─── TestMappingRecord_HasChanged ──────────────────────────────────────────────

func TestMappingRecord_HasChanged(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	os.WriteFile(p, []byte("original content"), 0644)
	hash, _ := ComputeFileHash(p)

	t.Run("哈希一致返回false", func(t *testing.T) {
		rec := &MappingRecord{FileHash: hash}
		changed, err := rec.HasChanged(p)
		if err != nil || changed {
			t.Errorf("HasChanged = %v, err = %v", changed, err)
		}
	})

	t.Run("哈希不同返回true", func(t *testing.T) {
		rec := &MappingRecord{FileHash: "sha256:stale"}
		changed, err := rec.HasChanged(p)
		if err != nil || !changed {
			t.Errorf("HasChanged = %v, err = %v", changed, err)
		}
	})

	t.Run("目录记录始终返回false", func(t *testing.T) {
		rec := &MappingRecord{IsDir: true, FileHash: ""}
		changed, err := rec.HasChanged(p)
		if err != nil || changed {
			t.Errorf("目录记录 HasChanged = %v, err = %v", changed, err)
		}
	})

	t.Run("文件不存在认为已变更", func(t *testing.T) {
		rec := &MappingRecord{FileHash: hash}
		changed, err := rec.HasChanged(filepath.Join(dir, "gone.md"))
		if err == nil || !changed {
			t.Errorf("文件不存在 HasChanged = %v, err = %v", changed, err)
		}
	})
}

// ─── TestFindFileUpward ───────────────────────────────────────────────────────

func TestFindFileUpward(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b")
	os.MkdirAll(child, 0755)

	t.Run("在当前目录找到文件", func(t *testing.T) {
		target := filepath.Join(root, ".wikitnow", "ignore")
		os.MkdirAll(filepath.Dir(target), 0755)
		os.WriteFile(target, []byte(""), 0644)

		got := findFileUpward(root, ".wikitnow/ignore")
		if got != target {
			t.Errorf("got %q, want %q", got, target)
		}
	})

	t.Run("在父目录找到文件", func(t *testing.T) {
		got := findFileUpward(child, ".wikitnow/ignore")
		expected := filepath.Join(root, ".wikitnow", "ignore")
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("文件不存在返回空字符串", func(t *testing.T) {
		got := findFileUpward(child, ".wikitnow/nonexist.txt")
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})
}

// ─── TestRebuildIndex ─────────────────────────────────────────────────────────

func TestRebuildIndex(t *testing.T) {
	s := &MappingStore{
		Version: "1.0",
		Mappings: []MappingRecord{
			{LocalPath: "a.md", NodeToken: "n1"},
			{LocalPath: "b/c.md", NodeToken: "n2"},
			{LocalPath: "dir", NodeToken: "n3", IsDir: true},
		},
	}
	s.rebuildIndex()

	for _, path := range []string{"a.md", "b/c.md", "dir"} {
		rec := s.GetByLocalPath(path)
		if rec == nil {
			t.Errorf("rebuildIndex 后查询 %q 失败", path)
		}
	}
	if s.GetByLocalPath("nonexist") != nil {
		t.Error("不存在的路径应返回 nil")
	}
}
