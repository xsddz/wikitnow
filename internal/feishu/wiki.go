package feishu

import (
	"errors"
	"fmt"
)

const wikiAPIBase = "https://open.feishu.cn/open-apis/wiki/v2"

// NodeInfo 存放获取到的知识库节点数据
type NodeInfo struct {
	Type      string `json:"obj_type"`
	ObjToken  string `json:"obj_token"`
	Title     string `json:"title"`
	NodeToken string `json:"node_token"`
	SpaceID   string `json:"space_id"`
}

// GetDocInfo 获取节点详情
func (c *Client) GetDocInfo(token string) (*NodeInfo, error) {
	url := fmt.Sprintf("%s/spaces/get_node", wikiAPIBase)

	type responseStruct struct {
		Data struct {
			Node NodeInfo `json:"node"`
		} `json:"data"`
	}

	var respBody responseStruct
	resp, err := c.resty.R().
		SetQueryParam("token", token).
		SetResult(&respBody).
		Get(url)

	if err := parseResponse(resp, err); err != nil {
		return nil, fmt.Errorf("GetDocInfo failed: %w", err)
	}

	return &respBody.Data.Node, nil
}

// GetSpaceID 从节点 token 中提前解析 Space ID
func (c *Client) GetSpaceID(parentToken string) (string, error) {
	docInfo, err := c.GetDocInfo(parentToken)
	if err != nil {
		return "", err
	}
	if docInfo.SpaceID == "" {
		return "", errors.New("无法从父节点获取 Space ID")
	}
	return docInfo.SpaceID, nil
}

// CreateNodeResult 返回创建结果
type CreateNodeResult struct {
	NodeToken string
	ObjToken  string
}

// CreateNode 在目标空间及父节点下创建新页面 (仅支持 docx)
func (c *Client) CreateNode(spaceID, parentToken, title, objType string) (*CreateNodeResult, error) {
	url := fmt.Sprintf("%s/spaces/%s/nodes", wikiAPIBase, spaceID)

	// 截断标题，飞书限制最大 100 字符
	if len([]rune(title)) > 100 {
		title = string([]rune(title)[:97]) + "..."
	}

	payload := map[string]string{
		"node_type":         "origin", // 直接新建
		"title":             title,
		"parent_node_token": parentToken,
		"obj_type":          objType, // 默认为 "docx"
	}

	type responseStruct struct {
		Data struct {
			Node struct {
				NodeToken string `json:"node_token"`
				ObjToken  string `json:"obj_token"`
			} `json:"node"`
		} `json:"data"`
	}

	var respBody responseStruct
	resp, err := c.resty.R().
		SetBody(payload).
		SetResult(&respBody).
		Post(url)

	if err := parseResponse(resp, err); err != nil {
		return nil, fmt.Errorf("CreateNode failed: %w", err)
	}

	return &CreateNodeResult{
		NodeToken: respBody.Data.Node.NodeToken,
		ObjToken:  respBody.Data.Node.ObjToken,
	}, nil
}
