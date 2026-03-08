package sync

import (
	"strings"
	"testing"
)

func TestRenderNodes(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []*treeNode
		padWidth  int
		wantLines []string
	}{
		{
			name: "单文件节点",
			nodes: []*treeNode{
				{displayStr: "README.md", displayLen: 9, statusStr: "✅ 将同步"},
			},
			padWidth: 13,
			// paddingCount = 13 - 9 = 4
			wantLines: []string{"README.md    [✅ 将同步]"},
		},
		{
			name: "目录+子文件+同级文件跨路径对齐",
			nodes: []*treeNode{
				{displayStr: "docs", displayLen: 4, statusStr: "📦 根目录"},
				{displayStr: "├── configuration.md", displayLen: 20, statusStr: "✅ 将同步"},
				{displayStr: "README.en.md", displayLen: 12, statusStr: "✅ 将同步"},
				{displayStr: "README.md", displayLen: 9, statusStr: "✅ 将同步"},
			},
			padWidth: 24,
			// maxLen=20, padWidth=24
			// "docs"(4): padding=20, "├── configuration.md"(20): padding=4
			// "README.en.md"(12): padding=12, "README.md"(9): padding=15
			wantLines: []string{
				"docs                    [📦 根目录]",
				"├── configuration.md    [✅ 将同步]",
				"README.en.md            [✅ 将同步]",
				"README.md               [✅ 将同步]",
			},
		},
		{
			name: "padding最小为1（padWidth小于displayLen）",
			nodes: []*treeNode{
				{displayStr: "a-very-long-filename.md", displayLen: 23, statusStr: "✅ 将同步"},
			},
			padWidth:  10, // < displayLen → paddingCount 被截断为 1
			wantLines: []string{"a-very-long-filename.md [✅ 将同步]"},
		},
		{
			name:      "空节点列表返回空字符串",
			nodes:     []*treeNode{},
			padWidth:  10,
			wantLines: []string{},
		},
		{
			name: "多层嵌套目录前缀对齐",
			nodes: []*treeNode{
				{displayStr: "src", displayLen: 3, statusStr: "📦 根目录"},
				{displayStr: "├── cmd", displayLen: 7, statusStr: "📁 将同步"},
				{displayStr: "│   └── main.go", displayLen: 15, statusStr: "✅ 将同步"},
				{displayStr: "└── README.md", displayLen: 13, statusStr: "✅ 将同步"},
			},
			padWidth: 19,
			// maxLen=15, padWidth=19
			wantLines: []string{
				"src                [📦 根目录]",
				"├── cmd            [📁 将同步]",
				"│   └── main.go    [✅ 将同步]",
				"└── README.md      [✅ 将同步]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderNodes(tt.nodes, tt.padWidth)

			if len(tt.wantLines) == 0 {
				if got != "" {
					t.Fatalf("期望空字符串，实际得到: %q", got)
				}
				return
			}

			lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
			if len(lines) != len(tt.wantLines) {
				t.Fatalf("行数不符: got %d, want %d\n实际输出:\n%s", len(lines), len(tt.wantLines), got)
			}
			for i, line := range lines {
				if line != tt.wantLines[i] {
					t.Errorf("第 %d 行不符:\n  got:  %q\n  want: %q", i, line, tt.wantLines[i])
				}
			}
		})
	}
}

// TestRenderNodesGlobalAlignment 验证 SyncAll 场景下的全局对齐：
// 若手动计算 maxLen 并传入 padWidth，renderNodes 应产生统一对齐的输出。
func TestRenderNodesGlobalAlignment(t *testing.T) {
	// 模拟 docs + README.en.md + README.md 的混合场景
	allNodes := []*treeNode{
		{displayStr: "docs", displayLen: 4, statusStr: "📦 根目录"},
		{displayStr: "├── aaa", displayLen: 7, statusStr: "📁 将同步"},
		{displayStr: "│   └── b.md", displayLen: 12, statusStr: "✅ 将同步"},
		{displayStr: "└── configuration.md", displayLen: 20, statusStr: "✅ 将同步"},
		{displayStr: "README.en.md", displayLen: 12, statusStr: "✅ 将同步"},
		{displayStr: "README.md", displayLen: 9, statusStr: "✅ 将同步"},
	}

	// 计算全局 maxLen（等同于 SyncAll 内部的逻辑）
	maxLen := 0
	for _, n := range allNodes {
		if n.displayLen > maxLen {
			maxLen = n.displayLen
		}
	}
	padWidth := maxLen + 4 // 20 + 4 = 24

	output := renderNodes(allNodes, padWidth)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if len(lines) != len(allNodes) {
		t.Fatalf("行数不符: got %d, want %d", len(lines), len(allNodes))
	}

	// ├、└、│、─ 等 box-drawing 字符是多字节 UTF-8，strings.Index 按字节偏移，
	// 不等于显示宽度。正确做法：找到 "[" 的字节偏移后，用 displayWidth 计算其前缀的显示宽度。
	for i, line := range lines {
		idx := strings.Index(line, "[")
		if idx == -1 {
			t.Errorf("第 %d 行没有 '[': %q", i, line)
			continue
		}
		prefixDisplayWidth := displayWidth(line[:idx])
		if prefixDisplayWidth != padWidth {
			t.Errorf("第 %d 行 '[' 的显示宽度位置=%d, 期望=%d\n  行内容: %q",
				i, prefixDisplayWidth, padWidth, line)
		}
	}
}
