package sync

import (
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Ignorer 定义了屏蔽逻辑模块
type Ignorer struct {
	matcher *ignore.GitIgnore
}

// NewIgnorer 按单一优先级瀑布链查找配置（找到即停止，不叠加）：
//
//  1. 从 baseDir 开始向上逐级查找 .wikitnow/ignore（复用 findFileUpward）
//  2. 用户全局: ~/.wikitnow/ignore（最终兜底）
//  3. 若整条链均找不到，返回空 Ignorer（仅保留隐藏文件硬性规则）
func NewIgnorer(baseDir string) *Ignorer {
	// 1. 从 baseDir 向上逐级查找项目配置
	if path := findFileUpward(baseDir, ".wikitnow/ignore"); path != "" {
		if m, err := ignore.CompileIgnoreFile(path); err == nil && m != nil {
			return &Ignorer{matcher: m}
		}
	}

	// 2. 用户全局: ~/.wikitnow/ignore（最终兜底）
	if home, err := os.UserHomeDir(); err == nil {
		globalIgnore := filepath.Join(home, ".wikitnow", "ignore")
		if _, err := os.Stat(globalIgnore); err == nil {
			if m, err := ignore.CompileIgnoreFile(globalIgnore); err == nil && m != nil {
				return &Ignorer{matcher: m}
			}
		}
	}

	// 3. 整条链均未找到：无排除规则，仅保留隐藏文件硬性规则
	return &Ignorer{matcher: nil}
}

// ShouldIgnore 判定当前文件或目录是否应该被忽略
func (ig *Ignorer) ShouldIgnore(path string, isDir bool) bool {
	baseName := filepath.Base(path)

	// 硬性规则：隐藏文件或目录（以 . 开头）统一忽略，不受配置文件影响
	if strings.HasPrefix(baseName, ".") && baseName != "." && baseName != ".." {
		return true
	}

	if ig.matcher != nil && ig.matcher.MatchesPath(path) {
		return true
	}

	return false
}
