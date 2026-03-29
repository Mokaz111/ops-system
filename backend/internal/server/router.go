package server

import (
	"net/http"

	"ops-system/backend/internal/config"
	"ops-system/backend/internal/grafana"
	"ops-system/backend/internal/handler"
	"ops-system/backend/internal/middleware"
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
					c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "database": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "postgresql"})
			})

			deptRepo := repository.NewDepartmentRepository(db)
			tenantRepo := repository.NewTenantRepository(db)
			userRepo := repository.NewUserRepository(db)
			instanceRepo := repository.NewInstanceRepository(db)

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
			tenantH := handler.NewTenantHandler(tenantSvc)

			instanceSvc := service.NewInstanceService(instanceRepo, tenantRepo, orch, log)
			scaleSvc := service.NewScaleService(nil, nil, instanceRepo, log)
			instanceH := handler.NewInstanceHandler(instanceSvc, scaleSvc)

			grafanaSvc := service.NewGrafanaService(grafanaClient, tenantRepo, log)
			grafanaH := handler.NewGrafanaHandler(grafanaSvc)

			api.POST("/auth/login", authH.Login)
			api.POST("/users/bootstrap", userH.Bootstrap)

			protected := api.Group("")
			protected.Use(middleware.JWTAuth(cfg.JWT.Secret))
			protected.GET("/auth/me", authH.Me)

			dg := protected.Group("/departments")
			dg.GET("/tree", deptH.Tree)
			dg.GET("", deptH.List)
			dg.POST("", deptH.Create)
			dg.GET("/:id/users", deptH.ListUsers)
			dg.GET("/:id", deptH.Get)
			dg.PUT("/:id", deptH.Update)
			dg.DELETE("/:id", deptH.Delete)

			tg := protected.Group("/tenants")
			tg.GET("", tenantH.List)
			tg.POST("", tenantH.Create)
			tg.GET("/:id/metrics", tenantH.Metrics)
			tg.GET("/:id", tenantH.Get)
			tg.PUT("/:id", tenantH.Update)
			tg.DELETE("/:id", tenantH.Delete)

			ug := protected.Group("/users")
			ug.GET("", userH.List)
			ug.POST("", userH.Create)
			ug.GET("/:id", userH.Get)
			ug.PUT("/:id", userH.Update)
			ug.DELETE("/:id", userH.Delete)

			ig := protected.Group("/instances")
			ig.GET("", instanceH.List)
			ig.POST("", instanceH.Create)
			ig.GET("/:id", instanceH.Get)
			ig.PUT("/:id", instanceH.Update)
			ig.DELETE("/:id", instanceH.Delete)
			ig.POST("/:id/scale", instanceH.Scale)
			ig.GET("/:id/metrics", instanceH.Metrics)

			gg := protected.Group("/grafana/orgs")
			gg.GET("", grafanaH.ListOrgs)
			gg.POST("", grafanaH.CreateOrg)

			gOrg := gg.Group("/:id")
			gOrg.DELETE("", grafanaH.DeleteOrg)
			gOrg.GET("/users", grafanaH.ListOrgUsers)
			gOrg.POST("/users", grafanaH.AddOrgUser)
			gOrg.DELETE("/users/:userId", grafanaH.RemoveOrgUser)
			gOrg.GET("/datasources", grafanaH.ListDatasources)
			gOrg.POST("/datasources", grafanaH.CreateDatasource)
			gOrg.DELETE("/datasources/:dsId", grafanaH.DeleteDatasource)
			gOrg.POST("/dashboards/import", grafanaH.ImportDashboard)
		}
	}

	return r
}
