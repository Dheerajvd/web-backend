package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MFAState string

const (
	MFANotEnrolled MFAState = "NOT_ENROLLED"
	MFADisabled    MFAState = "DISABLED"
	MFAEnrolled    MFAState = "ENROLLED"
)

type MfaSessionKind string

const (
	MfaSessionEnrollment MfaSessionKind = "enrollment"
	MfaSessionLogin      MfaSessionKind = "login"
)

// MfaSession is a short-lived server-side step between password auth and token issue (TOTP enrollment or login).
type MfaSession struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID `bson:"userId" json:"userId"`
	AppID           string             `bson:"appId" json:"appId"`
	Kind            MfaSessionKind     `bson:"kind" json:"kind"`
	PlainTotpSecret string             `bson:"plainTotpSecret,omitempty" json:"-"` // only for enrollment; never returned to client
	ExpiresAt       time.Time          `bson:"expiresAt" json:"expiresAt"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
}
