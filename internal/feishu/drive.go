package feishu

import (
	"bytes"
	"fmt"
	"mime/multipart"
)

const driveAPIBase = "https://open.feishu.cn/open-apis/drive/v1"

// UploadMedia 上传素材。如果是 docs 图片素材，parentType=docx_image, parentNode 为 Docx Image Block ID
func (c *Client) UploadMedia(fileName, parentType, parentNode string, size int, fileData []byte) (string, error) {
	url := fmt.Sprintf("%s/medias/upload_all", driveAPIBase)

	// Build multipart payload in memory
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Write metadata fields
	_ = writer.WriteField("file_name", fileName)
	_ = writer.WriteField("parent_type", parentType)
	_ = writer.WriteField("parent_node", parentNode)
	_ = writer.WriteField("size", fmt.Sprintf("%d", size))

	// Write file part
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("create form file failed: %w", err)
	}
	part.Write(fileData)

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("close multipart writer failed: %w", err)
	}

	type responseStruct struct {
		Data struct {
			FileToken string `json:"file_token"`
		} `json:"data"`
	}

	var respBody responseStruct
	resp, err := c.resty.R().
		SetHeader("Content-Type", writer.FormDataContentType()).
		SetBody(body.Bytes()).
		SetResult(&respBody).
		Post(url)

	if err := parseResponse(resp, err); err != nil {
		return "", fmt.Errorf("UploadMedia failed: %w", err)
	}

	return respBody.Data.FileToken, nil
}
