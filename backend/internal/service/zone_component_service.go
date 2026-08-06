package service

import (
	"context"

	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
)

// ZoneComponentStatus 可用区组件状态。
type ZoneComponentStatus struct {
	Component string `json:"component"`
	Status    string `json:"status"`
	Details   string `json:"details,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Release   string `json:"release,omitempty"`
}

// ZoneComponentService 聚合可用区组件状态。
type ZoneComponentService struct {
	zones         *repository.ZoneRepository
	vmClusters    *repository.VMClusterRepository
	logClusters   *repository.LogClusterRepository
	grafanaInst   *repository.GrafanaInstanceRepository
}

func NewZoneComponentService(
	zones *repository.ZoneRepository,
	vmClusters *repository.VMClusterRepository,
	logClusters *repository.LogClusterRepository,
	grafanaInst *repository.GrafanaInstanceRepository,
) *ZoneComponentService {
	return &ZoneComponentService{
		zones:       zones,
		vmClusters:  vmClusters,
		logClusters: logClusters,
		grafanaInst: grafanaInst,
	}
}

// GetComponents 返回 VMCluster / Kafka(VLogs pipeline) / VLogs / Grafana 状态摘要。
func (s *ZoneComponentService) GetComponents(ctx context.Context, zoneID uuid.UUID) ([]ZoneComponentStatus, error) {
	if s == nil || s.zones == nil {
		return nil, ErrZoneNotFound
	}
	z, err := s.zones.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrZoneNotFound
	}

	out := make([]ZoneComponentStatus, 0, 4)
	out = append(out, s.vmClusterStatus(ctx, zoneID))
	out = append(out, s.vlogsStatus(ctx, zoneID))
	out = append(out, s.kafkaStatus(ctx, zoneID))
	out = append(out, s.grafanaStatus(ctx, zoneID))
	return out, nil
}

func (s *ZoneComponentService) vmClusterStatus(ctx context.Context, zoneID uuid.UUID) ZoneComponentStatus {
	st := ZoneComponentStatus{Component: "vmcluster", Status: "missing"}
	if s.vmClusters == nil {
		st.Details = "repository unavailable"
		return st
	}
	c, err := s.vmClusters.GetActiveSharedByZone(ctx, zoneID)
	if err != nil || c == nil {
		st.Details = "shared VM cluster not registered"
		return st
	}
	st.Status = c.Status
	st.Namespace = c.Namespace
	st.Release = c.ReleaseName
	st.Details = c.Name
	return st
}

func (s *ZoneComponentService) vlogsStatus(ctx context.Context, zoneID uuid.UUID) ZoneComponentStatus {
	st := ZoneComponentStatus{Component: "vlogs", Status: "missing"}
	if s.logClusters == nil {
		st.Details = "repository unavailable"
		return st
	}
	c, err := s.logClusters.GetActiveByZone(ctx, zoneID)
	if err != nil || c == nil {
		st.Details = "shared log pipeline not registered"
		return st
	}
	st.Status = c.Status
	st.Namespace = c.Namespace
	st.Release = c.ReleaseName
	st.Details = c.SelectURL
	return st
}

func (s *ZoneComponentService) kafkaStatus(ctx context.Context, zoneID uuid.UUID) ZoneComponentStatus {
	st := ZoneComponentStatus{Component: "kafka", Status: "missing"}
	if s.logClusters == nil {
		st.Details = "repository unavailable"
		return st
	}
	c, err := s.logClusters.GetActiveByZone(ctx, zoneID)
	if err != nil || c == nil {
		st.Details = "kafka brokers not registered"
		return st
	}
	if c.KafkaBrokers == "" {
		st.Status = "warn"
		st.Details = "log pipeline registered without kafka brokers"
		return st
	}
	st.Status = c.Status
	st.Namespace = c.Namespace
	st.Release = c.ReleaseName
	st.Details = c.KafkaBrokers
	return st
}

func (s *ZoneComponentService) grafanaStatus(ctx context.Context, zoneID uuid.UUID) ZoneComponentStatus {
	st := ZoneComponentStatus{Component: "grafana", Status: "missing"}
	if s.grafanaInst == nil {
		st.Details = "repository unavailable"
		return st
	}
	list, _, err := s.grafanaInst.List(ctx, repository.GrafanaInstanceListFilter{
		Source: "platform",
		ZoneID: &zoneID,
		Offset: 0,
		Limit:  1,
	})
	if err != nil || len(list) == 0 {
		st.Details = "grafana instance not registered for zone"
		return st
	}
	g := list[0]
	st.Status = g.Status
	st.Details = g.URL
	return st
}
