package controllers

import (
	"html"
	"net/http"
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
	user.Email = html.EscapeString(strings.TrimSpace(strings.ToLower(userCreationRequest.Email)))
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

	// Hash the selected password, inside the shared bcrypt cost budget — this is an
	// unauthenticated endpoint, and the hash is a deliberate second of a core.
	hashRelease, hashSlotFree := middlewares.AcquirePasswordCostSlot()
	if !hashSlotFree {
		middlewares.AbortPasswordCostBusy(context)
		return
	}
	err = user.HashPassword(user.Password)
	hashRelease()
	if err != nil {
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

	// Reading your own account returns the full row (the /account page edits fields nobody
	// else may see); reading anyone else's returns models.PublicUser, whose field set is the
	// allowlist of what one user may learn about another. The two branches deliberately
	// respond separately rather than sharing a variable — that is what keeps the private
	// fields out of the cross-user path by type rather than by memory. See docs/wip.md, S15.
	if requesterUserID == user_id_int {
		userObject, err := database.GetAllUserInformation(requesterUserID)
		if err != nil {
			logger.Log.Info("Failed to get user details. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
			context.Abort()
			return
		}

		// Surface connection state without ever serializing the credentials themselves
		userObject.HevyConnected = userObject.HevyAPIKey != nil && *userObject.HevyAPIKey != ""
		userObject.StravaConnected = userObject.StravaCode != nil && *userObject.StravaCode != ""

		context.JSON(http.StatusOK, gin.H{"user": userObject, "message": "User retrieved."})
		return
	}

	publicUser, err := database.GetUserInformation(user_id_int)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	// Give achievement for visiting another user's profile, ignore outcome
	goSafely("achievement grant", func() {
		GiveUserAnAchievement(requesterUserID, uuid.MustParse("cbd81cd0-4caf-438b-989b-b5ca7e76605d"), time.Now(), 5)
	})

	// Reply
	context.JSON(http.StatusOK, gin.H{"user": publicUser, "message": "User retrieved."})
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

	// Issue a new OAuth token set for the verified user (auto-login)
	admin := user.Admin != nil && *user.Admin
	tokenSet, err := auth.IssueTokenSet(user.ID, admin, auth.ScopeForUser(admin), models.FirstPartyClientID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "User verified.", "data": tokenSet})

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

	checkRelease, checkSlotFree := middlewares.AcquirePasswordCostSlot()
	if !checkSlotFree {
		middlewares.AbortPasswordCostBusy(context)
		return
	}
	credentialError := userObject.CheckPassword(userUpdateRequest.OldPassword)
	checkRelease()
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

	userUpdateRequest.Email = html.EscapeString(strings.TrimSpace(strings.ToLower(userUpdateRequest.Email)))

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
	passwordChanged := userUpdateRequest.Password != ""
	if passwordChanged {
		hashRelease, hashSlotFree := middlewares.AcquirePasswordCostSlot()
		if !hashSlotFree {
			middlewares.AbortPasswordCostBusy(context)
			return
		}
		hashError := userOriginal.HashPassword(userUpdateRequest.Password)
		hashRelease()
		if hashError != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": hashError.Error()})
			context.Abort()
			return
		}
	}

	// Transfer share values
	userOriginal.ShareActivities = userUpdateRequest.ShareActivities
	userOriginal.ShareStatistics = userUpdateRequest.ShareStatistics

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
		goSafely("achievement grant", func() {
			GiveUserAnAchievement(userOriginal.ID, uuid.MustParse("05a3579f-aa8d-4814-b28f-5824a2d904ec"), time.Now(), 5)
		})
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

	// Validate heart-rate settings (optional; plausible physiological ranges). These feed
	// the activity heart-rate zones — an explicit max overrides the age-based estimate, and
	// a resting HR switches the zones to heart-rate reserve (Karvonen).
	if userUpdateRequest.MaxHeartrate != nil && (*userUpdateRequest.MaxHeartrate < 100 || *userUpdateRequest.MaxHeartrate > 240) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your maximum heart rate must be between 100 and 240 bpm."})
		context.Abort()
		return
	}
	if userUpdateRequest.RestingHeartrate != nil && (*userUpdateRequest.RestingHeartrate < 25 || *userUpdateRequest.RestingHeartrate > 120) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your resting heart rate must be between 25 and 120 bpm."})
		context.Abort()
		return
	}
	if userUpdateRequest.MaxHeartrate != nil && userUpdateRequest.RestingHeartrate != nil &&
		*userUpdateRequest.RestingHeartrate >= *userUpdateRequest.MaxHeartrate {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your resting heart rate must be below your maximum heart rate."})
		context.Abort()
		return
	}

	// Transfer heart-rate settings
	userOriginal.MaxHeartrate = userUpdateRequest.MaxHeartrate
	userOriginal.RestingHeartrate = userUpdateRequest.RestingHeartrate

	// Update user in database
	user, err := database.UpdateUser(userOriginal)
	if err != nil {
		logger.Log.Info("Failed to update user in the database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user in the database."})
		context.Abort()
		return
	}

	// A password change ends every existing session: otherwise a refresh token an
	// attacker obtained before the change keeps minting access tokens for 30 days,
	// and changing the password after a compromise achieves nothing. This runs before
	// the new token set is issued below, so the caller's own session survives (the
	// frontend stores the pair this handler returns) while every other one dies.
	// Personal Access Tokens are deliberately left alone — they are separate
	// credentials, listed and revocable on the account page.
	if passwordChanged {
		if err := database.RevokeAllRefreshTokensForUser(user.ID); err != nil {
			logger.Log.Error("failed to revoke refresh tokens after password change. error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end existing sessions."})
			context.Abort()
			return
		}
	}

	// Issue a new OAuth token set (email/admin changes may affect claims)
	admin := user.Admin != nil && *user.Admin
	tokenSet, err := auth.IssueTokenSet(user.ID, admin, auth.ScopeForUser(admin), models.FirstPartyClientID)
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
	context.JSON(http.StatusOK, gin.H{"message": "Account updated.", "data": tokenSet, "verified": user.Verified})

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

	// The reset mail needs the address, which the public view does not carry.
	fullUser, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		logger.Log.Info("Failed to retrieve data for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	err = utilities.SendSMTPResetEmail(fullUser)
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

	// Hash the selected password, inside the shared bcrypt cost budget.
	hashRelease, hashSlotFree := middlewares.AcquirePasswordCostSlot()
	if !hashSlotFree {
		middlewares.AbortPasswordCostBusy(context)
		return
	}
	err = user.HashPassword(userUpdatePasswordRequest.Password)
	hashRelease()
	if err != nil {
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

	// End every existing session — a password reset is the "I lost control of this
	// account" path, so any refresh token issued before it must stop working.
	// Personal Access Tokens survive; they are revoked from the account page.
	err = database.RevokeAllRefreshTokensForUser(user.ID)
	if err != nil {
		logger.Log.Error("failed to revoke refresh tokens after password reset. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end existing sessions."})
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

		// Uncensored: the reminder is an e-mail, so this job needs the address.
		usersWithAlerts, err := database.GetAllUsersWithSundayAlertsEnabledUncensored()
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
		context.JSON(http.StatusBadRequest, gin.H{"error": "User does not have a Strava connection."})
		context.Abort()
		return
	}

	goSafely("strava week sync", func() { StravaSyncWeekForUser(user, pointInTime) })

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

	// StravaIgnoreWalks is deprecated (superseded by UserActivityGoalSetting) and no longer
	// settable from the client — see models.User.

	if userUpdateRequest.StravaPublic != nil {
		userObject.StravaPublic = userUpdateRequest.StravaPublic
	}

	if userUpdateRequest.StravaSkipHevyDuplicates != nil {
		userObject.StravaSkipHevyDuplicates = userUpdateRequest.StravaSkipHevyDuplicates
	}

	if userUpdateRequest.HevyPublic != nil {
		userObject.HevyPublic = userUpdateRequest.HevyPublic
	}

	// Wheel appearance. An empty value clears the field (reverts to auto-assignment).
	if userUpdateRequest.WheelColor != nil {
		color := strings.TrimSpace(*userUpdateRequest.WheelColor)
		if color == "" {
			userObject.WheelColor = nil
		} else if utilities.ValidHexColor(color) {
			userObject.WheelColor = &color
		} else {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wheel color. Expected a hex value like #4363d8."})
			context.Abort()
			return
		}
	}

	if userUpdateRequest.WheelBorderColor != nil {
		border := strings.TrimSpace(*userUpdateRequest.WheelBorderColor)
		if border == "" {
			userObject.WheelBorderColor = nil
		} else if utilities.ValidHexColor(border) {
			userObject.WheelBorderColor = &border
		} else {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wheel border color. Expected a hex value like #000000."})
			context.Abort()
			return
		}
	}

	if userUpdateRequest.WheelEmoji != nil {
		emoji := strings.TrimSpace(*userUpdateRequest.WheelEmoji)
		if emoji == "" {
			userObject.WheelEmoji = nil
		} else if utilities.ValidWheelEmoji(emoji) {
			userObject.WheelEmoji = &emoji
		} else {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wheel emoji. Use a single emoji."})
			context.Abort()
			return
		}
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

	// ShareActivities is a visibility flag the owner sets, not something their peers may
	// read, so it is absent from models.PublicUser and the gate reads the real row.
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	if user.ShareActivities == nil || !*user.ShareActivities {
		context.JSON(http.StatusForbidden, gin.H{"error": "User does not share activities."})
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

	// One user's days, scoped in SQL — the query enforces share_activities itself, so the
	// gate above is defence in depth rather than the only check.
	//
	// This used to load *every* sharing user's days and then keep the ones whose owner was
	// NOT the requested user, which is inverted: the endpoint returned everyone else's
	// activities and excluded the profile's own. It went unnoticed because the gate above
	// read a censored user whose ShareActivities was nil-ed, so the handler returned 403 to
	// everyone and the filter never ran. Fixing either half alone would have turned a dead
	// endpoint into a leaking one, so both moved together. See docs/wip.md, S11.
	exerciseDays, err := database.GetExerciseDaysForSharingUsersInListUsingDates([]uuid.UUID{userID}, mondayStart, sundayEnd)
	if err != nil {
		logger.Log.Info("Failed to get exercise days from time frame. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get exercise days from time frame."})
		context.Abort()
		return
	}

	// The profile feed shares its shape and its visibility rules (switched-off, disabled and
	// private sessions excluded; Strava links per the owner's StravaPublic setting) with the
	// front-page and season feeds, so it flattens through the same builder.
	allActivities, err := buildActivitiesFromExerciseDays(exerciseDays)
	if err != nil {
		logger.Log.Info("Failed to build activities. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

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

	user, err := database.GetUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	if user.ShareStatistics == nil || *user.ShareStatistics == false {
		context.JSON(http.StatusForbidden, gin.H{"error": "User does not share statistics."})
		context.Abort()
		return
	}

	userStatisticsReply := models.UserStatisticsReply{MinimumSampleSize: userStatisticsMinSampleSize}

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

	// Personal day/week streaks are computed by the shared builder so the MCP
	// get_statistics tool and this endpoint cannot drift. See computePersonalStreaks.
	streaks := computePersonalStreaks(exerciseDayObjects)
	userStatisticsReply.StreakWeeks = streaks.WeekCurrent
	userStatisticsReply.StreakWeeksTop = streaks.WeekBest
	userStatisticsReply.StreakDays = streaks.DayCurrent
	userStatisticsReply.StreakDaysTop = streaks.DayBest

	// Candidate actions for the headline pick, gathered over two windows. The recent list
	// is what the section is normally about ("what you have been doing"); the all-time list
	// is the fallback when the recent window is too thin to say anything about a person
	// rather than about one workout. See chooseHeadlineAction.
	recentActions := []uuid.UUID{}
	allTimeActions := []uuid.UUID{}

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

			for _, operation := range exercise.Operations {
				if !operation.Enabled || operation.Action == nil {
					continue
				}
				allTimeActions = append(allTimeActions, operation.Action.ID)
				if time.Since(exerciseDayDate) <= time.Duration(time.Hour*24*31) {
					recentActions = append(recentActions, operation.Action.ID)
				}
			}
		}
	}

	goals, err := database.GetGoalsForUserUsingUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get goals for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goals for user."})
		context.Abort()
		return
	}

	userStatisticsReply.SeasonsJoined = len(goals)

	chosenAction := chooseHeadlineAction(recentActions, allTimeActions)

	// The three windows are accumulated as locals and only published at the end, because
	// whether a window may be published at all depends on the totals it ends up with.
	var (
		headlineAction *models.Action
		pastMonth      models.UserStatisticsCompilation
		pastYear       models.UserStatisticsCompilation
		allTime        models.UserStatisticsCompilation
	)

	for _, exerciseDay := range exerciseDayObjects {
		exerciseDayDate := exerciseDay.Date
		for _, exercise := range exerciseDay.Exercises {
			if !exercise.Enabled || !exercise.IsOn {
				continue
			}

			logger.Log.Tracef("exercise processing")

			for _, operation := range exercise.Operations {
				if !operation.Enabled {
					continue
				}

				logger.Log.Tracef("exercise operation with no object processing")

				actionDistance := 0.0
				actionTime := int64(0)
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
						actionTime += *operationSet.Time
					}
					if operationSet.Repetitions != nil {
						actionRepetitions += *operationSet.Repetitions
					}
					if operationSet.Weight != nil {
						actionWeight += *operationSet.Weight
					}
				}

				// Only the headline activity's operations feed this block. An operation with
				// no action used to match a nil pick, so a profile with nothing to headline
				// reported a nameless section built from actionless operations; now a nil
				// pick means the section is absent and the client says why.
				if chosenAction != nil && operation.Action != nil && operation.Action.ID == *chosenAction {
					headlineAction = operation.Action

					if time.Since(exerciseDayDate) <= time.Duration(time.Hour*24*31) {
						pastMonth.Sums.Operations += 1

						pastMonth.Sums.Distance += actionDistance
						pastMonth.Sums.Time += actionTime
						pastMonth.Sums.Weight += actionWeight

						if !exercise.Private {
							// A top is one nameable session with a date, so private sessions are
							// withheld here — unlike the sums and streaks above, which keep them so
							// the profile agrees with the season leaderboard. See docs/wip.md, S14.
							if actionDistance > pastMonth.Tops.Distance {
								pastMonth.Tops.Distance = actionDistance
							}
							if actionTime > pastMonth.Tops.Time {
								pastMonth.Tops.Time = actionTime
							}
							if actionWeight > pastMonth.Tops.Weight {
								pastMonth.Tops.Weight = actionWeight
							}
						}
					}
					if time.Since(exerciseDayDate) <= time.Duration(time.Hour*24*365) {
						pastYear.Sums.Operations += 1

						pastYear.Sums.Distance += actionDistance
						pastYear.Sums.Time += actionTime
						pastYear.Sums.Weight += actionWeight

						if !exercise.Private {
							// A top is one nameable session with a date, so private sessions are
							// withheld here — unlike the sums and streaks above, which keep them so
							// the profile agrees with the season leaderboard. See docs/wip.md, S14.
							if actionDistance > pastYear.Tops.Distance {
								pastYear.Tops.Distance = actionDistance
							}
							if actionTime > pastYear.Tops.Time {
								pastYear.Tops.Time = actionTime
							}
							if actionWeight > pastYear.Tops.Weight {
								pastYear.Tops.Weight = actionWeight
							}
						}
					}

					allTime.Sums.Operations += 1

					allTime.Sums.Distance += actionDistance
					allTime.Sums.Time += actionTime
					allTime.Sums.Weight += actionWeight

					if !exercise.Private {
						// A top is one nameable session with a date, so private sessions are
						// withheld here — unlike the sums and streaks above, which keep them so
						// the profile agrees with the season leaderboard. See docs/wip.md, S14.
						if actionDistance > allTime.Tops.Distance {
							allTime.Tops.Distance = actionDistance
						}
						if actionTime > allTime.Tops.Time {
							allTime.Tops.Time = actionTime
						}
						if actionWeight > allTime.Tops.Weight {
							allTime.Tops.Weight = actionWeight
						}
					}
				}
			}
		}
	}

	// A window holding fewer operations than the floor cannot be published: its distance
	// total, its longest session and its total time are all the same one or two workouts
	// said three ways (with one session, "distance" and "best" are literally equal). The
	// all-time window is the gate for the whole block — if even that is below the floor
	// there is nothing to build a breakdown from, so the section is dropped and the client
	// shows the empty state.
	if allTime.Sums.Operations >= userStatisticsMinSampleSize {
		userStatisticsReply.ActivityStatistics.Action = headlineAction
		userStatisticsReply.ActivityStatistics.PastMonth = publishWindow(pastMonth)
		userStatisticsReply.ActivityStatistics.PastYear = publishWindow(pastYear)
		userStatisticsReply.ActivityStatistics.AllTime = publishWindow(allTime)
	}

	context.JSON(http.StatusOK, gin.H{"data": userStatisticsReply})
}

// userStatisticsMinSampleSize is the smallest number of sessions a window may average over
// and still be published. Below it the "average" is really the sessions themselves — one
// operation makes avg distance an exact republication of that workout — which matters on a
// profile any authenticated user can read. The sums and counts stay regardless: they are
// already implied by the season leaderboard, so hiding them would only make the profile
// disagree with it. See docs/wip.md, S14.
const userStatisticsMinSampleSize = 3

// publishWindow decides whether a window may be reported at all, and fills in its averages
// if so. Below userStatisticsMinSampleSize it returns nil: every figure in the window — the
// distance total, the longest session, the total time — is then just the window's one or two
// sessions restated, which is the individual disclosure the floor exists to prevent, and it
// is plainly visible as one (with a single session the "distance" and "best" tiles show the
// same number). Nil means "not enough data", never zero; the count of sessions is not
// reported separately because it is only meaningful next to the figures it qualifies.
func publishWindow(window models.UserStatisticsCompilation) *models.UserStatisticsCompilation {
	if window.Sums.Operations < userStatisticsMinSampleSize {
		return nil
	}

	operations := float64(window.Sums.Operations)
	distance := window.Sums.Distance / operations
	seconds := int64(float64(window.Sums.Time) / operations)
	weight := window.Sums.Weight / operations

	window.Averages.Distance = &distance
	window.Averages.Time = &seconds
	window.Averages.Weight = &weight

	return &window
}

// chooseHeadlineAction picks the activity the profile's breakdown is built around.
//
// It prefers the past month, because "what they have been doing lately" is the more
// interesting thing to read on a profile. But at one or two activities that pick stops
// describing a person and starts describing a single workout — the same small-sample
// disclosure the averages floor guards against, and it is worse here because the pick also
// names the section. So a thin recent window falls back to the all-time pick, which answers
// the duller but safe question "what is this person's sport".
//
// Below the floor on both counts there is nothing honest to headline, so it returns nil and
// the whole activity block is omitted; the client renders an explicit empty state rather
// than a gap. Falling back rather than blanking matters because the section carries the
// month/year/all-time windows together — suppressing the pick outright would take a decade
// of all-time statistics off the page to hide a quiet month.
func chooseHeadlineAction(recentActions []uuid.UUID, allTimeActions []uuid.UUID) *uuid.UUID {
	if len(recentActions) >= userStatisticsMinSampleSize {
		return mostCommonAction(recentActions)
	}
	if len(allTimeActions) >= userStatisticsMinSampleSize {
		return mostCommonAction(allTimeActions)
	}
	return nil
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
