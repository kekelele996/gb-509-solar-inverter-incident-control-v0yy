package router

import (
	"log/slog"
	"net/http"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/config"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/handler"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/middleware"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/repository"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB, redisClient *redis.Client, logger *slog.Logger) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(middleware.RequestContext(logger))
	if len(cfg.TrustedProxies) > 0 {
		_ = engine.SetTrustedProxies(cfg.TrustedProxies)
	} else {
		_ = engine.SetTrustedProxies(nil)
	}

	securityRepository := repository.NewSecurityRepository(db)
	securityService := service.NewSecurityService(securityRepository, cfg)
	solarSiteRepository := repository.NewSolarSiteRepository(db)
	inverterUnitRepository := repository.NewInverterUnitRepository(db)
	faultEventRepository := repository.NewFaultEventRepository(db)
	mitigationActionRepository := repository.NewMitigationActionRepository(db)
	solarSiteService := service.NewSolarSiteService(solarSiteRepository, securityService)
	inverterUnitService := service.NewInverterUnitService(inverterUnitRepository, securityService)
	faultEventService := service.NewFaultEventService(faultEventRepository, securityService)
	mitigationActionService := service.NewMitigationActionService(mitigationActionRepository, securityService)
	solarSiteHandler := handler.NewSolarSiteHandler(solarSiteService)
	inverterUnitHandler := handler.NewInverterUnitHandler(inverterUnitService)
	faultEventHandler := handler.NewFaultEventHandler(faultEventService)
	mitigationActionHandler := handler.NewMitigationActionHandler(mitigationActionService)
	systemHandler := handler.NewSystemHandler(securityService, solarSiteService, inverterUnitService, faultEventService, mitigationActionService, db, redisClient)

	engine.GET("/healthz", systemHandler.Health)
	engine.POST("/api/auth/login", systemHandler.Login)

	limiter := middleware.NewLimiter(redisClient, cfg.RequestLimit)
	api := engine.Group("/api")
	api.Use(limiter.Middleware(), middleware.Authenticate(cfg))
	api.GET("/overview", systemHandler.Overview)
	api.GET("/audits", middleware.RequireMinimumRole("reviewer"), systemHandler.Audits)
	api.GET("/session", systemHandler.Session)
	api.GET("/runtime", middleware.RequireMinimumRole("reviewer"), systemHandler.Runtime)
	api.GET("/audit-summary", middleware.RequireMinimumRole("reviewer"), systemHandler.AuditSummary)
	api.GET("/audits/:entityType/:id", middleware.RequireMinimumRole("reviewer"), systemHandler.EntityHistory)
	solarSiteHandler.Register(api)
	inverterUnitHandler.Register(api)
	faultEventHandler.Register(api)
	mitigationActionHandler.Register(api)

	engine.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "route_not_found"})
	})
	return engine
}
