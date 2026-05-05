package controllers

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"web-manager/internal/db"
	"web-manager/internal/domain"
)

const siteDataMaxUploadBytes = 5 << 20 // 5 MiB

var siteDataNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

type SiteDataController struct {
	mongo *mongo.Database
	log   *zap.Logger
}

func NewSiteDataController(m *db.Mongo, logger *zap.Logger) *SiteDataController {
	return &SiteDataController{mongo: m.DB, log: logger}
}

func (c *SiteDataController) coll() *mongo.Collection {
	return c.mongo.Collection("siteData")
}

func parseSiteDataNameParam(ctx *fiber.Ctx) (string, error) {
	name := strings.TrimSpace(ctx.Params("name"))
	if name == "" || !siteDataNameRe.MatchString(name) {
		return "", fiber.NewError(fiber.StatusBadRequest, "invalid name")
	}
	return name, nil
}

func sitePayloadToBSON(doc domain.SiteDataDocument) (bson.M, error) {
	j, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var m bson.M
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *SiteDataController) upsert(ctx *fiber.Ctx, name string, doc domain.SiteDataDocument) error {
	data, err := sitePayloadToBSON(doc)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to encode data")
	}
	now := time.Now().UTC()
	filter := bson.M{"name": name}
	update := bson.M{
		"$set": bson.M{
			"name":      name,
			"data":      data,
			"updatedAt": now,
		},
		"$setOnInsert": bson.M{
			"createdAt": now,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err = c.coll().UpdateOne(ctx.UserContext(), filter, update, opts)
	return err
}

// Get godoc
// @Summary Get site JSON by name
// @Tags website
// @Security BearerAuth
// @Produce json
// @Param name path string true "Site key"
// @Success 200 {object} map[string]any
// @Router /website/site-data/{name} [get]
func (c *SiteDataController) Get(ctx *fiber.Ctx) error {
	name, err := parseSiteDataNameParam(ctx)
	if err != nil {
		return err
	}
	var rec struct {
		Data bson.M `bson:"data"`
	}
	if err := c.coll().FindOne(ctx.UserContext(), bson.M{"name": name}).Decode(&rec); err != nil {
		if err == mongo.ErrNoDocuments {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		c.log.Error("siteData find", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to load site data")
	}
	return ctx.JSON(rec.Data)
}

// ReplaceJSON godoc
// @Summary Replace site JSON (upsert)
// @Tags website
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param name path string true "Site key"
// @Success 200 {object} map[string]any
// @Router /website/site-data/{name} [post]
func (c *SiteDataController) ReplaceJSON(ctx *fiber.Ctx) error {
	name, err := parseSiteDataNameParam(ctx)
	if err != nil {
		return err
	}
	doc, err := domain.ParseAndValidateSiteDataJSON(ctx.Body())
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := c.upsert(ctx, name, doc); err != nil {
		c.log.Error("siteData upsert", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save site data")
	}
	return ctx.JSON(fiber.Map{"success": true})
}

// UploadJSON godoc
// @Summary Upload site JSON file (.json)
// @Tags website
// @Security BearerAuth
// @Accept mpfd
// @Produce json
// @Param name path string true "Site key"
// @Param file formData file true "JSON file"
// @Success 200 {object} map[string]any
// @Router /website/site-data/{name}/upload [post]
func (c *SiteDataController) UploadJSON(ctx *fiber.Ctx) error {
	name, err := parseSiteDataNameParam(ctx)
	if err != nil {
		return err
	}
	fh, err := ctx.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing multipart field \"file\"")
	}
	if !strings.HasSuffix(strings.ToLower(fh.Filename), ".json") {
		return fiber.NewError(fiber.StatusBadRequest, "file must use a .json extension")
	}
	src, err := fh.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not read uploaded file")
	}
	defer src.Close()
	raw, err := io.ReadAll(io.LimitReader(src, siteDataMaxUploadBytes+1))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not read uploaded file")
	}
	if len(raw) > siteDataMaxUploadBytes {
		return fiber.NewError(fiber.StatusBadRequest, "file too large")
	}
	doc, err := domain.ParseAndValidateSiteDataJSON(raw)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := c.upsert(ctx, name, doc); err != nil {
		c.log.Error("siteData upsert", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to save site data")
	}
	return ctx.JSON(fiber.Map{"success": true})
}
