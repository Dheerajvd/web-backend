package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RoleDefinition is a reference row for UI dropdowns (id + display name).
// Code matches the string stored on users.role (e.g. SUPER_USER).
type RoleDefinition struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      Role               `bson:"code" json:"code"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// ApplicationRecord is stored in the applicationIDs collection (app slug + display name).
type ApplicationRecord struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	AppID     string             `bson:"appId" json:"appId"`
	Name      string             `bson:"name" json:"name"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

const CollectionRoleDefinitions = "roles"
const CollectionApplicationIDs = "applicationIDs"
