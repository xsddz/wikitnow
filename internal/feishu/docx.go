package feishu

import (
	"fmt"
)

const docxAPIBase = "https://open.feishu.cn/open-apis/docx/v1"

// InsertCodeBlock 往指定的 ObjToken (docx 等类型文档) 中插入 Code Block
// maxCodeBlockChars 飞书单个代码块 text_run content 的字符上限。
const maxCodeBlockChars = 99000

func (c *Client) InsertCodeBlock(docToken, code string) error {
	for _, chunk := range splitByNaturalBoundary(code, maxCodeBlockChars) {
		if err := c.insertSingleCodeBlock(docToken, chunk); err != nil {
			return err
		}
	}
	return nil
}

// splitByNaturalBoundary 将 text 按自然段（空行）或退而求其次按换行拆分，
// 确保每片的 Unicode 字符数不超过 limit。
func splitByNaturalBoundary(text string, limit int) []string {
	if len([]rune(text)) <= limit {
		return []string{text}
	}

	var chunks []string
	remaining := text
	for len([]rune(remaining)) > 0 {
		runes := []rune(remaining)
		if len(runes) <= limit {
			chunks = append(chunks, remaining)
			break
		}

		// 在 limit 以内，优先找最后一个空行分隔符（自然段边界）
		window := string(runes[:limit])
		cutAt := -1

		// 尝试在空行处切分
		if idx := lastIndex(window, "\n\n"); idx > 0 {
			cutAt = idx + 2 // 保留空行本身，切到空行之后
		}
		// 退而求其次：在换行处切分
		if cutAt < 0 {
			if idx := lastIndex(window, "\n"); idx > 0 {
				cutAt = idx + 1
			}
		}
		// 兜底：硬切
		if cutAt < 0 {
			cutAt = len([]rune(window))
		}

		chunks = append(chunks, string([]rune(remaining)[:cutAt]))
		remaining = string([]rune(remaining)[cutAt:])
	}
	return chunks
}

// lastIndex 返回 sep 在 s 中最后一次出现的字节偏移，未找到返回 -1。
// 使用字节偏移是因为后续用 rune 切片重新定位，这里只用于找"切点"所在的 rune 索引。
func lastIndex(s, sep string) int {
	sr := []rune(s)
	sepr := []rune(sep)
	for i := len(sr) - len(sepr); i >= 0; i-- {
		if string(sr[i:i+len(sepr)]) == sep {
			return i
		}
	}
	return -1
}

func (c *Client) insertSingleCodeBlock(docToken, code string) error {
	// 往根节点插入，parent_id 等于 docToken
	url := fmt.Sprintf("%s/documents/%s/blocks/%s/children", docxAPIBase, docToken, docToken)

	payload := map[string]interface{}{
		"children": []map[string]interface{}{
			{
				"block_type": 14, // 14 代表 code block
				"code": map[string]interface{}{
					"elements": []map[string]interface{}{
						{
							"text_run": map[string]interface{}{
								"content": code,
							},
						},
					},
				},
			},
		},
	}

	resp, err := c.resty.R().
		SetBody(payload).
		Post(url)

	if err := parseResponse(resp, err); err != nil {
		return fmt.Errorf("InsertCodeBlock failed: %w", err)
	}

	return nil
}

// InsertTextBlock 往指定的 ObjToken (docx 等类型文档) 中插入纯文本 Block (Text)
func (c *Client) InsertTextBlock(docToken, text string) error {
	// 往根节点插入，parent_id 等于 docToken
	url := fmt.Sprintf("%s/documents/%s/blocks/%s/children", docxAPIBase, docToken, docToken)

	// docx 插入要求特定的 body 结构, block_type 2 代表纯文本文本块 (Text)
	payload := map[string]interface{}{
		"children": []map[string]interface{}{
			{
				"block_type": 2, // 2 代表 Text block
				"text": map[string]interface{}{
					"elements": []map[string]interface{}{
						{
							"text_run": map[string]interface{}{
								"content": text,
							},
						},
					},
				},
			},
		},
	}

	resp, err := c.resty.R().
		SetBody(payload).
		Post(url)

	if err := parseResponse(resp, err); err != nil {
		return fmt.Errorf("InsertTextBlock failed: %w", err)
	}

	return nil
}

// ReadDocxContent 简单读取 Docx 文本内容（目前仅获取纯文本 Markdown）
func (c *Client) ReadDocxContent(docToken string) (string, error) {
	url := "https://open.feishu.cn/open-apis/docs/v1/content"

	type responseStruct struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}

	var respBody responseStruct
	resp, err := c.resty.R().
		SetQueryParam("doc_token", docToken).
		SetQueryParam("doc_type", "docx").
		SetQueryParam("content_type", "markdown").
		SetResult(&respBody).
		Get(url)

	if err := parseResponse(resp, err); err != nil {
		return "", fmt.Errorf("ReadDocxContent failed: %w", err)
	}

	return respBody.Data.Content, nil
}

// ConvertContentToBlocks 将 Markdown 或 HTML 文本转换为文档块。
// 返回第一层 block ID 列表和完整的 block 索引表，供调用方逐层创建。
func (c *Client) ConvertContentToBlocks(content string) (firstLevelIDs []string, blockMap map[string]map[string]interface{}, err error) {
	apiURL := fmt.Sprintf("%s/documents/blocks/convert", docxAPIBase)
	payload := map[string]interface{}{
		"content":      content,
		"content_type": "markdown",
	}

	type responseStruct struct {
		Data struct {
			FirstLevelBlockIDs []string                 `json:"first_level_block_ids"`
			Blocks             []map[string]interface{} `json:"blocks"`
		} `json:"data"`
	}

	var respBody responseStruct
	resp, respErr := c.resty.R().
		SetBody(payload).
		SetResult(&respBody).
		Post(apiURL)

	if err = parseResponse(resp, respErr); err != nil {
		return nil, nil, fmt.Errorf("ConvertContentToBlocks failed: %w", err)
	}

	blockMap = make(map[string]map[string]interface{}, len(respBody.Data.Blocks))
	for _, b := range respBody.Data.Blocks {
		if id, ok := b["block_id"].(string); ok {
			blockMap[id] = b
		}
	}
	return respBody.Data.FirstLevelBlockIDs, blockMap, nil
}

// ListBlockChildren 获取指定块的直接子块列表（自动翻页）。
func (c *Client) ListBlockChildren(docToken, blockToken string) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/documents/%s/blocks/%s/children", docxAPIBase, docToken, blockToken)

	type responseStruct struct {
		Data struct {
			Items     []map[string]interface{} `json:"items"`
			HasMore   bool                     `json:"has_more"`
			PageToken string                   `json:"page_token"`
		} `json:"data"`
	}

	var all []map[string]interface{}
	pageToken := ""
	for {
		req := c.resty.R().SetResult(&responseStruct{})
		if pageToken != "" {
			req = req.SetQueryParam("page_token", pageToken)
		}
		resp, err := req.Get(apiURL)
		var body responseStruct
		if parseErr := parseResponse(resp, err); parseErr != nil {
			return nil, fmt.Errorf("ListBlockChildren failed: %w", parseErr)
		}
		if r, ok := resp.Result().(*responseStruct); ok {
			body = *r
		}
		all = append(all, body.Data.Items...)
		if !body.Data.HasMore {
			break
		}
		pageToken = body.Data.PageToken
	}
	return all, nil
}

// CreateDocumentBlockDescendants 调用「创建嵌套块」接口。
// childrenID：第一层子块的临时 ID 列表（对应 convert API 返回的 first_level_block_ids）。
// descendants：所有块的扁平列表，每个块保留 block_id（临时 ID）、block_type、内容字段，
//
//	children 字段为子块临时 ID 的字符串数组（不是嵌套对象）。
//
// 单次最多 1000 块，遇到限流自动重试。
func (c *Client) CreateDocumentBlockDescendants(docToken, blockToken string, childrenID []string, descendants []map[string]interface{}) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/documents/%s/blocks/%s/descendant", docxAPIBase, docToken, blockToken)

	payload := map[string]interface{}{
		"children_id": childrenID,
		"descendants": descendants,
	}

	type responseStruct struct {
		Data struct {
			Children []map[string]interface{} `json:"children"`
		} `json:"data"`
	}

	var respBody responseStruct
	err := retryOnRateLimit(5, func() error {
		resp, err := c.resty.R().
			SetBody(payload).
			SetResult(&respBody).
			Post(apiURL)
		return parseResponse(resp, err)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateDocumentBlockDescendants failed: %w", err)
	}
	return respBody.Data.Children, nil
}

// CreateDocumentBlockChildren 批量创建嵌套块，遇到限流自动重试。
func (c *Client) CreateDocumentBlockChildren(docToken, blockToken string, children []map[string]interface{}) ([]map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/documents/%s/blocks/%s/children", docxAPIBase, docToken, blockToken)

	payload := map[string]interface{}{
		"children": children,
	}

	type responseStruct struct {
		Data struct {
			Children []map[string]interface{} `json:"children"`
		} `json:"data"`
	}

	var respBody responseStruct
	err := retryOnRateLimit(5, func() error {
		resp, err := c.resty.R().
			SetBody(payload).
			SetResult(&respBody).
			Post(apiURL)
		return parseResponse(resp, err)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateDocumentBlockChildren failed: %w", err)
	}
	return respBody.Data.Children, nil
}

// PatchDocumentBlock 更新块 (如替换图片)，遇到限流自动重试。
func (c *Client) PatchDocumentBlock(docToken, blockToken string, payload map[string]interface{}) error {
	apiURL := fmt.Sprintf("%s/documents/%s/blocks/%s", docxAPIBase, docToken, blockToken)

	err := retryOnRateLimit(5, func() error {
		resp, err := c.resty.R().
			SetBody(payload).
			Patch(apiURL)
		return parseResponse(resp, err)
	})
	if err != nil {
		return fmt.Errorf("PatchDocumentBlock failed: %w", err)
	}
	return nil
}
