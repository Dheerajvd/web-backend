package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"web-manager/internal/db"
	"web-manager/internal/domain"
)

const appSettingsCollection = "app_settings"

type AppSettingsRepository struct {
	col *mongo.Collection
}

func NewAppSettingsRepository(m *db.Mongo) *AppSettingsRepository {
	return &AppSettingsRepository{col: m.DB.Collection(appSettingsCollection)}
}

// FindDefault returns the document with key=default, or the most recently updated row if none.
func (r *AppSettingsRepository) FindDefault(ctx context.Context) (*domain.AppSettings, error) {
	var doc domain.AppSettings
	err := r.col.FindOne(ctx, bson.M{"key": domain.AppSettingsKeyDefault}).Decode(&doc)
	if err == nil {
		return &doc, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updatedAt", Value: -1}})
	err = r.col.FindOne(ctx, bson.M{}, opts).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *AppSettingsRepository) Insert(ctx context.Context, s *domain.AppSettings) error {
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, s)
	return err
}

func (r *AppSettingsRepository) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{})
}
