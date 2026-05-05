package db

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureIndexes(ctx context.Context, m *Mongo) error {
	if err := ensureAppSettingsIndexes(ctx, m.DB.Collection("app_settings")); err != nil {
		return err
	}
	if err := ensureUsersIndexes(ctx, m.DB.Collection("users")); err != nil {
		return err
	}
	if err := ensureVideosIndexes(ctx, m.DB.Collection("videos")); err != nil {
		return err
	}
	if err := ensureSourcesIndexes(ctx, m.DB.Collection("sources")); err != nil {
		return err
	}
	if err := ensureRefreshTokensIndexes(ctx, m.DB.Collection("refresh_tokens")); err != nil {
		return err
	}
	if err := ensureMfaSessionsIndexes(ctx, m.DB.Collection("mfa_sessions")); err != nil {
		return err
	}
	if err := ensurePublicFormIndexes(ctx, m.DB.Collection("corrections"), "corrections"); err != nil {
		return err
	}
	if err := ensurePublicFormIndexes(ctx, m.DB.Collection("suggestions"), "suggestions"); err != nil {
		return err
	}
	if err := ensureContactsIndexes(ctx, m.DB.Collection("contacts")); err != nil {
		return err
	}
	if err := ensureRoleDefinitionsIndexes(ctx, m.DB.Collection("roles")); err != nil {
		return err
	}
	if err := ensureApplicationIDsIndexes(ctx, m.DB.Collection("applicationIDs")); err != nil {
		return err
	}
	if err := ensureSiteDataIndexes(ctx, m.DB.Collection("siteData")); err != nil {
		return err
	}
	return nil
}

func ensureAppSettingsIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("uniq_key"),
		},
	})
	if err != nil {
		return fmt.Errorf("app_settings indexes: %w", err)
	}
	return nil
}

func ensureUsersIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_email"),
		},
		{
			Keys:    bson.D{{Key: "role", Value: 1}},
			Options: options.Index().SetName("role"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status"),
		},
	})
	if err != nil {
		return fmt.Errorf("users indexes: %w", err)
	}
	return nil
}

func ensureVideosIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_slug"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "publishedAt", Value: -1}},
			Options: options.Index().SetName("status_publishedAt"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("createdAt"),
		},
	})
	if err != nil {
		return fmt.Errorf("videos indexes: %w", err)
	}
	return nil
}

func ensureSourcesIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "videoId", Value: 1}},
			Options: options.Index().SetName("videoId"),
		},
	})
	if err != nil {
		return fmt.Errorf("sources indexes: %w", err)
	}
	return nil
}

func ensureMfaSessionsIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("ttl_expiresAt"),
		},
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("userId"),
		},
	})
	if err != nil {
		return fmt.Errorf("mfa_sessions indexes: %w", err)
	}
	return nil
}

func ensureRefreshTokensIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "userId", Value: 1}},
			Options: options.Index().SetName("userId"),
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("ttl_expiresAt"),
		},
		{
			Keys:    bson.D{{Key: "tokenHash", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_tokenHash"),
		},
	})
	if err != nil {
		return fmt.Errorf("refresh_tokens indexes: %w", err)
	}
	return nil
}

func ensurePublicFormIndexes(ctx context.Context, col *mongo.Collection, name string) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("status_createdAt"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("createdAt"),
		},
	})
	if err != nil {
		return fmt.Errorf("%s indexes: %w", name, err)
	}
	return nil
}

func ensureContactsIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("createdAt"),
		},
	})
	if err != nil {
		return fmt.Errorf("contacts indexes: %w", err)
	}
	return nil
}

func ensureRoleDefinitionsIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "code", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_code"),
		},
	})
	if err != nil {
		return fmt.Errorf("roles indexes: %w", err)
	}
	return nil
}

func ensureApplicationIDsIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "appId", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_appId"),
		},
	})
	if err != nil {
		return fmt.Errorf("applicationIDs indexes: %w", err)
	}
	return nil
}

func ensureSiteDataIndexes(ctx context.Context, col *mongo.Collection) error {
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_name"),
		},
	})
	if err != nil {
		return fmt.Errorf("siteData indexes: %w", err)
	}
	return nil
}
