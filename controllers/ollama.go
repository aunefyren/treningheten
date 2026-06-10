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
	"sort"
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

// ollamaRecentWorkout is a single day the user actually worked out. Provided for
// flavour only; the authoritative weekly count lives in the top-level payload.
type ollamaRecentWorkout struct {
	Date          string `json:"date"`
	DayOfWeek     string `json:"day_of_week"`
	ExerciseCount int    `json:"exercise_count"`
	Note          string `json:"note,omitempty"`
}

// ollamaSeasonPayload is one season the user is currently in. Every field is
// pre-computed so the model never has to derive goal progress, streaks or stakes.
// All values are derived from the single shared weekly workout count.
type ollamaSeasonPayload struct {
	Name                     string `json:"name"`
	WeeklyGoalWorkouts       int    `json:"weekly_goal_workouts"`
	WeeklyGoalMet            bool   `json:"weekly_goal_met"`
	WorkoutsRemaining        int    `json:"workouts_remaining_to_meet_goal"`
	CurrentStreakWeeks       int    `json:"current_streak_weeks"`
	Competing                 bool `json:"competing"`
	PrizeEntriesIfYouMeetGoal int  `json:"prize_entries_to_win_if_a_rival_fails,omitempty"`
	SickleaveUsedThisWeek     bool `json:"sickleave_used_this_week"`
	SickleaveDaysLeft        int    `json:"sickleave_days_left"`
}

type ollamaPromptPayload struct {
	Today                  string                `json:"today"`
	DaysLeftInWeek         int                   `json:"days_left_in_week_including_today"`
	UserFirstName          string                `json:"user_first_name"`
	WorkoutsLoggedThisWeek int                   `json:"workouts_logged_this_week"`
	RecentWorkouts         []ollamaRecentWorkout `json:"recent_workouts"`
	InAnySeason            bool                  `json:"in_any_season"`
	Seasons                []ollamaSeasonPayload `json:"seasons"`
}

// buildOllamaPayload gathers everything the model needs for one user and flattens
// it into pre-computed data points. The model only reads these values; it never
// has to count workouts, filter by week, or reason about season rules.
func buildOllamaPayload(userID uuid.UUID, pointInTime time.Time) (string, error) {
	user, err := database.GetUserInformation(userID)
	if err != nil {
		return "", errors.New("failed to get user: " + err.Error())
	}

	// Shared weekly workout count, scored in the same unit as the goal
	// (number of valid exercises logged this week, Monday-Sunday).
	weekExercises, err := GetExercisesForWeekUsingUserID(pointInTime, userID)
	if err != nil {
		return "", errors.New("failed to get exercises for week: " + err.Error())
	}
	workoutsThisWeek := len(weekExercises)

	// Recent workouts (last 31 days) purely for flavour. Only days actually
	// worked out appear here.
	oneMonthAgo := pointInTime.AddDate(0, 0, -31)
	toNight := utilities.SetClockToMaximum(pointInTime)

	exerciseDays, err := database.GetExerciseDaysBetweenDatesUsingDatesAndUserID(userID, oneMonthAgo, toNight)
	if err != nil {
		return "", errors.New("failed to get exercise days: " + err.Error())
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		return "", errors.New("failed to convert exercise days: " + err.Error())
	}

	recentWorkouts := buildRecentWorkouts(exerciseDayObjects, 5)

	// Seasons the user is currently in, each flattened to its own pre-computed
	// progress and stakes.
	seasons, err := GetOngoingSeasonsFromDBForUserID(pointInTime, userID)
	if err != nil {
		return "", errors.New("failed to get seasons: " + err.Error())
	}

	seasonObjects, err := ConvertSeasonsToSeasonObjects(seasons)
	if err != nil {
		return "", errors.New("failed to convert seasons: " + err.Error())
	}

	seasonPayloads := buildSeasonPayloads(userID, seasonObjects, workoutsThisWeek, pointInTime)

	payload := ollamaPromptPayload{
		Today:                  pointInTime.Format("Monday 2006-01-02"),
		DaysLeftInWeek:         daysLeftInWeekIncludingToday(pointInTime),
		UserFirstName:          user.FirstName,
		WorkoutsLoggedThisWeek: workoutsThisWeek,
		RecentWorkouts:         recentWorkouts,
		InAnySeason:            len(seasonPayloads) > 0,
		Seasons:                seasonPayloads,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// daysLeftInWeekIncludingToday returns how many days remain in the current
// Monday-Sunday week, counting today (Monday=7, Sunday=1).
func daysLeftInWeekIncludingToday(pointInTime time.Time) int {
	weekday := int(pointInTime.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is 7 in ISO week numbering
	}
	return 8 - weekday
}

// buildRecentWorkouts returns the most recent worked-out days (newest first),
// counting only enabled, active exercises.
func buildRecentWorkouts(exerciseDayObjects []models.ExerciseDayObject, limit int) []ollamaRecentWorkout {
	recent := make([]ollamaRecentWorkout, 0, len(exerciseDayObjects))
	for _, day := range exerciseDayObjects {
		count := 0
		for _, exercise := range day.Exercises {
			if exercise.Enabled && exercise.IsOn {
				count++
			}
		}
		if count == 0 {
			continue
		}
		recent = append(recent, ollamaRecentWorkout{
			Date:          day.Date.Format("2006-01-02"),
			DayOfWeek:     day.Date.Format("Monday"),
			ExerciseCount: count,
			Note:          day.Note,
		})
	}

	// Newest first. ISO date strings sort chronologically.
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].Date > recent[j].Date
	})

	if len(recent) > limit {
		recent = recent[:limit]
	}
	return recent
}

// buildSeasonPayloads flattens each active season into pre-computed progress and
// stakes for this user, reusing the same week-result logic the rest of the app
// uses (so streaks, sick leave and wheel tickets stay consistent).
func buildSeasonPayloads(userID uuid.UUID, seasonObjects []models.SeasonObject, workoutsThisWeek int, pointInTime time.Time) []ollamaSeasonPayload {
	monday, err := utilities.FindEarlierMonday(pointInTime)
	if err != nil {
		logger.Log.Info("Ollama payload: failed to find earlier Monday. Error: " + err.Error())
		return []ollamaSeasonPayload{}
	}
	sunday, err := utilities.FindNextSunday(pointInTime)
	if err != nil {
		logger.Log.Info("Ollama payload: failed to find next Sunday. Error: " + err.Error())
		return []ollamaSeasonPayload{}
	}

	seasonPayloads := make([]ollamaSeasonPayload, 0, len(seasonObjects))
	for _, season := range seasonObjects {
		// Find this user's goal in the season (one goal per user per season).
		goalFound := false
		goalInterval := 0
		sickleaveLeft := 0
		for _, goal := range season.Goals {
			if goal.User.ID == userID {
				goalFound = true
				goalInterval = goal.ExerciseInterval
				sickleaveLeft = goal.SickleaveLeft
				break
			}
		}
		if !goalFound {
			continue
		}

		// Reuse the season week-result engine so the streak reflects the whole
		// season history, not just this week.
		weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(monday, sunday, season)
		if err != nil || len(weeks) != 1 {
			logger.Log.Info("Ollama payload: could not resolve current week for season '" + season.Name + "'. Skipping.")
			continue
		}

		var result *models.UserWeekResults
		for i := range weeks[0].UserWeekResults {
			if weeks[0].UserWeekResults[i].UserID == userID {
				result = &weeks[0].UserWeekResults[i]
				break
			}
		}
		if result == nil {
			continue
		}

		remaining := goalInterval - workoutsThisWeek
		if remaining < 0 {
			remaining = 0
		}

		seasonPayload := ollamaSeasonPayload{
			Name:                  season.Name,
			WeeklyGoalWorkouts:    goalInterval,
			WeeklyGoalMet:         result.WeekCompletion >= 1.0,
			WorkoutsRemaining:     remaining,
			CurrentStreakWeeks:    result.CurrentStreak,
			Competing:             result.Competing,
			SickleaveUsedThisWeek: result.SickLeave,
			SickleaveDaysLeft:     sickleaveLeft,
		}

		// Meeting the goal while competing earns streak+1 entries on the wheel.
		// These only pay off if a rival fails and has to spin.
		if result.Competing {
			seasonPayload.PrizeEntriesIfYouMeetGoal = result.CurrentStreak + 1
		}

		seasonPayloads = append(seasonPayloads, seasonPayload)
	}

	return seasonPayloads
}

func OllamaGenerateFrontPageMessage(ctx context.Context, userID uuid.UUID, pointInTime time.Time) (string, error) {
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

	userPayload, err := buildOllamaPayload(userID, pointInTime)
	if err != nil {
		return "", errors.New("failed to build Ollama payload: " + err.Error())
	}

	hash := ollamaPayloadHash(userPayload)
	userKey := userID.String()

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
			Content: `You are a fitness coach for Treningheten, a workout tracking app. You are given a JSON object of already-computed facts about one user. Every number and status in it is final: never recompute, infer, or invent anything that is not in the JSON.

READING THE DATA:
- "workouts_logged_this_week" is the exact number of workouts the user has logged since Monday. Use it as-is; do not count anything yourself.
- "recent_workouts" lists only days the user actually worked out. A day not in the list means no workout. Never claim a workout that is not listed.
- "seasons" lists the competitions the user is in right now. If "in_any_season" is false, the user is in no season: write a purely personal message and do not mention seasons, goals, season streaks, or the wheel.
- Each season is independent with its own goal and streak, all measured against the same shared workout count. When there is more than one season, address them separately by name.

PER-SEASON RULES (read carefully — the wheel is easy to get backwards):
- "weekly_goal_met" says whether this week's goal is already reached; "workouts_remaining_to_meet_goal" is how many more workouts are needed.
- The wheel only applies when "competing" is true. The user spins the wheel ONLY as a penalty for FAILING to meet their goal. Meeting the goal NEVER makes the user spin — do not tell the user to spin when they hit their goal.
- When the user meets their goal, they instead receive "prize_entries_to_win_if_a_rival_fails" entries on the wheel. These pay off only if a DIFFERENT competitor fails their own goal and is forced to spin; that spin might land on the user and win them the prize. It is conditional and not guaranteed — it depends on someone else failing.
- A higher "current_streak_weeks" means more entries and better odds of winning the prize when a rival fails.
- If "competing" is false, the user never spins and is simply participating.

OUTPUT: Write a short front-page greeting, 2 sentences normally, 4 maximum. Reference today's weekday and the user's progress this week, plus season standing when relevant. Friendly tone, light humor welcome. Plain text only: no markdown, no headings, no bullet points, no emojis. Reply in English.`,
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

	ollamaMessage, err := OllamaGenerateFrontPageMessage(context, userID, time.Now())
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

	_, err := OllamaGenerateFrontPageMessage(ctx, userID, time.Now())
	if err != nil {
		if ctx.Err() != nil {
			logger.Log.Debug("Ollama cache refresh: request cancelled for user " + userID.String() + ".")
		} else {
			logger.Log.Error("Ollama cache refresh: failed to generate message for user " + userID.String() + ". Error: " + err.Error())
		}
		return
	}

	logger.Log.Debug("Ollama cache refresh: cached message for user " + userID.String() + ".")
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
