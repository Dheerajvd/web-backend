package middlewares

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"web-manager/internal/db"
	"web-manager/internal/domain"
	"web-manager/internal/services"
)

const (
	LocalUserID = "userId"
	LocalAppID  = "appId"
	LocalRole   = "role"
)

// AuthScope describes what the authenticated route is allowed to do.
// Use Purpose* and Module* constants. Empty fields skip that dimension (except JWT appId vs user is always enforced).
type AuthScope struct {
	Purpose string
	Module  string
}

// RequireAuth validates JWT, user status, token appId vs user appIds, then scope (purpose/module/role/app rules).
func RequireAuth(auth *services.AuthService, mongo *db.Mongo, scope AuthScope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authz := strings.TrimSpace(c.Get("Authorization"))
		claims, err := auth.ParseAccessToken(authz)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		userIDHex, _ := claims["sub"].(string)
		oid, err := primitive.ObjectIDFromHex(userIDHex)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var doc struct {
			ID     primitive.ObjectID `bson:"_id"`
			Status string             `bson:"status"`
			Role   string             `bson:"role"`
			AppIDs []string           `bson:"appIds,omitempty"`
		}
		if err := mongo.DB.Collection("users").FindOne(c.UserContext(), bson.M{"_id": oid}).Decode(&doc); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		if doc.Status == "disabled" {
			return fiber.NewError(fiber.StatusForbidden, "user disabled")
		}

		role := domain.Role(doc.Role)
		appID, _ := claims["appId"].(string)
		if strings.TrimSpace(appID) == "" {
			appID, _ = claims["app_id"].(string)
		}
		appID = strings.TrimSpace(appID)
		if appID == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "token missing appId")
		}
		if !domain.UserCanAccessApp(role, doc.AppIDs, appID) {
			return fiber.NewError(fiber.StatusForbidden, "user is not assigned to this app")
		}

		if e := authScopeError(scope, role, appID); e != nil {
			return e
		}

		c.Locals(LocalUserID, oid)
		c.Locals(LocalAppID, appID)
		c.Locals(LocalRole, doc.Role)
		return c.Next()
	}
}

func RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if r, ok := c.Locals(LocalRole).(string); ok && r == role {
			return c.Next()
		}
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}
