package feishu

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xsddz/wikitnow/internal/feishu"
	"github.com/xsddz/wikitnow/internal/provider"
	url "github.com/xsddz/wikitnow/internal/utils"
)

type FeishuProvider struct {
	client *feishu.Client
}

// NewProvider 创建一个新的飞书平台提供者
func NewProvider(client *feishu.Client) *FeishuProvider {
	return &FeishuProvider{
		client: client,
	}
}

func (p *FeishuProvider) PlatformName() string {
	return "feishu"
}

// ExtractRoot 提取目标节点 Token 和对应的 SpaceID
func (p *FeishuProvider) ExtractRoot(wikiURL string) (string, string, error) {
	parentNodeToken, err := url.ExtractNodeToken(wikiURL)
	if err != nil {
		return "", "", fmt.Errorf("URL 解析失败: %w", err)
	}

	spaceID, err := p.client.GetSpaceID(parentNodeToken)
	if err != nil {
		return "", "", fmt.Errorf("获取知识库空间失败: %w", err)
	}

	return spaceID, parentNodeToken, nil
}

// CreateDir 建立远端目录节点
func (p *FeishuProvider) CreateDir(spaceID, parentID, name string) (*provider.Node, error) {
	res, err := p.client.CreateNode(spaceID, parentID, name, "docx")
	if err != nil {
		return nil, fmt.Errorf("建立远端目录节点失败 %s: %w", name, err)
	}
	return &provider.Node{
		ID:       res.NodeToken,
		ObjToken: res.ObjToken,
		ParentID: parentID,
	}, nil
}

// CreateDocument 创建文档并写入本地文件内容
func (p *FeishuProvider) CreateDocument(spaceID, parentID, filePath, fileName string, useCodeBlock bool) (*provider.Node, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	res, err := p.client.CreateNode(spaceID, parentID, fileName, "docx")
	if err != nil {
		return nil, err
	}

	var insertErr error
	if useCodeBlock {
		insertErr = p.client.InsertCodeBlock(res.ObjToken, string(content))
	} else if p.isMarkdown(fileName) {
		insertErr = p.uploadMarkdownAsBlocks(res.ObjToken, filePath, string(content))
	} else {
		insertErr = p.client.InsertTextBlock(res.ObjToken, string(content))
	}

	if insertErr != nil {
		return nil, insertErr
	}

	return &provider.Node{
		ID:       res.NodeToken,
		ObjToken: res.ObjToken,
		ParentID: parentID,
	}, nil
}

// UpdateDocument 清空并重写已有文档的全部内容（复用已有写入逻辑）。
func (p *FeishuProvider) UpdateDocument(objToken, filePath, fileName string, useCodeBlock bool) error {
	if err := p.client.ClearDocumentContent(objToken); err != nil {
		return fmt.Errorf("清空文档内容失败: %w", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if useCodeBlock {
		return p.client.InsertCodeBlock(objToken, string(content))
	} else if p.isMarkdown(fileName) {
		return p.uploadMarkdownAsBlocks(objToken, filePath, string(content))
	}
	return p.client.InsertTextBlock(objToken, string(content))
}

func (p *FeishuProvider) isMarkdown(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown"
}

// detectContentType 根据文件名后缀推断内容类型。
// 支持的后缀：
//   - .md, .markdown → "markdown"
//   - .html, .htm    → "html"
//   - 其他           → "markdown"（默认）
func (p *FeishuProvider) detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".html", ".htm":
		return "html"
	case ".md", ".markdown":
		return "markdown"
	default:
		return "markdown" // 默认为 markdown
	}
}

// uploadMarkdownAsBlocks 将 Markdown/HTML 内容写入文档块，遵循飞书「创建嵌套块」API 规范：
// 1. 根据文件后缀检测 contentType（.html → "html", 其他 → "markdown"）
// 2. ConvertContentToBlocks 转换，得到扁平块列表（blocks）和第一层 ID（first_level_block_ids）
// 3. 对每个块：去掉 parent_id（非 API 字段），去掉 Table.merge_info，清理 Image 只读字段
// 4. 以 children_id + descendants（扁平列表）格式调用 /descendant 接口，单次最多 1000 块
// 5. 以 Image BlockID 为 parent_node 上传图片素材，再通过 replace_image 写入 token
func (p *FeishuProvider) uploadMarkdownAsBlocks(docToken, filePath, content string) error {
	imageSources := extractAllImageSources(content)

	// 根据文件后缀推断内容类型
	contentType := p.detectContentType(filePath)

	firstLevelIDs, blockMap, err := p.client.ConvertContentToBlocks(content, contentType)
	if err != nil {
		return fmt.Errorf("转换 Markdown 到文档块失败: %w", err)
	}

	// 对每个块做清理，结果存入 sanitizedMap（以临时 block_id 为 key）
	sanitizedMap := make(map[string]map[string]interface{}, len(blockMap))
	for id, block := range blockMap {
		clean := make(map[string]interface{}, len(block))
		for k, v := range block {
			clean[k] = v
		}
		delete(clean, "parent_id") // parent_id 不是 descendants API 的合法字段
		if bt, _ := clean["block_type"].(float64); int(bt) == 31 {
			if table, ok := clean["table"].(map[string]interface{}); ok {
				delete(table, "merge_info") // 只读字段，传入会报错
				delete(table, "cells")      // 临时 ID 数组，descendants API 不接受
				// property 只保留 row_size / column_size，其余（column_width、merge_info 等）均为只读字段
				if prop, ok := table["property"].(map[string]interface{}); ok {
					cleaned := map[string]interface{}{}
					if v, ok := prop["row_size"]; ok {
						cleaned["row_size"] = v
					}
					if v, ok := prop["column_size"]; ok {
						cleaned["column_size"] = v
					}
					table["property"] = cleaned
				}
			}
		}
		sanitizeBlockForCreate(clean)
		sanitizedMap[id] = clean
	}

	// collectSubtree 递归收集一个块及其全部子树，结果为扁平列表
	var collectSubtree func(id string, visited map[string]bool) []map[string]interface{}
	collectSubtree = func(id string, visited map[string]bool) []map[string]interface{} {
		if visited[id] {
			return nil
		}
		visited[id] = true
		block := sanitizedMap[id]
		if block == nil {
			return nil
		}
		result := []map[string]interface{}{block}
		for _, childID := range blockChildIDs(blockMap[id]) {
			result = append(result, collectSubtree(childID, visited)...)
		}
		return result
	}

	// 将 firstLevelIDs 按子树大小分批，每批 descendants 不超过 1000 块
	type batchItem struct {
		childrenIDs []string
		descendants []map[string]interface{}
	}
	var batches []batchItem
	var cur batchItem
	visited := map[string]bool{}
	for _, id := range firstLevelIDs {
		subtree := collectSubtree(id, visited)
		if len(cur.descendants)+len(subtree) > 1000 && len(cur.childrenIDs) > 0 {
			batches = append(batches, cur)
			cur = batchItem{}
		}
		cur.childrenIDs = append(cur.childrenIDs, id)
		cur.descendants = append(cur.descendants, subtree...)
	}
	if len(cur.childrenIDs) > 0 {
		batches = append(batches, cur)
	}

	// 逐批调用「创建嵌套块」接口，响应的 children 是含真实 block_id 的扁平列表
	var allCreatedBlocks []map[string]interface{}
	for i, b := range batches {
		result, err := p.client.CreateDocumentBlockDescendants(docToken, docToken, b.childrenIDs, b.descendants)
		if err != nil {
			return fmt.Errorf("插入文档块 (批次 %d) 失败: %w", i, err)
		}
		allCreatedBlocks = append(allCreatedBlocks, result...)
		if i < len(batches)-1 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	p.processImageBlocks(docToken, filePath, allCreatedBlocks, imageSources)
	return nil
}

// blockChildIDs 从 block 的 children 字段中提取子块 ID 列表。
func blockChildIDs(block map[string]interface{}) []string {
	raw, ok := block["children"]
	if !ok {
		return nil
	}
	ids, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if s, ok := id.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// sanitizeBlockForCreate 清理 blockMap 中一个 block 内所有对 create API 无效的字段（原地修改）。
func sanitizeBlockForCreate(block map[string]interface{}) {
	bt, _ := block["block_type"].(float64)
	blockType := int(bt)

	switch blockType {
	case 14: // 代码块：language 0 无效，改为 1（PlainText）
		if code, ok := block["code"].(map[string]interface{}); ok {
			if lang, ok := code["language"].(float64); ok && int(lang) == 0 {
				code["language"] = float64(1)
			}
		}

	case 27: // 图片块：清除 convert 返回的只读字段，保留空 image 对象；后续通过 replace_image 写入实际 token
		// convert 返回的 image 对象包含 token、width、height 等只读字段，create API 不接受
		// 必须保留 "image": {} 以告知 API 这是图片块，否则 API 无法识别块类型
		block["image"] = map[string]interface{}{}

	case 32: // 表格单元格块：descendants API 示例中 table_cell 为空对象 {}
		// convert 响应可能携带额外字段，一律清空以匹配 API 要求
		block["table_cell"] = map[string]interface{}{}

	}

	// 对所有含 elements 的文本类块清理 text_element_style
	for _, key := range []string{
		"text", "heading1", "heading2", "heading3", "heading4",
		"heading5", "heading6", "heading7", "heading8", "heading9",
		"bullet", "ordered", "code", "quote", "todo",
	} {
		if textBlock, ok := block[key].(map[string]interface{}); ok {
			sanitizeTextBlockStyle(textBlock)
		}
	}
}

// sanitizeTextBlockStyle 清理文本块 style 和 elements 中无效字段（原地修改）。
func sanitizeTextBlockStyle(textBlock map[string]interface{}) {
	if style, ok := textBlock["style"].(map[string]interface{}); ok {
		// align: 合法值 1/2/3，0 表示未设置，删除让 API 使用默认值
		if v, ok := style["align"].(float64); ok && int(v) == 0 {
			delete(style, "align")
		}
		// background_color: 空字符串无效，删除
		if bg, ok := style["background_color"].(string); ok && bg == "" {
			delete(style, "background_color")
		}
		// indentation_level: 合法值 "NoIndent"/"OneLevelIndent"，空字符串无效，删除
		if v, ok := style["indentation_level"].(string); ok && v == "" {
			delete(style, "indentation_level")
		}
		// sequence: 有序列表编号，合法值为 "1"/"2"/... 或 "auto"，空字符串无效，删除
		if v, ok := style["sequence"].(string); ok && v == "" {
			delete(style, "sequence")
		}
	}
	// 清理 elements 中每个 text_run 的 text_element_style
	elems, ok := textBlock["elements"].([]interface{})
	if !ok {
		return
	}
	for _, raw := range elems {
		elem, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		textRun, ok := elem["text_run"].(map[string]interface{})
		if !ok {
			continue
		}
		style, ok := textRun["text_element_style"].(map[string]interface{})
		if !ok {
			continue
		}
		// background_color 和 text_color 值为 0 时无效（合法范围分别是 1-15 和 1-7）
		for _, numField := range []string{"background_color", "text_color"} {
			if v, ok := style[numField].(float64); ok && int(v) == 0 {
				delete(style, numField)
			}
		}
		// link.url 为空时删除整个 link
		if link, ok := style["link"].(map[string]interface{}); ok {
			if urlVal, _ := link["url"].(string); urlVal == "" {
				delete(style, "link")
			}
		}
		// create API 不支持传入 comment_ids（API 文档明确说明）
		delete(style, "comment_ids")
	}
}

// processImageBlocks 按位置将图片（本地或远程）上传并写入对应的 Image 块。
// imageSources 是从 Markdown/HTML 源码中按出现顺序提取的所有图片来源（路径或 URL）；
// insertedBlocks 中的 Image 块（type 27）与之一一对应。
func (p *FeishuProvider) processImageBlocks(docToken, filePath string, insertedBlocks []map[string]interface{}, imageSources []string) {
	if len(imageSources) == 0 {
		return
	}
	dir := filepath.Dir(filePath)
	imgIdx := 0
	for _, block := range insertedBlocks {
		if imgIdx >= len(imageSources) {
			break
		}
		blockType, ok := block["block_type"].(float64)
		if !ok || int(blockType) != 27 {
			continue
		}
		blockID, ok := block["block_id"].(string)
		if !ok || blockID == "" {
			imgIdx++
			continue
		}

		src := imageSources[imgIdx]
		imgIdx++

		var fileData []byte
		var fileName string
		var err error

		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			fileData, err = downloadImage(src)
			if err != nil {
				continue
			}
			fileName = path.Base(strings.SplitN(src, "?", 2)[0])
			if fileName == "." || fileName == "/" {
				fileName = "image.png"
			}
		} else {
			imagePath := filepath.Join(dir, src)
			fileData, err = os.ReadFile(imagePath)
			if err != nil {
				continue
			}
			fileName = filepath.Base(imagePath)
		}

		fileToken, err := p.client.UploadMedia(fileName, "docx_image", blockID, len(fileData), fileData)
		if err != nil || fileToken == "" {
			continue
		}
		patchPayload := map[string]interface{}{
			"replace_image": map[string]interface{}{
				"token": fileToken,
			},
		}
		_ = p.client.PatchDocumentBlock(docToken, blockID, patchPayload)
	}
}

// downloadImage 下载远程图片，返回字节内容。
func downloadImage(rawURL string) ([]byte, error) {
	resp, err := http.Get(rawURL) // #nosec G107 — URL 来自用户编写的 Markdown 文件
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d fetching image %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

// extractAllImageSources 从 Markdown/HTML 源码中按出现顺序提取所有图片来源（本地路径或远程 URL）。
// 同时支持 Markdown 语法 ![alt](src) 和 HTML <img src="..."> 标签。
var (
	mdImageRe   = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)[^)]*\)`)
	htmlImageRe = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["']`)
)

func extractAllImageSources(content string) []string {
	type pos struct {
		index int
		src   string
	}
	var all []pos

	for _, loc := range mdImageRe.FindAllStringSubmatchIndex(content, -1) {
		src := strings.TrimSpace(content[loc[2]:loc[3]])
		all = append(all, pos{loc[0], src})
	}
	for _, loc := range htmlImageRe.FindAllStringSubmatchIndex(content, -1) {
		src := strings.TrimSpace(content[loc[2]:loc[3]])
		all = append(all, pos{loc[0], src})
	}

	// 按在文档中的出现位置排序
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].index < all[j-1].index; j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}

	result := make([]string, 0, len(all))
	for _, p := range all {
		result = append(result, p.src)
	}
	return result
}
