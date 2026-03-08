package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

const internalTokenURL = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"

// TokenManager 管理飞书租户级别的 Access Token
type TokenManager struct {
	appID     string
	appSecret string

	token    string
	expireAt time.Time
	mu       sync.Mutex

	client *resty.Client
}

// NewTokenManager 初始化凭证管理器，按环境变量 -> ~/.wikitnow 配置文件顺序取值
func NewTokenManager() (*TokenManager, error) {
	appID := os.Getenv("FEISHU_APP_ID")
	appSecret := os.Getenv("FEISHU_APP_SECRET")

	if appID == "" || appSecret == "" {
		// 尝试从全局配置 fallback
		id, secret := readFswikiConfig()
		if id != "" && secret != "" {
			appID = id
			appSecret = secret
		}
	}

	if appID == "" || appSecret == "" {
		return nil, errors.New("缺乏飞书凭证: 请配置全局环境变量 FEISHU_APP_ID / FEISHU_APP_SECRET，或配置 ~/.wikitnow/credentials.json")
	}

	return &TokenManager{
		appID:     appID,
		appSecret: appSecret,
		client:    resty.New().SetTimeout(10 * time.Second),
	}, nil
}

// GetToken 获取当前有效的 Token，并在需要时自动刷新
func (m *TokenManager) GetToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 提前 60 秒判定过期
	if m.token != "" && time.Now().Before(m.expireAt.Add(-60*time.Second)) {
		return m.token, nil
	}

	type responseStruct struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}

	var respBody responseStruct
	resp, err := m.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"app_id":     m.appID,
			"app_secret": m.appSecret,
		}).
		SetResult(&respBody).
		Post(internalTokenURL)

	if err != nil {
		return "", err
	}

	if resp.IsError() || respBody.Code != 0 {
		return "", errors.New("Token 请求失败: " + resp.String())
	}

	m.token = respBody.TenantAccessToken
	m.expireAt = time.Now().Add(time.Duration(respBody.Expire) * time.Second)

	return m.token, nil
}

// readWikitnowConfig 尝试从 ~/.wikitnow/credentials.json 读取配置
func readFswikiConfig() (string, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}
	configPath := filepath.Join(home, ".wikitnow", "credentials.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", ""
	}

	var config struct {
		AppID     string `json:"app_id"`
		AppSecret string `json:"app_secret"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return "", ""
	}

	return config.AppID, config.AppSecret
}
