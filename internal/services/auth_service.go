package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"

	"web-manager/internal/config"
	"web-manager/internal/crypto"
	"web-manager/internal/db"
	"web-manager/internal/domain"
)

const mfaTokenTTL = 5 * time.Minute

type AuthService struct {
	cfg   config.Config
	mongo *db.Mongo
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

// LoginResult is returned from Login (password step).
// step is "mfa_enrollment" (QR returned), "mfa_required" (OTP required), or "tokens".
type LoginResult struct {
	Step        string     `json:"step"`
	Tokens      *TokenPair `json:"tokens,omitempty"`
	MfaToken    string     `json:"mfaToken,omitempty"`
	QrPngBase64 string     `json:"qrPngBase64,omitempty"`
	OtpauthURL  string     `json:"otpauthUrl,omitempty"`
	AppID       string     `json:"appId,omitempty"`
	UserName    string     `json:"userName,omitempty"`
}

func NewAuthService(cfg config.Config, mongo *db.Mongo) *AuthService {
	return &AuthService{cfg: cfg, mongo: mongo}
}

func effectiveMFAState(u domain.User) domain.MFAState {
	if !u.MfaEnabled {
		return domain.MFANotEnrolled
	}
	if u.MfaState == domain.MFAEnrolled && strings.TrimSpace(u.TotpSecretEnc) != "" {
		return domain.MFAEnrolled
	}
	if u.MfaState == "" || u.MfaState == domain.MFANotEnrolled || u.MfaState == domain.MFADisabled {
		return domain.MFANotEnrolled
	}
	return domain.MFANotEnrolled
}

func (s *AuthService) userCanAccessApp(u domain.User, appID string) bool {
	return domain.UserCanAccessApp(u.Role, u.AppIDs, appID)
}

// Login validates password and app access.
// If user has opted into MFA (mfaEnabled=true), this returns an MFA step response and tokens are only issued after ValidateOTP.
// totpIssuer is shown in the authenticator app (e.g. product name from app_settings).
func (s *AuthService) Login(ctx context.Context, email, password, appID, totpIssuer string) (*LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	appID = strings.TrimSpace(appID)
	if email == "" || password == "" {
		return nil, fiberErr(400, "email and password required")
	}
	if appID == "" {
		return nil, fiberErr(400, "appId required")
	}
	if strings.TrimSpace(totpIssuer) == "" {
		totpIssuer = "web-manager"
	}

	var user domain.User
	err := s.usersCol().FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fiberErr(401, "invalid credentials")
		}
		return nil, fiberErr(500, "db error")
	}
	if user.Status == domain.UserStatusDisabled {
		return nil, fiberErr(403, "user disabled")
	}
	if !s.userCanAccessApp(user, appID) {
		return nil, fiberErr(403, "user is not assigned to this app")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, fiberErr(401, "invalid credentials")
	}

	// MFA off => tokens immediately
	if !user.MfaEnabled {
		tokens, err := s.issueTokenPair(ctx, user, appID)
		if err != nil {
			return nil, err
		}
		return &LoginResult{Step: "tokens", Tokens: tokens, AppID: appID, UserName: user.Name}, nil
	}

	// MFA enabled but not enrolled yet => return QR (and store pending secret on user doc)
	if strings.TrimSpace(user.TotpSecretEnc) == "" {
		return s.beginMFAEnrollmentStateless(ctx, user, appID, totpIssuer)
	}

	// MFA enabled and enrolled => OTP required
	mfaTok, err := s.signMFAToken(user.ID, appID, "login")
	if err != nil {
		return nil, fiberErr(500, "failed to sign mfa token")
	}
	return &LoginResult{Step: "mfa_required", MfaToken: mfaTok, AppID: appID, UserName: user.Name}, nil
}

func (s *AuthService) beginMFAEnrollmentStateless(ctx context.Context, user domain.User, appID, issuer string) (*LoginResult, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fiberErr(500, "failed to generate TOTP key")
	}
	url := key.URL()
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return nil, fiberErr(500, "failed to encode QR")
	}
	qrB64 := base64.StdEncoding.EncodeToString(png)

	now := time.Now().UTC()
	enc, err := crypto.EncryptAESGCM([]byte(key.Secret()), crypto.DeriveMFAKey(s.cfg.Auth.JWTSecret))
	if err != nil {
		return nil, fiberErr(500, "failed to encrypt TOTP secret")
	}
	_, err = s.usersCol().UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{
		"totpPendingSecretEnc": enc,
		"totpPendingAt":        now,
		"updatedAt":            now,
	}})
	if err != nil {
		return nil, fiberErr(500, "failed to start MFA enrollment")
	}
	mfaTok, err := s.signMFAToken(user.ID, appID, "enroll")
	if err != nil {
		return nil, fiberErr(500, "failed to sign mfa token")
	}

	return &LoginResult{
		Step:        "mfa_enrollment",
		MfaToken:    mfaTok,
		QrPngBase64: qrB64,
		OtpauthURL:  url,
		AppID:       appID,
		UserName:    user.Name,
	}, nil
}

type MFATokenClaims struct {
	Sub     string `json:"sub"`
	AppID   string `json:"appId"`
	Purpose string `json:"purpose"` // "enroll" or "login"
	jwt.RegisteredClaims
}

func (s *AuthService) signMFAToken(userID primitive.ObjectID, appID, purpose string) (string, error) {
	now := time.Now().UTC()
	claims := MFATokenClaims{
		Sub:     userID.Hex(),
		AppID:   appID,
		Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(mfaTokenTTL)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func (s *AuthService) parseMFAToken(mfaToken string) (*MFATokenClaims, error) {
	mfaToken = strings.TrimSpace(mfaToken)
	if mfaToken == "" {
		return nil, fiberErr(400, "mfaToken required")
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	claims := &MFATokenClaims{}
	tok, err := parser.ParseWithClaims(mfaToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fiberErr(401, "invalid or expired mfaToken")
	}
	return claims, nil
}

// ValidateOTP validates an OTP for either initial enrollment or normal login and returns tokens.
func (s *AuthService) ValidateOTP(ctx context.Context, mfaToken, code string) (*TokenPair, error) {
	code = strings.TrimSpace(strings.ReplaceAll(code, " ", ""))
	if code == "" {
		return nil, fiberErr(400, "code required")
	}
	claims, err := s.parseMFAToken(mfaToken)
	if err != nil {
		return nil, err
	}
	uid, err := primitive.ObjectIDFromHex(claims.Sub)
	if err != nil {
		return nil, fiberErr(401, "invalid mfaToken")
	}

	var user domain.User
	if err := s.usersCol().FindOne(ctx, bson.M{"_id": uid}).Decode(&user); err != nil {
		return nil, fiberErr(401, "invalid mfaToken")
	}
	if user.Status == domain.UserStatusDisabled {
		return nil, fiberErr(403, "user disabled")
	}
	if !s.userCanAccessApp(user, claims.AppID) {
		return nil, fiberErr(403, "user is not assigned to this app")
	}
	if !user.MfaEnabled {
		return nil, fiberErr(400, "MFA not enabled")
	}

	switch claims.Purpose {
	case "enroll":
		if strings.TrimSpace(user.TotpPendingSecretEnc) == "" {
			return nil, fiberErr(400, "MFA enrollment not started")
		}
		plain, err := crypto.DecryptAESGCM(user.TotpPendingSecretEnc, crypto.DeriveMFAKey(s.cfg.Auth.JWTSecret))
		if err != nil {
			return nil, fiberErr(500, "failed to decrypt TOTP secret")
		}
		if !totp.Validate(code, string(plain)) {
			return nil, fiberErr(401, "invalid authenticator code")
		}
		now := time.Now().UTC()
		_, err = s.usersCol().UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
			"$set": bson.M{
				"mfaState":      domain.MFAEnrolled,
				"totpSecretEnc": user.TotpPendingSecretEnc,
				"mfaEnrolledAt": now,
				"updatedAt":     now,
			},
			"$unset": bson.M{
				"totpPendingSecretEnc": "",
				"totpPendingAt":        "",
			},
		})
		if err != nil {
			return nil, fiberErr(500, "failed to save MFA")
		}
		user.TotpSecretEnc = user.TotpPendingSecretEnc
		user.TotpPendingSecretEnc = ""
		return s.issueTokenPair(ctx, user, claims.AppID)

	case "login":
		if strings.TrimSpace(user.TotpSecretEnc) == "" {
			return nil, fiberErr(400, "MFA not configured")
		}
		plain, err := crypto.DecryptAESGCM(user.TotpSecretEnc, crypto.DeriveMFAKey(s.cfg.Auth.JWTSecret))
		if err != nil {
			return nil, fiberErr(500, "failed to decrypt TOTP secret")
		}
		if !totp.Validate(code, string(plain)) {
			return nil, fiberErr(401, "invalid authenticator code")
		}
		return s.issueTokenPair(ctx, user, claims.AppID)

	default:
		return nil, fiberErr(401, "invalid mfaToken")
	}
}

func (s *AuthService) issueTokenPair(ctx context.Context, user domain.User, appID string) (*TokenPair, error) {
	access, err := s.signAccessToken(user, appID)
	if err != nil {
		return nil, fiberErr(500, "failed to sign token")
	}
	refreshPlain, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, fiberErr(500, "failed to generate refresh token")
	}
	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(s.cfg.Auth.RefreshTokenTTLDays) * 24 * time.Hour)
	rt := domain.RefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		AppID:     appID,
		TokenHash: refreshHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	if _, err := s.refreshTokensCol().InsertOne(ctx, rt); err != nil {
		return nil, fiberErr(500, "failed to persist refresh token")
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refreshPlain,
		ExpiresIn:    s.cfg.Auth.AccessTokenTTLMinutes * 60,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken, ip string) (*TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fiberErr(400, "refreshToken required")
	}

	hash := hashRefreshToken(refreshToken)

	var stored domain.RefreshToken
	err := s.refreshTokensCol().FindOne(ctx, bson.M{"tokenHash": hash, "revokedAt": bson.M{"$exists": false}}).Decode(&stored)
	if err != nil {
		return nil, fiberErr(401, "invalid refresh token")
	}
	if time.Now().UTC().After(stored.ExpiresAt) {
		return nil, fiberErr(401, "refresh token expired")
	}

	var user domain.User
	if err := s.usersCol().FindOne(ctx, bson.M{"_id": stored.UserID}).Decode(&user); err != nil {
		return nil, fiberErr(401, "invalid refresh token")
	}
	if user.Status == domain.UserStatusDisabled {
		return nil, fiberErr(403, "user disabled")
	}
	appID := stored.AppID
	if appID == "" {
		appID = "default"
	}

	now := time.Now().UTC()
	_, _ = s.refreshTokensCol().UpdateOne(ctx, bson.M{"_id": stored.ID}, bson.M{"$set": bson.M{"revokedAt": now, "lastUsedAt": now}})

	newPlain, newHash, err := newRefreshToken()
	if err != nil {
		return nil, fiberErr(500, "failed to generate refresh token")
	}
	newStored := domain.RefreshToken{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		AppID:     appID,
		TokenHash: newHash,
		ExpiresAt: now.Add(time.Duration(s.cfg.Auth.RefreshTokenTTLDays) * 24 * time.Hour),
		CreatedAt: now,
	}
	if _, err := s.refreshTokensCol().InsertOne(ctx, newStored); err != nil {
		return nil, fiberErr(500, "failed to persist refresh token")
	}

	access, err := s.signAccessToken(user, appID)
	if err != nil {
		return nil, fiberErr(500, "failed to sign token")
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: newPlain,
		ExpiresIn:    s.cfg.Auth.AccessTokenTTLMinutes * 60,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) signAccessToken(user domain.User, appID string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":   user.ID.Hex(),
		"email": user.Email,
		"name":  user.Name,
		"role":  string(user.Role),
		"appId": appID,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Duration(s.cfg.Auth.AccessTokenTTLMinutes) * time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func (s *AuthService) ValidateAccessToken(tokenString string) (*jwt.RegisteredClaims, jwt.MapClaims, error) {
	tokenString = strings.TrimSpace(strings.TrimPrefix(tokenString, "Bearer "))
	if tokenString == "" {
		return nil, nil, fmt.Errorf("missing token")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	var claims jwt.MapClaims
	tok, err := parser.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return nil, nil, fmt.Errorf("invalid token")
	}

	rc, _ := tok.Claims.(jwt.MapClaims)
	_ = rc
	return nil, claims, nil
}

func (s *AuthService) ParseAccessToken(authHeader string) (jwt.MapClaims, error) {
	tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if tokenString == "" {
		return nil, fmt.Errorf("missing token")
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	claims := jwt.MapClaims{}
	tok, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *AuthService) EnsureSuperUser(ctx context.Context, name, email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return fmt.Errorf("email/password required")
	}

	count, err := s.usersCol().CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	appIDs := parseAppIDsCSV(os.Getenv("SUPER_USER_APP_IDS"))

	now := time.Now().UTC()
	u := domain.User{
		ID:           primitive.NewObjectID(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleSuperUser,
		Status:       domain.UserStatusActive,
		AppIDs:       appIDs,
		MfaEnabled:   false,
		MfaState:     domain.MFANotEnrolled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = s.usersCol().InsertOne(ctx, u, options.InsertOne())
	return err
}

func parseAppIDsCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *AuthService) usersCol() *mongo.Collection {
	return s.mongo.DB.Collection("users")
}

func (s *AuthService) refreshTokensCol() *mongo.Collection {
	return s.mongo.DB.Collection("refresh_tokens")
}

// (mfa_sessions collection no longer used; MFA is stateless via short-lived JWT mfaToken)

func newRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, hashRefreshToken(plain), nil
}

func hashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

type StatusError struct {
	Status  int
	Message string
}

func (e *StatusError) Error() string { return e.Message }

func fiberErr(status int, msg string) error {
	return &StatusError{Status: status, Message: msg}
}
