package url

import (
	"errors"
	"net/url"
	"strings"
)

// ExtractNodeToken 解析飞书 Wiki URL 并提取 node_token
// 支持的格式:
// 1. https://my.feishu.cn/wiki/SrR9whoxWiIcuuknF9Rc0TyWnxh
// 2. https://my.feishu.cn/wiki/wikcnXyz123?node=SrR9whoxWiIcuuknF9Rc0TyWnxh
func ExtractNodeToken(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// 1. 尝试从 ?node= 参数读取
	nodeParam := u.Query().Get("node")
	if nodeParam != "" {
		return nodeParam, nil
	}

	// 2. 尝试从路径中提取 /wiki/{token}
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) >= 2 && pathParts[0] == "wiki" {
		return pathParts[1], nil
	}

	return "", errors.New("无法从提供的 URL 中提取 node_token，请确保格式正确 (包含 /wiki/xxx 或 ?node=xxx)")
}
