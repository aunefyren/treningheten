package controllers

import (
	"aunefyren/treningheten/auth"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"log"
	"net/http"
	"strconv"
	"time"

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
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// check if email exists and password is correct
	record := database.Instance.Where("email = ?", request.Email).First(&user)
	if record.Error != nil {
		log.Println("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	credentialError := user.CheckPassword(request.Password)
	if credentialError != nil {
		log.Println("Invalid credentials. Error: " + credentialError.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, user.Email, int(user.ID), *user.Admin, user.Verified, user.SundayAlert)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": tokenString, "message": "Logged in!"})

}

func ValidateToken(context *gin.Context) {

	Claims, err := middlewares.GetTokenClaims(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to validate session. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	}

	token := ""

	// Refresh login if it is over 24 hours old
	if Claims.IssuedAt != nil {
		// Get user object by ID
		now := time.Now()
		difference := Claims.IssuedAt.Time.Sub(now)

		if int64(difference.Hours()/24/365) >= 24 && Claims.ExpiresAt.After(now) {
			user, err := database.GetAllUserInformation(Claims.UserID)
			if err != nil {
				log.Println("Failed to get user object for user '" + strconv.Itoa(Claims.UserID) + "'. Not returning token. Error: " + err.Error())
			} else {
				// Generate new token to refresh expiration time
				token, err = auth.GenerateJWT(user.FirstName, user.LastName, user.Email, int(user.ID), *user.Admin, user.Verified, user.SundayAlert)
				if err != nil {
					log.Println("Failed to create JWT token for user '" + strconv.Itoa(Claims.UserID) + "'. Not returning token. Error: " + err.Error())
				}
			}
		}

	}

	context.JSON(http.StatusOK, gin.H{"message": "Valid session!", "data": Claims, "token": token})

}
