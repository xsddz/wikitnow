package main

import (
	"reflect"
	"testing"
)

func TestParseSyncArgs(t *testing.T) {
	const url = "https://x.feishu.cn/wiki/TOKEN"

	tests := []struct {
		name       string
		args       []string
		wantPaths  []string
		wantTarget string
		wantCB     bool
		wantDebug  bool
		wantErr    bool
	}{
		// ── 基础路径 ──────────────────────────────────────────────
		{
			name:      "单路径无target",
			args:      []string{"README.md"},
			wantPaths: []string{"README.md"},
			wantCB:    true,
		},
		{
			name:      "多路径无target",
			args:      []string{"README.en.md", "README.md"},
			wantPaths: []string{"README.en.md", "README.md"},
			wantCB:    true,
		},
		// ── --target 位置变化 ─────────────────────────────────────
		{
			name:       "target在路径后-空格形式",
			args:       []string{"README.en.md", "README.md", "--target", url},
			wantPaths:  []string{"README.en.md", "README.md"},
			wantTarget: url,
			wantCB:     true,
		},
		{
			name:       "target在路径前-空格形式",
			args:       []string{"--target", url, "README.en.md", "README.md"},
			wantPaths:  []string{"README.en.md", "README.md"},
			wantTarget: url,
			wantCB:     true,
		},
		{
			name:       "target在路径中间-等号形式",
			args:       []string{"README.en.md", "--target=" + url, "README.md"},
			wantPaths:  []string{"README.en.md", "README.md"},
			wantTarget: url,
			wantCB:     true,
		},
		{
			name:       "单横线形式-target",
			args:       []string{"README.md", "-target", url},
			wantPaths:  []string{"README.md"},
			wantTarget: url,
			wantCB:     true,
		},
		{
			name:       "单横线等号形式-target",
			args:       []string{"README.md", "-target=" + url},
			wantPaths:  []string{"README.md"},
			wantTarget: url,
			wantCB:     true,
		},
		// ── --code-block ──────────────────────────────────────────
		{
			name:       "code-block=false",
			args:       []string{"README.md", "--target", url, "--code-block=false"},
			wantPaths:  []string{"README.md"},
			wantTarget: url,
			wantCB:     false,
		},
		{
			name:       "code-block显式true",
			args:       []string{"README.md", "--code-block=true"},
			wantPaths:  []string{"README.md"},
			wantCB:     true,
		},
		{
			name:       "code-block裸flag默认true",
			args:       []string{"README.md", "--code-block"},
			wantPaths:  []string{"README.md"},
			wantCB:     true,
		},
		// ── --debug ───────────────────────────────────────────────
		{
			name:      "debug flag",
			args:      []string{"README.md", "--debug"},
			wantPaths: []string{"README.md"},
			wantCB:    true,
			wantDebug: true,
		},
		// ── 错误场景 ──────────────────────────────────────────────
		{
			name:    "无任何参数",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "只有flags无路径",
			args:    []string{"--target", url},
			wantErr: true,
		},
		{
			name:    "--target无值",
			args:    []string{"README.md", "--target"},
			wantErr: true,
		},
		{
			name:    "未知flag",
			args:    []string{"README.md", "--apply"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSyncArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSyncArgs(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(got.localPaths, tt.wantPaths) {
				t.Errorf("localPaths = %v, want %v", got.localPaths, tt.wantPaths)
			}
			if got.target != tt.wantTarget {
				t.Errorf("target = %q, want %q", got.target, tt.wantTarget)
			}
			if got.useCodeBlock != tt.wantCB {
				t.Errorf("useCodeBlock = %v, want %v", got.useCodeBlock, tt.wantCB)
			}
			if got.debug != tt.wantDebug {
				t.Errorf("debug = %v, want %v", got.debug, tt.wantDebug)
			}
		})
	}
}
