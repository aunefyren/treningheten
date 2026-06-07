package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Principal is the authenticated caller, derived from either an access-token JWT
// or a Personal Access Token.
type Principal struct {
	UserID uuid.UUID
	Scope  string
	IsPAT  bool
}

// authenticate resolves an Authorization header value into a Principal. It
// dispatches on the PAT prefix: PATs are looked up (and validated) in the DB,
// everything else is parsed as a stateless access-token JWT.
func authenticate(authHeader string) (Principal, error) {
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return Principal{}, errors.New("no access token provided")
	}

	if strings.HasPrefix(tokenString, models.PATPrefix) {
		pat, err := database.GetPersonalAccessTokenByHash(auth.HashToken(tokenString))
		if err != nil {
			return Principal{}, errors.New("invalid personal access token")
		}
		if pat.RevokedAt != nil {
			return Principal{}, errors.New("personal access token has been revoked")
		}
		if pat.ExpiresAt.Before(time.Now()) {
			return Principal{}, errors.New("personal access token has expired")
		}
		// Best-effort usage tracking.
		if err := database.TouchPersonalAccessToken(pat.ID); err != nil {
			logger.Log.Info("failed to update PAT last-used. error: " + err.Error())
		}
		return Principal{UserID: pat.UserID, Scope: pat.Scope, IsPAT: true}, nil
	}

	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return Principal{}, err
	}
	return Principal{UserID: claims.UserID, Scope: claims.Scope, IsPAT: false}, nil
}

// Authenticate resolves a bearer credential into a Principal and verifies the
// account is enabled and (when SMTP is on) email-verified. It is the shared
// entry point used by both the Auth middleware and the MCP endpoint.
func Authenticate(authHeader string) (Principal, error) {
	principal, err := authenticate(authHeader)
	if err != nil {
		return Principal{}, err
	}

	enabled, err := database.VerifyUserIsEnabled(principal.UserID)
	if err != nil {
		return Principal{}, errors.New("failed to check account")
	}
	if !enabled {
		return Principal{}, errors.New("account disabled")
	}

	if files.ConfigFile.SMTPEnabled {
		verified, err := database.VerifyUserIsVerified(principal.UserID)
		if err != nil {
			return Principal{}, errors.New("failed to check verification")
		}
		if !verified {
			return Principal{}, errors.New("account not verified")
		}
	}

	return principal, nil
}

// bearerChallenge writes an RFC 6750 WWW-Authenticate challenge. When the
// external URL is configured it points clients at the protected-resource
// metadata (RFC 9728), which MCP clients use for discovery.
func bearerChallenge(context *gin.Context, status int, errCode string, description string) {
	parts := []string{`Bearer realm="Treningheten"`}
	if errCode != "" {
		parts = append(parts, `error="`+errCode+`"`)
		parts = append(parts, `error_description="`+description+`"`)
	}
	if external := strings.TrimRight(files.ConfigFile.TreninghetenExternalURL, "/"); external != "" {
		parts = append(parts, `resource_metadata="`+external+`/.well-known/oauth-protected-resource"`)
	}
	context.Header("WWW-Authenticate", strings.Join(parts, ", "))
	context.JSON(status, gin.H{"error": description})
	context.Abort()
}

// BearerChallenge emits an RFC 6750 WWW-Authenticate challenge (exported for the
// MCP endpoint so unauthenticated clients discover the OAuth flow).
func BearerChallenge(context *gin.Context, status int, errCode string, description string) {
	bearerChallenge(context, status, errCode, description)
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func Auth(admin bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		if context.GetHeader("Authorization") == "" {
			bearerChallenge(context, http.StatusUnauthorized, "invalid_request", "request does not contain an access token")
			return
		}

		principal, err := authenticate(context.GetHeader("Authorization"))
		if err != nil {
			logger.Log.Info("failed to validate token. error: " + err.Error())
			bearerChallenge(context, http.StatusUnauthorized, "invalid_token", "failed to validate token")
			return
		}

		userID := principal.UserID

		// Enforce admin privilege when required: the token must carry the admin
		// scope AND the user must still be an admin in the database.
		if admin {
			if !auth.ScopeHasAdmin(principal.Scope) {
				bearerChallenge(context, http.StatusForbidden, "insufficient_scope", "token lacks the admin scope")
				return
			}
			userObject, adminErr := database.GetUserInformation(userID)
			if adminErr != nil {
				logger.Log.Info("failed to check admin status. error: " + adminErr.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check account"})
				context.Abort()
				return
			} else if userObject.Admin == nil || !*userObject.Admin {
				bearerChallenge(context, http.StatusForbidden, "insufficient_scope", "admin privileges required")
				return
			}
		}

		// Read-only tokens (e.g. read-only PATs) may only perform safe methods.
		if !isReadMethod(context.Request.Method) && !auth.ScopeCanWrite(principal.Scope) {
			bearerChallenge(context, http.StatusForbidden, "insufficient_scope", "token is read-only")
			return
		}

		// Check if the user is enabled.
		enabled, err := database.VerifyUserIsEnabled(userID)
		if err != nil {
			logger.Log.Info("failed to check account. error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check account"})
			context.Abort()
			return
		} else if !enabled {
			context.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
			context.Abort()
			return
		}

		// If SMTP is enabled, verify the account is email-verified.
		if files.ConfigFile.SMTPEnabled {
			verified, err := database.VerifyUserIsVerified(userID)
			if err != nil {
				logger.Log.Info("failed to check verification. error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check verification"})
				context.Abort()
				return
			} else if !verified {
				// Ensure a verification code exists so the user can complete verification.
				hasVerificationCode, err := database.VerifyUserHasVerificationCode(userID)
				if err != nil {
					bearerChallenge(context, http.StatusUnauthorized, "invalid_token", "failed to validate token")
					return
				}
				if !hasVerificationCode {
					if _, err := database.GenerateRandomVerificationCodeForUser(userID); err != nil {
						bearerChallenge(context, http.StatusUnauthorized, "invalid_token", "failed to validate token")
						return
					}
				}
				context.JSON(http.StatusForbidden, gin.H{"error": "you must verify your account"})
				context.Abort()
				return
			}
		}

		context.Next()
	}
}

// GetAuthUsername resolves the calling user's ID from either token type.
func GetAuthUsername(tokenString string) (uuid.UUID, error) {
	if strings.TrimPrefix(tokenString, "Bearer ") == "" {
		return uuid.UUID{}, errors.New("no Authorization header given")
	}
	principal, err := authenticate(tokenString)
	if err != nil {
		return uuid.UUID{}, err
	}
	return principal.UserID, nil
}

// GetTokenClaims parses an access-token JWT's claims. It does not accept PATs.
func GetTokenClaims(tokenString string) (*auth.JWTClaim, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	if tokenString == "" {
		return &auth.JWTClaim{}, errors.New("no Authorization header given")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return &auth.JWTClaim{}, err
	}
	return claims, nil
}
