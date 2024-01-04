package middlewares

import (
	"aunefyren/treningheten/auth"
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Auth(admin bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		tokenString := context.GetHeader("Authorization")
		if tokenString == "" {
			context.JSON(401, gin.H{"error": "Request does not contain an access token"})
			context.Abort()
			return
		}

		err := auth.ValidateToken(tokenString, admin)
		if err != nil {
			log.Println("Failed to validate token. Error: " + err.Error())
			context.JSON(401, gin.H{"error": "Failed to validate token."})
			context.Abort()
			return
		}

		// Get configuration
		config, err := config.GetConfig()
		if err != nil {
			log.Println("Failed to get config. Error: " + err.Error())
			context.JSON(401, gin.H{"error": "Failed to validate token."})
			context.Abort()
			return
		}

		// Get userID from header
		userID, err := GetAuthUsername(context.GetHeader("Authorization"))
		if err != nil {
			context.JSON(401, gin.H{"error": "Failed to validate token."})
			context.Abort()
			return
		}

		// Check if the user is verified
		enabled, err := database.VerifyUserIsEnabled(userID)
		if err != nil {

			log.Println("Failed to check account. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check account."})
			context.Abort()
			return

		} else if !enabled {

			context.JSON(http.StatusInternalServerError, gin.H{"error": "Account disabled."})
			context.Abort()
			return

		}
		// If SMTP is enabled, verify if user is enabled
		if config.SMTPEnabled {

			// Check if the user is verified
			verified, err := database.VerifyUserIsVerified(userID)
			if err != nil {

				log.Println("Failed to check verification. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check verification."})
				context.Abort()
				return

			} else if !verified {

				// Verify user has verification code
				hasVerificationCode, err := database.VerifyUserHasVerificationCode(userID)
				if err != nil {
					context.JSON(401, gin.H{"error": "Failed to validate token."})
					context.Abort()
					return
				}

				// If the user doesn't have a code, set one
				if !hasVerificationCode {
					_, err := database.GenerateRandomVerificationCodeForUser(userID)
					if err != nil {
						context.JSON(401, gin.H{"error": "Failed to validate token."})
						context.Abort()
						return
					}
				}

				// Return error
				context.JSON(http.StatusForbidden, gin.H{"error": "You must verify your account."})
				context.Abort()
				return
			}

		}

		context.Next()
	}
}

func GetAuthUsername(tokenString string) (uuid.UUID, error) {

	if tokenString == "" {
		return uuid.UUID{}, errors.New("No Authorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return uuid.UUID{}, err
	}
	return claims.UserID, nil
}

func GetTokenClaims(tokenString string) (*auth.JWTClaim, error) {

	if tokenString == "" {
		return &auth.JWTClaim{}, errors.New("No Authorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return &auth.JWTClaim{}, err
	}
	return claims, nil
}
