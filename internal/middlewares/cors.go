package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"web-manager/internal/services"
)

// CORS allows browser origins listed in app_settings (Mongo). Uses AllowOriginsFunc so the list can be updated in memory later if you reload settings.
func CORS(settings *services.AppSettingsProvider) fiber.Handler {
	return cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return settings.AllowOrigin(origin)
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowCredentials: false,
	})
}
