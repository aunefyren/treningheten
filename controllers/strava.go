package controllers

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const stravaAPIBaseURL = "https://www.strava.com/api/v3"

func StravaAuthorize(config models.ConfigStruct, code string) (authorization models.StravaAuthorizeRequestReply, err error) {
	err = nil
	authorization = models.StravaAuthorizeRequestReply{}
	url := stravaAPIBaseURL + "/oauth/token"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("URL request generation threw error. Error: " + err.Error())
		return authorization, errors.New("URL request generation threw error.")
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")

	// Params
	q := req.URL.Query()
	q.Add("client_id", config.StravaClientID)
	q.Add("client_secret", config.StravaClientSecret)
	q.Add("code", code)
	q.Add("grant_type", "authorization_code")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("URL request threw error. Error: " + err.Error())
		return authorization, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	log.Println("Authorize gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		log.Println("HTTP code was not 200. Body:")
		log.Println(string(body))
	}

	err = json.Unmarshal(body, &authorization)
	if err != nil {
		log.Println("Failed to parse reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaReauthorize(config models.ConfigStruct, code string) (authorization models.StravaReauthorizationRequestReply, err error) {
	err = nil
	authorization = models.StravaReauthorizationRequestReply{}
	url := stravaAPIBaseURL + "/oauth/token"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("URL request generation threw error. Error: " + err.Error())
		return authorization, errors.New("URL request generation threw error.")
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")

	// Params
	q := req.URL.Query()
	q.Add("client_id", config.StravaClientID)
	q.Add("client_secret", config.StravaClientSecret)
	q.Add("refresh_token", code)
	q.Add("grant_type", "refresh_token")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("URL request threw error. Error: " + err.Error())
		return authorization, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	log.Println("Reauthorize gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		log.Println("HTTP code was not 200. Body:")
		log.Println(string(body))
	}

	err = json.Unmarshal(body, &authorization)
	if err != nil {
		log.Println("Failed to parse reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaGetActivities(config models.ConfigStruct, token string, before int, after int) (activities []models.StravaGetActivitiesRequestReply, err error) {
	err = nil
	activities = []models.StravaGetActivitiesRequestReply{}
	url := stravaAPIBaseURL + "/athlete/activities"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Println("URL request generation threw error. Error: " + err.Error())
		return activities, errors.New("URL request generation threw error.")
	}

	// Debug lines
	// log.Println(strconv.Itoa(before))
	// log.Println(strconv.Itoa(after))

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
		log.Println("URL request threw error. Error: " + err.Error())
		return activities, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	log.Println("Get activities gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read reply body. Error: " + err.Error())
		return activities, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		log.Println("HTTP code was not 200. Body:")
		log.Println(string(body))
	}

	err = json.Unmarshal(body, &activities)
	if err != nil {
		log.Println("Failed to parse reply body. Error: " + err.Error())
		return activities, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaSyncWeekForAllUsers() {
	configFile, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config file. Error: " + err.Error())
		return
	}

	users, err := database.GetStravaUsers()
	if err != nil {
		log.Println("Failed to get Strava users.")
		return
	}

	log.Println("Got '" + strconv.Itoa(len(users)) + "' users.")

	for _, user := range users {
		err = StravaSyncWeekForUser(user, *configFile, time.Now())
		if err != nil {
			log.Println("Sync Strava for user returned error. Error: " + err.Error())
		}
	}

	log.Println("Strava sync task finished.")
}

func StravaSyncWeekForUser(user models.User, configFile models.ConfigStruct, pointInTime time.Time) (err error) {
	err = nil
	log.Println("Strava sync for user '" + user.FirstName + " " + user.LastName + "'.")
	now := time.Now()

	stravaCodeData := strings.Split(*user.StravaCode, ":")

	if len(stravaCodeData) != 2 {
		log.Println("Invalid Strava code format for user. ID: " + string(user.ID.String()))
		return errors.New("Invalid Strava code format for user.")
	}

	token := ""

	// If totally new authorization
	if stravaCodeData[0] == "c" {
		authorization, err := StravaAuthorize(configFile, stravaCodeData[1])
		if err != nil {
			log.Println("Failed to authorize user. ID: " + user.ID.String())
			return errors.New("Failed to authorize user.")
		}

		newCode := "r:" + authorization.RefreshToken
		user.StravaCode = &newCode
		user, err = database.UpdateUser(user)
		if err != nil {
			log.Println("Failed to update user. ID: " + user.ID.String())
			return errors.New("Failed to update user.")
		}

		token = authorization.AccessToken
		// If re-authorization
	} else if stravaCodeData[0] == "r" {
		authorization, err := StravaReauthorize(configFile, stravaCodeData[1])
		if err != nil {
			log.Println("Failed to re-authorize user. ID: " + user.ID.String())
			return errors.New("Failed to re-authorize user.")
		}

		newCode := "r:" + authorization.RefreshToken
		user.StravaCode = &newCode
		user, err = database.UpdateUser(user)
		if err != nil {
			log.Println("Failed to update user. ID: " + user.ID.String())
			return errors.New("Failed to update user.")
		}

		token = authorization.AccessToken
	} else {
		log.Println("Invalid Strava code format for user. ID: " + string(user.ID.String()))
		return errors.New("Invalid Strava code format for user.")
	}

	monday, err := utilities.FindEarlierMonday(pointInTime)
	if err != nil {
		log.Println("Failed to find monday. ID: " + user.ID.String())
		return errors.New("Failed to find monday.")
	}

	sunday, err := utilities.FindNextSunday(pointInTime)
	if err != nil {
		log.Println("Failed to find sunday. ID: " + user.ID.String())
		return errors.New("Failed to find sunday.")
	}

	activities, err := StravaGetActivities(configFile, token, int(sunday.Unix()), int(monday.Unix()))
	if err != nil {
		log.Println("Failed to get activities. ID: " + user.ID.String())
		return errors.New("Failed to get activities.")
	}

	log.Println("Got '" + strconv.Itoa((len(activities))) + "' activities for user.")

	for _, activity := range activities {
		// Skip walks if enabled
		if user.StravaWalks != nil && *user.StravaWalks && strings.ToLower(activity.SportType) == "walk" {
			log.Println("Skipping activity because user has 'ignore walks' enabled.")
			continue
		} else {
			log.Println("Sport type is: " + activity.SportType)
		}

		exercise, err := database.GetExerciseForUserWithStravaID(user.ID, int(activity.ID))
		if err != nil {
			log.Println("Failed to get exercise. ID: " + user.ID.String())
			return errors.New("Failed to get exercise.")
		} else if exercise == nil {
			// Get exercise day
			exerciseDay, err := database.GetExerciseDayByDateAndUserID(user.ID, activity.StartDate)
			if err != nil {
				log.Println("Failed to get exercise day. ID: " + user.ID.String())
				return errors.New("Failed to get exercise day.")
			} else if exerciseDay == nil {
				log.Println("Creating new exercise day.")

				exerciseDay = &models.ExerciseDay{}
				exerciseDay.ID = uuid.New()
				exerciseDay.Enabled = true
				exerciseDay.CreatedAt = now
				exerciseDay.UpdatedAt = now
				exerciseDay.UserID = &user.ID

				dateObject := time.Date(activity.StartDate.Year(), activity.StartDate.Month(), activity.StartDate.Day(), 0, 0, 0, activity.StartDate.Nanosecond(), activity.StartDate.Location())
				exerciseDay.Date = dateObject

				err = database.CreateExerciseDayInDB(*exerciseDay)
				if err != nil {
					log.Println("Failed to create exercise day. ID: " + user.ID.String())
					return errors.New("Failed to create exercise day.")
				}
			}
			log.Println("Creating new exercise.")

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
		exercise.On = true
		exercise.StravaID = &newStravaID

		finalExercise, err := database.UpdateExerciseInDB(*exercise)
		if err != nil {
			log.Println("Failed to get exercise. ID: " + user.ID.String())
			return errors.New("Failed to get exercise.")
		}

		log.Println("Updated exercise.")

		operation, err := StravaSyncOperationForActivity(activity, user, finalExercise)
		if err != nil {
			log.Println("Failed to sync operation. Error: " + err.Error())
			log.Println("Sport type was: " + activity.SportType)
		} else if operation == nil {
			log.Println("Failed to sync operation. No error.")
			log.Println("Sport type was: " + activity.SportType)
		}

		log.Println("Synced operations.")
	}

	return
}

func StravaSyncOperationForActivity(activity models.StravaGetActivitiesRequestReply, user models.User, exercise models.Exercise) (finalOperation *models.Operation, err error) {
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
		log.Println("Creating new operation.")
		operation = models.Operation{}
		operation.ID = uuid.New()
	} else {
		log.Println("Updating operation.")
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

	log.Println("Updated operation.")

	finalOperation = &newOperation

	// Get or create operation set
	var operationSet models.OperationSet = models.OperationSet{}
	oldOperationSet, err := database.GetOperationSetByStravaIDAndUserIDAndOperationID(user.ID, int(activity.ID), operation.ID)
	if err != nil {
		return finalOperation, err
	} else if oldOperationSet == nil {
		log.Println("Creating new operation set.")
		operationSet = models.OperationSet{}
		operationSet.ID = uuid.New()
	} else {
		log.Println("Updating operation set.")
		operationSet = *oldOperationSet
	}

	operationSet.StravaID = &stravaID
	operationSet.OperationID = operation.ID
	movingTime := time.Duration(activity.MovingTime)
	operationSet.Time = &movingTime

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

	log.Println("Updated operation set.")

	// Sync duration of operations to exercise
	err = SyncStravaOperationsToExerciseSession(exercise.ID, user.ID)
	if err != nil {
		return finalOperation, err
	}

	return
}

func SyncStravaOperationsToExerciseSession(exerciseID uuid.UUID, userID uuid.UUID) (err error) {
	err = nil

	log.Println("Syncing Strava operations to exercise.")

	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		log.Println("Failed to get exercise object. Error: " + err.Error())
		return errors.New("Failed to get exercise object.")
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		log.Println("Failed to convert exercise to exercise object. Error: " + err.Error())
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

	_, err = database.UpdateExerciseInDB(exercise)
	if err != nil {
		log.Println("Failed to update exercise. Error: " + err.Error())
		return errors.New("Failed to update exercise.")
	}

	log.Println("Updated exercise with operations.")

	return
}
