package logstore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// VictoriaLogsStore VictoriaLogs vlselect 驱动。
type VictoriaLogsStore struct {
	selectURL string
	http      *http.Client
}

func NewVictoriaLogsStore(cfg Config) (*VictoriaLogsStore, error) {
	selectURL := strings.TrimSpace(cfg.SelectURL)
	if selectURL == "" {
		return nil, ErrEmptySelectURL
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &VictoriaLogsStore{
		selectURL: strings.TrimRight(selectURL, "/"),
		http:      &http.Client{Timeout: timeout},
	}, nil
}

func (s *VictoriaLogsStore) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.selectURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("victorialogs ping http %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *VictoriaLogsStore) Query(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	if err := validateTenantFilter(req.TenantFilter); err != nil {
		return nil, err
	}
	limit := normalizeLimit(req.Limit)
	query := buildLogsQL(req)

	values := url.Values{}
	values.Set("query", query)
	values.Set("limit", strconv.Itoa(limit))
	if req.Start != nil {
		values.Set("start", req.Start.UTC().Format(time.RFC3339Nano))
	}
	if req.End != nil {
		values.Set("end", req.End.UTC().Format(time.RFC3339Nano))
	}

	u := s.selectURL + "/select/logsql/query?" + values.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("victorialogs query http %d: %s", resp.StatusCode, string(body))
	}

	entries, err := parseVictoriaLogsNDJSON(resp.Body)
	if err != nil {
		return nil, err
	}
	return &QueryResult{
		Entries: entries,
		Stats: QueryStats{
			Returned: len(entries),
			Limit:    limit,
		},
	}, nil
}

// buildLogsQL 强制拼接租户过滤（U-001）。
func buildLogsQL(req QueryRequest) string {
	parts := []string{fmt.Sprintf(`ops_tenant_id:"%s"`, escapeLogsQLValue(req.TenantFilter.TenantID))}
	if z := strings.TrimSpace(req.TenantFilter.Zone); z != "" {
		parts = append(parts, fmt.Sprintf(`ops_zone:"%s"`, escapeLogsQLValue(z)))
	}
	if ws := strings.TrimSpace(req.TenantFilter.WorkspaceID); ws != "" {
		parts = append(parts, fmt.Sprintf(`ops_workspace:"%s"`, escapeLogsQLValue(ws)))
	}
	if cl := strings.TrimSpace(req.TenantFilter.Cluster); cl != "" {
		parts = append(parts, fmt.Sprintf(`ops_cluster:"%s"`, escapeLogsQLValue(cl)))
	}
	tenantFilter := strings.Join(parts, " AND ")
	raw := strings.TrimSpace(req.RawQuery)
	if raw == "" || raw == "*" {
		return tenantFilter
	}
	return tenantFilter + " AND (" + raw + ")"
}

func escapeLogsQLValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	return strings.ReplaceAll(v, `"`, `\"`)
}

func parseVictoriaLogsNDJSON(r io.Reader) ([]LogEntry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var entries []LogEntry
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		entry, err := parseVictoriaLogsLine(line)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func parseVictoriaLogsLine(line []byte) (LogEntry, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return LogEntry{}, err
	}
	entry := LogEntry{Fields: make(map[string]string)}
	for k, v := range raw {
		switch k {
		case "_time", "time", "@timestamp":
			var ts string
			if err := json.Unmarshal(v, &ts); err == nil {
				if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
					entry.Time = t
				}
			}
		case "_msg", "message", "msg":
			var msg string
			if err := json.Unmarshal(v, &msg); err == nil {
				entry.Message = msg
			}
		default:
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				entry.Fields[k] = s
			} else {
				// 数字/布尔/嵌套对象：保留原始 JSON 文本，避免字段丢失。
				entry.Fields[k] = string(v)
			}
		}
	}
	if entry.Time.IsZero() {
		entry.Time = time.Now().UTC()
	}
	return entry, nil
}
