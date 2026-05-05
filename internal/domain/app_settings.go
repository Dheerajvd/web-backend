package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const AppSettingsKeyDefault = "default"

// AppSettings is stored in collection app_settings (one logical row per key, usually key=default).
type AppSettings struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key       string             `bson:"key" json:"key"`
	AppName   string             `bson:"appName" json:"appName"`
	Origins   []string           `bson:"origins" json:"origins"`
	SMTP      *SMTPSettings      `bson:"smtp,omitempty" json:"smtp,omitempty"`
	Extra     map[string]any     `bson:"extra,omitempty" json:"extra,omitempty"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type SMTPSettings struct {
	Host     string `bson:"host" json:"host"`
	Port     int    `bson:"port" json:"port"`
	User     string `bson:"user" json:"user"`
	Password string `bson:"password" json:"password"`
	From     string `bson:"from" json:"from"`
	FromName string `bson:"fromName,omitempty" json:"fromName,omitempty"`
}
