package controllers

import (
	"aunefyren/treningheten/auth"
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/logger"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

func RegisterUser(context *gin.Context) {

	// Initialize variables
	var user models.User
	var userCreationRequest models.UserCreationRequest

	// Parse creation request
	if err := context.ShouldBindJSON(&userCreationRequest); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Make sure password match
	if userCreationRequest.Password != userCreationRequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Make password is strong enough
	valid, requirements, err := utilities.ValidatePasswordFormat(userCreationRequest.Password)
	if err != nil {
		logger.Log.Info("Failed to verify password quality. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password quality."})
		context.Abort()
		return
	} else if !valid {
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Move values from request to object
	user.Email = html.EscapeString(strings.TrimSpace(userCreationRequest.Email))
	user.Password = userCreationRequest.Password
	user.FirstName = html.EscapeString(strings.TrimSpace(userCreationRequest.FirstName))
	user.LastName = html.EscapeString(strings.TrimSpace(userCreationRequest.LastName))
	user.Enabled = true
	user.ID = uuid.New()

	randomString := randstr.String(8)
	finalRandomString := strings.ToUpper(randomString)
	user.VerificationCode = &finalRandomString

	timeExpir := time.Now().Add(time.Hour * 24 * 2)
	user.VerificationCodeExpiration = &timeExpir

	randomString = randstr.String(8)
	finalRandomString = strings.ToUpper(randomString)
	user.ResetCode = &finalRandomString

	timeResetExp := time.Now()
	user.ResetExpiration = &timeResetExp

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// If SMTP is disabled, create the user as verified
	if config.SMTPEnabled {
		user.Verified = false
	} else {
		user.Verified = true
	}

	// Hash the selected password
	if err := user.HashPassword(user.Password); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify unused invite code exists
	uniqueInviteCode, err := database.VerifyUnusedUserInviteCode(strings.TrimSpace(userCreationRequest.InviteCode))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !uniqueInviteCode {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Initiation code is not valid."})
		context.Abort()
		return
	}

	// Verify e-mail is not in use
	unique_email, err := database.VerifyUniqueUserEmail(user.Email)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !unique_email {
		context.JSON(http.StatusBadRequest, gin.H{"error": "E-mail is already in use."})
		context.Abort()
		return
	}

	// Create user in DB
	user, err = database.RegisterUserInDB(user)
	if err != nil {
		logger.Log.Info("Failed to save user in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user in database."})
		context.Abort()
		return
	}

	// Set code to used
	err = database.SetUsedUserInviteCode(strings.TrimSpace(userCreationRequest.InviteCode), user.ID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// If user is not verified and SMTP is enabled, send verification e-mail
	if !user.Verified && config.SMTPEnabled {

		logger.Log.Info("Sending verification e-mail to new user: " + user.FirstName + " " + user.LastName + ".")

		err = utilities.SendSMTPVerificationEmail(user)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "User created!"})

}

func GetUser(context *gin.Context) {

	// Create user request
	var user = context.Param("user_id")
	userObject := models.User{}

	// Parse requested user id
	user_id_int, err := uuid.Parse(user)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user ID from requestor
	requesterUserID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get requesting user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get requesting user ID."})
		context.Abort()
		return
	}

	if requesterUserID == user_id_int {
		userObject, err = database.GetAllUserInformation(requesterUserID)
		if err != nil {
			logger.Log.Info("Failed to get user details. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
			context.Abort()
			return
		}
	} else {
		userObject, err = database.GetUserInformation(user_id_int)
		if err != nil {
			logger.Log.Info("Failed to get user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
			context.Abort()
			return
		}

		// Give achievement for visiting another user's profile
		err := GiveUserAnAchievement(requesterUserID, uuid.MustParse("cbd81cd0-4caf-438b-989b-b5ca7e76605d"), time.Now())
		if err != nil {
			logger.Log.Info("Failed to give achievement for user '" + requesterUserID.String() + "'. Ignoring. Error: " + err.Error())
		}
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"user": userObject, "message": "User retrieved."})
}

func GetUsers(context *gin.Context) {

	// Get users from DB
	users, err := database.GetUsersInformation()
	if err != nil {
		logger.Log.Info("Failed to get users. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users."})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"users": users, "message": "Users retrieved."})
}

func VerifyUser(context *gin.Context) {

	// Get code from URL
	var code = context.Param("code")

	// Check if the string is empty
	if code == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "No code found."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify if code matches
	match, expiration, err := database.VerifyUserVerificationCodeMatches(userID, code)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Check if code matches
	if !match {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Verification code invalid."})
		context.Abort()
		return
	} else if expiration == nil || time.Now().After(*expiration) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Verification code has expired, request a new one."})
		context.Abort()
		return
	}

	// Set account to verified
	err = database.SetUserVerification(userID, true)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user object
	var user models.User
	record := database.Instance.Where("ID = ?", userID).First(&user)
	if record.Error != nil {
		logger.Log.Info("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	// Generate new JWT token
	tokenString, err := auth.GenerateJWT(user.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "User verified.", "token": tokenString})

}

func SendUserVerificationCode(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Create a new code
	_, err = database.GenerateRandomVerificationCodeForUser(userID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user object
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Send new e-mail
	err = utilities.SendSMTPVerificationEmail(user)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "New verification code sent."})

}

func UpdateUser(context *gin.Context) {

	// Initialize variables
	var userUpdateRequest models.UserUpdateRequest
	var err error

	// Parse creation request
	if err := context.ShouldBindJSON(&userUpdateRequest); err != nil {
		logger.Log.Info("Failed to prase update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to prase update request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	userObject, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user information. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information."})
		context.Abort()
		return
	}

	credentialError := userObject.CheckPassword(userUpdateRequest.OldPassword)
	if credentialError != nil {
		logger.Log.Info("Invalid credentials. Error: " + credentialError.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	// Make sure password match
	if userUpdateRequest.Password != "" && userUpdateRequest.Password != userUpdateRequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Make password is strong enough
	valid, requirements, err := utilities.ValidatePasswordFormat(userUpdateRequest.Password)
	if err != nil {
		logger.Log.Info("Failed to verify password quality. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password quality."})
		context.Abort()
		return
	} else if !valid && userUpdateRequest.Password != "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Get user object
	var userOriginal models.User
	record := database.Instance.Where("ID = ?", userID).First(&userOriginal)
	if record.Error != nil {
		logger.Log.Info("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	userUpdateRequest.Email = html.EscapeString(strings.TrimSpace(userUpdateRequest.Email))

	if userOriginal.Email != userUpdateRequest.Email {

		// Verify e-mail is not in use
		unique_email, err := database.VerifyUniqueUserEmail(userUpdateRequest.Email)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		} else if !unique_email {
			context.JSON(http.StatusBadRequest, gin.H{"error": "E-mail is already in use."})
			context.Abort()
			return
		}

		// Set account to not verified
		err = database.SetUserVerification(userID, false)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		userOriginal.Email = userUpdateRequest.Email

	}

	// Hash the selected password
	if userUpdateRequest.Password != "" {
		if err := userOriginal.HashPassword(userUpdateRequest.Password); err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
	}

	// Transfer share activity value
	userOriginal.ShareActivities = userUpdateRequest.ShareActivities

	// Update profile image
	if userUpdateRequest.ProfileImage != "" {
		err = UpdateUserProfileImage(userOriginal.ID, userUpdateRequest.ProfileImage)
		if err != nil {
			logger.Log.Info("Failed to update profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile image."})
			context.Abort()
			return
		}

		// Give achievement to user for changing profile photo
		err := GiveUserAnAchievement(userOriginal.ID, uuid.MustParse("05a3579f-aa8d-4814-b28f-5824a2d904ec"), time.Now())
		if err != nil {
			logger.Log.Info("Failed to give achievement for user '" + userOriginal.ID.String() + "'. Ignoring. Error: " + err.Error())
		}
	}

	// Validate birth date
	if userUpdateRequest.BirthDate != nil {
		ThirteenYearsDuration := time.Hour * 24 * 365 * 13
		if userUpdateRequest.BirthDate.After(time.Now().Add(-ThirteenYearsDuration)) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Your birth date must be more than thirteen years ago."})
			context.Abort()
			return
		}
	}

	// Transfer birth date
	userOriginal.BirthDate = userUpdateRequest.BirthDate

	// Update user in database
	user, err := database.UpdateUser(userOriginal)
	if err != nil {
		logger.Log.Info("Failed to update user in the database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user in the database."})
		context.Abort()
		return
	}

	// Generate new JWT token
	tokenString, err := auth.GenerateJWT(user.ID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// If user is not verified and SMTP is enabled, send verification e-mail
	if config.SMTPEnabled && !user.Verified {

		verificationCode, err := database.GenerateRandomVerificationCodeForUser(userID)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		user.VerificationCode = &verificationCode

		logger.Log.Info("Sending verification e-mail to new user: " + user.FirstName + " " + user.LastName + ".")

		err = utilities.SendSMTPVerificationEmail(user)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Account updated.", "token": tokenString, "verified": user.Verified})

}

func APIResetPassword(context *gin.Context) {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if !config.SMTPEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The website administrator has not enabled SMTP."})
		context.Abort()
		return
	}

	if config.TreninghetenExternalURL == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The website administrator has not setup an external website URL."})
		context.Abort()
		return
	}

	type resetRequest struct {
		Email string `json:"email"`
	}

	var resetRequestVar resetRequest

	// Parse reset request
	if err := context.ShouldBindJSON(&resetRequestVar); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	user, err := database.GetUserInformationByEmail(resetRequestVar.Email)
	if err != nil {
		logger.Log.Info("Failed to find user using email during password reset. Replied with okay 200. Error: " + err.Error())
		context.JSON(http.StatusOK, gin.H{"message": "If the user exists, an email with a password reset has been sent."})
		context.Abort()
		return
	}

	_, err = database.GenerateRandomResetCodeForUser(user.ID)
	if err != nil {
		logger.Log.Info("Failed to generate reset code for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	user, err = database.GetAllUserInformation(user.ID)
	if err != nil {
		logger.Log.Info("Failed to retrieve data for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	err = utilities.SendSMTPResetEmail(user)
	if err != nil {
		logger.Log.Info("Failed to send email to user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "If the user exists, an email with a password reset has been sent."})

}

func APIChangePassword(context *gin.Context) {

	// Initialize variables
	var user models.User
	var userUpdatePasswordRequest models.UserUpdatePasswordRequest

	// Parse creation request
	if err := context.ShouldBindJSON(&userUpdatePasswordRequest); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Make sure password match
	if userUpdatePasswordRequest.Password != userUpdatePasswordRequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Make password is strong enough
	valid, requirements, err := utilities.ValidatePasswordFormat(userUpdatePasswordRequest.Password)
	if err != nil {
		logger.Log.Info("Failed to verify password quality. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password quality."})
		context.Abort()
		return
	} else if !valid {
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Get user object using reset code
	user, err = database.GetAllUserInformationByResetCode(userUpdatePasswordRequest.ResetCode)
	if err != nil {
		logger.Log.Info("Failed to retrieve user using reset code. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Reset code has expired."})
		context.Abort()
		return
	}

	now := time.Now()

	// Check if code has expired
	if user.ResetExpiration.Before(now) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Reset code has expired."})
		context.Abort()
		return
	}

	// Hash the selected password
	if err = user.HashPassword(userUpdatePasswordRequest.Password); err != nil {
		logger.Log.Info("Failed to hash password. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password."})
		context.Abort()
		return
	}

	// Save new password
	err = database.UpdateUserValuesByUserID(user.ID, user.Email, user.Password, user.SundayAlert, user.BirthDate)
	if err != nil {
		logger.Log.Info("Failed to update password. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password."})
		context.Abort()
		return
	}

	// Change the reset code
	_, err = database.GenerateRandomResetCodeForUser(user.ID)
	if err != nil {
		logger.Log.Info("Failed to generate reset code for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Password reset. You can now log in."})

}

func SendSundayReminders() {

	now := time.Now()

	// Get current season
	seasons, err := GetOngoingSeasonsFromDB(now)
	if err != nil {
		logger.Log.Info("Failed to verify current season status. Returning. Error: " + err.Error())
		return
	} else if len(seasons) == 0 {
		logger.Log.Info("Failed to verify current season status. Returning. Error: No active or future seasons found.")
		return
	}

	for _, season := range seasons {
		if season.Start.After(now) || season.End.Before(now) {
			logger.Log.Info("Not in the middle of a season. Returning.")
			return
		}

		usersWithAlerts, err := database.GetAllUsersWithSundayAlertsEnabled()
		if err != nil {
			logger.Log.Info("Failed to get users with alerts enabled. Returning. Error: " + err.Error())
			return
		}

		usersToAlert := []models.User{}

		for _, user := range usersWithAlerts {

			goalStatus, _, err := database.VerifyUserGoalInSeason(user.ID, season.ID)
			if err != nil {
				logger.Log.Info("Failed to verify user '" + user.ID.String() + "'. Skipping.")
			} else if goalStatus {
				usersToAlert = append(usersToAlert, user)
			}

		}

		for _, user := range usersToAlert {
			utilities.SendSMTPSundayReminderEmail(user, season, time.Now())
		}

		// Send push notifications
		err = PushNotificationsForSundayAlerts()
		if err != nil {
			logger.Log.Info("Failed to send push notifications for Sunday reminders.")
		}
	}
}

func APISetStravaCode(context *gin.Context) {
	// Initialize variables
	var user models.User
	var userStravaCodeUpdateRequest models.UserStravaCodeUpdateRequest

	// Parse creation request
	err := context.ShouldBindJSON(&userStravaCodeUpdateRequest)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	configFile, err := config.GetConfig()
	if err != nil {
		logger.Log.Info("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	if !configFile.StravaEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Strava is not enabled."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err = database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	newCode := "c:" + userStravaCodeUpdateRequest.StravaCode
	user.StravaCode = &newCode

	_, err = database.UpdateUser(user)
	if err != nil {
		logger.Log.Info("Failed to update user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user object."})
		context.Abort()
		return
	}

	err = StravaSyncWeekForUser(user, configFile, time.Now())
	if err != nil {
		logger.Log.Info("Failed to sync Strava for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync Strava for user."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Code updated!"})
}

func APISyncStravaForUser(context *gin.Context) {
	// Initialize variables
	var user models.User

	configFile, err := config.GetConfig()
	if err != nil {
		logger.Log.Info("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	if !configFile.StravaEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Strava is not enabled."})
		context.Abort()
		return
	}

	pointInTime := time.Now()
	pointInTimeInput, okay := context.GetQuery("pointInTime")
	if okay {
		pointInTimeInt, err := strconv.ParseInt(pointInTimeInput, 10, 64)
		if err != nil {
			logger.Log.Info("Failed to parse UNIX timestamp. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse UNIX timestamp."})
			context.Abort()
			return
		}

		pointInTime = time.Unix(pointInTimeInt, 0)
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err = database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	if user.StravaCode == nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "User does not have a Strava connection."})
		context.Abort()
		return
	}

	err = StravaSyncWeekForUser(user, configFile, pointInTime)
	if err != nil {
		logger.Log.Info("Failed to sync Strava for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync Strava for user."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Strava sync started!"})
}

func APIPartialUpdateUser(context *gin.Context) {
	// Initialize variables
	var userUpdateRequest models.UserPartialUpdateRequest
	var err error

	// Parse creation request
	err = context.ShouldBindJSON(&userUpdateRequest)
	if err != nil {
		logger.Log.Info("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user from header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user from header."})
		context.Abort()
		return
	}

	userObject, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user information. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user information."})
		context.Abort()
		return
	}

	if userUpdateRequest.SundayAlert != nil {
		userObject.SundayAlert = *userUpdateRequest.SundayAlert
	}

	if userUpdateRequest.StravaPadel != nil {
		userObject.StravaPadel = userUpdateRequest.StravaPadel
	}

	if userUpdateRequest.StravaWalks != nil {
		userObject.StravaWalks = userUpdateRequest.StravaWalks
	}

	if userUpdateRequest.StravaPublic != nil {
		userObject.StravaPublic = userUpdateRequest.StravaPublic
	}

	// Update user in database
	_, err = database.UpdateUser(userObject)
	if err != nil {
		logger.Log.Info("Failed to update user in the database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user in the database."})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Account updated."})
}
