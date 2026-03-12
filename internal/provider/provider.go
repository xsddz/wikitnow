package provider

// Node 代表远程知识库中的一个节点（文件或目录）
type Node struct {
	ID       string // 平台通用的唯一标识（例如飞书的 node_token）
	ObjToken string // 实际对象实体的标识（例如飞书的 obj_token）
	ParentID string // 父节点标识
}

// Provider 定义了一个知识库平台所需实现的核心能力
type Provider interface {
	// PlatformName 返回平台的名称（如 "feishu", "notion"）
	PlatformName() string

	// ExtractRoot 解析用户传入的 URL，并提取目标根节点的上下文信息
	// 例如从 Wiki URL 中提取 SpaceID 和 NodeToken
	ExtractRoot(wikiURL string) (spaceID string, parentID string, err error)

	// CreateDir 在指定空间和父级下创建一个目录/文件夹节点
	CreateDir(spaceID string, parentID string, name string) (*Node, error)

	// CreateDocument 在指定空间和父级下创建一个文档记录，并将其本地文件写入知识库
	CreateDocument(spaceID string, parentID string, filePath string, fileName string, useCodeBlock bool) (*Node, error)

	// UpdateDocument 清空并重写已有文档的内容（objToken 为平台文档的 obj_token）
	UpdateDocument(objToken string, filePath string, fileName string, useCodeBlock bool) error
}
