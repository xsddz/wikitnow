package feishu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/xsddz/wikitnow/internal/auth"

	"github.com/go-resty/resty/v2"
)

// Client 封装对于飞书的 API 调用
type Client struct {
	resty *resty.Client
	authM *auth.TokenManager
	debug bool
}

// NewClient 初始化一个绑定了认证管理器的飞书 SDK。
// 当 debug 为 true 时，每次 HTTP 请求前会向 stderr 打印请求方法和 URL。
func NewClient(authManager *auth.TokenManager, debug bool) *Client {
	r := resty.New().
		SetTimeout(30*time.Second).
		SetHeader("Content-Type", "application/json; charset=utf-8")

	// 统一注入 Authorization 头部
	r.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		if authManager == nil {
			return fmt.Errorf("未配置凭证，无法发起网络请求")
		}
		token, err := authManager.GetToken()
		if err != nil {
			return err
		}
		req.SetHeader("Authorization", "Bearer "+token)
		return nil
	})

	client := &Client{
		resty: r,
		authM: authManager,
		debug: debug,
	}

	if debug {
		r.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
			fmt.Fprintf(os.Stderr, "[DEBUG] %s %s\n", req.Method, req.URL)
			if req.Body != nil {
				if b, err := json.Marshal(req.Body); err == nil {
					fmt.Fprintf(os.Stderr, "[DEBUG] Body: %s\n", string(b))
				}
			}
			return nil
		})
	}

	return client
}

// FeishuError 定义飞书 API 的标准错误响应
type FeishuError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *FeishuError) Error() string {
	return fmt.Sprintf("Feishu API Error: [%d] %s", e.Code, e.Msg)
}

// parseResponse 处理 Resty 响应，将其转换为内部错误
func parseResponse(resp *resty.Response, err error) error {
	if err != nil {
		return err
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusTooManyRequests {
			return &rateLimitError{}
		}
		var fe FeishuError
		if unmarshalErr := json.Unmarshal(resp.Body(), &fe); unmarshalErr == nil && fe.Code != 0 {
			return &fe
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	// 针对 code != 0 的业务错误处理 (飞书 API 规范)
	var fe FeishuError
	if unmarshalErr := json.Unmarshal(resp.Body(), &fe); unmarshalErr == nil && fe.Code != 0 {
		return &fe
	}

	return nil
}

// rateLimitError 表示 HTTP 429 限流错误
type rateLimitError struct{}

func (e *rateLimitError) Error() string { return "rate limited (HTTP 429)" }

// isRateLimit 判断错误是否为限流
func isRateLimit(err error) bool {
	_, ok := err.(*rateLimitError)
	return ok
}

// retryOnRateLimit 在遇到 429 时以指数退避最多重试 maxRetries 次执行 fn。
func retryOnRateLimit(maxRetries int, fn func() error) error {
	wait := 2 * time.Second
	for attempt := 0; ; attempt++ {
		err := fn()
		if err == nil || !isRateLimit(err) || attempt >= maxRetries {
			return err
		}
		time.Sleep(wait)
		wait *= 2
	}
}
