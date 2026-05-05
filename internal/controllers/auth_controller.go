package controllers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"web-manager/internal/db"
	"web-manager/internal/middlewares"
	"web-manager/internal/services"
)

type AuthController struct {
	auth     *services.AuthService
	mongo    *db.Mongo
	log      *zap.Logger
	settings *services.AppSettingsProvider
}

func NewAuthController(auth *services.AuthService, mongo *db.Mongo, logger *zap.Logger, settings *services.AppSettingsProvider) *AuthController {
	return &AuthController{
		auth:     auth,
		mongo:    mongo,
		log:      logger,
		settings: settings,
	}
}

func (c *AuthController) totpIssuer() string {
	if c.settings == nil {
		return "web-manager"
	}
	name := strings.TrimSpace(c.settings.AppName())
	if name == "" {
		return "web-manager"
	}
	return name
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	AppID    string `json:"appId"`
}

// Login godoc
// @Summary Login (password step)
// @Description If `mfaEnabled=false`, returns tokens. If `mfaEnabled=true`, returns enrollment QR on first time or OTP required otherwise.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "Login request"
// @Success 200 {object} map[string]any
// @Router /auth/login [post]
func (c *AuthController) Login(ctx *fiber.Ctx) error {
	var req loginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	res, err := c.auth.Login(ctx.UserContext(), req.Email, req.Password, req.AppID, c.totpIssuer())
	if err != nil {
		return err
	}
	return ctx.JSON(fiber.Map{"success": true, "data": res})
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type validateOTPRequest struct {
	MfaToken string `json:"mfaToken"`
	Code     string `json:"code"`
}

// ValidateOTP godoc
// @Summary Validate OTP (TOTP)
// @Description Validates OTP using the `mfaToken` returned by `/auth/login` and returns tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body validateOTPRequest true "OTP validation request"
// @Success 200 {object} map[string]any
// @Router /auth/otp/validate [post]
func (c *AuthController) ValidateOTP(ctx *fiber.Ctx) error {
	var req validateOTPRequest
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	tokens, err := c.auth.ValidateOTP(ctx.UserContext(), req.MfaToken, req.Code)
	if err != nil {
		return err
	}
	return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"step":   "tokens",
		"tokens": tokens,
	}})
}

// Refresh godoc
// @Summary Refresh access token
// @Description Exchanges a valid refresh token for a new access + refresh token pair (refresh rotation).
// @Tags auth
// @Accept json
// @Produce json
// @Param body body refreshRequest true "Refresh request"
// @Success 200 {object} map[string]any
// @Router /auth/refresh [post]
func (c *AuthController) Refresh(ctx *fiber.Ctx) error {
	var req refreshRequest
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	res, err := c.auth.Refresh(ctx.UserContext(), req.RefreshToken, ctx.IP())
	if err != nil {
		return err
	}
	return ctx.JSON(fiber.Map{"success": true, "data": res})
}

// Logout godoc
// @Summary Logout
// @Description Invalidates refresh tokens for the current user.
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]any
// @Router /auth/logout [post]
func (c *AuthController) Logout(ctx *fiber.Ctx) error {
	authz := strings.TrimSpace(ctx.Get("Authorization"))
	claims, err := c.auth.ParseAccessToken(authz)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	sub, _ := claims["sub"].(string)
	oid, err := primitive.ObjectIDFromHex(sub)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	now := time.Now().UTC()
	_, _ = c.refreshTokens().UpdateMany(ctx.UserContext(),
		bson.M{"userId": oid, "revokedAt": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"revokedAt": now, "lastUsedAt": now}},
	)

	return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{"loggedOutAt": now}})
}

func (c *AuthController) Me(ctx *fiber.Ctx) error {
	userID, ok := ctx.Locals(middlewares.LocalUserID).(primitive.ObjectID)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	appID, _ := ctx.Locals(middlewares.LocalAppID).(string)

	var user struct {
		ID         primitive.ObjectID `bson:"_id" json:"id"`
		Name       string             `bson:"name" json:"name"`
		Email      string             `bson:"email" json:"email"`
		Role       string             `bson:"role" json:"role"`
		Status     string             `bson:"status" json:"status"`
		AppIDs     []string           `bson:"appIds,omitempty" json:"appIds,omitempty"`
		MfaEnabled bool               `bson:"mfaEnabled,omitempty" json:"mfaEnabled,omitempty"`
		MfaState   string             `bson:"mfaState,omitempty" json:"mfaState,omitempty"`
	}
	if err := c.authUsers().FindOne(ctx.UserContext(), bson.M{"_id": userID}).Decode(&user); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"appIds":     user.AppIDs,
		"mfaEnabled": user.MfaEnabled,
		"mfaState":   user.MfaState,
		"appId":      appID,
	}})
}

func (c *AuthController) authUsers() *mongo.Collection { return c.mongo.DB.Collection("users") }
func (c *AuthController) refreshTokens() *mongo.Collection {
	return c.mongo.DB.Collection("refresh_tokens")
}
