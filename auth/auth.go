package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

const (
	// AccessTokenTTL is the lifetime of a stateless access token JWT.
	AccessTokenTTL = 60 * time.Minute
	// RefreshTokenTTL is the lifetime of an opaque refresh token.
	RefreshTokenTTL = 30 * 24 * time.Hour
)

type JWTClaim struct {
	UserID   uuid.UUID `json:"id"`
	Admin    bool      `json:"admin"`
	Scope    string    `json:"scope,omitempty"`
	ClientID string    `json:"client_id,omitempty"`
	jwt.RegisteredClaims
}

// ScopeForUser returns the default granted scope for a first-party user login.
func ScopeForUser(admin bool) string {
	if admin {
		return models.ScopeAPI + " " + models.ScopeAdmin
	}
	return models.ScopeAPI
}

// OAuthIssuer returns the issuer/resource identifier for tokens, derived from
// the configured external URL. Empty when no external URL is configured, in
// which case audience binding is skipped (MCP requires it to be set).
func OAuthIssuer() string {
	return strings.TrimRight(files.ConfigFile.TreninghetenExternalURL, "/")
}

// GenerateAccessToken builds a signed, stateless access-token JWT.
func GenerateAccessToken(userID uuid.UUID, admin bool, scope string, clientID string) (tokenString string, expiresAt time.Time, err error) {
	now := time.Now()
	expiresAt = now.Add(AccessTokenTTL)

	claims := &JWTClaim{
		UserID:   userID,
		Admin:    admin,
		Scope:    scope,
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "Treningheten",
		},
	}

	// Bind issuer and audience to the resource when an external URL is set.
	if issuer := OAuthIssuer(); issuer != "" {
		claims.Issuer = issuer
		claims.Audience = jwt.ClaimStrings{issuer}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(files.GetPrivateKey(1))
	return
}

// GenerateJWTFromClaims re-signs an existing set of claims.
func GenerateJWTFromClaims(claims *JWTClaim) (tokenString string, err error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(files.GetPrivateKey(1))
	return
}

// ParseToken validates the signature and temporal claims and returns the claims.
func ParseToken(signedToken string) (*JWTClaim, error) {
	jwtKey := files.GetPrivateKey(1)
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		return nil, errors.New("could not parse claims")
	} else if claims.ExpiresAt == nil || claims.NotBefore == nil {
		return nil, errors.New("claims not present")
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		return nil, errors.New("token has expired")
	}
	if claims.NotBefore.Time.After(now) {
		return nil, errors.New("token has not begun")
	}

	// Validate audience binding when the resource is configured.
	if issuer := OAuthIssuer(); issuer != "" {
		if !audienceContains(claims.Audience, issuer) {
			return nil, errors.New("token audience mismatch")
		}
	}

	return claims, nil
}

// ValidateToken validates a token and optionally enforces admin privileges.
func ValidateToken(signedToken string, admin bool) error {
	claims, err := ParseToken(signedToken)
	if err != nil {
		return err
	}

	if admin {
		if !claims.Admin {
			return errors.New("token is not an admin session")
		}
		userObject, userErr := database.GetUserInformation(claims.UserID)
		if userErr != nil {
			return errors.New("failed to check admin status")
		} else if userObject.Admin == nil || !*userObject.Admin {
			return errors.New("token is not an admin session")
		}
	}

	return nil
}

func audienceContains(audience jwt.ClaimStrings, value string) bool {
	for _, a := range audience {
		if a == value {
			return true
		}
	}
	return false
}

// --- Refresh tokens ---

// HashToken returns the hex-encoded SHA-256 hash used to store opaque tokens.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// generateOpaqueToken returns a cryptographically random, URL-safe token string.
func generateOpaqueToken() string {
	return base64.RawURLEncoding.EncodeToString([]byte(randstr.String(48)))
}

// GeneratePATToken returns a new Personal Access Token string (prefix + random).
func GeneratePATToken() string {
	return models.PATPrefix + base64.RawURLEncoding.EncodeToString([]byte(randstr.String(40)))
}

// IssueTokenSet creates and persists a new access + refresh token pair for a user.
func IssueTokenSet(userID uuid.UUID, admin bool, scope string, clientID string) (models.OAuthTokenResponse, error) {
	accessToken, expiresAt, err := GenerateAccessToken(userID, admin, scope, clientID)
	if err != nil {
		return models.OAuthTokenResponse{}, err
	}

	refreshToken := generateOpaqueToken()
	stored := models.OAuthRefreshToken{
		TokenHash: HashToken(refreshToken),
		UserID:    userID,
		ClientID:  clientID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
	}
	if err := database.CreateRefreshToken(&stored); err != nil {
		return models.OAuthTokenResponse{}, err
	}

	return models.OAuthTokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(expiresAt).Seconds()),
		RefreshToken: refreshToken,
		Scope:        scope,
	}, nil
}

// RefreshTokenSet validates and rotates a refresh token, returning a fresh token
// pair. Reuse of an already-rotated token revokes the whole chain.
func RefreshTokenSet(presentedToken string, clientID string) (models.OAuthTokenResponse, error) {
	stored, err := database.GetRefreshTokenByHash(HashToken(presentedToken))
	if err != nil {
		return models.OAuthTokenResponse{}, errors.New("invalid refresh token")
	}

	// Reuse detection: a presented-but-already-revoked token means the chain is
	// compromised; revoke everything reachable from it.
	if stored.RevokedAt != nil {
		if chainErr := database.RevokeRefreshTokenChain(stored.ID); chainErr != nil {
			logger.Log.Error("failed to revoke refresh token chain. error: " + chainErr.Error())
		}
		return models.OAuthTokenResponse{}, errors.New("refresh token has been revoked")
	}

	if stored.ExpiresAt.Before(time.Now()) {
		return models.OAuthTokenResponse{}, errors.New("refresh token has expired")
	}

	// Refresh tokens are bound to the client they were issued to.
	if clientID != "" && stored.ClientID != clientID {
		return models.OAuthTokenResponse{}, errors.New("refresh token was issued to a different client")
	}

	// Re-derive admin status so privilege changes propagate on refresh.
	userObject, err := database.GetUserInformation(stored.UserID)
	if err != nil {
		return models.OAuthTokenResponse{}, errors.New("failed to load user")
	}
	admin := userObject.Admin != nil && *userObject.Admin

	accessToken, expiresAt, err := GenerateAccessToken(stored.UserID, admin, stored.Scope, stored.ClientID)
	if err != nil {
		return models.OAuthTokenResponse{}, err
	}

	newRefreshToken := generateOpaqueToken()
	newStored := models.OAuthRefreshToken{
		TokenHash: HashToken(newRefreshToken),
		UserID:    stored.UserID,
		ClientID:  stored.ClientID,
		Scope:     stored.Scope,
		ExpiresAt: time.Now().Add(RefreshTokenTTL),
	}
	if err := database.RotateRefreshToken(stored.ID, &newStored); err != nil {
		return models.OAuthTokenResponse{}, err
	}

	return models.OAuthTokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(expiresAt).Seconds()),
		RefreshToken: newRefreshToken,
		Scope:        stored.Scope,
	}, nil
}
