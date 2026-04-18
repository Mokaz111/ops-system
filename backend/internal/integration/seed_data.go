package integration

// SeededTemplate 单个种子模版（template 元信息 + 一个初始版本 spec）。
type SeededTemplate struct {
	Name        string
	DisplayName string
	Category    string
	Component   string
	Description string
	Icon        string
	Tags        []string
	Version     string
	Spec        TemplateSpec
	Changelog   string
}

// SeedTemplates 返回内置模版清单（幂等 upsert 依靠 Name）。
// M2 先提供 3 个：node / mysql / redis。
func SeedTemplates() []SeededTemplate {
	return []SeededTemplate{
		nodeExporterTemplate(),
		mysqldExporterTemplate(),
		redisExporterTemplate(),
	}
}

func nodeExporterTemplate() SeededTemplate {
	collectorYAML := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMPodScrape
metadata:
  name: node-exporter
  namespace: {{ .Values.namespace }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: node-exporter
  podMetricsEndpoints:
    - port: metrics
      interval: {{ .Values.scrape_interval }}
      path: /metrics
# metrics: node_cpu_seconds_total, node_load1, node_load5, node_load15, node_memory_MemTotal_bytes, node_memory_MemAvailable_bytes, node_filesystem_avail_bytes, node_filesystem_size_bytes, node_network_receive_bytes_total, node_network_transmit_bytes_total, node_disk_read_bytes_total, node_disk_written_bytes_total
`

	vmruleYAML := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMRule
metadata:
  name: node-exporter-alerts
  namespace: {{ .Values.namespace }}
spec:
  groups:
    - name: node-exporter
      rules:
        - alert: NodeHighCPU
          expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > {{ .Values.cpu_threshold }}
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "{{"{{ $labels.instance }}"}} CPU usage high"
        - alert: NodeLowMemory
          expr: (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 < {{ .Values.memory_threshold }}
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "{{"{{ $labels.instance }}"}} memory low"
        - alert: NodeFilesystemAlmostFull
          expr: (node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100 < 10
          for: 10m
          labels:
            severity: critical
`

	dashboardJSON := `{
  "uid": "node-exporter",
  "title": "Node Exporter / 主机监控",
  "panels": [
    {"id":1,"title":"CPU 使用率","targets":[{"expr":"100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"}]},
    {"id":2,"title":"负载","targets":[{"expr":"node_load1"},{"expr":"node_load5"},{"expr":"node_load15"}]},
    {"id":3,"title":"内存使用","targets":[{"expr":"(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100"}]},
    {"id":4,"title":"磁盘使用","targets":[{"expr":"(1 - node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100"}]},
    {"id":5,"title":"网络入","targets":[{"expr":"rate(node_network_receive_bytes_total[5m])"}]},
    {"id":6,"title":"网络出","targets":[{"expr":"rate(node_network_transmit_bytes_total[5m])"}]}
  ]
}`

	return SeededTemplate{
		Name:        "node-exporter",
		DisplayName: "Node Exporter 主机监控",
		Category:    "infra",
		Component:   "node",
		Description: "采集 Linux 主机 CPU / 内存 / 磁盘 / 网络等系统级指标。",
		Icon:        "server",
		Tags:        []string{"infra", "node", "host"},
		Version:     "v1.0.0",
		Changelog:   "初始版本：VMPodScrape + 基础告警 + 默认大盘。",
		Spec: TemplateSpec{
			Variables: []Variable{
				{Name: "namespace", Label: "命名空间", Type: "string", Default: "monitoring", Required: true},
				{Name: "scrape_interval", Label: "采集间隔", Type: "string", Default: "30s"},
				{Name: "cpu_threshold", Label: "CPU 告警阈值(%)", Type: "int", Default: "85"},
				{Name: "memory_threshold", Label: "剩余内存告警阈值(%)", Type: "int", Default: "10"},
			},
			Collector: CollectorSpec{
				Resources: []ResourceTemplate{
					{Kind: "VMPodScrape", APIVersion: "operator.victoriametrics.com/v1beta1", Name: "node-exporter", Manifest: collectorYAML},
				},
			},
			Alert: AlertSpec{
				Targets: []string{"vmrule"},
				VMRules: []ResourceTemplate{
					{Kind: "VMRule", APIVersion: "operator.victoriametrics.com/v1beta1", Name: "node-exporter-alerts", Manifest: vmruleYAML},
				},
			},
			Dashboards: []DashboardSpec{
				{UID: "node-exporter", Title: "Node Exporter / 主机监控", JSON: dashboardJSON},
			},
		},
	}
}

func mysqldExporterTemplate() SeededTemplate {
	collectorYAML := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMServiceScrape
metadata:
  name: mysqld-exporter
  namespace: {{ .Values.namespace }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: mysqld-exporter
  endpoints:
    - port: metrics
      interval: {{ .Values.scrape_interval }}
      path: /metrics
# metrics: mysql_up, mysql_global_status_threads_connected, mysql_global_status_threads_running, mysql_global_status_queries, mysql_global_status_slow_queries, mysql_global_status_innodb_buffer_pool_reads, mysql_global_status_innodb_buffer_pool_read_requests, mysql_global_status_commands_total, mysql_global_variables_max_connections
`

	vmruleYAML := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMRule
metadata:
  name: mysqld-exporter-alerts
  namespace: {{ .Values.namespace }}
spec:
  groups:
    - name: mysql
      rules:
        - alert: MySQLDown
          expr: mysql_up == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "MySQL {{"{{ $labels.instance }}"}} is down"
        - alert: MySQLTooManyConnections
          expr: mysql_global_status_threads_connected / mysql_global_variables_max_connections * 100 > {{ .Values.conn_threshold }}
          for: 5m
          labels:
            severity: warning
        - alert: MySQLSlowQueries
          expr: rate(mysql_global_status_slow_queries[5m]) > {{ .Values.slow_query_threshold }}
          for: 5m
          labels:
            severity: warning
`

	dashboardJSON := `{
  "uid": "mysqld-exporter",
  "title": "MySQL 监控",
  "panels": [
    {"id":1,"title":"MySQL 存活","targets":[{"expr":"mysql_up"}]},
    {"id":2,"title":"连接数","targets":[{"expr":"mysql_global_status_threads_connected"},{"expr":"mysql_global_status_threads_running"}]},
    {"id":3,"title":"QPS","targets":[{"expr":"rate(mysql_global_status_queries[5m])"}]},
    {"id":4,"title":"慢查询","targets":[{"expr":"rate(mysql_global_status_slow_queries[5m])"}]},
    {"id":5,"title":"InnoDB Buffer Pool 命中率","targets":[{"expr":"1 - rate(mysql_global_status_innodb_buffer_pool_reads[5m]) / rate(mysql_global_status_innodb_buffer_pool_read_requests[5m])"}]}
  ]
}`

	return SeededTemplate{
		Name:        "mysqld-exporter",
		DisplayName: "MySQL 监控",
		Category:    "db",
		Component:   "mysql",
		Description: "采集 MySQL 的连接数、QPS、慢查询、InnoDB 缓冲池等关键指标。",
		Icon:        "database",
		Tags:        []string{"db", "mysql"},
		Version:     "v1.0.0",
		Changelog:   "初始版本。",
		Spec: TemplateSpec{
			Variables: []Variable{
				{Name: "namespace", Label: "命名空间", Type: "string", Default: "monitoring", Required: true},
				{Name: "scrape_interval", Label: "采集间隔", Type: "string", Default: "30s"},
				{Name: "conn_threshold", Label: "连接使用率阈值(%)", Type: "int", Default: "80"},
				{Name: "slow_query_threshold", Label: "慢查询阈值(次/秒)", Type: "int", Default: "5"},
			},
			Collector: CollectorSpec{
				Resources: []ResourceTemplate{
					{Kind: "VMServiceScrape", APIVersion: "operator.victoriametrics.com/v1beta1", Name: "mysqld-exporter", Manifest: collectorYAML},
				},
			},
			Alert: AlertSpec{
				Targets: []string{"vmrule"},
				VMRules: []ResourceTemplate{
					{Kind: "VMRule", APIVersion: "operator.victoriametrics.com/v1beta1", Name: "mysqld-exporter-alerts", Manifest: vmruleYAML},
				},
			},
			Dashboards: []DashboardSpec{
				{UID: "mysqld-exporter", Title: "MySQL 监控", JSON: dashboardJSON},
			},
		},
	}
}

func redisExporterTemplate() SeededTemplate {
	collectorYAML := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMServiceScrape
metadata:
  name: redis-exporter
  namespace: {{ .Values.namespace }}
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: redis-exporter
  endpoints:
    - port: metrics
      interval: {{ .Values.scrape_interval }}
      path: /metrics
# metrics: redis_up, redis_connected_clients, redis_memory_used_bytes, redis_memory_max_bytes, redis_commands_processed_total, redis_keyspace_hits_total, redis_keyspace_misses_total, redis_evicted_keys_total, redis_net_input_bytes_total, redis_net_output_bytes_total
`

	vmruleYAML := `apiVersion: operator.victoriametrics.com/v1beta1
kind: VMRule
metadata:
  name: redis-exporter-alerts
  namespace: {{ .Values.namespace }}
spec:
  groups:
    - name: redis
      rules:
        - alert: RedisDown
          expr: redis_up == 0
          for: 2m
          labels:
            severity: critical
        - alert: RedisMemoryHigh
          expr: (redis_memory_used_bytes / redis_memory_max_bytes) * 100 > {{ .Values.memory_threshold }}
          for: 5m
          labels:
            severity: warning
        - alert: RedisEvictingKeys
          expr: rate(redis_evicted_keys_total[5m]) > 0
          for: 10m
          labels:
            severity: warning
`

	dashboardJSON := `{
  "uid": "redis-exporter",
  "title": "Redis 监控",
  "panels": [
    {"id":1,"title":"Redis 存活","targets":[{"expr":"redis_up"}]},
    {"id":2,"title":"连接客户端","targets":[{"expr":"redis_connected_clients"}]},
    {"id":3,"title":"内存使用率","targets":[{"expr":"(redis_memory_used_bytes / redis_memory_max_bytes) * 100"}]},
    {"id":4,"title":"命中率","targets":[{"expr":"rate(redis_keyspace_hits_total[5m]) / (rate(redis_keyspace_hits_total[5m]) + rate(redis_keyspace_misses_total[5m]))"}]},
    {"id":5,"title":"QPS","targets":[{"expr":"rate(redis_commands_processed_total[5m])"}]}
  ]
}`

	return SeededTemplate{
		Name:        "redis-exporter",
		DisplayName: "Redis 监控",
		Category:    "db",
		Component:   "redis",
		Description: "采集 Redis 的连接数、内存占用、命令速率与命中率等指标。",
		Icon:        "database",
		Tags:        []string{"db", "redis", "cache"},
		Version:     "v1.0.0",
		Changelog:   "初始版本。",
		Spec: TemplateSpec{
			Variables: []Variable{
				{Name: "namespace", Label: "命名空间", Type: "string", Default: "monitoring", Required: true},
				{Name: "scrape_interval", Label: "采集间隔", Type: "string", Default: "30s"},
				{Name: "memory_threshold", Label: "内存告警阈值(%)", Type: "int", Default: "85"},
			},
			Collector: CollectorSpec{
				Resources: []ResourceTemplate{
					{Kind: "VMServiceScrape", APIVersion: "operator.victoriametrics.com/v1beta1", Name: "redis-exporter", Manifest: collectorYAML},
				},
			},
			Alert: AlertSpec{
				Targets: []string{"vmrule"},
				VMRules: []ResourceTemplate{
					{Kind: "VMRule", APIVersion: "operator.victoriametrics.com/v1beta1", Name: "redis-exporter-alerts", Manifest: vmruleYAML},
				},
			},
			Dashboards: []DashboardSpec{
				{UID: "redis-exporter", Title: "Redis 监控", JSON: dashboardJSON},
			},
		},
	}
}

// DescribeMetric 返回内置指标的中文描述 / 单位 / 类型（用于 seed 时 enrich ops_metrics）。
func DescribeMetric(name string) (metricType, unit, descCN, descEN string) {
	if info, ok := metricCatalog[name]; ok {
		return info.Type, info.Unit, info.DescCN, info.DescEN
	}
	return "", "", "", ""
}

type metricInfo struct {
	Type   string
	Unit   string
	DescCN string
	DescEN string
}

var metricCatalog = map[string]metricInfo{
	// node
	"node_cpu_seconds_total":             {Type: "counter", Unit: "seconds", DescCN: "CPU 在各 mode 下累计运行秒数", DescEN: "CPU time per mode"},
	"node_load1":                         {Type: "gauge", Unit: "", DescCN: "1 分钟系统负载", DescEN: "1m load average"},
	"node_load5":                         {Type: "gauge", Unit: "", DescCN: "5 分钟系统负载", DescEN: "5m load average"},
	"node_load15":                        {Type: "gauge", Unit: "", DescCN: "15 分钟系统负载", DescEN: "15m load average"},
	"node_memory_MemTotal_bytes":         {Type: "gauge", Unit: "bytes", DescCN: "总内存", DescEN: "total memory"},
	"node_memory_MemAvailable_bytes":     {Type: "gauge", Unit: "bytes", DescCN: "可用内存", DescEN: "available memory"},
	"node_filesystem_avail_bytes":        {Type: "gauge", Unit: "bytes", DescCN: "文件系统可用空间", DescEN: "filesystem available bytes"},
	"node_filesystem_size_bytes":         {Type: "gauge", Unit: "bytes", DescCN: "文件系统总空间", DescEN: "filesystem size"},
	"node_network_receive_bytes_total":   {Type: "counter", Unit: "bytes", DescCN: "累计接收网络字节", DescEN: "received network bytes"},
	"node_network_transmit_bytes_total":  {Type: "counter", Unit: "bytes", DescCN: "累计发送网络字节", DescEN: "transmitted network bytes"},
	"node_disk_read_bytes_total":         {Type: "counter", Unit: "bytes", DescCN: "累计读磁盘字节", DescEN: "disk read bytes"},
	"node_disk_written_bytes_total":      {Type: "counter", Unit: "bytes", DescCN: "累计写磁盘字节", DescEN: "disk written bytes"},

	// mysql
	"mysql_up":                                       {Type: "gauge", Unit: "", DescCN: "MySQL 是否存活", DescEN: "MySQL up"},
	"mysql_global_status_threads_connected":          {Type: "gauge", Unit: "", DescCN: "已连接线程数", DescEN: "threads connected"},
	"mysql_global_status_threads_running":            {Type: "gauge", Unit: "", DescCN: "运行中线程数", DescEN: "threads running"},
	"mysql_global_status_queries":                    {Type: "counter", Unit: "", DescCN: "累计执行查询数", DescEN: "queries"},
	"mysql_global_status_slow_queries":               {Type: "counter", Unit: "", DescCN: "累计慢查询数", DescEN: "slow queries"},
	"mysql_global_status_innodb_buffer_pool_reads":   {Type: "counter", Unit: "", DescCN: "InnoDB 缓冲池物理读次数", DescEN: "innodb buffer pool physical reads"},
	"mysql_global_status_innodb_buffer_pool_read_requests": {Type: "counter", Unit: "", DescCN: "InnoDB 缓冲池逻辑读请求次数", DescEN: "innodb buffer pool read requests"},
	"mysql_global_status_commands_total":             {Type: "counter", Unit: "", DescCN: "按命令分类累计次数", DescEN: "commands total"},
	"mysql_global_variables_max_connections":         {Type: "gauge", Unit: "", DescCN: "最大连接数配置", DescEN: "max connections"},

	// redis
	"redis_up":                         {Type: "gauge", Unit: "", DescCN: "Redis 是否存活", DescEN: "Redis up"},
	"redis_connected_clients":          {Type: "gauge", Unit: "", DescCN: "已连接客户端数", DescEN: "connected clients"},
	"redis_memory_used_bytes":          {Type: "gauge", Unit: "bytes", DescCN: "已使用内存字节", DescEN: "used memory"},
	"redis_memory_max_bytes":           {Type: "gauge", Unit: "bytes", DescCN: "最大内存上限", DescEN: "max memory"},
	"redis_commands_processed_total":   {Type: "counter", Unit: "", DescCN: "累计执行命令数", DescEN: "commands processed"},
	"redis_keyspace_hits_total":        {Type: "counter", Unit: "", DescCN: "累计键命中数", DescEN: "keyspace hits"},
	"redis_keyspace_misses_total":      {Type: "counter", Unit: "", DescCN: "累计键未命中数", DescEN: "keyspace misses"},
	"redis_evicted_keys_total":         {Type: "counter", Unit: "", DescCN: "累计被驱逐键数", DescEN: "evicted keys"},
	"redis_net_input_bytes_total":      {Type: "counter", Unit: "bytes", DescCN: "累计网络入字节", DescEN: "net input bytes"},
	"redis_net_output_bytes_total":     {Type: "counter", Unit: "bytes", DescCN: "累计网络出字节", DescEN: "net output bytes"},
}
