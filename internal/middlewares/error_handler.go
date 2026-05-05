package middlewares

import (
	"github.com/gofiber/fiber/v2"

	"web-manager/internal/services"
)

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := err.Error()
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		msg = e.Message
	}
	if e, ok := err.(*services.StatusError); ok {
		code = e.Status
		msg = e.Message
	}
	return c.Status(code).JSON(ErrorResponse{Success: false, Message: msg})
}

