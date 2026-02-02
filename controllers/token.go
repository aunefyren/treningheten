package controllers

import (
	"net/http"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
)

type TokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func GenerateToken(context *gin.Context) {

	var request TokenRequest

	var user models.User
	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	// check if email exists and password is correct
	record := database.Instance.Where("email = ?", request.Email).First(&user)
	if record.Error != nil {
		logger.Log.Info("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	credentialError := user.CheckPassword(request.Password)
	if credentialError != nil {
		logger.Log.Info("Invalid credentials. Error: " + credentialError.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	tokenString, err := auth.GenerateJWT(user.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": tokenString, "message": "Logged in!"})

}

func ValidateToken(context *gin.Context) {

	now := time.Now()

	claims, err := middlewares.GetTokenClaims(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to validate session. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	} else if claims.ExpiresAt.Time.Before(now) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	}

	token := ""

	// Refresh login if it is over 24 hours old
	if claims.IssuedAt != nil {

		// Get time difference between now and token issue time
		difference := now.Sub(claims.IssuedAt.Time)

		if float64(difference.Hours()/24/365) < 1.0 && claims.ExpiresAt.After(now) {

			// Change expiration to now + seven days
			claims.ExpiresAt.Time = now.Add(time.Hour * 24 * 7)

			// Get user object by ID and check and update admin status
			userObject, userErr := database.GetUserInformation(claims.UserID)
			if userErr != nil {
				logger.Log.Info("Failed to check admin status during token refresh.")
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
				context.Abort()
				return
			} else if *userObject.Admin != claims.Admin {
				claims.Admin = *userObject.Admin
			}

			// Re-generate token with updated claims
			token, err = auth.GenerateJWTFromClaims(claims)
			if err != nil {
				logger.Log.Info("Failed to re-sign JWT from claims. Error: " + err.Error())
				token = ""
			}
		}

	}

	// Get public VAPID key
	VAPIDSettings, err := GetVAPIDSettings()
	if err != nil {
		logger.Log.Info("Failed to get public VAPID key from config. Error: " + err.Error())
	}

	context.JSON(http.StatusOK, gin.H{
		"message": "Valid session!",
		"data":    claims, "token": token,
		"vapid_public_key":    VAPIDSettings.VAPIDPublicKey,
		"strava_enabled":      files.ConfigFile.StravaEnabled,
		"strava_client_id":    files.ConfigFile.StravaClientID,
		"strava_redirect_uri": files.ConfigFile.StravaRedirectURI,
	})

}
