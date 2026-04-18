package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/grafana"
	"ops-system/backend/internal/handler"
	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/idempotency"
	integrationpkg "ops-system/backend/internal/integration"
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/middleware"
	"ops-system/backend/internal/n9e"
	"ops-system/backend/internal/notify"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/service"
	"ops-system/backend/internal/vm"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NewRouter 注册路由与全局中间件。
func NewRouter(cfg *config.Config, log *zap.Logger, db *gorm.DB) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.Server.Mode == "test" {
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	lim := middleware.NewIPRateLimiter(cfg.RateLimit.RequestsPerSecond, cfg.RateLimit.Burst)
	r.Use(lim.RateLimit())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "version": "0.3.0"})
		})
		if db != nil {
			api.GET("/health/db", func(c *gin.Context) {
				sqlDB, err := db.DB()
				if err != nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "database": "no_sql_driver"})
					return
				}
				if err := sqlDB.PingContext(c.Request.Context()); err != nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "database": "unavailable"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "postgresql"})
			})

			deptRepo := repository.NewDepartmentRepository(db)
			tenantRepo := repository.NewTenantRepository(db)
			userRepo := repository.NewUserRepository(db)
			instanceRepo := repository.NewInstanceRepository(db)
			alertRepo := repository.NewAlertRuleRepository(db)
			alertEventRepo := repository.NewAlertEventRepository(db)
			channelRepo := repository.NewNotificationChannelRepository(db)
			platformAuditRepo := repository.NewPlatformScaleAuditRepository(db)
			logInstanceRepo := repository.NewLogInstanceRepository(db)
			integrationTemplateRepo := repository.NewIntegrationTemplateRepository(db)
			integrationInstallRepo := repository.NewIntegrationInstallationRepository(db)
			metricRepo := repository.NewMetricRepository(db)
			grafanaHostRepo := repository.NewGrafanaHostRepository(db)
			clusterRepo := repository.NewClusterRepository(db)

			userSvc := service.NewUserService(userRepo)
			authSvc := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
			authH := handler.NewAuthHandler(authSvc, userSvc, cfg.JWT.Secret)
			userH := handler.NewUserHandler(userSvc, cfg.JWT.Secret)

			deptSvc := service.NewDepartmentService(deptRepo, tenantRepo, userRepo)
			deptH := handler.NewDepartmentHandler(deptSvc)

			vmSync := vm.NewSyncService(&cfg.VM, log)
			grafanaClient := grafana.NewClient(&cfg.Grafana, log)
			orch, err := service.NewOrchestratorService(cfg, log)
			if err != nil {
				log.Fatal("orchestrator_init", zap.Error(err))
			}
			tenantSvc := service.NewTenantService(deptRepo, tenantRepo, instanceRepo, vmSync, grafanaClient, orch, log)
			tenantH := handler.NewTenantHandler(tenantSvc, userSvc)

			instanceSvc := service.NewInstanceService(instanceRepo, tenantRepo, orch, log)
			var (
				helmClient *helm.Client
				k8sClient  *k8s.Client
			)
			if cfg.Kubernetes.InCluster || strings.TrimSpace(cfg.Kubernetes.Kubeconfig) != "" {
				hc, herr := helm.NewClient(cfg.Kubernetes.Kubeconfig)
				if herr != nil {
					log.Warn("scale_helm_client_init_failed", zap.Error(herr))
				} else {
					helmClient = hc
				}
				kc, kerr := k8s.NewClient(cfg.Kubernetes.Kubeconfig, cfg.Kubernetes.InCluster)
				if kerr != nil {
					log.Warn("scale_k8s_client_init_failed", zap.Error(kerr))
				} else {
					k8sClient = kc
				}
			}
			scaleEventRepo := repository.NewScaleEventRepository(db)
			scaleSvc := service.NewScaleService(helmClient, k8sClient, instanceRepo, scaleEventRepo, log)
			instanceH := handler.NewInstanceHandler(instanceSvc, scaleSvc, userSvc)
			k8sOps := service.NewK8sResourceOperator(k8sClient, log)
			platformScaleSvc := service.NewPlatformScaleService(k8sOps)
			platformBootstrapSvc := service.NewPlatformBootstrapService(helmClient, k8sClient)
			var idemStore idempotency.Store
			redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
			if cfg.Redis.Host != "" && cfg.Redis.Port > 0 {
				redisStore := idempotency.NewRedisStore(redisAddr)
				if err := redisStore.Ping(context.Background()); err != nil {
					log.Warn("platform_idempotency_redis_unavailable", zap.String("addr", redisAddr), zap.Error(err))
				} else {
					idemStore = redisStore
				}
			}
			platformH := handler.NewPlatformHandler(platformScaleSvc, platformBootstrapSvc, log, idemStore, platformAuditRepo)

			grafanaSvc := service.NewGrafanaService(grafanaClient, tenantRepo, log)
			grafanaH := handler.NewGrafanaHandler(grafanaSvc)
			n9eClient := n9e.NewClient(&cfg.N9E, log)
			notifySvc := notify.NewNotifyService(log)
			alertSvc := service.NewAlertService(alertRepo, tenantRepo, n9eClient, log)
			alertEventSvc := service.NewAlertEventService(alertEventRepo, alertRepo, channelRepo, n9eClient, notifySvc, log)
			channelSvc := service.NewNotificationChannelService(channelRepo, tenantRepo, log)
			alertH := handler.NewAlertHandler(alertSvc, alertEventSvc, channelSvc, userSvc)

			logInstanceSvc := service.NewLogInstanceService(logInstanceRepo)
			logInstanceH := handler.NewLogInstanceHandler(logInstanceSvc)

			renderer := integrationpkg.NewRenderer()

			// 集群 resolver：按 cluster id 动态构造 k8s.Client；nil / 查询失败 / 创建失败 → 回退默认 client。
			// 用闭包缓存已建 client，避免每次 Apply 反复初始化 discovery。
			type clusterClientEntry struct {
				client *k8s.Client
				fp     string // kubeconfig 指纹，用于失效检测
			}
			clusterClientCache := map[uuid.UUID]*clusterClientEntry{}
			k8sResolver := integrationpkg.K8sClientResolver(func(ctx context.Context, clusterID *uuid.UUID) (*k8s.Client, error) {
				if clusterID == nil {
					return k8sClient, nil
				}
				cluster, err := clusterRepo.GetByID(ctx, *clusterID)
				if err != nil {
					return k8sClient, err
				}
				if cluster == nil || cluster.Status != "active" {
					return k8sClient, nil
				}
				fp := fmt.Sprintf("%v|%s|%s", cluster.InCluster, cluster.KubeconfigPath, cluster.Kubeconfig)
				if entry, ok := clusterClientCache[*clusterID]; ok && entry.fp == fp {
					return entry.client, nil
				}
				kubeconfigPath := strings.TrimSpace(cluster.KubeconfigPath)
				inCluster := cluster.InCluster
				// 若只提供了 kubeconfig 原文，可在临时目录里写一个文件；M3 先只支持 path/incluster，
				// inline kubeconfig 暂不自动落盘（可在后续加 helper）。
				if strings.TrimSpace(cluster.Kubeconfig) != "" && kubeconfigPath == "" {
					log.Warn("cluster_inline_kubeconfig_ignored",
						zap.String("cluster", cluster.Name),
						zap.String("hint", "supply kubeconfig_path instead"))
				}
				if !inCluster && kubeconfigPath == "" {
					return k8sClient, nil
				}
				cli, kerr := k8s.NewClient(kubeconfigPath, inCluster)
				if kerr != nil {
					log.Warn("cluster_k8s_client_init_failed",
						zap.String("cluster", cluster.Name),
						zap.Error(kerr))
					return k8sClient, kerr
				}
				clusterClientCache[*clusterID] = &clusterClientEntry{client: cli, fp: fp}
				return cli, nil
			})

			// Grafana resolver：按 host id 动态构造 client；hostID 为 nil 或查询失败时回退平台 client。
			grafanaResolver := integrationpkg.GrafanaClientResolver(func(ctx context.Context, hostID *uuid.UUID) (*grafana.Client, error) {
				if hostID == nil {
					return grafanaClient, nil
				}
				host, err := grafanaHostRepo.GetByID(ctx, *hostID)
				if err != nil {
					return grafanaClient, err
				}
				if host == nil || host.Status != "active" || strings.TrimSpace(host.URL) == "" {
					return grafanaClient, nil
				}
				subCfg := config.GrafanaConfig{
					Enabled:                 true,
					BaseURL:                 host.URL,
					APIKey:                  host.AdminTokenEnc,
					HTTPTimeoutSeconds:      cfg.Grafana.HTTPTimeoutSeconds,
					PrometheusDatasourceURL: cfg.Grafana.PrometheusDatasourceURL,
					OrgNamePrefix:           cfg.Grafana.OrgNamePrefix,
				}
				return grafana.NewClient(&subCfg, log), nil
			})

			var integrationApplier integrationpkg.Applier
			if k8sClient != nil || (grafanaClient != nil && grafanaClient.Enabled()) {
				integrationApplier = integrationpkg.NewCompositeApplier(
					k8sClient, k8sResolver,
					grafanaClient, grafanaResolver,
					log,
				)
				log.Info("integration_applier_enabled",
					zap.Bool("k8s", k8sClient != nil),
					zap.Bool("grafana", grafanaClient != nil && grafanaClient.Enabled()))
			} else {
				integrationApplier = integrationpkg.NewNoopApplier()
				log.Warn("integration_applier_noop", zap.String("reason", "k8s & grafana unavailable"))
			}
			_ = clusterRepo // 下方 cluster handler 也会用到
			integrationTemplateSvc := service.NewIntegrationTemplateService(integrationTemplateRepo, integrationInstallRepo)
			integrationInstallSvc := service.NewIntegrationInstallationService(
				integrationInstallRepo, integrationTemplateRepo, instanceRepo, renderer, integrationApplier,
			)
			integrationH := handler.NewIntegrationHandler(integrationTemplateSvc, integrationInstallSvc, userSvc)

			metricSvc := service.NewMetricService(metricRepo, integrationTemplateRepo)
			metricH := handler.NewMetricHandler(metricSvc)

			seeder := service.NewIntegrationSeeder(db, integrationTemplateRepo, metricRepo, log)
			if err := seeder.SeedBuiltin(context.Background()); err != nil {
				log.Warn("integration_seed_failed", zap.Error(err))
			}

			grafanaHostSvc := service.NewGrafanaHostService(grafanaHostRepo)
			grafanaHostH := handler.NewGrafanaHostHandler(grafanaHostSvc)

			clusterSvc := service.NewClusterService(clusterRepo)
			clusterH := handler.NewClusterHandler(clusterSvc)

			api.POST("/auth/login", authH.Login)
			api.POST("/users/bootstrap", userH.Bootstrap)

			protected := api.Group("")
			protected.Use(middleware.JWTAuth(cfg.JWT.Secret))
			protected.GET("/auth/me", authH.Me)

			dg := protected.Group("/departments")
			dg.GET("/tree", deptH.Tree)
			dg.GET("", deptH.List)
			dg.GET("/:id/users", deptH.ListUsers)
			dg.GET("/:id", deptH.Get)

			tg := protected.Group("/tenants")
			tg.GET("", tenantH.List)
			tg.GET("/:id/metrics", tenantH.Metrics)
			tg.GET("/:id", tenantH.Get)

			ug := protected.Group("/users")
			ug.GET("", userH.List)
			ug.GET("/:id", userH.Get)
			ug.PUT("/:id", userH.Update)

			ig := protected.Group("/instances")
			ig.GET("", instanceH.List)
			ig.GET("/:id", instanceH.Get)
			ig.GET("/:id/metrics", instanceH.Metrics)
			ig.GET("/:id/scale-events", instanceH.ListScaleEvents)

			gg := protected.Group("/grafana/orgs")
			gg.GET("", grafanaH.ListOrgs)
			gg.GET("/:id/users", grafanaH.ListOrgUsers)
			gg.GET("/:id/datasources", grafanaH.ListDatasources)

			ag := protected.Group("/alerts")
			ag.GET("/rules", alertH.ListRules)
			ag.GET("/events", alertH.ListEvents)
			ag.GET("/events/:id", alertH.GetEvent)
			ag.PUT("/events/:id/ack", alertH.AckEvent)
			ag.GET("/channels", alertH.ListChannels)
			ag.GET("/stats/summary", alertH.Summary)
			ag.GET("/stats/trend", alertH.Trend)
			ag.GET("/stats/by-level", alertH.StatsByLevel)
			ag.GET("/stats/by-rule", alertH.StatsByRule)

			// 日志实例（查询）
			lg := protected.Group("/log-instances")
			lg.GET("", logInstanceH.List)
			lg.GET("/:id", logInstanceH.Get)
			lg.POST("/:id/query", logInstanceH.Query)

			// 接入中心（查询）
			intg := protected.Group("/integrations")
			intg.GET("/categories", integrationH.ListCategories)
			intg.GET("/templates", integrationH.ListTemplates)
			intg.GET("/templates/:id", integrationH.GetTemplate)
			intg.GET("/templates/:id/versions", integrationH.ListVersions)
			intg.POST("/install/plan", integrationH.InstallPlan)
			intg.POST("/install", integrationH.Install)
			intg.GET("/installations", integrationH.ListInstallations)
			intg.GET("/installations/:id", integrationH.GetInstallation)
			intg.GET("/installations/:id/revisions", integrationH.ListInstallationRevisions)
			intg.DELETE("/installations/:id", integrationH.Uninstall)

			// 指标库（查询）
			mg := protected.Group("/metrics")
			mg.GET("", metricH.List)
			mg.GET("/:id", metricH.Get)
			mg.GET("/:id/related", metricH.Related)

			// Grafana 主机（查询）
			ghg := protected.Group("/grafana/hosts")
			ghg.GET("", grafanaHostH.List)
			ghg.GET("/:id", grafanaHostH.Get)

			// K8s 集群（查询）
			cg := protected.Group("/clusters")
			cg.GET("", clusterH.List)
			cg.GET("/:id", clusterH.Get)

			admin := protected.Group("")
			admin.Use(middleware.RequireRole("admin"))

			adminDG := admin.Group("/departments")
			adminDG.POST("", deptH.Create)
			adminDG.PUT("/:id", deptH.Update)
			adminDG.DELETE("/:id", deptH.Delete)

			adminTG := admin.Group("/tenants")
			adminTG.POST("", tenantH.Create)
			adminTG.PUT("/:id", tenantH.Update)
			adminTG.DELETE("/:id", tenantH.Delete)

			adminUG := admin.Group("/users")
			adminUG.POST("", userH.Create)
			adminUG.DELETE("/:id", userH.Delete)

			adminIG := admin.Group("/instances")
			adminIG.POST("", instanceH.Create)
			adminIG.PUT("/:id", instanceH.Update)
			adminIG.DELETE("/:id", instanceH.Delete)
			adminIG.POST("/:id/scale", instanceH.Scale)

			adminGG := admin.Group("/grafana/orgs")
			adminGG.POST("", grafanaH.CreateOrg)
			adminGG.DELETE("/:id", grafanaH.DeleteOrg)
			adminGG.POST("/:id/users", grafanaH.AddOrgUser)
			adminGG.DELETE("/:id/users/:userId", grafanaH.RemoveOrgUser)
			adminGG.POST("/:id/datasources", grafanaH.CreateDatasource)
			adminGG.DELETE("/:id/datasources/:dsId", grafanaH.DeleteDatasource)
			adminGG.POST("/:id/dashboards/import", grafanaH.ImportDashboard)

			adminAG := admin.Group("/alerts")
			adminAG.POST("/rules", alertH.CreateRule)
			adminAG.PUT("/rules/:id", alertH.UpdateRule)
			adminAG.DELETE("/rules/:id", alertH.DeleteRule)
			adminAG.POST("/channels", alertH.CreateChannel)
			adminAG.PUT("/channels/:id", alertH.UpdateChannel)
			adminAG.DELETE("/channels/:id", alertH.DeleteChannel)

			adminPG := admin.Group("/platform/scaling")
			adminPG.POST("/bootstrap/shared/init", platformH.InitSharedCluster)
			adminPG.GET("/audits", platformH.ListAudits)
			adminPG.GET("/vmcluster/targets", platformH.ListVMClusterTargets)
			adminPG.POST("/vmcluster", platformH.ScaleVMCluster)

			// 日志实例（写）
			adminLG := admin.Group("/log-instances")
			adminLG.POST("", logInstanceH.Create)
			adminLG.PUT("/:id", logInstanceH.Update)
			adminLG.DELETE("/:id", logInstanceH.Delete)

			// 接入中心模版（写）
			adminINTG := admin.Group("/integrations")
			adminINTG.POST("/templates", integrationH.CreateTemplate)
			adminINTG.PUT("/templates/:id", integrationH.UpdateTemplate)
			adminINTG.DELETE("/templates/:id", integrationH.DeleteTemplate)
			adminINTG.POST("/templates/:id/versions", integrationH.CreateVersion)
			adminINTG.DELETE("/templates/:id/versions/:version", integrationH.DeleteVersion)

			// 指标库（写）
			adminMG := admin.Group("/metrics")
			adminMG.POST("", metricH.Create)
			adminMG.PUT("/:id", metricH.Update)
			adminMG.DELETE("/:id", metricH.Delete)
			adminMG.POST("/reparse/:templateId", metricH.Reparse)

			// Grafana 主机（写）
			adminGHG := admin.Group("/grafana/hosts")
			adminGHG.POST("", grafanaHostH.Create)
			adminGHG.PUT("/:id", grafanaHostH.Update)
			adminGHG.DELETE("/:id", grafanaHostH.Delete)

			// K8s 集群（写）
			adminCG := admin.Group("/clusters")
			adminCG.POST("", clusterH.Create)
			adminCG.PUT("/:id", clusterH.Update)
			adminCG.DELETE("/:id", clusterH.Delete)
		}
	}

	return r
}
