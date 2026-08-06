package service

import (
	"context"
	"fmt"
	"strings"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// PreflightCheck Zone 预检结果。
type PreflightCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok / warn / fail
	Message string `json:"message"`
}

// ZonePreflightService 可用区部署前检查。
type ZonePreflightService struct {
	zones       *repository.ZoneRepository
	clusters    *repository.ClusterRepository
	clientCache *k8s.ClusterClientCache
	helmCfg     *config.HelmConfig
	log         *zap.Logger
}

func NewZonePreflightService(
	zones *repository.ZoneRepository,
	clusters *repository.ClusterRepository,
	clientCache *k8s.ClusterClientCache,
	helmCfg *config.HelmConfig,
	log *zap.Logger,
) *ZonePreflightService {
	if log == nil {
		log = zap.NewNop()
	}
	return &ZonePreflightService{
		zones:       zones,
		clusters:    clusters,
		clientCache: clientCache,
		helmCfg:     helmCfg,
		log:         log,
	}
}

// Preflight 检查 Zone 绑定的 K8s 集群是否满足组件部署要求。
func (s *ZonePreflightService) Preflight(ctx context.Context, zoneID uuid.UUID) ([]PreflightCheck, error) {
	z, err := s.zones.GetByID(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	if z == nil {
		return nil, ErrZoneNotFound
	}
	cluster, err := s.clusters.GetByID(ctx, z.ClusterID)
	if err != nil {
		return nil, err
	}
	if cluster == nil || cluster.Status != "active" {
		return []PreflightCheck{{
			Name:    "cluster",
			Status:  "fail",
			Message: "zone cluster is not active",
		}}, nil
	}

	checks := make([]PreflightCheck, 0, 4)
	k8sCli := s.resolveK8sClient(cluster)
	checks = append(checks, s.checkK8sConnectivity(ctx, k8sCli))
	checks = append(checks, s.checkVMOperatorCRD(ctx, k8sCli))
	checks = append(checks, s.checkHelmRepos())
	checks = append(checks, s.checkStorageClass(ctx, k8sCli))
	return checks, nil
}

func (s *ZonePreflightService) resolveK8sClient(cluster *model.Cluster) *k8s.Client {
	if s == nil || s.clientCache == nil || cluster == nil {
		return nil
	}
	fp := fmt.Sprintf("%v|%s|%s", cluster.InCluster, cluster.KubeconfigPath, cluster.Kubeconfig)
	if cached := s.clientCache.Get(cluster.ID, fp); cached != nil {
		return cached
	}
	kubeconfigPath := strings.TrimSpace(cluster.KubeconfigPath)
	if !cluster.InCluster && kubeconfigPath == "" {
		return nil
	}
	cli, err := k8s.NewClient(kubeconfigPath, cluster.InCluster)
	if err != nil {
		s.log.Warn("zone_preflight_k8s_client_failed", zap.Error(err))
		return nil
	}
	s.clientCache.Put(cluster.ID, fp, cli)
	return cli
}

func (s *ZonePreflightService) checkK8sConnectivity(ctx context.Context, cli *k8s.Client) PreflightCheck {
	if cli == nil {
		return PreflightCheck{Name: "k8s_connectivity", Status: "fail", Message: "k8s client unavailable"}
	}
	if _, _, err := cli.ResolveGVRByString("v1", "Namespace"); err != nil {
		return PreflightCheck{Name: "k8s_connectivity", Status: "fail", Message: err.Error()}
	}
	return PreflightCheck{Name: "k8s_connectivity", Status: "ok", Message: "cluster reachable"}
}

func (s *ZonePreflightService) checkVMOperatorCRD(ctx context.Context, cli *k8s.Client) PreflightCheck {
	if cli == nil {
		return PreflightCheck{Name: "vm_operator_crd", Status: "fail", Message: "k8s client unavailable"}
	}
	if _, _, err := cli.ResolveGVRByString("operator.victoriametrics.com/v1beta1", "VMCluster"); err != nil {
		return PreflightCheck{Name: "vm_operator_crd", Status: "fail", Message: errors.Wrap(err, "VMCluster CRD").Error()}
	}
	return PreflightCheck{Name: "vm_operator_crd", Status: "ok", Message: "VictoriaMetrics operator CRDs registered"}
}

func (s *ZonePreflightService) checkHelmRepos() PreflightCheck {
	if s == nil || s.helmCfg == nil || len(s.helmCfg.Repos) == 0 {
		return PreflightCheck{Name: "helm_repos", Status: "warn", Message: "no helm repos configured"}
	}
	names := make([]string, 0, len(s.helmCfg.Repos))
	for _, r := range s.helmCfg.Repos {
		if strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.URL) == "" {
			return PreflightCheck{Name: "helm_repos", Status: "fail", Message: "invalid helm repo entry"}
		}
		names = append(names, r.Name)
	}
	return PreflightCheck{Name: "helm_repos", Status: "ok", Message: strings.Join(names, ", ")}
}

func (s *ZonePreflightService) checkStorageClass(ctx context.Context, cli *k8s.Client) PreflightCheck {
	if cli == nil {
		return PreflightCheck{Name: "storage_class", Status: "fail", Message: "k8s client unavailable"}
	}
	// 通过 core API 解析确认 storageclass 资源可用；具体默认 SC 由集群管理员配置。
	if _, _, err := cli.ResolveGVRByString("storage.k8s.io/v1", "StorageClass"); err != nil {
		return PreflightCheck{Name: "storage_class", Status: "warn", Message: "StorageClass API unavailable"}
	}
	return PreflightCheck{Name: "storage_class", Status: "ok", Message: "StorageClass API available"}
}
