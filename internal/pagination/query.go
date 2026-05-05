package pagination

import (
	"github.com/gofiber/fiber/v2"
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Parse reads page and limit from query (?page=1&limit=20). Defaults: page=1, limit=20. Every list response uses these bounds.
func Parse(c *fiber.Ctx) (page int, limit int, skip int64, err error) {
	page = c.QueryInt("page", 1)
	limit = c.QueryInt("limit", DefaultLimit)
	if page < 1 {
		return 0, 0, 0, fiber.NewError(fiber.StatusBadRequest, "page must be >= 1")
	}
	if limit < 1 || limit > MaxLimit {
		return 0, 0, 0, fiber.NewError(fiber.StatusBadRequest, "limit must be between 1 and 100")
	}
	skip = int64((page - 1) * limit)
	return page, limit, skip, nil
}

// TotalPages returns ceiling(total / limit) for limit > 0.
func TotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	if total == 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
