package controllers

import (
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

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

	var trueVariable = true

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

	// If SMTP is disabled, create the user as verified
	if files.ConfigFile.SMTPEnabled {
		user.Verified = false
	} else {
		user.Verified = true
	}

	// Check if any users exist, if not, make new user admin
	userAmount, err := database.GetAmountOfEnabledUsers()
	if err != nil {
		logger.Log.Error("failed to verify user amount. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify user amount"})
		context.Abort()
		return
	} else if userAmount == 0 {
		user.Admin = &trueVariable
		logger.Log.Info("No other users found. New user is set to admin.")
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
	if !user.Verified && files.ConfigFile.SMTPEnabled {

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

		// Give achievement for visiting another user's profile, ignore outcome
		go GiveUserAnAchievement(requesterUserID, uuid.MustParse("cbd81cd0-4caf-438b-989b-b5ca7e76605d"), time.Now(), 5)
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

		// Give achievement to user for changing profile photo, ignore outcome
		go GiveUserAnAchievement(userOriginal.ID, uuid.MustParse("05a3579f-aa8d-4814-b28f-5824a2d904ec"), time.Now(), 5)
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

	// If user is not verified and SMTP is enabled, send verification e-mail
	if files.ConfigFile.SMTPEnabled && !user.Verified {

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
	if !files.ConfigFile.SMTPEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The website administrator has not enabled SMTP."})
		context.Abort()
		return
	}

	if files.ConfigFile.TreninghetenExternalURL == "" {
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

	_, err = database.GenerateRandomResetCodeForUser(user.ID, true)
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

func APIVerifyResetCode(context *gin.Context) {
	// Get code from URL
	var resetCode = context.Param("resetCode")

	// Parse creation request
	if resetCode == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user object using reset code
	user, err := database.GetAllUserInformationByResetCode(resetCode)
	if err != nil {
		logger.Log.Error("Failed to retrieve user using reset code. Error: " + err.Error())
		context.JSON(http.StatusOK, gin.H{"message": "Reset code retrieved.", "expired": true})
		context.Abort()
		return
	}

	now := time.Now()

	// Check if code has expired
	if user.ResetExpiration.Before(now) {
		context.JSON(http.StatusOK, gin.H{"message": "Reset code retrieved.", "expired": true})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Reset code retrieved.", "expired": false})
}

func APIChangePassword(context *gin.Context) {

	// Initialize variables
	var user models.User
	var userUpdatePasswordRequest models.UserUpdatePasswordRequest

	// Parse creation request
	err := context.ShouldBindJSON(&userUpdatePasswordRequest)
	if err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
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
	_, err = database.GenerateRandomResetCodeForUser(user.ID, false)
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

	if !files.ConfigFile.StravaEnabled {
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

	err = StravaSyncWeekForUser(user, time.Now())
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

	if !files.ConfigFile.StravaEnabled {
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

	go StravaSyncWeekForUser(user, pointInTime)

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

func APIGetUserActivities(context *gin.Context) {
	var userIDString = context.Param("user_id")
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		logger.Log.Info("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	// Get current time
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	mondayStart, err := utilities.FindEarlierMonday(lastMonth)
	if err != nil {
		logger.Log.Info("Failed to find Monday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find Monday."})
		context.Abort()
		return
	}

	sundayEnd, err := utilities.FindNextSunday(now)
	if err != nil {
		logger.Log.Info("Failed to find Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find Sunday."})
		context.Abort()
		return
	}

	allExerciseDays, err := database.GetExerciseDaysForSharingUsersUsingDates(mondayStart, sundayEnd)
	if err != nil {
		logger.Log.Info("Failed to get exercise days from time frame. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get exercise days from time frame."})
		context.Abort()
		return
	}

	filteredExerciseDays := []models.ExerciseDay{}
	validatedUsers := []uuid.UUID{}
	for _, exerciseDay := range allExerciseDays {
		if exerciseDay.UserID == nil {
			continue
		}

		foundInCache := false
		for _, validatedUserID := range validatedUsers {
			if validatedUserID == *exerciseDay.UserID {
				foundInCache = true
				break
			}
		}

		if foundInCache {
			filteredExerciseDays = append(filteredExerciseDays, exerciseDay)
			continue
		} else {
			if exerciseDay.UserID != nil && userID != *exerciseDay.UserID {
				filteredExerciseDays = append(filteredExerciseDays, exerciseDay)
				validatedUsers = append(validatedUsers, *exerciseDay.UserID)
				continue
			}
		}
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(filteredExerciseDays)
	if err != nil {
		logger.Log.Info("Failed to convert exercise day to exercise day objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day objects."})
		context.Abort()
		return
	}

	allActivities := []models.Activity{}
	for _, exerciseDayObject := range exerciseDayObjects {
		for _, exercise := range exerciseDayObject.Exercises {
			if exercise.IsOn && exercise.Enabled {
				newActivity := models.Activity{}
				newActivity.ExerciseID = exercise.ID
				newActivity.User = exerciseDayObject.User
				newActivity.Time = exercise.Time
				newActivity.Actions = []models.Action{}

				if exerciseDayObject.User.StravaPublic != nil && *exerciseDayObject.User.StravaPublic {
					newActivity.StravaIDs = exercise.StravaID
				} else {
					newActivity.StravaIDs = []string{}
				}

				for _, operation := range exercise.Operations {
					if operation.Action != nil {
						newActivity.Actions = append(newActivity.Actions, *operation.Action)
					}
				}

				allActivities = append(allActivities, newActivity)
			}
		}
	}

	sort.Slice(allActivities, func(i, j int) bool {
		return allActivities[j].Time.Before(allActivities[i].Time)
	})

	// Return activities
	context.JSON(http.StatusOK, gin.H{"activities": allActivities})
}

func APIGetUserStatistics(context *gin.Context) {
	var userIDString = context.Param("user_id")
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		logger.Log.Info("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	userStatisticsReply := models.UserStatisticsReply{}

	exerciseDays, err := database.GetAllExerciseDaysWithExerciseByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise days for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise days for user."})
		context.Abort()
		return
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		logger.Log.Info("Failed to convert exercise days to objects for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise days to objects for user"})
		context.Abort()
		return
	}

	var lastRegisteredWeek *int
	var lastRegisteredYear *int
	var lastRegisteredDay *time.Time
	lastWeekChecked := false
	yesterdayChecked := false

	lastWeekDate := time.Now().AddDate(0, 0, -7)
	lastWeekYear, lastWeek := lastWeekDate.ISOWeek()
	yesterday := time.Now().AddDate(0, 0, -1)

	sort.Slice(exerciseDayObjects, func(i, j int) bool {
		return exerciseDayObjects[i].Date.Before(exerciseDayObjects[j].Date)
	})

	allActions := []uuid.UUID{}

	for _, exerciseDay := range exerciseDayObjects {
		exerciseDayDate := exerciseDay.Date
		logger.Log.Trace("exercise day: " + exerciseDayDate.String())
		for _, exercise := range exerciseDay.Exercises {
			if !exercise.Enabled || !exercise.IsOn {
				continue
			}

			userStatisticsReply.ExercisesAllTime += 1

			if time.Since(exerciseDayDate) <= time.Duration(time.Hour*24*365) {
				userStatisticsReply.ExercisesPastYear += 1
			}

			if time.Since(exerciseDayDate) <= time.Duration(time.Hour*24*31) {
				userStatisticsReply.ExercisesPastMonth += 1
			}

			currentYear, currentWeek := exerciseDayDate.ISOWeek()
			logger.Log.Tracef("week %d, year %d", currentWeek, currentYear)
			if lastRegisteredWeek != nil && lastRegisteredYear != nil && (currentWeek != *lastRegisteredWeek || currentYear != *lastRegisteredYear) {
				logger.Log.Tracef("not the same week %d", currentWeek)
				lastWeekDate := exerciseDayDate.AddDate(0, 0, -7)
				lastWeekYear, lastWeek := lastWeekDate.ISOWeek()

				if lastWeek == *lastRegisteredWeek && lastWeekYear == *lastRegisteredYear {
					userStatisticsReply.StreakWeeks += 1
					logger.Log.Tracef("adding week %d exercises", currentWeek)

					if userStatisticsReply.StreakWeeks > userStatisticsReply.StreakWeeksTop {
						userStatisticsReply.StreakWeeksTop = userStatisticsReply.StreakWeeks
					}
				} else {
					userStatisticsReply.StreakWeeks = 1
					logger.Log.Tracef("reset week %d", currentWeek)
				}
			} else {
				if lastRegisteredWeek != nil {
					logger.Log.Tracef("the same week %d, %d | %d, %d", *lastRegisteredWeek, *lastRegisteredYear, currentWeek, currentYear)
				}
			}

			currentDay := exerciseDayDate
			if lastRegisteredDay != nil && currentDay.Format("2006-01-02") != lastRegisteredDay.Format("2006-01-02") {
				dayBefore := currentDay.AddDate(0, 0, -1)
				if dayBefore.Format("2006-01-02") == lastRegisteredDay.Format("2006-01-02") {
					userStatisticsReply.StreakDays += 1

					if userStatisticsReply.StreakDays > userStatisticsReply.StreakDaysTop {
						userStatisticsReply.StreakDaysTop = userStatisticsReply.StreakDays
					}
				} else {
					userStatisticsReply.StreakDays = 1
				}
			}

			lastRegisteredWeek = &currentWeek
			lastRegisteredYear = &currentYear
			lastRegisteredDay = &exerciseDayDate

			if currentWeek == lastWeek && currentYear == lastWeekYear {
				lastWeekChecked = true
			}
			if currentDay.Format("2006-01-02") != yesterday.Format("2006-01-02") {
				yesterdayChecked = true
			}

			for _, operation := range exercise.Operations {
				if !operation.Enabled {
					continue
				}
				if operation.Action != nil {
					allActions = append(allActions, *&operation.Action.ID)
				}
			}
		}
	}

	if !lastWeekChecked {
		userStatisticsReply.StreakWeeks = 0
	}
	if !yesterdayChecked {
		userStatisticsReply.StreakDays = 0
	}

	goals, err := database.GetGoalsForUserUsingUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get goals for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goals for user."})
		context.Abort()
		return
	}

	userStatisticsReply.SeasonsJoined = len(goals)

	chosenAction := mostCommonAction(allActions)

	if chosenAction != nil {
		for _, exerciseDay := range exerciseDayObjects {
			exerciseDayDate := exerciseDay.Date
			for _, exercise := range exerciseDay.Exercises {
				if !exercise.Enabled || !exercise.IsOn {
					continue
				}

				for _, operation := range exercise.Operations {
					if !operation.Enabled {
						continue
					}

					actionDistance := 0.0
					actionTime := time.Duration(0)
					actionRepetitions := 0.0
					actionWeight := 0.0

					for _, operationSet := range operation.OperationSets {
						if !operationSet.Enabled {
							continue
						}
						if operationSet.Distance != nil {
							actionDistance += *operationSet.Distance
						}
						if operationSet.Time != nil {
							actionTime = time.Duration(actionTime + *operationSet.Time)
						}
						if operationSet.Repetitions != nil {
							actionRepetitions += *operationSet.Repetitions
						}
						if operationSet.Weight != nil {
							actionWeight += *operationSet.Weight
						}
					}

					if operation.Action != nil && operation.Action.ID == *chosenAction {
						userStatisticsReply.ActivityStatistics.Action = *operation.Action

						if time.Since(exerciseDayDate) < time.Duration(time.Hour*24*31) {
							userStatisticsReply.ActivityStatistics.PastMonth.Sums.Operations += 1

							userStatisticsReply.ActivityStatistics.PastMonth.Sums.Distance += actionDistance
							userStatisticsReply.ActivityStatistics.PastMonth.Sums.Time += actionTime
							userStatisticsReply.ActivityStatistics.PastMonth.Sums.Weight += actionWeight

							if actionDistance > userStatisticsReply.ActivityStatistics.PastMonth.Tops.Distance {
								userStatisticsReply.ActivityStatistics.PastMonth.Tops.Distance = actionDistance
								userStatisticsReply.ActivityStatistics.PastMonth.Tops.DistanceExerciseDayID = &exercise.ExerciseDay
							}
							if actionTime > userStatisticsReply.ActivityStatistics.PastMonth.Tops.Time {
								userStatisticsReply.ActivityStatistics.PastMonth.Tops.Time = actionTime
								userStatisticsReply.ActivityStatistics.PastMonth.Tops.TimeExerciseDayID = &exercise.ExerciseDay
							}
							if actionWeight > userStatisticsReply.ActivityStatistics.PastMonth.Tops.Weight {
								userStatisticsReply.ActivityStatistics.PastMonth.Tops.Weight = actionWeight
								userStatisticsReply.ActivityStatistics.PastMonth.Tops.WeightExerciseDayID = &exercise.ExerciseDay
							}
						}
						if time.Since(exerciseDayDate) < time.Duration(time.Hour*24*365) {
							userStatisticsReply.ActivityStatistics.PastYear.Sums.Operations += 1

							userStatisticsReply.ActivityStatistics.PastYear.Sums.Distance += actionDistance
							userStatisticsReply.ActivityStatistics.PastYear.Sums.Time += time.Duration(actionTime)
							userStatisticsReply.ActivityStatistics.PastYear.Sums.Weight += actionWeight

							if actionDistance > userStatisticsReply.ActivityStatistics.PastYear.Tops.Distance {
								userStatisticsReply.ActivityStatistics.PastYear.Tops.Distance = actionDistance
								userStatisticsReply.ActivityStatistics.PastYear.Tops.DistanceExerciseDayID = &exercise.ExerciseDay
							}
							if actionTime > userStatisticsReply.ActivityStatistics.PastYear.Tops.Time {
								userStatisticsReply.ActivityStatistics.PastYear.Tops.Time = actionTime
								userStatisticsReply.ActivityStatistics.PastYear.Tops.TimeExerciseDayID = &exercise.ExerciseDay
							}
							if actionWeight > userStatisticsReply.ActivityStatistics.PastYear.Tops.Weight {
								userStatisticsReply.ActivityStatistics.PastYear.Tops.Weight = actionWeight
								userStatisticsReply.ActivityStatistics.PastYear.Tops.WeightExerciseDayID = &exercise.ExerciseDay
							}
						}

						userStatisticsReply.ActivityStatistics.AllTime.Sums.Operations += 1

						userStatisticsReply.ActivityStatistics.AllTime.Sums.Distance += actionDistance
						userStatisticsReply.ActivityStatistics.AllTime.Sums.Time += actionTime
						userStatisticsReply.ActivityStatistics.AllTime.Sums.Weight += actionWeight

						if actionDistance > userStatisticsReply.ActivityStatistics.AllTime.Tops.Distance {
							userStatisticsReply.ActivityStatistics.AllTime.Tops.Distance = actionDistance
							userStatisticsReply.ActivityStatistics.AllTime.Tops.DistanceExerciseDayID = &exercise.ExerciseDay
						}
						if actionTime > userStatisticsReply.ActivityStatistics.AllTime.Tops.Time {
							userStatisticsReply.ActivityStatistics.AllTime.Tops.Time = actionTime
							userStatisticsReply.ActivityStatistics.AllTime.Tops.TimeExerciseDayID = &exercise.ExerciseDay
						}
						if actionWeight > userStatisticsReply.ActivityStatistics.AllTime.Tops.Weight {
							userStatisticsReply.ActivityStatistics.AllTime.Tops.Weight = actionWeight
							userStatisticsReply.ActivityStatistics.AllTime.Tops.WeightExerciseDayID = &exercise.ExerciseDay
						}
					}
				}
			}
		}
	}

	if userStatisticsReply.ActivityStatistics.PastMonth.Sums.Operations > 0 {
		userStatisticsReply.ActivityStatistics.PastMonth.Averages.Distance = float64(userStatisticsReply.ActivityStatistics.PastMonth.Sums.Distance / float64(userStatisticsReply.ActivityStatistics.PastMonth.Sums.Operations))
		userStatisticsReply.ActivityStatistics.PastMonth.Averages.Time = time.Duration(float64(userStatisticsReply.ActivityStatistics.PastMonth.Sums.Time) / float64(userStatisticsReply.ActivityStatistics.PastMonth.Sums.Operations))
		userStatisticsReply.ActivityStatistics.PastMonth.Averages.Weight = float64(userStatisticsReply.ActivityStatistics.PastMonth.Sums.Weight / float64(userStatisticsReply.ActivityStatistics.PastMonth.Sums.Operations))
	}
	if userStatisticsReply.ActivityStatistics.PastYear.Sums.Operations > 0 {
		userStatisticsReply.ActivityStatistics.PastYear.Averages.Distance = float64(userStatisticsReply.ActivityStatistics.PastYear.Sums.Distance / float64(userStatisticsReply.ActivityStatistics.PastYear.Sums.Operations))
		userStatisticsReply.ActivityStatistics.PastYear.Averages.Time = time.Duration(float64(userStatisticsReply.ActivityStatistics.PastYear.Sums.Time) / float64(userStatisticsReply.ActivityStatistics.PastYear.Sums.Operations))
		userStatisticsReply.ActivityStatistics.PastYear.Averages.Weight = float64(userStatisticsReply.ActivityStatistics.PastYear.Sums.Weight / float64(userStatisticsReply.ActivityStatistics.PastYear.Sums.Operations))
	}
	if userStatisticsReply.ActivityStatistics.AllTime.Sums.Operations > 0 {
		userStatisticsReply.ActivityStatistics.AllTime.Averages.Distance = float64(userStatisticsReply.ActivityStatistics.AllTime.Sums.Distance / float64(userStatisticsReply.ActivityStatistics.AllTime.Sums.Operations))
		userStatisticsReply.ActivityStatistics.AllTime.Averages.Time = time.Duration(float64(userStatisticsReply.ActivityStatistics.AllTime.Sums.Time) / float64(userStatisticsReply.ActivityStatistics.AllTime.Sums.Operations))
		userStatisticsReply.ActivityStatistics.AllTime.Averages.Weight = float64(userStatisticsReply.ActivityStatistics.AllTime.Sums.Weight / float64(userStatisticsReply.ActivityStatistics.AllTime.Sums.Operations))
	}

	context.JSON(http.StatusOK, gin.H{"data": userStatisticsReply})
}

func mostCommonAction(uuidArray []uuid.UUID) *uuid.UUID {
	if len(uuidArray) == 0 {
		return nil
	}

	counts := make(map[uuid.UUID]int)
	for _, id := range uuidArray {
		counts[id]++
	}

	var best uuid.UUID
	bestCount := 0
	for id, count := range counts {
		if count > bestCount || (count == bestCount && id.String() < best.String()) {
			best = id
			bestCount = count
		}
	}

	return &best
}
