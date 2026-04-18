package k8s

import (
	"sync"

	"github.com/google/uuid"
)

// ClusterClientCache 按 cluster_id 缓存已经初始化好的 *Client，避免每次 Apply 重新做 discovery。
// 使用 fingerprint 字符串检测后端配置变化后失效。对外方法全部线程安全。
type ClusterClientCache struct {
	mu sync.RWMutex
	m  map[uuid.UUID]cacheEntry
}

type cacheEntry struct {
	client *Client
	fp     string
}

// NewClusterClientCache 构造一个空缓存。
func NewClusterClientCache() *ClusterClientCache {
	return &ClusterClientCache{m: make(map[uuid.UUID]cacheEntry)}
}

// Get 查询指定 clusterID 对应的 client；若 fingerprint 不匹配视为失效并返回 nil。
func (c *ClusterClientCache) Get(id uuid.UUID, fp string) *Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.m[id]; ok && entry.fp == fp {
		return entry.client
	}
	return nil
}

// Put 写入 / 更新缓存。
func (c *ClusterClientCache) Put(id uuid.UUID, fp string, client *Client) {
	if client == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[id] = cacheEntry{client: client, fp: fp}
}

// Invalidate 删除指定 clusterID 的缓存（例如配置变更后）。
func (c *ClusterClientCache) Invalidate(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, id)
}

// Clear 清空全部缓存。
func (c *ClusterClientCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[uuid.UUID]cacheEntry)
}
