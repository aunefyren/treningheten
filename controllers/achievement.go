package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func APIGetAchivements(context *gin.Context) {

	achivementsArray, err := database.GetAllEnabledAchievements()
	if err != nil {
		log.Println("Failed to get achivements. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get achivements."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Achivements found.", "achivements": achivementsArray})

}

func APIGetPersonalAchivements(context *gin.Context) {

	// Get user ID from URL
	var userIDRequested = context.Param("user_id")

	// Check if the string is empty
	if userIDRequested == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "No user ID found."})
		context.Abort()
		return
	}

	userIDRequestedInt, err := strconv.Atoi(userIDRequested)
	if err != nil {
		log.Println("Invalid user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID."})
		context.Abort()
		return
	}

	achivementsArray, _, err := database.GetDelegatedAchievementsByUserID(int(userIDRequestedInt))
	if err != nil {
		log.Println("Failed to get achivements. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get achivements."})
		context.Abort()
		return
	}

	achivementObjectsArray, err := ConvertAchivementDelegationsToAchivementObjects(achivementsArray)
	if err != nil {
		log.Println("Failed to convert to achivement objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to achivement objects."})
		context.Abort()
		return
	}

	sort.Slice(achivementObjectsArray, func(i, j int) bool {
		return achivementObjectsArray[i].GivenAt.After(achivementObjectsArray[j].GivenAt)
	})

	context.JSON(http.StatusOK, gin.H{"message": "Achivements for user found.", "achivements": achivementObjectsArray})

}

func CheckIfAchivementsExist() (bool, error) {
	found, err := database.CheckIfAchivementsExistinDB()
	return found, err
}

func CreateDefaultAchivements() error {

	achievements := []models.Achievement{}

	leapAchievement := models.Achievement{
		Name:        "One of us",
		Description: "Join a season by creating a goal.",
	}
	achievements = append(achievements, leapAchievement)

	weekAchievement := models.Achievement{
		Name:        "It's everyday, bro",
		Description: "Exercise everyday for a week.",
	}
	achievements = append(achievements, weekAchievement)

	noteAchievement := models.Achievement{
		Name:        "Dear diary...",
		Description: "Write a long workout note.",
	}
	achievements = append(achievements, noteAchievement)

	deserveAchievement := models.Achievement{
		Name:        "What you deserve",
		Description: "Spin the wheel after failing a week.",
	}
	achievements = append(achievements, deserveAchievement)

	winAchievement := models.Achievement{
		Name:        "Lucky bastard",
		Description: "Have someone else spin the wheel and win.",
	}
	achievements = append(achievements, winAchievement)

	anotherAchievement := models.Achievement{
		Name:        "Another one",
		Description: "Exercise more than once in a day.",
	}
	achievements = append(achievements, anotherAchievement)

	overAchievement := models.Achievement{
		Name:        "Overachiever",
		Description: "Exercise more than required in a week.",
	}
	achievements = append(achievements, overAchievement)

	mayAchievement := models.Achievement{
		Name:        "Norwegian heritage",
		Description: "Exercise on the 17th of May.",
	}
	achievements = append(achievements, mayAchievement)

	christmasAchievement := models.Achievement{
		Name:        "The gift of lifting",
		Description: "Exercise on the 24th of December.",
	}
	achievements = append(achievements, christmasAchievement)

	sickAchievement := models.Achievement{
		Name:        "Your week off",
		Description: "Use a week of sick leave.",
	}
	achievements = append(achievements, sickAchievement)

	weekendAchievement := models.Achievement{
		Name:        "I'll do it later",
		Description: "Only exercise during the weekend.",
	}
	achievements = append(achievements, weekendAchievement)

	easyAchievement := models.Achievement{
		Name:        "Making it look easy",
		Description: "Exercise more than seven times a week.",
	}
	achievements = append(achievements, easyAchievement)

	threeAchievement := models.Achievement{
		Name:        "Three weeks",
		Description: "Get a three week streak.",
	}
	achievements = append(achievements, threeAchievement)

	tenAchievement := models.Achievement{
		Name:        "10 weeks",
		Description: "Get a 10 week streak.",
	}
	achievements = append(achievements, tenAchievement)

	fiteenAchievement := models.Achievement{
		Name:        "15 weeks",
		Description: "Get a 15 week streak.",
	}
	achievements = append(achievements, fiteenAchievement)

	completeAchievement := models.Achievement{
		Name:        "Fun run",
		Description: "Complete every week in a season.",
	}
	achievements = append(achievements, completeAchievement)

	for _, achievement := range achievements {

		_, err := database.RegisterAchievementInDB(achievement)
		if err != nil {
			log.Println("Failed to create new achievement. Error: " + err.Error())
			return errors.New("Failed to create new achievement")
		}

	}

	return nil

}

func ConvertAchivementDelegationToAchivementObject(achievementDelegation models.AchievementDelegation) (models.AchievementObject, error) {

	achievement, err := database.GetAchievementByID(achievementDelegation.Achievement)
	if err != nil {
		log.Println("Failed to get achivement. Error: " + err.Error())
		return models.AchievementObject{}, errors.New("Failed to get achivement. Returning...")
	}

	user, err := database.GetUserInformation(achievementDelegation.User)
	if err != nil {
		log.Println("Failed to get user. Error: " + err.Error())
		return models.AchievementObject{}, errors.New("Failed to get user. Returning...")
	}

	achivementObject := models.AchievementObject{
		Name:        achievement.Name,
		Description: achievement.Description,
		ID:          achievement.ID,
		Enabled:     achievement.Enabled,
		GivenAt:     achievementDelegation.GivenAt,
		GivenTo:     user,
	}

	return achivementObject, nil

}

func ConvertAchivementDelegationsToAchivementObjects(achievementDelegations []models.AchievementDelegation) ([]models.AchievementObject, error) {

	achievementObjects := []models.AchievementObject{}

	for _, achievementDelegation := range achievementDelegations {

		achievementObject, err := ConvertAchivementDelegationToAchivementObject(achievementDelegation)
		if err != nil {
			log.Println("Failed to convert achievement delegation to achievement object. Skipping...")
			continue
		}

		achievementObjects = append(achievementObjects, achievementObject)

	}

	return achievementObjects, nil

}

func GiveUserAnAchivement(userID int, achivementID int, achivementTime time.Time) error {

	_, found, err := database.GetAchievementDelegationByAchivementIDAndUserID(userID, achivementID)
	if err != nil {
		log.Println("Failed to check achivement delegation. Error: " + err.Error())
		return errors.New("Failed to check achivement delegation.")
	} else if found {
		return errors.New("User already has achivement.")
	}

	delegation := models.AchievementDelegation{
		User:        userID,
		Achievement: achivementID,
		GivenAt:     achivementTime,
	}

	_, err = database.RegisterAchievementDelegationInDB(delegation)
	if err != nil {
		log.Println("Failed to give achivement. Error: " + err.Error())
		return errors.New("Failed to give achivement.")
	}

	return nil

}

func GenerateAchivementsForWeek(weekResults models.WeekResults) error {

	sundayDate, err := utilities.FindNextSunday(weekResults.WeekDate)
	if err != nil {
		log.Println("Failed to find next Sunday. Error: " + err.Error())
		return errors.New("Failed to find next Sunday.")
	}

	for _, user := range weekResults.UserWeekResults {

		if user.WeekCompletion > 1.0 {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 7, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

		if user.CurrentStreak >= 2 && user.WeekCompletion >= 1 {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 13, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

		if user.CurrentStreak >= 9 && user.WeekCompletion >= 1 {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 14, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

		if user.CurrentStreak >= 14 && user.WeekCompletion >= 1 {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 15, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

		week, err := GetExercisesForWeekUsingGoal(weekResults.WeekDate, user.Goal)
		if err != nil {
			log.Println("Failed to get week exercises for user '" + strconv.Itoa(int(user.User.ID)) + "'. Returning. Error: " + err.Error())
			return errors.New("Failed to get week exercises for user.")
		}

		everyday := true
		weekday := false
		weekend := false
		exerciseSum := 0

		for _, day := range week.Days {

			dayDate := day.Date.Day()
			dayMonth := day.Date.Month()

			if dayDate == 17 && dayMonth == 5 && day.ExerciseInterval > 0 {

				// Give achivement to user
				err := GiveUserAnAchivement(int(user.User.ID), 8, day.Date)
				if err != nil {
					log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
				}

			}

			if dayDate == 24 && dayMonth == 12 && day.ExerciseInterval > 0 {

				// Give achivement to user
				err := GiveUserAnAchivement(int(user.User.ID), 9, day.Date)
				if err != nil {
					log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
				}

			}

			if len(day.Note) > 59 {

				// Give achivement to user
				err := GiveUserAnAchivement(int(user.User.ID), 3, day.Date)
				if err != nil {
					log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.ExerciseInterval > 1 {

				// Give achivement to user
				err := GiveUserAnAchivement(int(user.User.ID), 6, day.Date)
				if err != nil {
					log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.ExerciseInterval == 0 {
				everyday = false
			}

			if day.Date.Weekday() > 0 && day.Date.Weekday() < 6 && day.ExerciseInterval > 1 {
				weekday = true
			}

			if (day.Date.Weekday() == 0 || day.Date.Weekday() == 6) && day.ExerciseInterval > 1 {
				weekend = true
			}

			exerciseSum += day.ExerciseInterval

		}

		if everyday {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 2, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

		if !weekday && weekend {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 11, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

		if exerciseSum > 7 {

			// Give achivement to user
			err := GiveUserAnAchivement(int(user.User.ID), 12, sundayDate)
			if err != nil {
				log.Println("Failed to give achivement for user '" + strconv.Itoa(int(user.User.ID)) + "'. Ignoring. Error: " + err.Error())
			}

		}

	}

	return nil

}

func GenerateAchivementsForSeason(seasonResults []models.WeekResults) error {

	if len(seasonResults) == 0 {
		return errors.New("Empty season, returning...")
	}

	winningUsers := []int{}

	for _, weekResults := range seasonResults {

		for _, weekResult := range weekResults.UserWeekResults {

			if weekResult.WeekCompletion >= 1 {

				found := false
				for _, userID := range winningUsers {

					if userID == int(weekResult.User.ID) {
						found = true
						break
					}

				}

				if !found {
					winningUsers = append(winningUsers, int(weekResult.User.ID))
				}

			} else {

				winningUsers = utilities.RemoveIntFromArray(winningUsers, int(weekResult.User.ID))

			}

		}

	}

	seasonSunday, err := utilities.FindNextSunday(seasonResults[0].WeekDate)
	if err != nil {
		return errors.New("Failed to find Sunday for last week of the season.")
	}

	for _, userID := range winningUsers {

		// Give achivement to user
		err := GiveUserAnAchivement(userID, 16, seasonSunday)
		if err != nil {
			log.Println("Failed to give achivement for user '" + strconv.Itoa(userID) + "'. Ignoring. Error: " + err.Error())
		}

	}

	return nil

}
