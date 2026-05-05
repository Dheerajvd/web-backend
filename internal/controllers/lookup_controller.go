package controllers

import (
	"github.com/gofiber/fiber/v2"

	"web-manager/internal/db"
	"web-manager/internal/pagination"
	"web-manager/internal/services"
)

type LookupController struct {
	svc *services.LookupService
}

func NewLookupController(m *db.Mongo) *LookupController {
	return &LookupController{
		svc: services.NewLookupService(m),
	}
}

// ByPurpose godoc
// @Summary Lookup lists by purpose (authenticated)
// @Description purpose: `roles` — role id, code, display name; `appIds` — application id, appId slug, display name (aliases: apps, applications).
// @Tags lookups
// @Security BearerAuth
// @Produce json
// @Param purpose path string true "roles | appIds"
// @Success 200 {object} map[string]any
// @Router /lookups/{purpose} [get]
func (c *LookupController) ByPurpose(ctx *fiber.Ctx) error {
	page, limit, skip, perr := pagination.Parse(ctx)
	if perr != nil {
		return perr
	}
	items, total, err := c.svc.ListLookupsPage(ctx.UserContext(), ctx.Params("purpose"), skip, limit)
	if err != nil {
		return err
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"items":      items,
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": pagination.TotalPages(total, limit),
		},
	})
}
