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

func SetupAdminRoutes(router fiber.Router, cfg config.Config, mongo *db.Mongo, logger *zap.Logger) {
	_ = cfg
	authSvc := services.NewAuthService(cfg, mongo)

	// Users CRUD
	users := controllers.NewUserController(mongo, logger)
	router.Get("/users", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeRead,
		Module:  middlewares.ModuleUM,
	}), users.List)
	router.Get("/users/:id", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeRead,
		Module:  middlewares.ModuleUM,
	}), users.Get)
	router.Post("/users", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeCreate,
		Module:  middlewares.ModuleUM,
	}), users.Create)
	router.Patch("/users/:id", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeUpdate,
		Module:  middlewares.ModuleUM,
	}), users.Update)
	router.Delete("/users/:id", middlewares.RequireAuth(authSvc, mongo, middlewares.AuthScope{
		Purpose: middlewares.PurposeDelete,
		Module:  middlewares.ModuleUM,
	}), users.Delete)
}
