package sync

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// LoadMappingStore 从文件加载映射，支持项目级和全局查找
// workingDir：工作目录路径（用于向上查找项目级映射）
// 不存在则返回新的空映射
func LoadMappingStore(workingDir string) *MappingStore {
	mappingPath := findMappingFile(workingDir)
	if mappingPath == "" {
		return &MappingStore{
			Version:  "1.0",
			Mappings: []MappingRecord{},
		}
	}

	data, err := os.ReadFile(mappingPath)
	if err != nil {
		// 文件不存在，返回新的空映射
		return &MappingStore{
			Version:  "1.0",
			Mappings: []MappingRecord{},
		}
	}

	var store MappingStore
	if err := json.Unmarshal(data, &store); err != nil {
		// 反序列化失败，返回新的空映射
		return &MappingStore{
			Version:  "1.0",
			Mappings: []MappingRecord{},
		}
	}

	return &store
}

// Save 将映射保存到文件
// workingDir：工作目录路径
// 保存策略：
// 1. 如果项目级映射文件存在，保存到项目级
// 2. 否则保存到项目目录的 .wikitnow/mapping.json
func (s *MappingStore) Save(workingDir string) error {
	// 首先尝试找到已存在的映射文件
	existingPath := findMappingFile(workingDir)

	var mappingPath string

	// 如果存在映射文件或文件在项目级位置，使用项目级路径
	if existingPath != "" && !filepath.HasPrefix(existingPath, filepath.Join(os.Getenv("HOME"), ".wikitnow")) {
		mappingPath = existingPath
	} else {
		// 否则创建项目级映射
		if workingDir == "" {
			workingDir = "."
		}
		mappingPath = filepath.Join(workingDir, ".wikitnow", "mapping.json")
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

// findMappingFile 按优先级查找映射文件位置
// 查找顺序：
// 1. 从工作目录向上查找 .wikitnow/mapping.json（参考 ignore 文件查找模式）
// 2. 主目录 ~/.wikitnow/mappings/{projectHash}.json
func findMappingFile(workingDir string) string {
	// 第一优先级：项目级映射（从工作目录向上查找）
	if workingDir != "" {
		searchPath := workingDir
		for {
			candidate := filepath.Join(searchPath, ".wikitnow", "mapping.json")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}

			parent := filepath.Dir(searchPath)
			if parent == searchPath {
				break // 到达根目录
			}
			searchPath = parent
		}
	}

	// 第二优先级：全局映射（主目录）
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// 获取工作目录的绝对路径
	absPath, err := filepath.Abs(workingDir)
	if err != nil {
		return ""
	}

	projectHash := getProjectHash(absPath)
	globalMappingPath := filepath.Join(home, ".wikitnow", "mappings", projectHash+".json")

	return globalMappingPath
}

// getOrCreateMappingDir 获取或创建映射文件的目录
func getOrCreateMappingDir(mappingPath string) (string, error) {
	dir := filepath.Dir(mappingPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
