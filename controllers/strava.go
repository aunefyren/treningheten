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

	now := time.Now()

	season, seasonFound, err := GetOngoingSeasonFromDB(now)
	if err != nil {
		log.Println("Failed to check for ongoing season.")
		return
	} else if !seasonFound {
		log.Println("No ongoing season found.")
		return
	}

	users, err := database.GetStravaUsersWithinSeason(season.ID)
	if err != nil {
		log.Println("Failed to get Strava users.")
		return
	}

	log.Println("Got '" + strconv.Itoa(len(users)) + "' users.")

	for _, user := range users {
		err = StravaSyncWeekForUser(user, *configFile, season)
	}
	log.Println("Strava sync task finished.")
}

func StravaSyncWeekForUser(user models.User, configFile models.ConfigStruct, season models.Season) (err error) {
	err = nil
	now := time.Now()
	log.Println("Strava sync for user '" + user.FirstName + " " + user.LastName + "'.")

	goal, err := database.GetGoalFromUserWithinSeason(season.ID, user.ID)
	if err != nil {
		log.Println("Failed to get goal. ID: " + user.ID.String())
		return
	}

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

	monday, err := utilities.FindEarlierMonday(now)
	if err != nil {
		log.Println("Failed to find monday. ID: " + user.ID.String())
		return errors.New("Failed to find monday.")
	}

	sunday, err := utilities.FindNextSunday(now)
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
		tmp := true
		if user.StravaWalks == &tmp && strings.ToLower(activity.SportType) == "walk" {
			continue
		}

		newExercise := false
		exercise, err := database.GetExerciseForUserWithStravaID(user.ID, int(activity.ID))
		if err != nil {
			log.Println("Failed to get exercise. ID: " + user.ID.String())
			return errors.New("Failed to get exercise.")
		} else if exercise == nil {
			// Get exercise day
			exerciseDay, err := database.GetExerciseDayByDateAndGoal(goal.ID, activity.StartDate)
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
				exerciseDay.GoalID = goal.ID

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

			newExercise = true
		}

		exercise.Enabled = true
		exercise.Note = activity.Name
		elapsedTime := time.Duration(activity.ElapsedTime)
		exercise.Duration = &elapsedTime
		exercise.On = true
		exercise.StravaID = strconv.Itoa(int(activity.ID))

		finalExercise, err := database.UpdateExerciseInDB(*exercise)
		if err != nil {
			log.Println("Failed to get exercise. ID: " + user.ID.String())
			return errors.New("Failed to get exercise.")
		}

		if newExercise {
			log.Println("Creating new operation for exercise based on activity.")
			operation, err := StravaCreateOperationForActivity(activity, user, finalExercise)
			if err != nil {
				log.Println("Failed to create operation. Error: " + err.Error())
				log.Println("Sport type was: " + activity.SportType)
			} else if operation == nil {
				log.Println("Failed to create operation. No error.")
				log.Println("Sport type was: " + activity.SportType)
			}

		}

		log.Println("Updated exercise.")
	}

	return
}

func StravaCreateOperationForActivity(activity models.StravaGetActivitiesRequestReply, user models.User, exercise models.Exercise) (finalOperation *models.Operation, err error) {
	err = nil
	finalOperation = nil

	operations, err := database.GetOperationsByExerciseID(exercise.ID)
	if len(operations) > 0 {
		return
	}

	if strings.ToLower(activity.SportType) == "pickleball" {
		activity.SportType = "Padel"
	}

	action, err := database.GetActionByStravaName(activity.SportType)
	if err != nil {
		return finalOperation, err
	} else if action == nil {
		return finalOperation, nil
	}

	operation := models.Operation{}
	operation.ID = uuid.New()
	operation.ExerciseID = exercise.ID
	operation.ActionID = &action.ID
	operation.Type = action.Type

	operation, err = database.CreateOperationInDB(operation)
	if err != nil {
		return finalOperation, err
	}

	finalOperation = &operation

	operationSet := models.OperationSet{}
	operationSet.ID = uuid.New()
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

	_, err = database.CreateOperationSetInDB(operationSet)
	if err != nil {
		return finalOperation, err
	}

	return
}
