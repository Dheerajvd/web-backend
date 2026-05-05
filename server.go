package webmanager

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"go.uber.org/zap"

	_ "web-manager/docs"
	"web-manager/internal/config"
	"web-manager/internal/db"
	"web-manager/internal/middlewares"
	"web-manager/internal/routes"
	"web-manager/internal/services"
)

// NewServer builds the Fiber app, wires routes, and returns a cleanup that closes Mongo.
func NewServer(cfg config.Config, logger *zap.Logger) (*fiber.App, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongo, mongoCleanup, err := db.ConnectMongo(ctx, cfg.Mongo.URI, cfg.Mongo.Database)
	if err != nil {
		return nil, func() {}, err
	}
	if err := db.EnsureIndexes(ctx, mongo); err != nil {
		_ = mongoCleanup(ctx)
		return nil, func() {}, err
	}

	appSettingsProv := services.NewAppSettingsProvider()
	appSettingsSvc := services.NewAppSettingsService(mongo, logger)
	if err := appSettingsSvc.LoadOrSeedDefault(ctx, appSettingsProv); err != nil {
		_ = mongoCleanup(ctx)
		return nil, func() {}, err
	}

	lookupSvc := services.NewLookupService(mongo)
	if err := lookupSvc.EnsureLookupDefaults(ctx); err != nil {
		_ = mongoCleanup(ctx)
		return nil, func() {}, err
	}

	appName := strings.TrimSpace(appSettingsProv.AppName())
	if appName == "" {
		appName = "Web Manager"
	}

	app := fiber.New(fiber.Config{
		AppName:      appName,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: middlewares.ErrorHandler,
	})

	app.Use(middlewares.CORS(appSettingsProv))
	app.Use(middlewares.RequestLogger(logger))

	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	if email := os.Getenv("SUPER_USER_EMAIL"); email != "" && os.Getenv("SUPER_USER_PASSWORD") != "" {
		authSvc := services.NewAuthService(cfg, mongo)
		if err := authSvc.EnsureSuperUser(ctx, os.Getenv("SUPER_USER_NAME"), email, os.Getenv("SUPER_USER_PASSWORD")); err != nil {
			logger.Warn("failed to ensure super user", zap.Error(err))
		} else {
			logger.Info("super user ensured", zap.String("email", email))
		}
	}

	routes.SetupHealthRoutes(app.Group("/health"), mongo.DB.Name())
	routes.SetupAuthRoutes(app.Group("/auth"), cfg, mongo, logger, appSettingsProv)
	routes.SetupLookupRoutes(app.Group("/lookups"), cfg, mongo)
	routes.SetupAdminRoutes(app.Group("/admin"), cfg, mongo, logger)
	routes.SetupPublicRoutes(app, cfg, mongo, logger)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = mongoCleanup(ctx)
	}
	return app, cleanup, nil
}
