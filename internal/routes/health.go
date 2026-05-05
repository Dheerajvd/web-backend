package routes

import (
	"github.com/gofiber/fiber/v2"

	"web-manager/internal/controllers"
)

func SetupHealthRoutes(router fiber.Router, mongoDBName string) {
	h := controllers.NewHealthController(mongoDBName)
	router.Get("/", h.Health)
}
