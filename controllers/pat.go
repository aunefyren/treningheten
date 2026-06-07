package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APICreatePersonalAccessToken creates a PAT for the calling user and returns
// the plaintext token exactly once.
func APICreatePersonalAccessToken(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to identify user."})
		context.Abort()
		return
	}

	var request models.PATCreationRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request."})
		context.Abort()
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "A name is required."})
		context.Abort()
		return
	}

	// Determine the base access scope (read-only by default).
	base := models.ScopeAPIRead
	switch request.Scope {
	case "", models.ScopeAPIRead:
		base = models.ScopeAPIRead
	case models.ScopeAPIWrite:
		base = models.ScopeAPIWrite
	default:
		context.JSON(http.StatusBadRequest, gin.H{"error": "Scope must be 'api:read' or 'api:write'."})
		context.Abort()
		return
	}
	scopes := []string{base}

	// The admin scope requires the user to actually be an admin.
	if request.Admin {
		userObject, err := database.GetUserInformation(userID)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check account."})
			context.Abort()
			return
		}
		if userObject.Admin == nil || !*userObject.Admin {
			context.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create admin tokens."})
			context.Abort()
			return
		}
		scopes = append(scopes, models.ScopeAdmin)
	}

	// Expiry is required and capped.
	if request.ExpiresInDays < 1 || request.ExpiresInDays > models.PATMaxLifetimeDays {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Expiry must be between 1 and 365 days."})
		context.Abort()
		return
	}

	plaintext := auth.GeneratePATToken()
	pat := models.PersonalAccessToken{
		GormModel: models.GormModel{ID: uuid.New()},
		UserID:    userID,
		Name:      name,
		TokenHash: auth.HashToken(plaintext),
		Scope:     strings.Join(scopes, " "),
		ExpiresAt: time.Now().AddDate(0, 0, request.ExpiresInDays),
	}
	if err := database.CreatePersonalAccessToken(&pat); err != nil {
		logger.Log.Error("failed to create PAT. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{
		"message": "Token created.",
		"data":    models.PATCreationResponse{Token: plaintext, PAT: pat},
	})
}

// APIGetPersonalAccessTokens lists the calling user's active PATs (metadata only).
func APIGetPersonalAccessTokens(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to identify user."})
		context.Abort()
		return
	}

	pats, err := database.GetPersonalAccessTokensByUser(userID)
	if err != nil {
		logger.Log.Error("failed to list PATs. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Tokens retrieved.", "data": pats})
}

// APIDeletePersonalAccessToken revokes one of the calling user's PATs.
func APIDeletePersonalAccessToken(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to identify user."})
		context.Abort()
		return
	}

	patID, err := uuid.Parse(context.Param("pat_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID."})
		context.Abort()
		return
	}

	if err := database.RevokePersonalAccessToken(patID, userID); err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "Token not found."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Token revoked."})
}
