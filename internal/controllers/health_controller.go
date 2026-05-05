package controllers

import "github.com/gofiber/fiber/v2"

type HealthController struct {
	mongoDBName string
}

func NewHealthController(mongoDBName string) *HealthController {
	return &HealthController{mongoDBName: mongoDBName}
}

type healthResponse struct {
	Status string `json:"status"`
	Mongo  string `json:"mongo"`
}

// Health godoc
// @Summary Health check
// @Tags health
// @Success 200 {object} map[string]any
// @Router /health [get]
func (c *HealthController) Health(ctx *fiber.Ctx) error {
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": healthResponse{
			Status: "ok",
			Mongo:  c.mongoDBName,
		},
	})
}
