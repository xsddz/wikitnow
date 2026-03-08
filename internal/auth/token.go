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

// NewTokenManager 初始化凭证管理器，按环境变量 -> ~/.wikitnow/config.json 顺序取值
func NewTokenManager() (*TokenManager, error) {
	appID := os.Getenv("WIKITNOW_FEISHU_APP_ID")
	appSecret := os.Getenv("WIKITNOW_FEISHU_APP_SECRET")

	if appID == "" || appSecret == "" {
		id, secret := readFeishuConfig()
		if id != "" && secret != "" {
			appID = id
			appSecret = secret
		}
	}

	if appID == "" || appSecret == "" {
		return nil, errors.New("飞书凭证未配置（环境变量 WIKITNOW_FEISHU_APP_ID / WIKITNOW_FEISHU_APP_SECRET，或 ~/.wikitnow/config.json）")
	}

	return &TokenManager{
		appID:     appID,
		appSecret: appSecret,
		client:    resty.New().SetTimeout(10 * time.Second),
	}, nil
}

// ConfigPath 返回全局配置文件路径
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".wikitnow", "config.json"), nil
}

// GlobalConfig 表示 ~/.wikitnow/config.json 的完整结构
type GlobalConfig struct {
	DefaultProvider string        `json:"default_provider,omitempty"`
	Feishu          *FeishuConfig `json:"feishu,omitempty"`
}

// FeishuConfig 表示飞书凭证配置
type FeishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// ReadGlobalConfig 读取并解析全局配置文件
func ReadGlobalConfig() (*GlobalConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// WriteGlobalConfig 将配置写入全局配置文件（权限 600）
func WriteGlobalConfig(cfg *GlobalConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
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

// readFeishuConfig 从 ~/.wikitnow/config.json 读取飞书凭证
func readFeishuConfig() (string, string) {
	cfg, err := ReadGlobalConfig()
	if err != nil || cfg.Feishu == nil {
		return "", ""
	}
	return cfg.Feishu.AppID, cfg.Feishu.AppSecret
}
