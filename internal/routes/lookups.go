package routes

import (
	"github.com/gofiber/fiber/v2"

	"web-manager/internal/config"
	"web-manager/internal/controllers"
	"web-manager/internal/db"
	"web-manager/internal/middlewares"
	"web-manager/internal/services"
)

func SetupLookupRoutes(router fiber.Router, cfg config.Config, mongo *db.Mongo) {
	authSvc := services.NewAuthService(cfg, mongo)
	ctrl := controllers.NewLookupController(mongo)
	router.Get("/:purpose", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeRead,
	}), ctrl.ByPurpose)
}
