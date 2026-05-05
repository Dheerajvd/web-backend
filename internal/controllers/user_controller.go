package controllers

import (
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"web-manager/internal/db"
	"web-manager/internal/domain"
	"web-manager/internal/middlewares"
	"web-manager/internal/pagination"
)

type UserController struct {
	mongo *mongo.Database
	log   *zap.Logger
}

func NewUserController(m *db.Mongo, logger *zap.Logger) *UserController {
	return &UserController{mongo: m.DB, log: logger}
}

type createUserRequest struct {
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Password   string            `json:"password"`
	Role       domain.Role       `json:"role"`
	Status     domain.UserStatus `json:"status"`
	AppIDs     []string          `json:"appIds"`
	MfaEnabled bool              `json:"mfaEnabled"`
}

type updateUserRequest struct {
	Name       *string            `json:"name"`
	Email      *string            `json:"email"`
	Password   *string            `json:"password"`
	Role       *domain.Role       `json:"role"`
	Status     *domain.UserStatus `json:"status"`
	AppIDs     *[]string          `json:"appIds"`
	MfaEnabled *bool              `json:"mfaEnabled"`
}

func (c *UserController) actor(ctx *fiber.Ctx) (domain.User, error) {
	actorID, ok := ctx.Locals(middlewares.LocalUserID).(primitive.ObjectID)
	if !ok {
		return domain.User{}, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var u domain.User
	if err := c.users().FindOne(ctx.UserContext(), bson.M{"_id": actorID}).Decode(&u); err != nil {
		return domain.User{}, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return u, nil
}

func normalizeAppIDs(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, a := range in {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out
}

func isSubset(sub []string, sup []string) bool {
	if len(sub) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, s := range sup {
		set[s] = struct{}{}
	}
	for _, s := range sub {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}

func (c *UserController) requireCreatePermission(actor domain.User, reqRole domain.Role, reqAppIDs []string) error {
	switch actor.Role {
	case domain.RoleSuperUser:
		return nil
	case domain.RoleManager:
		if reqRole == domain.RoleSuperUser {
			return fiber.NewError(fiber.StatusForbidden, "managers cannot create super users")
		}
		if !isSubset(reqAppIDs, actor.AppIDs) {
			return fiber.NewError(fiber.StatusForbidden, "cannot assign apps outside your scope")
		}
		return nil
	default:
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}

func (c *UserController) requireModifyPermission(actor domain.User, target domain.User, newRole *domain.Role, newAppIDs *[]string) error {
	switch actor.Role {
	case domain.RoleSuperUser:
		return nil
	case domain.RoleManager:
		// managers cannot modify super users
		if target.Role == domain.RoleSuperUser {
			return fiber.NewError(fiber.StatusForbidden, "managers cannot modify super users")
		}
		if newRole != nil && *newRole == domain.RoleSuperUser {
			return fiber.NewError(fiber.StatusForbidden, "managers cannot promote to super user")
		}
		// managers can only manage users within their app scope
		if !isSubset(target.AppIDs, actor.AppIDs) {
			return fiber.NewError(fiber.StatusForbidden, "target user is outside your app scope")
		}
		if newAppIDs != nil && !isSubset(normalizeAppIDs(*newAppIDs), actor.AppIDs) {
			return fiber.NewError(fiber.StatusForbidden, "cannot assign apps outside your scope")
		}
		return nil
	default:
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
}

// Create godoc
// @Summary Create user
// @Tags admin-users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body createUserRequest true "Create user request"
// @Success 201 {object} map[string]any
// @Router /admin/users [post]
func (c *UserController) Create(ctx *fiber.Ctx) error {
	actor, err := c.actor(ctx)
	if err != nil {
		return err
	}
	var req createUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	req.Password = strings.TrimSpace(req.Password)
	req.AppIDs = normalizeAppIDs(req.AppIDs)
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name, email and password required")
	}
	if req.Role == "" {
		req.Role = domain.RoleAppUser
	}
	if req.Status == "" {
		req.Status = domain.UserStatusActive
	}
	if err := c.requireCreatePermission(actor, req.Role, req.AppIDs); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	now := time.Now().UTC()
	u := domain.User{
		ID:           primitive.NewObjectID(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         req.Role,
		Status:       req.Status,
		AppIDs:       req.AppIDs,
		MfaEnabled:   req.MfaEnabled,
		MfaState:     domain.MFANotEnrolled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := c.users().InsertOne(ctx.UserContext(), u); err != nil {
		// likely unique email
		return fiber.NewError(fiber.StatusBadRequest, "failed to create user")
	}
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": u.ID.Hex()}})
}

// List godoc
// @Summary List users
// @Tags admin-users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]any
// @Param search query string false "Case-insensitive substring match on name or email"
// @Router /admin/users [get]
func (c *UserController) List(ctx *fiber.Ctx) error {
	actor, err := c.actor(ctx)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperUser && actor.Role != domain.RoleManager {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	conds := make([]bson.M, 0, 2)
	if actor.Role == domain.RoleManager {
		conds = append(conds, bson.M{"appIds": bson.M{"$in": actor.AppIDs}})
	}
	if q := strings.TrimSpace(ctx.Query("search")); q != "" {
		pattern := primitive.Regex{Pattern: regexp.QuoteMeta(q), Options: "i"}
		conds = append(conds, bson.M{"$or": []bson.M{
			{"name": pattern},
			{"email": pattern},
		}})
	}

	var filter bson.M
	switch len(conds) {
	case 0:
		filter = bson.M{}
	case 1:
		filter = conds[0]
	default:
		filter = bson.M{"$and": conds}
	}

	page, limit, skip, perr := pagination.Parse(ctx)
	if perr != nil {
		return perr
	}

	total, err := c.users().CountDocuments(ctx.UserContext(), filter)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cur, err := c.users().Find(ctx.UserContext(), filter, findOpts)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	defer cur.Close(ctx.UserContext())

	type row struct {
		ID         primitive.ObjectID `bson:"_id" json:"id"`
		Name       string             `bson:"name" json:"name"`
		Email      string             `bson:"email" json:"email"`
		Role       domain.Role        `bson:"role" json:"role"`
		Status     domain.UserStatus  `bson:"status" json:"status"`
		AppIDs     []string           `bson:"appIds,omitempty" json:"appIds,omitempty"`
		MfaEnabled bool               `bson:"mfaEnabled,omitempty" json:"mfaEnabled,omitempty"`
		MfaState   domain.MFAState    `bson:"mfaState,omitempty" json:"mfaState,omitempty"`
		CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
		UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
	}

	out := make([]row, 0)
	for cur.Next(ctx.UserContext()) {
		var r row
		if err := cur.Decode(&r); err == nil {
			out = append(out, r)
		}
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"items":      out,
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": pagination.TotalPages(total, limit),
		},
	})
}

// Get godoc
// @Summary Get user by id
// @Tags admin-users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]any
// @Router /admin/users/{id} [get]
func (c *UserController) Get(ctx *fiber.Ctx) error {
	actor, err := c.actor(ctx)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperUser && actor.Role != domain.RoleManager {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	oid, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var u domain.User
	if err := c.users().FindOne(ctx.UserContext(), bson.M{"_id": oid}).Decode(&u); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	if actor.Role == domain.RoleManager && !isSubset(u.AppIDs, actor.AppIDs) {
		return fiber.NewError(fiber.StatusForbidden, "user is outside your app scope")
	}

	return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"id":         u.ID.Hex(),
		"name":       u.Name,
		"email":      u.Email,
		"role":       u.Role,
		"status":     u.Status,
		"appIds":     u.AppIDs,
		"mfaEnabled": u.MfaEnabled,
		"mfaState":   u.MfaState,
		"createdAt":  u.CreatedAt,
		"updatedAt":  u.UpdatedAt,
	}})
}

// Update godoc
// @Summary Update user
// @Tags admin-users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body updateUserRequest true "Update user request"
// @Success 200 {object} map[string]any
// @Router /admin/users/{id} [patch]
func (c *UserController) Update(ctx *fiber.Ctx) error {
	actor, err := c.actor(ctx)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperUser && actor.Role != domain.RoleManager {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	oid, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var target domain.User
	if err := c.users().FindOne(ctx.UserContext(), bson.M{"_id": oid}).Decode(&target); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	var req updateUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := c.requireModifyPermission(actor, target, req.Role, req.AppIDs); err != nil {
		return err
	}

	set := bson.M{}
	unset := bson.M{}

	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name cannot be empty")
		}
		set["name"] = n
	}
	if req.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*req.Email))
		if e == "" {
			return fiber.NewError(fiber.StatusBadRequest, "email cannot be empty")
		}
		set["email"] = e
	}
	if req.Password != nil {
		p := strings.TrimSpace(*req.Password)
		if p == "" {
			return fiber.NewError(fiber.StatusBadRequest, "password cannot be empty")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(p), 12)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
		}
		set["passwordHash"] = string(hash)
	}
	if req.Role != nil {
		set["role"] = *req.Role
	}
	if req.Status != nil {
		set["status"] = *req.Status
	}
	if req.AppIDs != nil {
		set["appIds"] = normalizeAppIDs(*req.AppIDs)
	}
	if req.MfaEnabled != nil {
		set["mfaEnabled"] = *req.MfaEnabled
		if !*req.MfaEnabled {
			// clear secrets when disabling MFA (reset)
			set["mfaState"] = domain.MFADisabled
			unset["totpSecretEnc"] = ""
			unset["totpPendingSecretEnc"] = ""
			unset["totpPendingAt"] = ""
			unset["mfaEnrolledAt"] = ""
		} else {
			// enabling does not automatically enroll; first login will produce QR.
			if target.MfaState == domain.MFADisabled {
				set["mfaState"] = domain.MFANotEnrolled
			}
		}
	}

	if len(set) == 0 && len(unset) == 0 {
		return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{"updated": false}})
	}
	set["updatedAt"] = time.Now().UTC()

	update := bson.M{}
	if len(set) > 0 {
		update["$set"] = set
	}
	if len(unset) > 0 {
		update["$unset"] = unset
	}

	if _, err := c.users().UpdateOne(ctx.UserContext(), bson.M{"_id": oid}, update); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update user")
	}
	return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{"updated": true}})
}

// Delete godoc
// @Summary Delete user
// @Tags admin-users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]any
// @Router /admin/users/{id} [delete]
func (c *UserController) Delete(ctx *fiber.Ctx) error {
	actor, err := c.actor(ctx)
	if err != nil {
		return err
	}
	if actor.Role != domain.RoleSuperUser && actor.Role != domain.RoleManager {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	oid, err := primitive.ObjectIDFromHex(ctx.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var target domain.User
	if err := c.users().FindOne(ctx.UserContext(), bson.M{"_id": oid}).Decode(&target); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	if err := c.requireModifyPermission(actor, target, nil, nil); err != nil {
		return err
	}
	if _, err := c.users().DeleteOne(ctx.UserContext(), bson.M{"_id": oid}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete user")
	}
	return ctx.JSON(fiber.Map{"success": true, "data": fiber.Map{"deleted": true}})
}

func (c *UserController) users() *mongo.Collection { return c.mongo.Collection("users") }
