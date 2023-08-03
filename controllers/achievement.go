package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"errors"
	"log"
	"net/http"
	"strconv"

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

	context.JSON(http.StatusOK, gin.H{"message": "Achivements for user found.", "achivements": achivementObjectsArray})

}

func CheckIfAchivementsExist() (bool, error) {
	found, err := database.CheckIfAchivementsExistinDB()
	return found, err
}

func CreateDefaultAchivements() error {

	achievements := []models.Achievement{}

	leapAchievement := models.Achievement{
		Name:        "New year, new me",
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
		Description: "Spin the wheel after losing a week.",
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
		Name:        "Averachiever",
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
		Description: "Use a day of sick leave.",
	}
	achievements = append(achievements, sickAchievement)

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
		GivenAt:     achievementDelegation.CreatedAt,
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

func GiveUserAnAchivement(userID int, achivementID int) error {

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
	}

	_, err = database.RegisterAchievementDelegationInDB(delegation)
	if err != nil {
		log.Println("Failed to give achivement. Error: " + err.Error())
		return errors.New("Failed to give achivement.")
	}

	return nil

}
