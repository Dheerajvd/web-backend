package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	Role         Role               `bson:"role" json:"role"`
	Status       UserStatus         `bson:"status" json:"status"`
	// AppIDs lists application ids this user may access (e.g. "admin-web", "public-site"). SUPER_USER may have an empty list = all apps.
	AppIDs []string `bson:"appIds,omitempty" json:"appIds,omitempty"`
	// MFA (TOTP)
	// MfaEnabled gates the entire MFA flow. If false, login never requires MFA.
	MfaEnabled bool     `bson:"mfaEnabled,omitempty" json:"mfaEnabled,omitempty"`
	MfaState   MFAState `bson:"mfaState,omitempty" json:"mfaState,omitempty"`
	// TotpSecretEnc is the enrolled secret (AES-GCM encrypted).
	TotpSecretEnc string `bson:"totpSecretEnc,omitempty" json:"-"`
	// TotpPendingSecretEnc is a temporary secret used during first-time enrollment (AES-GCM encrypted).
	// It is cleared after successful OTP validation.
	TotpPendingSecretEnc string     `bson:"totpPendingSecretEnc,omitempty" json:"-"`
	TotpPendingAt        *time.Time `bson:"totpPendingAt,omitempty" json:"totpPendingAt,omitempty"`
	MfaEnrolledAt        *time.Time `bson:"mfaEnrolledAt,omitempty" json:"mfaEnrolledAt,omitempty"`
	CreatedAt            time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time  `bson:"updatedAt" json:"updatedAt"`
}

type VideoStatus string

const (
	VideoStatusDraft     VideoStatus = "draft"
	VideoStatusPublished VideoStatus = "published"
)

type Video struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string             `bson:"title" json:"title"`
	Slug            string             `bson:"slug" json:"slug"`
	YouTubeURL      string             `bson:"youtubeUrl" json:"youtubeUrl"`
	ThumbnailURL    string             `bson:"thumbnailUrl" json:"thumbnailUrl"`
	ShortDesc       string             `bson:"shortDescription" json:"shortDescription"`
	FullDesc        string             `bson:"fullDescription" json:"fullDescription"`
	Category        string             `bson:"category" json:"category"`
	Tags            []string           `bson:"tags" json:"tags"`
	Status          VideoStatus        `bson:"status" json:"status"`
	PublishedAt     *time.Time         `bson:"publishedAt,omitempty" json:"publishedAt,omitempty"`
	CreatedByUserID primitive.ObjectID `bson:"createdBy" json:"createdBy"`
	UpdatedByUserID primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type SourceType string

const (
	SourceTypeArticle SourceType = "article"
	SourceTypeBook    SourceType = "book"
	SourceTypeArchive SourceType = "archive"
	SourceTypeVideo   SourceType = "video"
)

type Source struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VideoID          primitive.ObjectID `bson:"videoId" json:"videoId"`
	Title            string             `bson:"title" json:"title"`
	Type             SourceType         `bson:"type" json:"type"`
	URL              string             `bson:"url" json:"url"`
	Note             string             `bson:"note" json:"note"`
	CredibilityScore int                `bson:"credibilityScore" json:"credibilityScore"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type CorrectionStatus string

const (
	CorrectionStatusNew       CorrectionStatus = "new"
	CorrectionStatusReviewing CorrectionStatus = "reviewing"
	CorrectionStatusApplied   CorrectionStatus = "applied"
	CorrectionStatusRejected  CorrectionStatus = "rejected"
)

type Correction struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	VideoID   primitive.ObjectID `bson:"videoId" json:"videoId"`
	Message   string             `bson:"message" json:"message"`
	SourceURL string             `bson:"sourceUrl" json:"sourceUrl"`
	Email     *string            `bson:"email,omitempty" json:"email,omitempty"`
	Status    CorrectionStatus   `bson:"status" json:"status"`
	IP        string             `bson:"ip" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type SuggestionStatus string

const (
	SuggestionStatusNew       SuggestionStatus = "new"
	SuggestionStatusReviewing SuggestionStatus = "reviewing"
	SuggestionStatusAccepted  SuggestionStatus = "accepted"
	SuggestionStatusRejected  SuggestionStatus = "rejected"
)

type Suggestion struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Links       []string           `bson:"links" json:"links"`
	Email       *string            `bson:"email,omitempty" json:"email,omitempty"`
	Status      SuggestionStatus   `bson:"status" json:"status"`
	IP          string             `bson:"ip" json:"-"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type Contact struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Message   string             `bson:"message" json:"message"`
	IP        string             `bson:"ip" json:"-"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type AuditLog struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ActorID   *primitive.ObjectID `bson:"actorId,omitempty" json:"actorId,omitempty"`
	Action    string              `bson:"action" json:"action"`
	Entity    string              `bson:"entity" json:"entity"`
	EntityID  *primitive.ObjectID `bson:"entityId,omitempty" json:"entityId,omitempty"`
	Meta      map[string]any      `bson:"meta,omitempty" json:"meta,omitempty"`
	IP        string              `bson:"ip" json:"ip"`
	CreatedAt time.Time           `bson:"createdAt" json:"createdAt"`
}

type RefreshToken struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"userId" json:"userId"`
	AppID      string             `bson:"appId,omitempty" json:"appId,omitempty"`
	TokenHash  string             `bson:"tokenHash" json:"-"`
	RevokedAt  *time.Time         `bson:"revokedAt,omitempty" json:"revokedAt,omitempty"`
	ExpiresAt  time.Time          `bson:"expiresAt" json:"expiresAt"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	LastUsedAt *time.Time         `bson:"lastUsedAt,omitempty" json:"lastUsedAt,omitempty"`
}
