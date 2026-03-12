package sync

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MappingStore 记录本地文件与远端文档的映射关系
type MappingStore struct {
	Version       string          `json:"version"`
	SpaceID       string          `json:"space_id"`
	RootNodeToken string          `json:"root_node_token"`
	SyncedAt      time.Time       `json:"synced_at"`
	Mappings      []MappingRecord `json:"mappings"`
}

// MappingRecord 单个本地文件与远端节点的映射
type MappingRecord struct {
	LocalPath string `json:"local_path"`
	NodeToken string `json:"node_token"`
	ObjToken  string `json:"obj_token,omitempty"`
	IsDir     bool   `json:"is_dir"`
	FileHash  string `json:"file_hash,omitempty"`
}

// LoadMappingStore 从文件加载映射，支持项目级和全局查找。
// 返回 (store, anchorDir)：
//   - anchorDir 是 relPath 计算的基准根，调用方（Engine）应将 baseDir 设为此值，
//     以保证 local_path 键与映射文件始终一致，不受当前工作目录影响。
//
// workingDir：工作目录路径（用于向上查找项目级映射）；不存在则返回新的空映射。
func LoadMappingStore(workingDir string) (*MappingStore, string) {
	mappingPath, anchorDir := findMappingFile(workingDir)
	if mappingPath == "" {
		return &MappingStore{
			Version:  "1.0",
			Mappings: []MappingRecord{},
		}, workingDir
	}

	data, err := os.ReadFile(mappingPath)
	if err != nil {
		// 文件不存在，返回新的空映射
		return &MappingStore{
			Version:  "1.0",
			Mappings: []MappingRecord{},
		}, anchorDir
	}

	var store MappingStore
	if err := json.Unmarshal(data, &store); err != nil {
		// 反序列化失败，返回新的空映射
		return &MappingStore{
			Version:  "1.0",
			Mappings: []MappingRecord{},
		}, anchorDir
	}

	return &store, anchorDir
}

// Save 将映射保存到文件
// workingDir：工作目录路径（传入 e.baseDir，即 anchorDir）
// 保存策略：
// 1. 如果项目级映射文件已存在（workingDir 向上找到），保存到原处
// 2. 否则兜底到 ~/.wikitnow/mappings/{projectHash}.json，不污染工作目录
func (s *MappingStore) Save(workingDir string) error {
	mappingPath, _ := findMappingFile(workingDir)
	if mappingPath == "" {
		return fmt.Errorf("无法确定映射文件保存路径（HOME 目录不可访问）")
	}

	if _, err := getOrCreateMappingDir(mappingPath); err != nil {
		return err
	}

	data, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(mappingPath, data, 0644)
}

// GetByLocalPath 根据本地路径查找映射
func (s *MappingStore) GetByLocalPath(localPath string) *MappingRecord {
	for i := range s.Mappings {
		if s.Mappings[i].LocalPath == localPath {
			return &s.Mappings[i]
		}
	}
	return nil
}

// AddOrUpdate 添加或更新映射
func (s *MappingStore) AddOrUpdate(localPath, nodeToken, objToken string, isDir bool, fileHash string) {
	for i := range s.Mappings {
		if s.Mappings[i].LocalPath == localPath {
			s.Mappings[i].NodeToken = nodeToken
			s.Mappings[i].ObjToken = objToken
			s.Mappings[i].IsDir = isDir
			s.Mappings[i].FileHash = fileHash
			return
		}
	}

	s.Mappings = append(s.Mappings, MappingRecord{
		LocalPath: localPath,
		NodeToken: nodeToken,
		ObjToken:  objToken,
		IsDir:     isDir,
		FileHash:  fileHash,
	})
}

// Clear 清空所有映射（切换 target 时使用）
func (s *MappingStore) Clear() {
	s.Mappings = []MappingRecord{}
}

// UpdateMetadata 更新知识库信息
func (s *MappingStore) UpdateMetadata(spaceID, rootNodeToken string) {
	s.SpaceID = spaceID
	s.RootNodeToken = rootNodeToken
	s.SyncedAt = time.Now()
}

// HasDifferentTarget 检查是否指向不同的目标
func (s *MappingStore) HasDifferentTarget(spaceID, rootNodeToken string) bool {
	// 如果映射为空，不认为是不同的目标
	if s.SpaceID == "" && s.RootNodeToken == "" {
		return false
	}

	return s.SpaceID != spaceID || s.RootNodeToken != rootNodeToken
}

// ComputeFileHash 计算文件的 SHA256 哈希值
func ComputeFileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// HasChanged 检查文件是否改变
func (record *MappingRecord) HasChanged(filePath string) (bool, error) {
	if record.IsDir {
		return false, nil // 目录不需要检查
	}

	newHash, err := ComputeFileHash(filePath)
	if err != nil {
		return true, err // 读取失败认为改变了，重新上传
	}

	return newHash != record.FileHash, nil
}

// getProjectHash 根据绝对路径生成项目哈希（用于全局映射存储）
func getProjectHash(absPath string) string {
	hash := md5.Sum([]byte(absPath))
	return hex.EncodeToString(hash[:])
}

// findFileUpward 从 baseDir 开始向上逐级查找 relPath（相对于各层目录），
// 找到第一个存在的文件即返回其绝对路径；到达文件系统根目录仍未找到则返回 ""。
// ignore 和 mapping 均通过此函数完成向上查找，保证机制完全一致。
func findFileUpward(baseDir, relPath string) string {
	searchPath := baseDir
	for {
		candidate := filepath.Join(searchPath, relPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(searchPath)
		if parent == searchPath {
			return ""
		}
		searchPath = parent
	}
}

// findMappingFile 按优先级查找映射文件位置，同时返回锚定目录（anchorDir）：
//   - anchorDir 是计算 local_path 相对路径的基准根，确保无论从哪个子目录运行
//     relPath 都保持一致，不会因 baseDir 不同而产生键冲突或重复条目。
//
// 1. 从工作目录向上查找 .wikitnow/mapping.json（与 ignore 查找机制一致）
//   - mappingPath = 找到的文件路径
//   - anchorDir  = 含有 .wikitnow/ 的目录（即 `filepath.Dir(filepath.Dir(path))`）
//
// 2. 家目录 ~/.wikitnow/mappings/{projectHash}.json（最终兜底）
//   - anchorDir  = workingDir（每个工作目录独享一个哈希文件，天然隔离）
func findMappingFile(workingDir string) (string, string) {
	// 1. 从工作目录向上查找已存在的项目级映射
	if workingDir != "" {
		if path := findFileUpward(workingDir, ".wikitnow/mapping.json"); path != "" {
			// anchorDir = 包含 .wikitnow/ 的目录
			anchorDir := filepath.Dir(filepath.Dir(path))
			return path, anchorDir
		}
	}

	// 2. 家目录兜底：路径始终有效（Save 会按需创建目录和文件）
	home, err := os.UserHomeDir()
	if err != nil {
		return "", workingDir
	}

	// 获取工作目录的绝对路径
	absPath, err := filepath.Abs(workingDir)
	if err != nil {
		return "", workingDir
	}

	projectHash := getProjectHash(absPath)
	return filepath.Join(home, ".wikitnow", "mappings", projectHash+".json"), workingDir
}

// getOrCreateMappingDir 获取或创建映射文件的目录
func getOrCreateMappingDir(mappingPath string) (string, error) {
	dir := filepath.Dir(mappingPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
