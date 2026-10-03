package database

import (
	"errors"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

// receive a user struct and save it in the database
func RegisterUserInDB(user models.User) (models.User, error) {
	// New users start with a concrete (zero) observed max heart rate rather than NULL, so a
	// NULL only ever marks a legacy row the observed-max backfill still owes (see
	// backfillObservedMaxHeartrate). The value is bumped as their activities sync in.
	if user.ObservedMaxHeartrate == nil {
		zero := 0
		user.ObservedMaxHeartrate = &zero
	}

	dbRecord := Instance.Create(&user)

	if dbRecord.Error != nil {
		return models.User{}, dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to update DB.")
	}

	return user, nil
}

// BumpObservedMaxHeartrate raises the user's stored all-time observed max heart rate to
// candidate, but only when candidate is higher (or the value is still NULL). A single
// atomic conditional update, so concurrent syncs can't lower it.
func BumpObservedMaxHeartrate(userID uuid.UUID, candidate int) error {
	if candidate <= 0 {
		return nil
	}
	return Instance.Model(&models.User{}).
		Where("id = ? AND (observed_max_heartrate IS NULL OR observed_max_heartrate < ?)", userID, candidate).
		Update("observed_max_heartrate", candidate).Error
}

// generate a random verification code an return it
func GenerateRandomVerificationCodeForUser(userID uuid.UUID) (string, error) {

	randomString := randstr.String(8)
	verificationCode := strings.ToUpper(randomString)

	newTime := time.Now().Add(time.Hour * 24 * 2)

	var user models.User
	userrecord := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("verification_code", verificationCode)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Verification code not changed in database.")
	}

	userrecord = Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("verification_code_expiration", newTime)
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
	userrecords := Instance.Where("users.enabled = ?", true).Where("users.email= ?", providedEmail).Find(&user)
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
	userrecords := Instance.Where("users.enabled = ?", true).Where("users.ID = ?", userID).Find(&user)
	if userrecords.Error != nil {
		return false, userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return false, errors.New("Couldn't find the user.")
	}

	if user.VerificationCode == nil {
		return false, nil
	} else {
		return true, nil
	}
}

// Verify if user has a verification code set
func VerifyUserVerificationCodeMatches(userID uuid.UUID, verificationCode string) (bool, *time.Time, error) {

	var user models.User
	var now = time.Now()

	userrecords := Instance.Where("users.enabled = ?", true).Where("users.ID = ?", userID).Where("users.verification_code = ?", verificationCode).Find(&user)

	if userrecords.Error != nil {
		return false, &now, userrecords.Error
	}

	if userrecords.RowsAffected != 1 {
		return false, &now, nil
	} else {
		return true, user.VerificationCodeExpiration, nil
	}

}

// Verify if user is verified
func VerifyUserIsVerified(userID uuid.UUID) (bool, error) {

	var user models.User
	userrecords := Instance.Where("users.id = ?", userID).Find(&user)
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
	userrecords := Instance.Where("users.id = ?", userID).Find(&user)
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

	userrecords := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("verified", verified)
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

	userrecords := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("email", email)
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

	userrecords := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("password", password)
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

	userrecords := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("sunday_alert", sundayAlert)
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

	userrecords := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("birth_date", &birthDate)
	if userrecords.Error != nil {
		return userrecords.Error
	}
	if userrecords.RowsAffected != 1 {
		return errors.New("Birth date not changed in database.")
	}

	return nil

}

// Get enabled user information by user ID (censored)
func GetUserInformation(UserID uuid.UUID) (models.PublicUser, error) {
	var user models.User
	userrecord := Instance.Where("users.enabled = ?", true).Where("users.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.PublicUser{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.PublicUser{}, errors.New("Failed to find correct user in DB.")
	}

	return CensorUserObject(user), nil
}

// Get all enabled users information (censored)
func GetUsersInformation() ([]models.PublicUser, error) {
	users, err := GetAllUsersUncensored()
	if err != nil {
		return []models.PublicUser{}, err
	}

	publicUsers := make([]models.PublicUser, 0, len(users))
	for _, user := range users {
		publicUsers = append(publicUsers, CensorUserObject(user))
	}

	return publicUsers, nil
}

// GetAllUsersUncensored returns every enabled user with credential fields intact. Only for
// server-side jobs that genuinely need them (e.g. the Strava sync needs StravaCode) — never
// for anything that reaches a response body. Use GetUsersInformation for that.
func GetAllUsersUncensored() ([]models.User, error) {
	var users []models.User
	userrecord := Instance.Where("users.enabled = ?", true).Find(&users)
	if userrecord.Error != nil {
		return []models.User{}, userrecord.Error
	} else if userrecord.RowsAffected == 0 {
		return []models.User{}, nil
	}

	return users, nil
}

// Get user information using email (censored)
// GetUsersByIDs returns the enabled, censored users for the given IDs in a single query,
// so callers converting many goals don't issue one GetUserInformation per goal. Users that
// are missing/disabled are simply absent from the result. Returns an empty slice when no
// IDs are supplied.
func GetUsersByIDs(userIDs []uuid.UUID) ([]models.PublicUser, error) {
	var users []models.User

	if len(userIDs) == 0 {
		return []models.PublicUser{}, nil
	}

	userrecord := Instance.Where("users.enabled = ?", true).Where("users.id IN ?", userIDs).Find(&users)
	if userrecord.Error != nil {
		return []models.PublicUser{}, userrecord.Error
	}

	publicUsers := make([]models.PublicUser, 0, len(users))
	for _, user := range users {
		publicUsers = append(publicUsers, CensorUserObject(user))
	}

	return publicUsers, nil
}

func GetUserInformationByEmail(email string) (models.PublicUser, error) {
	var user models.User
	userrecord := Instance.Where("users.enabled = ?", true).Where("users.email = ?", email).Find(&user)
	if userrecord.Error != nil {
		return models.PublicUser{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.PublicUser{}, errors.New("Failed to find correct user in DB.")
	}

	return CensorUserObject(user), nil
}

// Get all user information using email (uncensored)
func GetAllUserInformationByEmail(email string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("users.enabled = ?", true).Where("users.email = ?", email).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Get ALL user information by user ID (uncensored)
func GetAllUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("users.enabled = ?", true).Where("users.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Get all users with sunday alerts configured (censored)
func GetAllUsersWithSundayAlertsEnabled() ([]models.PublicUser, error) {
	users, err := GetAllUsersWithSundayAlertsEnabledUncensored()
	if err != nil {
		return []models.PublicUser{}, err
	}

	publicUsers := make([]models.PublicUser, 0, len(users))
	for _, user := range users {
		publicUsers = append(publicUsers, CensorUserObject(user))
	}

	return publicUsers, nil
}

// GetAllUsersWithSundayAlertsEnabledUncensored returns the same users with their e-mail
// address intact, for the reminder job that has to mail them. Never use it for a response
// body — see GetAllUsersUncensored.
func GetAllUsersWithSundayAlertsEnabledUncensored() ([]models.User, error) {
	var users []models.User
	userrecord := Instance.Where("users.enabled = ?", true).Where("users.sunday_alert = ?", true).Find(&users)
	if userrecord.Error != nil {
		return []models.User{}, userrecord.Error
	} else if userrecord.RowsAffected == 0 {
		return []models.User{}, nil
	}

	return users, nil
}

// Get ALL user information by user reset code (uncensored)
func GetAllUserInformationByResetCode(resetCode string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("users.enabled = ?", true).Where("users.reset_code = ?", resetCode).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Generate a random reset code and return it
func GenerateRandomResetCodeForUser(userID uuid.UUID, valid bool) (string, error) {
	randomString := randstr.String(16)
	resetCode := strings.ToUpper(randomString)

	expirationDate := time.Now()
	if valid {
		expirationDate = expirationDate.AddDate(0, 0, 1)
	}

	var user models.User
	userrecord := Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("reset_code", resetCode)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Reset code not changed in database.")
	}

	userrecord = Instance.Model(user).Where("users.enabled = ?", true).Where("users.ID = ?", userID).Update("reset_expiration", expirationDate)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Reset code expiration not changed in database.")
	}

	return resetCode, nil

}

// CensorUserObject reduces a full user row to the subset one user may see of another.
//
// It returns models.PublicUser rather than a redacted models.User on purpose: an allowlist
// the compiler enforces cannot be defeated by someone adding a field to models.User and not
// thinking about this function, which is exactly how the health fields (birth date, resting
// and max heart rate) ended up being served to every authenticated caller. To expose
// something new about a user to their peers, add it to models.PublicUser deliberately.
//
// Callers that need the real row — a background job that must mail the user, or the owner
// reading their own account — want GetAllUserInformation and friends instead.
func CensorUserObject(user models.User) models.PublicUser {
	publicUser := models.PublicUser{
		ID:               user.ID,
		CreatedAt:        user.CreatedAt,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		Admin:            user.Admin,
		HevyProfileURL:   user.HevyProfileURL,
		HevyPublic:       user.HevyPublic,
		ShareStatistics:  user.ShareStatistics,
		WheelColor:       user.WheelColor,
		WheelBorderColor: user.WheelBorderColor,
		WheelEmoji:       user.WheelEmoji,
	}

	// A Strava id is only a profile link while its owner keeps the link public; otherwise
	// it is an identifier for an account they did not choose to advertise.
	if user.StravaPublic != nil && *user.StravaPublic {
		publicUser.StravaPublic = user.StravaPublic
		publicUser.StravaID = user.StravaID
	}

	return publicUser
}

// Get user email by UserID
func GetUserEmailByUserID(userID uuid.UUID) (string, bool, error) {

	var user models.User

	userrecords := Instance.Where("users.id= ?", userID).Find(&user)
	if userrecords.Error != nil {
		return "", false, userrecords.Error
	}

	if userrecords.RowsAffected != 1 {
		return "", false, errors.New("No user found.")
	}

	return user.Email, true, nil
}

func UpdateUser(user models.User) (models.User, error) {
	record := Instance.Save(&user)
	if record.Error != nil {
		return user, record.Error
	}
	return user, nil
}

func GetStravaUsersWithinSeason(seasonID uuid.UUID) (users []models.User, err error) {
	err = nil
	users = []models.User{}

	record := Instance.Where("users.enabled = ?", true).
		Where("users.strava_code IS NOT NULL").
		Joins("JOIN goals on goals.user_id = users.id").
		Where("goals.enabled = ?", true).
		Joins("JOIN seasons on goals.season_id = seasons.id").
		Where("seasons.enabled = ?", true).
		Where("seasons.id = ?", seasonID).
		Find(&users)
	if record.Error != nil {
		return users, record.Error
	}

	return
}

// ClearStravaConnectionForUser disconnects Strava by NULLing the stored credential
// and athlete id. A map is used (not a struct) so GORM writes the NULLs rather than
// skipping the zero values.
func ClearStravaConnectionForUser(userID uuid.UUID) error {
	record := Instance.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"strava_code": nil,
		"strava_id":   nil,
	})
	return record.Error
}

// SetStravaCredentialsForUser stores only the Strava credential, plus the athlete id when one
// is given (nil leaves the stored id alone). Like SetHevyLastSyncForUser it never writes the
// whole row: the sync holds a copy of the user loaded before it started, and saving that copy
// would undo any profile change made in the meantime.
func SetStravaCredentialsForUser(userID uuid.UUID, stravaCode string, stravaID *string) error {
	updates := map[string]interface{}{"strava_code": stravaCode}
	if stravaID != nil {
		updates["strava_id"] = *stravaID
	}
	record := Instance.Model(&models.User{}).Where("id = ?", userID).Updates(updates)
	return record.Error
}

// SetHevyLastSyncForUser advances only the Hevy sync baseline, and only while the user is
// still connected. The sync runs in the background on a copy of the user loaded before it
// started, so saving that whole copy would write back a Hevy key the user removed in the
// meantime (and undo any other profile change made during the import).
func SetHevyLastSyncForUser(userID uuid.UUID, at time.Time) error {
	record := Instance.Model(&models.User{}).
		Where("id = ?", userID).
		Where("hevy_api_key IS NOT NULL").
		Update("hevy_last_sync", at)
	return record.Error
}

func GetStravaUsers() (users []models.User, err error) {
	err = nil
	users = []models.User{}

	record := Instance.Where("users.enabled = ?", true).
		Where("users.strava_code IS NOT NULL").
		Find(&users)
	if record.Error != nil {
		return users, record.Error
	}

	return
}

func GetHevyUsers() (users []models.User, err error) {
	err = nil
	users = []models.User{}

	record := Instance.Where("users.enabled = ?", true).
		Where("users.hevy_api_key IS NOT NULL").
		Find(&users)
	if record.Error != nil {
		return users, record.Error
	}

	return
}

// Retrieves the amount of enabled users in the user table
func GetAmountOfEnabledUsers() (int, error) {
	var users []models.User

	userRecords := Instance.
		Where(&models.User{Enabled: true}).
		Find(&users)

	if userRecords.Error != nil {
		return 0, userRecords.Error
	}

	return int(userRecords.RowsAffected), nil
}
