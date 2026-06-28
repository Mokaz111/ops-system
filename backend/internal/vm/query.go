package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/model"
)

type QueryClient struct {
	cfg    *config.VMConfig
	http   *http.Client
	routes *RouteBuilder
}

func NewQueryClient(cfg *config.VMConfig) *QueryClient {
	if cfg == nil {
		cfg = &config.VMConfig{}
	}
	sec := cfg.HTTPTimeoutSeconds
	if sec <= 0 {
		sec = 15
	}
	return &QueryClient{
		cfg:    cfg,
		http:   &http.Client{Timeout: time.Duration(sec) * time.Second},
		routes: NewRouteBuilder(cfg),
	}
}

type QueryResult struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error,omitempty"`
}

type SampleValue struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
}

func (c *QueryClient) Enabled() bool {
	return c != nil && c.cfg != nil && strings.TrimSpace(c.cfg.VMAuthBaseURL) != ""
}

func (c *QueryClient) SelectURL(t *model.Workspace) string {
	if c == nil || t == nil {
		return ""
	}
	if strings.TrimSpace(t.VMSelectURL) != "" {
		return strings.TrimRight(t.VMSelectURL, "/")
	}
	routes := c.routes.BuildWorkspaceRoutes(t)
	return routes.SelectURL
}

func (c *QueryClient) Query(ctx context.Context, t *model.Workspace, query string, ts *time.Time) (*QueryResult, error) {
	v := url.Values{}
	v.Set("query", query)
	if ts != nil {
		v.Set("time", strconv.FormatInt(ts.Unix(), 10))
	}
	return c.get(ctx, t, "/api/v1/query", v)
}

func (c *QueryClient) QueryRange(ctx context.Context, t *model.Workspace, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	if step <= 0 {
		step = time.Minute
	}
	v := url.Values{}
	v.Set("query", query)
	v.Set("start", strconv.FormatInt(start.Unix(), 10))
	v.Set("end", strconv.FormatInt(end.Unix(), 10))
	v.Set("step", strconv.Itoa(int(step.Seconds())))
	return c.get(ctx, t, "/api/v1/query_range", v)
}

func (c *QueryClient) Scalar(ctx context.Context, t *model.Workspace, query string) (float64, error) {
	res, err := c.Query(ctx, t, query, nil)
	if err != nil {
		return 0, err
	}
	if res == nil || res.Status != "success" {
		if res != nil && res.Error != "" {
			return 0, errors.New(res.Error)
		}
		return 0, fmt.Errorf("victoriametrics query failed")
	}
	var data struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value []any `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(res.Data, &data); err != nil {
		return 0, err
	}
	if len(data.Result) == 0 || len(data.Result[0].Value) < 2 {
		return 0, nil
	}
	raw, ok := data.Result[0].Value[1].(string)
	if !ok {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, nil
	}
	return v, nil
}

func (c *QueryClient) get(ctx context.Context, t *model.Workspace, path string, values url.Values) (*QueryResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("victoriametrics query client disabled")
	}
	selectURL := c.SelectURL(t)
	if selectURL == "" {
		return nil, fmt.Errorf("tenant select url is empty")
	}
	u := APIURL(selectURL, path)
	if encoded := values.Encode(); encoded != "" {
		u += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if t != nil && t.VMUserID != "" && t.VMUserKey != "" {
		req.SetBasicAuth(t.VMUserID, t.VMUserKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("victoriametrics http %d: %s", resp.StatusCode, string(body))
	}
	var out QueryResult
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
