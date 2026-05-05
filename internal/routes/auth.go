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

func SetupAuthRoutes(router fiber.Router, cfg config.Config, mongo *db.Mongo, logger *zap.Logger, settings *services.AppSettingsProvider) {
	authSvc := services.NewAuthService(cfg, mongo)
	authController := controllers.NewAuthController(authSvc, mongo, logger, settings)

	router.Post("/login", authController.Login)
	router.Post("/otp/validate", authController.ValidateOTP)
	router.Post("/logout", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{}), authController.Logout)
}
