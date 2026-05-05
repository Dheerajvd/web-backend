package routes

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"web-manager/internal/config"
	"web-manager/internal/db"
)

func SetupPublicRoutes(app *fiber.App, cfg config.Config, mongo *db.Mongo, logger *zap.Logger) {
	// TODO: public endpoints per PRD (videos, sources, suggestions/corrections/contact)
	_ = cfg
	_ = mongo
	_ = logger
	_ = app
}
