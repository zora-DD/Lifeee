package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"life-core/service"
)

const codeToSessionURL = "https://api.weixin.qq.com/sns/jscode2session"

// Client 调用微信 jscode2session 接口。
type Client struct {
	appID      string
	appSecret  string
	httpClient *http.Client
	endpoint   string
}

func NewClient(appID, appSecret string) *Client {
	return &Client{
		appID: strings.TrimSpace(appID), appSecret: strings.TrimSpace(appSecret),
		httpClient: &http.Client{Timeout: 5 * time.Second}, endpoint: codeToSessionURL,
	}
}

func (c *Client) Exchange(ctx context.Context, code string) (*service.WechatSession, error) {
	if c == nil || c.appID == "" || c.appSecret == "" || strings.TrimSpace(code) == "" {
		return nil, errors.New("wechat: app credentials and code are required")
	}
	params := url.Values{
		"appid":      {c.appID},
		"secret":     {c.appSecret},
		"js_code":    {strings.TrimSpace(code)},
		"grant_type": {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("wechat: create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("wechat: request code2session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wechat: code2session status %d", resp.StatusCode)
	}
	var payload struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("wechat: decode code2session: %w", err)
	}
	if payload.ErrCode != 0 || payload.OpenID == "" {
		return nil, fmt.Errorf("wechat: code2session failed: %d %s", payload.ErrCode, payload.ErrMsg)
	}
	return &service.WechatSession{OpenID: payload.OpenID, UnionID: payload.UnionID}, nil
}

// MockClient 仅供本地联调，不调用微信服务器。
type MockClient struct {
	OpenID string
}

func (c *MockClient) Exchange(_ context.Context, code string) (*service.WechatSession, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(c.OpenID) == "" {
		return nil, errors.New("wechat: mock openid and code are required")
	}
	return &service.WechatSession{OpenID: strings.TrimSpace(c.OpenID)}, nil
}
