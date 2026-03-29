package n9e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"ops-system/backend/internal/config"

	"go.uber.org/zap"
)

// Client 夜莺 / N9E HTTP 客户端（§2.4）。
type Client struct {
	cfg    *config.N9EConfig
	log    *zap.Logger
	http   *http.Client
	mu     sync.Mutex
	token  string
	prefix string
}

// NewClient 创建客户端；cfg 为 nil 时 Enabled() 为 false。
func NewClient(cfg *config.N9EConfig, log *zap.Logger) *Client {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg == nil {
		cfg = &config.N9EConfig{}
	}
	sec := cfg.HTTPTimeoutSeconds
	if sec <= 0 {
		sec = 20
	}
	p := strings.TrimSuffix(cfg.APIPrefix, "/")
	if p == "" {
		p = "/api/n9e"
	}
	return &Client{
		cfg: cfg,
		log: log,
		http: &http.Client{
			Timeout: time.Duration(sec) * time.Second,
		},
		prefix: p,
	}
}

// Enabled 是否启用（需 base_url 与认证信息）。
func (c *Client) Enabled() bool {
	if c == nil || c.cfg == nil || !c.cfg.Enabled {
		return false
	}
	base := strings.TrimSpace(c.cfg.BaseURL)
	if base == "" {
		return false
	}
	if strings.TrimSpace(c.cfg.Token) != "" {
		return true
	}
	return strings.TrimSpace(c.cfg.Username) != "" && strings.TrimSpace(c.cfg.Password) != ""
}

func (c *Client) base() string {
	return strings.TrimRight(strings.TrimSpace(c.cfg.BaseURL), "/")
}

func (c *Client) ensureToken(ctx context.Context) error {
	if strings.TrimSpace(c.cfg.Token) != "" {
		c.mu.Lock()
		c.token = strings.TrimSpace(c.cfg.Token)
		c.mu.Unlock()
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" {
		return nil
	}
	return c.loginLocked(ctx)
}

func (c *Client) refreshToken(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	_ = c.loginLocked(ctx)
}

func (c *Client) loginLocked(ctx context.Context) error {
	u := c.base() + c.prefix + "/auth/login"
	body := map[string]string{
		"username": c.cfg.Username,
		"password": c.cfg.Password,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("n9e login http %d: %s", resp.StatusCode, string(b))
	}
	var wrap struct {
		Err string          `json:"err"`
		Dat json.RawMessage `json:"dat"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return err
	}
	if wrap.Err != "" {
		return errors.New(wrap.Err)
	}
	tok, err := parseTokenFromDat(wrap.Dat)
	if err != nil {
		return err
	}
	c.token = tok
	c.log.Info("n9e_login_ok")
	return nil
}

func parseTokenFromDat(dat json.RawMessage) (string, error) {
	if len(dat) == 0 {
		return "", errors.New("empty dat")
	}
	var s string
	if err := json.Unmarshal(dat, &s); err == nil && s != "" {
		return s, nil
	}
	var m map[string]any
	if err := json.Unmarshal(dat, &m); err != nil {
		return "", err
	}
	for _, k := range []string{"access_token", "token", "accessToken"} {
		if v, ok := m[k].(string); ok && v != "" {
			return v, nil
		}
	}
	return "", errors.New("no token in dat")
}

// RequestJSON 调用 N9E 接口并返回原始 dat。遇到认证失败自动刷新 token 重试一次。
func (c *Client) RequestJSON(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	dat, err := c.doRequest(ctx, method, path, body)
	if err != nil && isAuthError(err) {
		c.refreshToken(ctx)
		return c.doRequest(ctx, method, path, body)
	}
	return dat, err
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "密码错误") ||
		strings.Contains(s, "unauthorized") ||
		strings.Contains(s, "http 401") ||
		strings.Contains(s, "token") && strings.Contains(s, "invalid")
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(raw)
	}
	u := c.base() + path
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("n9e %s %s http %d: %s", method, path, resp.StatusCode, string(b))
	}
	var wrap struct {
		Err string          `json:"err"`
		Dat json.RawMessage `json:"dat"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	if wrap.Err != "" {
		return nil, errors.New(wrap.Err)
	}
	return wrap.Dat, nil
}

// doJSON POST/DELETE 等，自动带 Bearer，将 dat 解码到 out。
func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	dat, err := c.RequestJSON(ctx, method, path, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if len(dat) == 0 || string(dat) == "null" {
		return nil
	}
	return json.Unmarshal(dat, out)
}
