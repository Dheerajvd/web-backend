package services

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"web-manager/internal/db"
	"web-manager/internal/domain"
)

type LookupService struct {
	mongo *db.Mongo
}

func NewLookupService(m *db.Mongo) *LookupService {
	return &LookupService{mongo: m}
}

func (s *LookupService) rolesCol() *mongo.Collection {
	return s.mongo.DB.Collection(domain.CollectionRoleDefinitions)
}

func (s *LookupService) appsCol() *mongo.Collection {
	return s.mongo.DB.Collection(domain.CollectionApplicationIDs)
}

// EnsureLookupDefaults seeds roles and a default application if collections are empty for those keys.
func (s *LookupService) EnsureLookupDefaults(ctx context.Context) error {
	now := time.Now().UTC()
	for _, row := range []struct {
		code domain.Role
		name string
	}{
		{domain.RoleSuperUser, "Super User"},
		{domain.RoleManager, "Manager"},
		{domain.RoleAppUser, "App User"},
	} {
		n, err := s.rolesCol().CountDocuments(ctx, bson.M{"code": row.code})
		if err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		_, err = s.rolesCol().InsertOne(ctx, domain.RoleDefinition{
			ID:        primitive.NewObjectID(),
			Code:      row.code,
			Name:      row.name,
			CreatedAt: now,
			UpdatedAt: now,
		})
		if err != nil {
			return err
		}
	}

	n, err := s.appsCol().CountDocuments(ctx, bson.M{"appId": "web-manager"})
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = s.appsCol().InsertOne(ctx, domain.ApplicationRecord{
		ID:        primitive.NewObjectID(),
		AppID:     "web-manager",
		Name:      "Web Manager",
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}

type RoleLookupItem struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type AppLookupItem struct {
	ID    string `json:"id"`
	AppID string `json:"appId"`
	Name  string `json:"name"`
}

// ListLookupsPage returns a page of lookup rows for purpose: "roles", "appIds" (aliases: apps, applications).
func (s *LookupService) ListLookupsPage(ctx context.Context, purpose string, skip int64, limit int) (any, int64, error) {
	p := strings.ToLower(strings.TrimSpace(purpose))
	lim := int64(limit)
	switch p {
	case "roles":
		total, err := s.rolesCol().CountDocuments(ctx, bson.M{})
		if err != nil {
			return nil, 0, fiberErr(500, "failed to list roles")
		}
		cur, err := s.rolesCol().Find(ctx, bson.M{}, options.Find().
			SetSort(bson.D{{Key: "code", Value: 1}}).
			SetSkip(skip).
			SetLimit(lim))
		if err != nil {
			return nil, 0, fiberErr(500, "failed to list roles")
		}
		defer cur.Close(ctx)
		out := make([]RoleLookupItem, 0)
		for cur.Next(ctx) {
			var doc domain.RoleDefinition
			if err := cur.Decode(&doc); err != nil {
				continue
			}
			out = append(out, RoleLookupItem{
				ID:   doc.ID.Hex(),
				Code: string(doc.Code),
				Name: doc.Name,
			})
		}
		return out, total, nil

	case "appids", "apps", "applications":
		total, err := s.appsCol().CountDocuments(ctx, bson.M{})
		if err != nil {
			return nil, 0, fiberErr(500, "failed to list applications")
		}
		cur, err := s.appsCol().Find(ctx, bson.M{}, options.Find().
			SetSort(bson.D{{Key: "appId", Value: 1}}).
			SetSkip(skip).
			SetLimit(lim))
		if err != nil {
			return nil, 0, fiberErr(500, "failed to list applications")
		}
		defer cur.Close(ctx)
		out := make([]AppLookupItem, 0)
		for cur.Next(ctx) {
			var doc domain.ApplicationRecord
			if err := cur.Decode(&doc); err != nil {
				continue
			}
			out = append(out, AppLookupItem{
				ID:    doc.ID.Hex(),
				AppID: doc.AppID,
				Name:  doc.Name,
			})
		}
		return out, total, nil

	default:
		return nil, 0, fiberErr(400, "unknown lookup purpose; use roles or appIds")
	}
}
