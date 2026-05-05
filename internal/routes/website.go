package routes

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"web-manager/internal/config"
	"web-manager/internal/controllers"
	"web-manager/internal/db"
	"web-manager/internal/middlewares"
	"web-manager/internal/services"
)

func SetupWebsiteRoutes(router fiber.Router, cfg config.Config, mongo *db.Mongo, logger *zap.Logger) {
	_ = cfg
	authSvc := services.NewAuthService(cfg, mongo)
	ctrl := controllers.NewSiteDataController(mongo, logger)

	scopeRead := middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeRead,
		Module:  middlewares.ModuleWebsite,
	})
	scopeWrite := middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeUpdate,
		Module:  middlewares.ModuleWebsite,
	})

	sd := router.Group("/site-data")
	sd.Post("/:name/upload", scopeWrite, ctrl.UploadJSON)
	sd.Get("/:name", scopeRead, ctrl.Get)
	sd.Post("/:name", scopeWrite, ctrl.ReplaceJSON)
}
