package controllers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ollamaChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type ollamaCacheEntry struct {
	hash    string
	message string
}

var (
	ollamaCache     = make(map[string]ollamaCacheEntry)
	ollamaCacheMu   sync.RWMutex
	ollamaCancelFns = make(map[string]context.CancelFunc)
	ollamaCancelSeq = make(map[string]uint64)
	ollamaCancelMu  sync.Mutex
)

func ollamaPayloadHash(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", sum)
}

type ollamaUserPayload struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type ollamaExercisePayload struct {
	Date      string `json:"date"`
	DayOfWeek string `json:"day_of_week"`
	Note      string `json:"note,omitempty"`
	Count     int    `json:"exercise_count"`
}

type ollamaGoalPayload struct {
	WeeklyGoal    int    `json:"weekly_goal"`
	Competing     bool   `json:"competing"`
	SickleaveLeft int    `json:"sickleave_days_left"`
	UserFirstName string `json:"user_first_name"`
}

type ollamaSeasonPayload struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Start       string              `json:"start"`
	End         string              `json:"end"`
	Prize       string              `json:"prize,omitempty"`
	Goals       []ollamaGoalPayload `json:"goals"`
}

type ollamaPromptPayload struct {
	PointInTime    string                  `json:"point_in_time"`
	User           ollamaUserPayload       `json:"user"`
	ActiveSeasons  []ollamaSeasonPayload   `json:"active_seasons"`
	RecentWorkouts []ollamaExercisePayload `json:"recent_workouts_past_31_days"`
}

func buildOllamaPayload(user models.User, exerciseDays []models.ExerciseDayObject, activeSeasons []models.SeasonObject, pointInTime time.Time) (string, error) {
	seasons := make([]ollamaSeasonPayload, 0, len(activeSeasons))
	for _, s := range activeSeasons {
		goals := make([]ollamaGoalPayload, 0, len(s.Goals))
		for _, g := range s.Goals {
			goals = append(goals, ollamaGoalPayload{
				WeeklyGoal:    g.ExerciseInterval,
				Competing:     g.Competing,
				SickleaveLeft: g.SickleaveLeft,
				UserFirstName: g.User.FirstName,
			})
		}
		seasons = append(seasons, ollamaSeasonPayload{
			Name:        s.Name,
			Description: s.Description,
			Start:       s.Start.Format("2006-01-02"),
			End:         s.End.Format("2006-01-02"),
			Prize:       s.Prize.Name,
			Goals:       goals,
		})
	}

	workouts := make([]ollamaExercisePayload, 0, len(exerciseDays))
	for _, d := range exerciseDays {
		workouts = append(workouts, ollamaExercisePayload{
			Date:      d.Date.Format("2006-01-02"),
			DayOfWeek: d.Date.Format("Monday"),
			Note:      d.Note,
			Count:     len(d.Exercises),
		})
	}

	payload := ollamaPromptPayload{
		PointInTime: pointInTime.Format("2006-01-02 (Monday)"),
		User: ollamaUserPayload{
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
		ActiveSeasons:  seasons,
		RecentWorkouts: workouts,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func OllamaGenerateFrontPageMessage(ctx context.Context, user models.User, exerciseDays []models.ExerciseDayObject, activeSeasons []models.SeasonObject, pointInTime time.Time) (string, error) {
	config := files.ConfigFile.Ollama

	if !config.Enabled {
		return "", errors.New("Ollama is not enabled")
	}

	if config.URL == "" {
		return "", errors.New("Ollama URL is not configured")
	}

	if config.Model == "" {
		return "", errors.New("Ollama model is not configured")
	}

	userPayload, err := buildOllamaPayload(user, exerciseDays, activeSeasons, pointInTime)
	if err != nil {
		return "", errors.New("failed to build Ollama payload: " + err.Error())
	}

	hash := ollamaPayloadHash(userPayload)
	userKey := user.ID.String()

	ollamaCacheMu.RLock()
	entry, found := ollamaCache[userKey]
	ollamaCacheMu.RUnlock()

	if found && entry.hash == hash {
		logger.Log.Debug(("returning cached Ollama response"))
		return entry.message, nil
	}

	messages := []ChatMessage{
		{
			Role: "system",
			Content: `
				You are a fitness coach for Treningheten, a workout tracking app.
				Users join seasons with a weekly workout goal. Hit the goal each week to build a streak; miss it and you must spin a
				wheel if competing — the person it lands on wins the season prize (meaning someone else failed). Not relevant outside of seasons.
				Sick leave is per-season and expires unused. Streaks increase when you exercise every week. Weeks start on Mondays.
				Generate a short front-page message (2 sentences normally, 4 sentences maximum) for the user based on their data. Look at the dates of exercises.
				Be specific: reference the current weekday, their progress this week, and where they stand in the season, if applicable. 
				Friendly tone, humor welcome. No text formatting, reply in plain text. Emoji's are allowed. Respond in English.`,
		},
		{
			Role:    "user",
			Content: userPayload,
		},
	}

	reqBody := ollamaChatRequest{
		Model:    config.Model,
		Messages: messages,
		Stream:   false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", errors.New("failed to marshal Ollama request: " + err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.URL+"/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", errors.New("failed to create Ollama request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.New("failed to reach Ollama: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read Ollama response: " + err.Error())
	}

	var chatResp ollamaChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", errors.New("failed to parse Ollama response: " + err.Error())
	}

	if len(chatResp.Choices) == 0 {
		return "", errors.New("Ollama returned no choices")
	}

	message := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	ollamaCacheMu.Lock()
	ollamaCache[userKey] = ollamaCacheEntry{hash: hash, message: message}
	ollamaCacheMu.Unlock()

	return message, nil
}

func APIGetOllamaFrontPageMessageForUser(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	user, err := database.GetUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	// Current time
	now := time.Now()
	oneMonthAgo := now.AddDate(0, 0, -31)
	toNight := utilities.SetClockToMaximum(now)

	exerciseDays, err := database.GetExerciseDaysBetweenDatesUsingDatesAndUserID(userID, oneMonthAgo, toNight)
	if err != nil {
		logger.Log.Info("Failed to get exercises. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get exercises."})
		context.Abort()
		return
	}

	exerciseDaysObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		logger.Log.Info("Failed to convert exercise days to exercise day objects. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to convert exercise days to exercise day objects."})
		context.Abort()
		return
	}

	// clean up data set for Ollama
	tmp := []models.ExerciseDayObject{}
	for _, exerciseDay := range exerciseDaysObjects {
		if len(exerciseDay.Exercises) == 0 {
			continue
		}

		tmp2 := []models.ExerciseObject{}
		for _, exercise := range exerciseDay.Exercises {
			if exercise.Enabled && exercise.IsOn {
				tmp2 = append(tmp2, exercise)
			}
		}

		exerciseDay.Exercises = tmp2

		if len(exerciseDay.Exercises) > 0 {
			tmp = append(tmp, exerciseDay)
		}
	}
	exerciseDaysObjects = tmp

	seasons, err := GetOngoingSeasonsFromDBForUserID(now, userID)
	if err != nil {
		logger.Log.Info("Failed to get seasons. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get seasons."})
		context.Abort()
		return
	}

	seasonObjects, err := ConvertSeasonsToSeasonObjects(seasons)
	if err != nil {
		logger.Log.Info("Failed to convert seasons to season objects. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to convert seasons to season objects."})
		context.Abort()
		return
	}

	ollamaMessage, err := OllamaGenerateFrontPageMessage(context, user, exerciseDaysObjects, seasonObjects, now)
	if err != nil {
		logger.Log.Info("Failed to get Ollama message. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get Ollama message."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Ollama message retrieved.", "data": ollamaMessage})
}

func OllamaAsyncRefreshCacheForUser(userID uuid.UUID) {
	if !files.ConfigFile.Ollama.Enabled {
		return
	}

	// Cancel any in-flight request for this user and replace it with a new one
	userKey := userID.String()
	ctx, cancel := context.WithCancel(context.Background())

	ollamaCancelMu.Lock()
	if prev, ok := ollamaCancelFns[userKey]; ok {
		prev()
	}
	ollamaCancelFns[userKey] = cancel
	ollamaCancelSeq[userKey]++
	seq := ollamaCancelSeq[userKey]
	ollamaCancelMu.Unlock()

	defer func() {
		ollamaCancelMu.Lock()
		if ollamaCancelSeq[userKey] == seq {
			delete(ollamaCancelFns, userKey)
			delete(ollamaCancelSeq, userKey)
		}
		ollamaCancelMu.Unlock()
		cancel()
	}()

	logger.Log.Error("Ollama cache refresh: starting for user " + userID.String())

	now := time.Now()
	oneMonthAgo := now.AddDate(0, 0, -31)
	toNight := utilities.SetClockToMaximum(now)

	user, err := database.GetUserInformation(userID)
	if err != nil {
		logger.Log.Error("Ollama cache refresh: failed to get user " + userID.String() + ". Error: " + err.Error())
		return
	}

	exerciseDays, err := database.GetExerciseDaysBetweenDatesUsingDatesAndUserID(userID, oneMonthAgo, toNight)
	if err != nil {
		logger.Log.Error("Ollama cache refresh: failed to get exercise days for user " + userID.String() + ". Error: " + err.Error())
		return
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		logger.Log.Error("Ollama cache refresh: failed to convert exercise days for user " + userID.String() + ". Error: " + err.Error())
		return
	}

	// clean up data set for Ollama
	tmp := []models.ExerciseDayObject{}
	for _, exerciseDay := range exerciseDayObjects {
		if len(exerciseDay.Exercises) == 0 {
			continue
		}

		tmp2 := []models.ExerciseObject{}
		for _, exercise := range exerciseDay.Exercises {
			if exercise.Enabled && exercise.IsOn {
				tmp2 = append(tmp2, exercise)
			}
		}

		exerciseDay.Exercises = tmp2

		if len(exerciseDay.Exercises) > 0 {
			tmp = append(tmp, exerciseDay)
		}
	}
	exerciseDayObjects = tmp

	seasons, err := GetOngoingSeasonsFromDBForUserID(now, userID)
	if err != nil {
		logger.Log.Error("Ollama cache refresh: failed to get seasons for user " + userID.String() + ". Error: " + err.Error())
		return
	}

	seasonObjects, err := ConvertSeasonsToSeasonObjects(seasons)
	if err != nil {
		logger.Log.Error("Ollama cache refresh: failed to convert seasons for user " + userID.String() + ". Error: " + err.Error())
		return
	}

	_, err = OllamaGenerateFrontPageMessage(ctx, user, exerciseDayObjects, seasonObjects, now)
	if err != nil {
		if ctx.Err() != nil {
			logger.Log.Debug("Ollama cache refresh: request cancelled for user " + user.FirstName + " " + user.LastName + ".")
		} else {
			logger.Log.Error("Ollama cache refresh: failed to generate message for user " + userID.String() + ". Error: " + err.Error())
		}
		return
	}

	logger.Log.Debug("Ollama cache refresh: cached message for user " + user.FirstName + " " + user.LastName + ".")
}

func OllamaPreCacheForAllUsers() {
	if !files.ConfigFile.Ollama.Enabled {
		return
	}

	users, err := database.GetUsersInformation()
	if err != nil {
		logger.Log.Error("Ollama pre-cache: failed to get users. Error: " + err.Error())
		return
	}

	logger.Log.Info("Ollama pre-cache: warming cache for " + fmt.Sprintf("%d", len(users)) + " users.")

	for _, user := range users {
		OllamaAsyncRefreshCacheForUser(user.ID)
	}

	logger.Log.Info("Ollama pre-cache: finished.")
}
