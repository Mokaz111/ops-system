package logstore

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	BackendVictoriaLogs = "victorialogs"
	BackendClickHouse   = "clickhouse"
	BackendDoris        = "doris"

	defaultQueryLimit = 100
	maxQueryLimit     = 1000
)

var (
	ErrUnsupportedBackend = errors.New("unsupported log backend")
	ErrEmptySelectURL     = errors.New("log select url is empty")
	ErrTenantRequired     = errors.New("tenant filter required")
)

// TenantFilter U-001 租户隔离字段。
type TenantFilter struct {
	TenantID    string
	Zone        string
	WorkspaceID string
	Cluster     string
}

// QueryRequest 统一日志查询请求。
type QueryRequest struct {
	TenantFilter TenantFilter
	RawQuery     string
	Start        *time.Time
	End          *time.Time
	Limit        int
}

// LogEntry 统一日志条目。
type LogEntry struct {
	Time    time.Time         `json:"time"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// QueryStats 查询统计。
type QueryStats struct {
	Returned int `json:"returned"`
	Limit    int `json:"limit"`
}

// QueryResult 统一查询结果。
type QueryResult struct {
	Entries []LogEntry `json:"entries"`
	Stats   QueryStats `json:"stats"`
}

// Config 存储后端连接配置。
type Config struct {
	SelectURL string
	InsertURL string
	Timeout   time.Duration
}

// Store 日志存储抽象接口。
type Store interface {
	Query(ctx context.Context, req QueryRequest) (*QueryResult, error)
	Ping(ctx context.Context) error
}

// New 按 backend_type 构造 Store 驱动。
func New(backendType string, cfg Config) (Store, error) {
	switch backendType {
	case "", BackendVictoriaLogs:
		return NewVictoriaLogsStore(cfg)
	case BackendClickHouse, BackendDoris:
		return nil, fmt.Errorf("%w: %s (not implemented)", ErrUnsupportedBackend, backendType)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBackend, backendType)
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultQueryLimit
	}
	if limit > maxQueryLimit {
		return maxQueryLimit
	}
	return limit
}

func validateTenantFilter(f TenantFilter) error {
	if f.TenantID == "" {
		return ErrTenantRequired
	}
	return nil
}
