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
	"ops-system/backend/internal/k8s"
	"ops-system/backend/internal/middleware"
	"ops-system/backend/internal/n9e"
	"ops-system/backend/internal/notify"
	"ops-system/backend/internal/repository"
	"ops-system/backend/internal/service"
	"ops-system/backend/internal/vm"

	"github.com/gin-gonic/gin"
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
			scaleSvc := service.NewScaleService(helmClient, k8sClient, instanceRepo, log)
			instanceH := handler.NewInstanceHandler(instanceSvc, scaleSvc, userSvc)
			k8sOps := service.NewK8sResourceOperator(k8sClient, log)
			platformScaleSvc := service.NewPlatformScaleService(k8sOps)
			platformBootstrapSvc := service.NewPlatformBootstrapService(helmClient)
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
		}
	}

	return r
}
