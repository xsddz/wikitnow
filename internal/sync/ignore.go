package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Ignorer 定义了屏蔽逻辑模块
type Ignorer struct {
	matcher *ignore.GitIgnore
}

// systemIgnorePath 是命令安装后系统级默认配置的标准路径。
// 由 install.sh 将 internal/configs/default_ignore 安装至此处。
const systemIgnorePath = "/usr/local/etc/wikitnow/ignore"

// NewIgnorer 按单一优先级瀑布链查找配置（找到即停止，不叠加）：
//
//  1. 从 baseDir 开始向上逐级查找 .wikitnow/ignore
//  2. 用户全局: ~/.wikitnow/ignore
//  3. 系统级默认: /usr/local/etc/wikitnow/ignore  (由 install.sh 部署)
//  4. 若整条链均找不到，打印警告并返回空 Ignorer（仅保留隐藏文件硬性规则）
func NewIgnorer(baseDir string) *Ignorer {
	// 1. 从 baseDir 向上逐级查找项目配置
	searchPath := baseDir
	for {
		ignoreFile := filepath.Join(searchPath, ".wikitnow", "ignore")
		if _, err := os.Stat(ignoreFile); err == nil {
			if m, err := ignore.CompileIgnoreFile(ignoreFile); err == nil && m != nil {
				return &Ignorer{matcher: m}
			}
		}

		parentDir := filepath.Dir(searchPath)
		if parentDir == searchPath {
			break // 已到达文件系统根目录
		}
		searchPath = parentDir
	}

	// 2. 用户全局: ~/.wikitnow/ignore
	if home, err := os.UserHomeDir(); err == nil {
		globalIgnore := filepath.Join(home, ".wikitnow", "ignore")
		if _, err := os.Stat(globalIgnore); err == nil {
			if m, err := ignore.CompileIgnoreFile(globalIgnore); err == nil && m != nil {
				return &Ignorer{matcher: m}
			}
		}
	}

	// 3. 系统级默认配置（由 install.sh 安装至 /usr/local/etc/wikitnow/ignore）
	if _, err := os.Stat(systemIgnorePath); err == nil {
		if m, err := ignore.CompileIgnoreFile(systemIgnorePath); err == nil && m != nil {
			return &Ignorer{matcher: m}
		}
	}

	// 4. 整条链均未找到：安装可能不完整，给出提示
	fmt.Fprintf(os.Stderr, "⚠️  未找到任何 ignore 配置（项目/.wikitnow/ignore、~/.wikitnow/ignore、%s 均不存在）\n", systemIgnorePath)
	fmt.Fprintf(os.Stderr, "   建议运行 wikitnow config init-ignore 在当前目录生成默认配置，或手动创建该文件。\n")
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
