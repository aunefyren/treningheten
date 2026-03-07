package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"
	"github.com/gin-gonic/gin"

	"github.com/google/uuid"
)

const stravaAPIBaseURL = "https://www.strava.com/api/v3"

// package-level rate limiter
var stravaRateLimiter = time.NewTicker(time.Minute / 30)

func stravaWait() {
	<-stravaRateLimiter.C
}

func StravaAuthorize(code string) (authorization models.StravaAuthorizeRequestReply, err error) {
	err = nil
	authorization = models.StravaAuthorizeRequestReply{}
	url := stravaAPIBaseURL + "/oauth/token"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Error("URL request generation threw error. Error: " + err.Error())
		return authorization, errors.New("URL request generation threw error.")
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")

	// Params
	q := req.URL.Query()
	q.Add("client_id", files.ConfigFile.StravaClientID)
	q.Add("client_secret", files.ConfigFile.StravaClientSecret)
	q.Add("code", code)
	q.Add("grant_type", "authorization_code")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("URL request threw error. Error: " + err.Error())
		return authorization, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Trace("Authorize gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Error("HTTP code was not 200. Body:")
		logger.Log.Error(string(body))
	}

	err = json.Unmarshal(body, &authorization)
	if err != nil {
		logger.Log.Error("Failed to parse reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaReauthorize(code string) (authorization models.StravaReauthorizationRequestReply, err error) {
	err = nil
	authorization = models.StravaReauthorizationRequestReply{}
	url := stravaAPIBaseURL + "/oauth/token"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Error("URL request generation threw error. Error: " + err.Error())
		return authorization, errors.New("URL request generation threw error.")
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")

	// Params
	q := req.URL.Query()
	q.Add("client_id", files.ConfigFile.StravaClientID)
	q.Add("client_secret", files.ConfigFile.StravaClientSecret)
	q.Add("refresh_token", code)
	q.Add("grant_type", "refresh_token")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("URL request threw error. Error: " + err.Error())
		return authorization, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Trace("Reauthorize gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Error("HTTP code was not 200. Body:")
		logger.Log.Error(string(body))
	}

	err = json.Unmarshal(body, &authorization)
	if err != nil {
		logger.Log.Error("Failed to parse reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaGetActivities(token string, before int, after int) (activities []models.StravaGetActivitiesRequestReply, err error) {
	stravaWait() // wait until a slot is available

	err = nil
	activities = []models.StravaGetActivitiesRequestReply{}
	url := stravaAPIBaseURL + "/athlete/activities"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return activities, errors.New("URL request generation threw error.")
	}

	// Debug lines
	// logger.Log.Info(strconv.Itoa(before))
	// logger.Log.Info(strconv.Itoa(after))

	// Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Params
	q := req.URL.Query()
	q.Add("before", strconv.Itoa(before))
	q.Add("after", strconv.Itoa(after))
	q.Add("page", "1")
	q.Add("per_page", "30")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return activities, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Info("Get activities gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return activities, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("HTTP code was not 200. Body:")
		logger.Log.Info(string(body))
	}

	err = json.Unmarshal(body, &activities)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return activities, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaSyncWeekForAllUsers() {
	users, err := database.GetStravaUsers()
	if err != nil {
		logger.Log.Error("Failed to get Strava users.")
		return
	}

	logger.Log.Debug("Got '" + strconv.Itoa(len(users)) + "' users.")

	for _, user := range users {
		err = StravaSyncWeekForUser(user, time.Now())
		if err != nil {
			logger.Log.Error("Sync Strava for user returned error. Error: " + err.Error())
		}
	}

	logger.Log.Info("Strava sync task finished.")
}

func StravaSyncWeekForUser(user models.User, pointInTime time.Time) (err error) {
	err = nil
	logger.Log.Debug("Strava sync for user '" + user.FirstName + " " + user.LastName + "'.")

	monday, err := utilities.FindEarlierMonday(pointInTime)
	if err != nil {
		logger.Log.Error("Failed to find monday. ID: " + user.ID.String())
		return errors.New("Failed to find monday.")
	}

	sunday, err := utilities.FindNextSunday(pointInTime)
	if err != nil {
		logger.Log.Error("Failed to find sunday. ID: " + user.ID.String())
		return errors.New("Failed to find sunday.")
	}

	token, err := StravaGetAuthorizationForUser(user)
	if err != nil {
		logger.Log.Error("failed to find authorize user toward Strava. error: " + err.Error())
		return errors.New("failed to find authorize user toward Strava")
	}

	activities, err := StravaGetActivities(token, int(sunday.Unix()), int(monday.Unix()))
	if err != nil {
		logger.Log.Error("Failed to get activities. ID: " + user.ID.String())
		return errors.New("Failed to get activities.")
	}

	logger.Log.Debug("Got '" + strconv.Itoa((len(activities))) + "' activities for user.")

	// Give user achievement for connecting Strava, ignore outcome
	go GiveUserAnAchievement(user.ID, uuid.MustParse("fb4f6c1f-dfad-4df7-8007-4cfd6f351b17"), time.Now(), 5)

	for _, activity := range activities {
		err = StravaSyncActivityForUser(activity, user, token)
		if err != nil {
			logger.Log.Errorf("failed to sync activity '%d' for user '%s %s'. error: %s", activity.ID, user.FirstName, user.LastName, err.Error())
		}
	}

	return
}

func StravaGetAuthorizationForUser(user models.User) (token string, err error) {
	token = ""
	err = nil

	if user.StravaCode == nil {
		logger.Log.Error("no Strava code")
		return token, errors.New("no Strava code")
	}

	stravaCodeData := strings.Split(*user.StravaCode, ":")

	if len(stravaCodeData) != 2 {
		logger.Log.Error("Invalid Strava code format for user. ID: " + string(user.ID.String()))
		return token, errors.New("Invalid Strava code format for user.")
	}

	// If totally new authorization
	switch strings.ToLower(stravaCodeData[0]) {
	case "c":
		authorization, err := StravaAuthorize(stravaCodeData[1])
		if err != nil {
			logger.Log.Error("Failed to authorize user. ID: " + user.ID.String())
			return token, errors.New("Failed to authorize user.")
		}

		newCode := "r:" + authorization.RefreshToken
		user.StravaCode = &newCode
		user, err = database.UpdateUser(user)
		if err != nil {
			logger.Log.Error("Failed to update user. ID: " + user.ID.String())
			return token, errors.New("Failed to update user.")
		}

		token = authorization.AccessToken
		// If re-authorization
	case "r":
		authorization, err := StravaReauthorize(stravaCodeData[1])
		if err != nil {
			logger.Log.Error("Failed to re-authorize user. ID: " + user.ID.String())
			return token, errors.New("Failed to re-authorize user.")
		}

		newCode := "r:" + authorization.RefreshToken
		user.StravaCode = &newCode
		user, err = database.UpdateUser(user)
		if err != nil {
			logger.Log.Error("Failed to update user. ID: " + user.ID.String())
			return token, errors.New("Failed to update user.")
		}

		token = authorization.AccessToken
	default:
		logger.Log.Error("Invalid Strava code format for user. ID: " + string(user.ID.String()))
		return token, errors.New("Invalid Strava code format for user.")
	}

	return
}

func StravaSyncActivityForUser(activity models.StravaGetActivitiesRequestReply, user models.User, token string) (err error) {
	err = nil
	now := time.Now()

	// skip walks if enabled
	if user.StravaWalks != nil && *user.StravaWalks && strings.ToLower(activity.SportType) == "walk" {
		logger.Log.Trace("Skipping activity because user has 'ignore walks' enabled.")
		return nil
	} else {
		logger.Log.Trace("Sport type is: " + activity.SportType)
	}

	// add Strava ID to user if missing
	if user.StravaID == nil {
		stravaID := strconv.Itoa(activity.Athlete.ID)
		user.StravaID = &stravaID
		user, err = database.UpdateUser(user)
		if err != nil {
			logger.Log.Error("Failed to update user Strava ID. Error: " + err.Error())
			return errors.New("Failed to update user Strava ID.")
		}
	}

	// check for data streams
	var stravaStreams *models.StravaActivityStreams
	stravaActivityStreams, err := StravaGetActivityStreams(token, strconv.FormatInt(activity.ID, 10))
	if err != nil {
		logger.Log.Warn("Failed to get activity streams. Error: " + err.Error())
	} else {
		stravaStreams = &stravaActivityStreams
	}

	exercise, err := database.GetExerciseForUserWithStravaID(user.ID, strconv.Itoa(int(activity.ID)))
	if err != nil {
		logger.Log.Error("Failed to get exercise. ID: " + user.ID.String())
		return errors.New("Failed to get exercise.")
	} else if exercise == nil {
		// Get exercise day
		exerciseDay, err := database.GetExerciseDayByDateAndUserID(user.ID, activity.StartDateLocal)
		if err != nil {
			logger.Log.Error("Failed to get exercise day. ID: " + user.ID.String())
			return errors.New("Failed to get exercise day.")
		} else if exerciseDay == nil {
			logger.Log.Trace("Creating new exercise day.")

			exerciseDay = &models.ExerciseDay{}
			exerciseDay.ID = uuid.New()
			exerciseDay.Enabled = true
			exerciseDay.CreatedAt = now
			exerciseDay.UpdatedAt = now
			exerciseDay.UserID = &user.ID

			logger.Log.Trace("activity start: " + activity.StartDateLocal.String())

			dateObject := time.Date(activity.StartDateLocal.Year(), activity.StartDateLocal.Month(), activity.StartDateLocal.Day(), 0, 0, 0, 0, time.Local)
			exerciseDay.Date = dateObject

			logger.Log.Trace("exercise day date: " + dateObject.String())

			err = database.CreateExerciseDayInDB(*exerciseDay)
			if err != nil {
				logger.Log.Error("Failed to create exercise day. ID: " + user.ID.String())
				return errors.New("Failed to create exercise day.")
			}
		}
		logger.Log.Trace("Creating new exercise.")

		exercise = &models.Exercise{}
		exercise.ID = uuid.New()
		exercise.CreatedAt = now
		exercise.UpdatedAt = now
		exercise.ExerciseDayID = exerciseDay.ID
	}

	// Strava ID list
	oldStravaID := ""
	idString := exercise.StravaID
	if idString != nil {
		oldStravaID = *idString
	}

	newStravaID := ""
	if oldStravaID != "" {
		stravaIDArray := strings.Split(oldStravaID, ";")
		idFound := false
		for _, stravaID := range stravaIDArray {
			if stravaID == strconv.Itoa(int(activity.ID)) {
				idFound = true
				break
			}
		}
		if !idFound {
			stravaIDArray = append(stravaIDArray, strconv.Itoa(int(activity.ID)))
		}
		for index, stravaID := range stravaIDArray {
			if index != 0 {
				newStravaID += ";"
			}
			newStravaID += stravaID
		}
	} else {
		newStravaID = strconv.Itoa(int(activity.ID))
	}

	exercise.Enabled = true
	exercise.Note = activity.Name
	elapsedTime := time.Duration(activity.ElapsedTime)
	exercise.Duration = &elapsedTime
	exercise.IsOn = true
	exercise.StravaID = &newStravaID
	exercise.Time = &activity.StartDate

	logger.Log.Tracef("Strava activity start time %s for Strava ID %s", activity.StartDate, newStravaID)
	logger.Log.Tracef("Strava activity local start time %s for Strava ID %s", activity.StartDateLocal, newStravaID)

	finalExercise, err := database.UpdateExerciseInDB(*exercise)
	if err != nil {
		logger.Log.Error("Failed to get exercise. ID: " + user.ID.String())
		return errors.New("Failed to get exercise.")
	}

	logger.Log.Trace("Updated exercise.")

	operation, err := StravaSyncOperationForActivity(activity, user, finalExercise, stravaStreams)
	if err != nil {
		logger.Log.Error("Failed to sync operation. Error: " + err.Error())
		logger.Log.Error("Sport type was: " + activity.SportType)
	} else if operation == nil {
		logger.Log.Error("Failed to sync operation. No error.")
		logger.Log.Error("Sport type was: " + activity.SportType)
	}

	logger.Log.Trace("Synced operations.")
	return nil
}

func StravaSyncOperationForActivity(activity models.StravaGetActivitiesRequestReply, user models.User, exercise models.Exercise, streams *models.StravaActivityStreams) (finalOperation *models.Operation, err error) {
	err = nil
	finalOperation = nil

	if strings.ToLower(activity.SportType) == "pickleball" {
		activity.SportType = "Padel"
	}

	// Get action by Strava activity
	action, err := database.GetActionByStravaName(activity.SportType)
	if err != nil {
		return finalOperation, err
	} else if action == nil {
		return finalOperation, nil
	}

	// Get or create operation
	var operation models.Operation = models.Operation{}
	oldOperation, err := database.GetOperationByStravaIDAndUserIDAndExerciseID(user.ID, int(activity.ID), exercise.ID)
	if err != nil {
		return finalOperation, err
	} else if oldOperation == nil {
		logger.Log.Info("Creating new operation.")
		operation = models.Operation{}
		operation.ID = uuid.New()
	} else {
		logger.Log.Info("Updating operation.")
		operation = *oldOperation
	}

	operation.ExerciseID = exercise.ID
	operation.ActionID = &action.ID
	operation.Type = action.Type
	stravaID := strconv.Itoa(int(activity.ID))
	operation.StravaID = &stravaID
	durationTime := time.Duration(activity.ElapsedTime)
	operation.Duration = &durationTime
	operation.Note = &activity.Name

	newOperation, err := database.UpdateOperationInDB(operation)
	if err != nil {
		return finalOperation, err
	}

	logger.Log.Info("Updated operation.")

	finalOperation = &newOperation

	// Get or create operation set
	var operationSet models.OperationSet = models.OperationSet{}
	oldOperationSet, err := database.GetOperationSetByStravaIDAndUserIDAndOperationID(user.ID, int(activity.ID), operation.ID)
	if err != nil {
		return finalOperation, err
	} else if oldOperationSet == nil {
		logger.Log.Info("Creating new operation set.")
		operationSet = models.OperationSet{}
		operationSet.ID = uuid.New()
	} else {
		logger.Log.Info("Updating operation set.")
		operationSet = *oldOperationSet
	}

	operationSet.StravaID = &stravaID
	operationSet.OperationID = operation.ID
	movingTime := time.Duration(activity.MovingTime)
	operationSet.MovingTime = &movingTime
	totalTime := time.Duration(activity.ElapsedTime)
	operationSet.Time = &totalTime

	now := time.Now()
	operationSet.StravaDataRetrievedAt = &now

	if streams != nil {
		operationSet.StravaStreams = &models.StravaStreamsJSON{StravaActivityStreams: *streams}
	}

	if activity.Distance != 0.0 {
		var newFloat float64
		var newDistance float64
		newFloat = activity.Distance
		newDistance = (newFloat / 1000)
		operationSet.Distance = &newDistance
	}

	_, err = database.UpdateOperationSetInDB(operationSet)
	if err != nil {
		return finalOperation, err
	}

	logger.Log.Info("Updated operation set.")

	// Sync duration of operations to exercise
	err = SyncStravaOperationsToExerciseSession(exercise.ID, user.ID)
	if err != nil {
		return finalOperation, err
	}

	return
}

func SyncStravaOperationsToExerciseSession(exerciseID uuid.UUID, userID uuid.UUID) (err error) {
	err = nil

	logger.Log.Info("Syncing Strava operations to exercise.")

	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise object. Error: " + err.Error())
		return errors.New("Failed to get exercise object.")
	} else if exercise == nil {
		logger.Log.Info("Failed to find exercise object.")
		return errors.New("Failed to find exercise object.")
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(*exercise)
	if err != nil {
		logger.Log.Info("Failed to convert exercise to exercise object. Error: " + err.Error())
		return errors.New("Failed to convert exercise to exercise object.")
	}

	var newDuration time.Duration = time.Duration(0)
	for _, operation := range exerciseObject.Operations {
		if operation.Duration != nil {
			newDuration += *operation.Duration
		}
	}

	var newNote string = ""
	for index, operation := range exerciseObject.Operations {
		if operation.Note != nil {
			if index != 0 {
				newNote += " + "
			}
			newNote += *operation.Note
		}
	}

	exercise.Duration = &newDuration
	exercise.Note = newNote

	_, err = database.UpdateExerciseInDB(*exercise)
	if err != nil {
		logger.Log.Info("Failed to update exercise. Error: " + err.Error())
		return errors.New("Failed to update exercise.")
	}

	logger.Log.Info("Updated exercise with operations.")

	return
}

func StravaGetActivityStreams(token string, activityID string) (streams models.StravaActivityStreams, err error) {
	stravaWait() // wait until a slot is available

	err = nil
	streams = models.StravaActivityStreams{}

	url := stravaAPIBaseURL + "/activities/" + activityID + "/streams"
	var jsonStr = []byte(``)

	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return streams, errors.New("URL request generation threw error.")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	q := req.URL.Query()
	q.Add("keys", "time,latlng,altitude,heartrate,cadence,watts,temp,velocity_smooth")
	q.Add("key_by_type", "true")

	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return streams, errors.New("URL request threw error.")
	}

	defer resp.Body.Close()

	logger.Log.Info("Get activity streams gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return streams, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("HTTP code was not 200. Body:")
		logger.Log.Info(string(body))
		return streams, errors.New("Strava returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &streams)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return streams, errors.New("Failed to parse reply body.")
	}

	return
}

func APISyncStravaActivitiesForUsers(context *gin.Context) {
	var syncRequest models.StravaSyncActivitiesForUsersRequest

	if !files.ConfigFile.StravaEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Strava is not enabled"})
		context.Abort()
		return
	}

	if err := context.ShouldBindJSON(&syncRequest); err != nil {
		logger.Log.Error("failed to parse request. error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		context.Abort()
		return
	}

	usersToSync := []models.User{}
	if syncRequest.UserIDs != nil && len(*syncRequest.UserIDs) > 0 {
		for _, userID := range *syncRequest.UserIDs {
			parsedID, err := uuid.Parse(userID)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse user ID"})
				context.Abort()
				return
			}

			user, err := database.GetUserInformation(parsedID)
			if err != nil {
				context.JSON(http.StatusNotFound, gin.H{"error": "failed to find user by ID"})
				context.Abort()
				return
			}

			if user.Enabled && user.StravaCode != nil && *user.StravaCode != "" {
				usersToSync = append(usersToSync, user)
			}
		}
	} else {
		users, err := database.GetUsersInformation()
		if err != nil {
			logger.Log.Info("Failed to get user objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user objects."})
			context.Abort()
			return
		}

		for _, user := range users {
			if user.Enabled && user.StravaCode != nil && *user.StravaCode != "" {
				usersToSync = append(usersToSync, user)
			}
		}
	}

	go SyncStravaActivitiesForUsers(usersToSync)

	context.JSON(http.StatusAccepted, gin.H{"message": "Strava sync started!"})
}

func SyncStravaActivitiesForUsers(usersToSync []models.User) {
	for _, user := range usersToSync {
		userOperationSets, err := database.GetStravaOperationSetsByUserID(user.ID)
		if err != nil {
			logger.Log.Info("failed to get user operation sets. error: " + err.Error())
			continue
		}

		if len(userOperationSets) < 1 {
			continue
		}

		stravaToken, err := StravaGetAuthorizationForUser(user)
		if err != nil {
			logger.Log.Info("failed to authorize user toward Strava. error: " + err.Error())
			continue
		}

		logger.Log.Debugf("syncing activities for user '%s %s'", user.FirstName, user.LastName)

		sort.Slice(userOperationSets, func(i, j int) bool {
			return userOperationSets[i].CreatedAt.After(userOperationSets[j].CreatedAt)
		})

		for _, operationSet := range userOperationSets {
			logger.Log.Debugf("getting activity from Strava '%s'", *operationSet.StravaID)

			if operationSet.StravaDataRetrievedAt != nil && time.Since(*operationSet.StravaDataRetrievedAt) < time.Duration(time.Hour*24*7) {
				logger.Log.Debugf("Strava activity '%s' data is too new for full refresh", *operationSet.StravaID)
				continue
			}

			stravaActivity, err := StravaGetActivity(stravaToken, *operationSet.StravaID)
			if err != nil {
				logger.Log.Info("failed to authorize user toward Strava. error: " + err.Error())
				continue
			}

			err = StravaSyncActivityForUser(stravaActivity, user, stravaToken)
			if err != nil {
				logger.Log.Info("failed to update Strava activity. error: " + err.Error())
				continue
			}

			logger.Log.Infof("updated Strava activity '%d' for user '%s %s'", stravaActivity.ID, user.FirstName, user.LastName)
		}
	}

	logger.Log.Infof("Strava activities synced for %d users", len(usersToSync))
}

func StravaGetActivity(token string, activityID string) (activity models.StravaGetActivitiesRequestReply, err error) {
	stravaWait()
	err = nil
	activity = models.StravaGetActivitiesRequestReply{}
	url := stravaAPIBaseURL + "/activities/" + activityID

	var jsonStr = []byte(``)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return activity, errors.New("URL request generation threw error.")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return activity, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Info("Get activity gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return activity, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("HTTP code was not 200. Body:")
		logger.Log.Info(string(body))
		return activity, errors.New("Strava returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &activity)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return activity, errors.New("Failed to parse reply body.")
	}

	return
}
