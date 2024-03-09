package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

// receive a user strcut and save it in the database
func RegisterUserInDB(user models.User) (models.User, error) {
	dbRecord := Instance.Create(&user)

	if dbRecord.Error != nil {
		return models.User{}, dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to update DB.")
	}

	return user, nil
}

// Generate a random verification code an return ut
func GenerateRandomVerificationCodeForUser(userID uuid.UUID) (string, error) {

	randomString := randstr.String(8)
	verificationCode := strings.ToUpper(randomString)

	newTime := time.Now().Add(time.Hour * 24 * 2)

	var user models.User
	userrecord := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("verification_code", verificationCode)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Verification code not changed in database.")
	}

	userrecord = Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("verification_code_expiration", newTime)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Verification code reset time not changed in database.")
	}

	return verificationCode, nil

}

// Verify e-mail is not in use
func VerifyUniqueUserEmail(providedEmail string) (bool, error) {
	var user models.User
	userrecords := Instance.Where("`users`.enabled = ?", 1).Where("`users`.email= ?", providedEmail).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 0 {
		return false, nil
	}
	return true, nil
}

// Verify if user has a verification code set
func VerifyUserHasVerificationCode(userID uuid.UUID) (bool, error) {
	var user models.User
	userrecords := Instance.Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return false, errors.New("Couldn't find the user.")
	}

	if user.VerificationCode == "" {
		return false, nil
	} else {
		return true, nil
	}
}

// Verify if user has a verification code set
func VerifyUserVerificationCodeMatches(userID uuid.UUID, verificationCode string) (bool, time.Time, error) {

	var user models.User

	userrecords := Instance.Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Where("`users`.verification_code = ?", verificationCode).Find(&user)

	if userrecords.Error != nil {
		return false, time.Now(), userrecords.Error
	}

	if userrecords.RowsAffected != 1 {
		return false, time.Now(), nil
	} else {
		return true, user.VerificationCodeExpiration, nil
	}

}

// Verify if user is verified
func VerifyUserIsVerified(userID uuid.UUID) (bool, error) {

	var user models.User
	userrecords := Instance.Where("`users`.id = ?", userID).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return false, errors.New("No user found.")
	}

	return user.Verified, nil
}

// Verify if user is enabled
func VerifyUserIsEnabled(userID uuid.UUID) (bool, error) {

	var user models.User
	userrecords := Instance.Where("`users`.id = ?", userID).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return false, errors.New("No user found.")
	}

	return user.Enabled, nil
}

// Set user to verified
func SetUserVerification(userID uuid.UUID, verified bool) error {

	var user models.User
	var verInt int

	if verified {
		verInt = 1
	} else {
		verInt = 0
	}

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("verified", verInt)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Verification not changed in database.")
	}

	return nil
}

// Update user values
func UpdateUserValuesByUserID(userID uuid.UUID, email string, password string, sundayAlert bool, birthDate *time.Time) (err error) {

	err = nil

	err = UpdateEmailValueByUserID(userID, email)
	if err != nil {
		return err
	}

	err = UpdatePasswordValueByUserID(userID, password)
	if err != nil {
		return err
	}

	err = UpdateSundayAlertValueByUserID(userID, sundayAlert)
	if err != nil {
		return err
	}

	err = UpdateBirthDateValueByUserID(userID, birthDate)
	if err != nil {
		return err
	}

	return nil

}

func UpdateEmailValueByUserID(userID uuid.UUID, email string) error {

	var user models.User

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("email", email)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Email not changed in database.")
	}

	return nil

}

func UpdatePasswordValueByUserID(userID uuid.UUID, password string) error {

	var user models.User

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("password", password)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Password not changed in database.")
	}

	return nil

}

func UpdateSundayAlertValueByUserID(userID uuid.UUID, sundayAlert bool) error {

	var user models.User

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("sunday_alert", sundayAlert)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Sunday alert not changed in database.")
	}

	return nil

}

func UpdateBirthDateValueByUserID(userID uuid.UUID, birthDate *time.Time) error {

	var user models.User

	userrecords := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("birth_date", &birthDate)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Birth date not changed in database.")
	}

	return nil

}

// Get user information by user ID (censored)
func GetUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user = CensorUserObject(user)

	return user, nil
}

// Get all users information (censored)
func GetUsersInformation() ([]models.User, error) {
	var users []models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Find(&users)
	if userrecord.Error != nil {
		return []models.User{}, userrecord.Error
	} else if userrecord.RowsAffected == 0 {
		return []models.User{}, nil
	}

	for _, user := range users {

		user = CensorUserObject(user)

	}

	return users, nil
}

// Get user information using email (censored)
func GetUserInformationByEmail(email string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.email = ?", email).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	user = CensorUserObject(user)

	return user, nil
}

// Get ALL user information by user ID (uncensored)
func GetAllUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Get all users with sunday alerts configured (censored)
func GetAllUsersWithSundayAlertsEnabled() ([]models.User, error) {
	var users []models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.sunday_alert = ?", 1).Find(&users)
	if userrecord.Error != nil {
		return []models.User{}, userrecord.Error
	} else if userrecord.RowsAffected == 0 {
		return []models.User{}, nil
	}

	for _, user := range users {

		user = CensorUserObject(user)

	}

	return users, nil
}

// Get ALL user information by user reset code (uncensored)
func GetAllUserInformationByResetCode(resetCode string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.reset_code = ?", resetCode).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Generate a random reset code and return it
func GenerateRandomResetCodeForUser(userID uuid.UUID) (string, error) {

	randomString := randstr.String(8)
	resetCode := strings.ToUpper(randomString)

	expirationDate := time.Now().Add(time.Hour * 24 * 2)

	var user models.User
	userrecord := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("reset_code", resetCode)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Reset code not changed in database.")
	}

	userrecord = Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("reset_expiration", expirationDate)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Reset code expiration not changed in database.")
	}

	return resetCode, nil

}

func CensorUserObject(user models.User) models.User {

	// Redact user information
	user.Password = "REDACTED"
	user.Email = "REDACTED"
	user.VerificationCode = "REDACTED"
	user.ResetCode = "REDACTED"
	user.ResetExpiration = time.Now()
	user.SundayAlert = false

	return user
}

// Get user email by UserID
func GetUserEmailByUserID(userID uuid.UUID) (string, bool, error) {

	var user models.User

	userrecords := Instance.Where("`users`.id= ?", userID).Find(&user)
	if userrecords.Error != nil {
		return "", false, userrecords.Error
	}

	if userrecords.RowsAffected != 1 {
		return "", false, errors.New("No user found.")
	}

	return user.Email, true, nil
}
