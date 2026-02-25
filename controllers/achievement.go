package controllers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func APIGetAchievements(context *gin.Context) {
	var achievementObjectsArray = []models.AchievementUserObject{}
	var requestedUser *uuid.UUID
	requestedUser = nil

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	user, okay := context.GetQuery("user")
	if !okay {
		achievementsArray, err := database.GetAllEnabledAchievements()
		if err != nil {
			logger.Log.Info("Failed to get achievements. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get achievements."})
			context.Abort()
			return
		}

		achievementObjectsArray, err = ConvertAchievementsToAchievementUserObjects(achievementsArray, &userID)
		if err != nil {
			logger.Log.Info("Failed to convert to achievement objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to achievement objects."})
			context.Abort()
			return
		}
	} else {
		requestedUserNew, err := uuid.Parse(user)
		if err != nil {
			logger.Log.Info("Failed to parse user ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
			context.Abort()
			return
		}
		requestedUser = &requestedUserNew

		achievementsArray, _, err := database.GetDistinctDelegatedAchievementsByUserID(*requestedUser)
		if err != nil {
			logger.Log.Info("Failed to get achievements. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get achievements."})
			context.Abort()
			return
		}

		achievementObjectsArray, err = ConvertAchievementDelegationsToAchievementUserObjects(achievementsArray, &userID)
		if err != nil {
			logger.Log.Info("Failed to convert to achievement objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to achievement objects."})
			context.Abort()
			return
		}

		sort.Slice(achievementObjectsArray, func(i, j int) bool {
			return achievementObjectsArray[i].LastGivenAt.After(*achievementObjectsArray[j].LastGivenAt)
		})
	}

	context.JSON(http.StatusOK, gin.H{"message": "Achievements found.", "achievements": achievementObjectsArray})

	// Mark all achievements as seen
	if requestedUser != nil && userID == *requestedUser {
		_, err = database.SetAchievementsToSeenForUser(userID)
		if err != nil {
			logger.Log.Info("Failed to set achievements to seen for user. Error: " + err.Error())
		}
	}
}

func CheckIfAchievementsExist() (bool, error) {
	found, err := database.CheckIfAchievementsExistsInDB()
	return found, err
}

func ValidateAchievements() error {
	achievements := []models.Achievement{}
	trueVariable := true
	falseVariable := false

	leapAchievement := models.Achievement{
		Name:          "One of us",
		Description:   "Join a season by creating a goal.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	leapAchievement.ID = uuid.MustParse("7f2d49ad-d056-415e-aa80-0ada6db7cc00")
	leapAchievement.MultipleDelegations = &falseVariable
	achievements = append(achievements, leapAchievement)

	weekAchievement := models.Achievement{
		Name:          "It's everyday, bro",
		Description:   "Exercise everyday for a week.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	weekAchievement.ID = uuid.MustParse("a8c62293-6090-4b16-a070-ad65404836ae")
	weekAchievement.MultipleDelegations = &trueVariable
	weekAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, weekAchievement)

	noteAchievement := models.Achievement{
		Name:          "Dear diary...",
		Description:   "Write a long workout note.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	noteAchievement.ID = uuid.MustParse("ae27d8bf-dfc8-4be1-b7a9-01183b375ebf")
	noteAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, noteAchievement)

	deserveAchievement := models.Achievement{
		Name:          "What you deserve",
		Description:   "Spin the wheel after failing a week.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	deserveAchievement.ID = uuid.MustParse("d415fffc-ea99-4b27-8929-aeb02ae44da3")
	deserveAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, deserveAchievement)

	winAchievement := models.Achievement{
		Name:          "Lucky bastard",
		Description:   "Have someone else spin the wheel and win.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	winAchievement.ID = uuid.MustParse("bb964360-6413-47c2-8400-ee87b40365a7")
	winAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, winAchievement)

	anotherAchievement := models.Achievement{
		Name:             "Another one",
		Description:      "Exercise more than once in a day.",
		Category:         "Default",
		CategoryColor:    "lightblue",
		AchievementOrder: 11,
	}
	anotherAchievement.ID = uuid.MustParse("51c48b42-4429-4b82-8fb2-d2bb2bfe907a")
	anotherAchievement.MultipleDelegations = &trueVariable
	anotherAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, anotherAchievement)

	overAchievement := models.Achievement{
		Name:          "Overachiever",
		Description:   "Exercise more than required in a week.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	overAchievement.ID = uuid.MustParse("f7fad558-3e59-4812-9b13-4c30a91c04b9")
	overAchievement.MultipleDelegations = &trueVariable
	overAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, overAchievement)

	mayAchievement := models.Achievement{
		Name:          "Norwegian heritage",
		Description:   "Exercise on the 17th of May.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	mayAchievement.ID = uuid.MustParse("ab0b1bf0-c57b-469f-a6ba-5d195f1b896d")
	mayAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, mayAchievement)

	christmasAchievement := models.Achievement{
		Name:             "The gift of lifting",
		Description:      "Exercise on the 24th of December.",
		Category:         "Christmas",
		CategoryColor:    "lightgreen",
		AchievementOrder: 5,
	}
	christmasAchievement.ID = uuid.MustParse("c4a131a6-2aa6-49fb-98e5-fa797152a9a4")
	christmasAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, christmasAchievement)

	sickAchievement := models.Achievement{
		Name:          "Your week off",
		Description:   "Use a week of sick leave.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	sickAchievement.ID = uuid.MustParse("420b020c-2cad-4898-bb94-d86dc0031203")
	sickAchievement.MultipleDelegations = &trueVariable
	sickAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, sickAchievement)

	weekendAchievement := models.Achievement{
		Name:          "I'll do it later",
		Description:   "Only exercise during the weekend.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	weekendAchievement.ID = uuid.MustParse("31fa2681-eec7-43e4-bc69-35dee352eaee")
	weekendAchievement.MultipleDelegations = &trueVariable
	weekendAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, weekendAchievement)

	easyAchievement := models.Achievement{
		Name:          "Making it look easy",
		Description:   "Exercise more than seven times in a week.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	easyAchievement.ID = uuid.MustParse("e7ee36d4-f39e-40a3-af92-2f7e1f707d07")
	easyAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, easyAchievement)

	threeAchievement := models.Achievement{
		Name:             "Three weeks",
		Description:      "Get a three week streak.",
		Category:         "Default",
		CategoryColor:    "lightblue",
		AchievementOrder: 5,
	}
	threeAchievement.ID = uuid.MustParse("8875597e-d8f5-4514-b96f-c51ecce4eb1f")
	threeAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, threeAchievement)

	tenAchievement := models.Achievement{
		Name:             "10 weeks",
		Description:      "Get a 10 week streak.",
		Category:         "Default",
		CategoryColor:    "lightblue",
		AchievementOrder: 6,
	}
	tenAchievement.ID = uuid.MustParse("ca6a4692-153b-47a7-8444-457b906d0666")
	tenAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, tenAchievement)

	fifteenAchievement := models.Achievement{
		Name:             "15 weeks",
		Description:      "Get a 15 week streak.",
		Category:         "Default",
		CategoryColor:    "lightblue",
		AchievementOrder: 7,
	}
	fifteenAchievement.ID = uuid.MustParse("2a84df89-9976-443b-a093-19f8d73b5eff")
	fifteenAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, fifteenAchievement)

	twentyAchievement := models.Achievement{
		Name:             "20 weeks",
		Description:      "Get a 20 week streak.",
		Category:         "Default",
		CategoryColor:    "lightblue",
		AchievementOrder: 8,
	}
	twentyAchievement.ID = uuid.MustParse("09da2ab1-393d-4c43-a1d0-daa45520b49f")
	twentyAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, twentyAchievement)

	completeAchievement := models.Achievement{
		Name:             "Fun run",
		Description:      "Complete every week in a season.",
		Category:         "Season",
		CategoryColor:    "yellow-light",
		AchievementOrder: 9,
	}
	completeAchievement.ID = uuid.MustParse("01dc9c4b-cf65-4d3c-9596-1417b67bd86f")
	completeAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, completeAchievement)

	comebackAchievement := models.Achievement{
		Name:          "Underdog",
		Description:   "Complete a week after failing two in a row.",
		Category:      "Season",
		CategoryColor: "yellow-light",
	}
	comebackAchievement.ID = uuid.MustParse("38524a0a-f0b6-4cbf-b221-05ebfa0797f7")
	comebackAchievement.MultipleDelegations = &trueVariable
	comebackAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, comebackAchievement)

	deadAchievement := models.Achievement{
		Name:          "Back from the dead",
		Description:   "Complete a week after using sick leave.",
		Category:      "Season",
		CategoryColor: "yellow-light",
	}
	deadAchievement.ID = uuid.MustParse("b342cd1b-1812-4384-967f-51d2be772eab")
	deadAchievement.MultipleDelegations = &trueVariable
	deadAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, deadAchievement)

	fullAchievement := models.Achievement{
		Name:             "The boyband",
		Description:      "Exercise three times in a day.",
		Category:         "Default",
		CategoryColor:    "lightblue",
		AchievementOrder: 12,
	}
	fullAchievement.ID = uuid.MustParse("c92178b4-753a-4624-a7f6-ae5afd0a9ca3")
	fullAchievement.MultipleDelegations = &trueVariable
	fullAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, fullAchievement)

	photoAchievement := models.Achievement{
		Name:          "Looking good",
		Description:   "Change your profile photo.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	photoAchievement.ID = uuid.MustParse("05a3579f-aa8d-4814-b28f-5824a2d904ec")
	photoAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, photoAchievement)

	treatyoselfAchievement := models.Achievement{
		Name:          "Treat yo self",
		Description:   "Exercise on your birthday.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	treatyoselfAchievement.ID = uuid.MustParse("5e0f5605-b3e5-4350-a408-1c9f5b5a99a4")
	treatyoselfAchievement.MultipleDelegations = &trueVariable
	treatyoselfAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, treatyoselfAchievement)

	shameAchievement := models.Achievement{
		Name:          "Badge of shame",
		Description:   "Forget to log your workouts and have it fixed later.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	shameAchievement.ID = uuid.MustParse("96cf246b-5d16-4fc8-8887-d95815a89683")
	shameAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, shameAchievement)

	immuneAchievement := models.Achievement{
		Name:             "Superior immune system",
		Description:      "Don't use any sick leave throughout a season.",
		Category:         "Season",
		CategoryColor:    "yellow-light",
		AchievementOrder: 10,
	}
	immuneAchievement.ID = uuid.MustParse("b566e486-d476-40f1-a9f2-28035bb43f37")
	immuneAchievement.MultipleDelegations = &trueVariable
	immuneAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, immuneAchievement)

	firstAdventAchievement := models.Achievement{
		Name:             "Advent (1)",
		Description:      "Exercise on the first Sunday in Advent.",
		Category:         "Christmas",
		CategoryColor:    "lightgreen",
		AchievementOrder: 1,
	}
	firstAdventAchievement.ID = uuid.MustParse("5276382c-fdae-410b-a298-5107a3ff3089")
	firstAdventAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, firstAdventAchievement)

	secondAdventAchievement := models.Achievement{
		Name:             "Advent (2)",
		Description:      "Exercise on the second Sunday in Advent.",
		Category:         "Christmas",
		CategoryColor:    "lightgreen",
		AchievementOrder: 2,
	}
	secondAdventAchievement.ID = uuid.MustParse("6c991ba6-d0ae-4022-9410-6558e376ec5e")
	secondAdventAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, secondAdventAchievement)

	thirdAdventAchievement := models.Achievement{
		Name:             "Advent (3)",
		Description:      "Exercise on the third Sunday in Advent.",
		Category:         "Christmas",
		CategoryColor:    "lightgreen",
		AchievementOrder: 3,
	}
	thirdAdventAchievement.ID = uuid.MustParse("7ef923b5-21aa-4478-a658-68078f499620")
	thirdAdventAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, thirdAdventAchievement)

	fourthAdventAchievement := models.Achievement{
		Name:             "Advent (4)",
		Description:      "Exercise on the last Sunday in Advent.",
		Category:         "Christmas",
		CategoryColor:    "lightgreen",
		AchievementOrder: 4,
	}
	fourthAdventAchievement.ID = uuid.MustParse("720b036c-7d24-418f-88e6-a0e84147efda")
	fourthAdventAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, fourthAdventAchievement)

	mondaysAchievement := models.Achievement{
		Name:          "I hate Mondays",
		Description:   "Exercise on a Monday.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	mondaysAchievement.ID = uuid.MustParse("47f04b1f-4e19-40fe-ace3-3afa18378751")
	mondaysAchievement.MultipleDelegations = &trueVariable
	mondaysAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, mondaysAchievement)

	drewBloodAchievement := models.Achievement{
		Name:          "They drew first blood",
		Description:   "Be the only victor in a week.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	drewBloodAchievement.ID = uuid.MustParse("6cc0f1b0-c894-4b12-a9ed-569cdfde3b16")
	drewBloodAchievement.MultipleDelegations = &trueVariable
	achievements = append(achievements, drewBloodAchievement)

	creeperAchievement := models.Achievement{
		Name:          "Creeper",
		Description:   "Visit another user's profile.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	creeperAchievement.ID = uuid.MustParse("cbd81cd0-4caf-438b-989b-b5ca7e76605d")
	creeperAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, creeperAchievement)

	stravaAchievement := models.Achievement{
		Name:          "Influencer",
		Description:   "Add a Strava connection to Treningheten.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	stravaAchievement.ID = uuid.MustParse("fb4f6c1f-dfad-4df7-8007-4cfd6f351b17")
	stravaAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, stravaAchievement)

	historianAchievement := models.Achievement{
		Name:          "Historian",
		Description:   "Add details to a workout.",
		Category:      "Default",
		CategoryColor: "lightblue",
	}
	historianAchievement.ID = uuid.MustParse("3d745d3a-b4b8-4194-bc72-653cfe4c351b")
	historianAchievement.HiddenDescription = &trueVariable
	achievements = append(achievements, historianAchievement)

	achievementsOld, err := database.GetAllAchievements()
	if err != nil {
		logger.Log.Info("Failed to get all achievements. Error: " + err.Error())
		return errors.New("Failed to get all achievements.")
	}

	achievementsFinal, err := MergeAchievements(achievementsOld, achievements)
	if err != nil {
		logger.Log.Info("Failed to create/update achievement. Error: " + err.Error())
		return errors.New("Failed to merge achievement lists.")
	}

	for _, achievement := range achievementsFinal {
		_, err := database.SaveAchievementInDB(achievement)
		if err != nil {
			logger.Log.Info("Failed to create/update achievement. Error: " + err.Error())
			return errors.New("Failed to create/update achievement.")
		}
	}

	return nil
}

func MergeAchievements(primaryAchievements []models.Achievement, otherAchievements []models.Achievement) ([]models.Achievement, error) {
	finalAchievements := []models.Achievement{}

	for _, newAchievement := range otherAchievements {
		found := false

		for _, oldAchievement := range primaryAchievements {
			if oldAchievement.ID == newAchievement.ID {
				found = true

				oldAchievement.AchievementOrder = newAchievement.AchievementOrder
				oldAchievement.Category = newAchievement.Category
				oldAchievement.CategoryColor = newAchievement.CategoryColor
				oldAchievement.Description = newAchievement.Description
				oldAchievement.MultipleDelegations = newAchievement.MultipleDelegations
				oldAchievement.Name = newAchievement.Name
				oldAchievement.HiddenDescription = newAchievement.HiddenDescription

				finalAchievements = append(finalAchievements, oldAchievement)
			}
		}

		if !found {
			finalAchievements = append(finalAchievements, newAchievement)
			logger.Log.Info("Added achievement '" + newAchievement.Name + "'.")
		}
	}

	for _, oldAchievement := range primaryAchievements {
		found := false

		for _, newAchievement := range otherAchievements {
			if newAchievement.ID == oldAchievement.ID {
				found = true
			}
		}

		if !found {
			finalAchievements = append(finalAchievements, oldAchievement)
		}
	}

	return finalAchievements, nil
}

func ConvertAchievementDelegationToAchievementUserObject(achievementDelegation models.AchievementDelegation, userChecking *uuid.UUID) (models.AchievementUserObject, error) {
	achievement, err := database.GetAchievementByID(achievementDelegation.AchievementID)
	if err != nil {
		logger.Log.Info("Failed to get achievement. Error: " + err.Error())
		return models.AchievementUserObject{}, errors.New("Failed to get achievement. Returning...")
	}

	user, err := database.GetUserInformation(achievementDelegation.UserID)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		return models.AchievementUserObject{}, errors.New("Failed to get user. Returning...")
	}

	achievementDelegations, err := database.GetAchievementDelegationByAchievementIDAndUserID(achievementDelegation.UserID, achievementDelegation.AchievementID)
	if err != nil {
		logger.Log.Info("Failed to get achievement delegations. Error: " + err.Error())
		return models.AchievementUserObject{}, errors.New("Failed to get achievement delegations. Returning...")
	}

	achievementObject := models.AchievementUserObject{
		Name:                achievement.Name,
		Description:         achievement.Description,
		Enabled:             achievement.Enabled,
		Category:            achievement.Category,
		CategoryColor:       achievement.CategoryColor,
		AchievementOrder:    achievement.AchievementOrder,
		MultipleDelegations: achievement.MultipleDelegations,
		LastGivenAt:         nil,
	}
	achievementObject.ID = achievement.ID

	for i := 0; i < len(achievementDelegations); i++ {
		achievementDelegations[i].User = user

		if achievementObject.LastGivenAt == nil || achievementDelegations[i].GivenAt.After(*achievementObject.LastGivenAt) {
			achievementObject.LastGivenAt = &achievementDelegations[i].GivenAt
		}
	}

	achievedByUserChecking := false
	if userChecking != nil {
		achievedByUser, err := database.VerifyIfAchievedByUser(achievement.ID, *userChecking)
		achievedByUserChecking = achievedByUser
		if err != nil {
			logger.Log.Info("Failed to verify if user has achievement. Error: " + err.Error())
		}

		if achievement.ID.String() == "cbd81cd0-4caf-438b-989b-b5ca7e76605d" {
			logger.Log.Info(achievedByUserChecking)
		}
	}

	if achievement.HiddenDescription != nil && *achievement.HiddenDescription && !achievedByUserChecking {
		achievementObject.Description = "Hidden"
	}

	// Add all delegations if multiple delegations is active
	if achievement.MultipleDelegations != nil && *achievement.MultipleDelegations == true {
		achievementObject.AchievementDelegation = &achievementDelegations
	} else {
		// if not, find oldest and replace given date
		oldestDelegation := models.AchievementDelegation{}

		for i := 0; i < len(achievementDelegations); i++ {
			achievementObject.LastGivenAt = nil
			if achievementObject.LastGivenAt == nil || achievementDelegations[i].GivenAt.Before(*achievementObject.LastGivenAt) {
				achievementObject.LastGivenAt = &achievementDelegations[i].GivenAt
				oldestDelegation = achievementDelegations[i]
			}
		}

		oldestDelegationArray := []models.AchievementDelegation{}
		oldestDelegationArray = append(oldestDelegationArray, oldestDelegation)
		achievementObject.AchievementDelegation = &oldestDelegationArray
	}

	return achievementObject, nil
}

func ConvertAchievementDelegationsToAchievementUserObjects(achievementDelegations []models.AchievementDelegation, userChecking *uuid.UUID) ([]models.AchievementUserObject, error) {
	achievementObjects := []models.AchievementUserObject{}

	for _, achievementDelegation := range achievementDelegations {

		achievementObject, err := ConvertAchievementDelegationToAchievementUserObject(achievementDelegation, userChecking)
		if err != nil {
			logger.Log.Info("Failed to convert achievement delegation to achievement object. Skipping...")
			continue
		}

		achievementObjects = append(achievementObjects, achievementObject)

	}

	return achievementObjects, nil
}

func ConvertAchievementToAchievementUserObject(achievement models.Achievement, userChecking *uuid.UUID) (models.AchievementUserObject, error) {
	achievementObject := models.AchievementUserObject{
		Name:                  achievement.Name,
		Description:           achievement.Description,
		Enabled:               achievement.Enabled,
		Category:              achievement.Category,
		CategoryColor:         achievement.CategoryColor,
		AchievementOrder:      achievement.AchievementOrder,
		MultipleDelegations:   achievement.MultipleDelegations,
		AchievementDelegation: nil,
	}
	achievementObject.ID = achievement.ID

	achievedByUserChecking := false
	if userChecking != nil {
		achievedByUser, err := database.VerifyIfAchievedByUser(achievement.ID, *userChecking)
		achievedByUserChecking = achievedByUser
		if err != nil {
			logger.Log.Info("Failed to verify if user has achievement. Error: " + err.Error())
		}

		if achievement.ID.String() == "cbd81cd0-4caf-438b-989b-b5ca7e76605d" {
			logger.Log.Info(achievedByUserChecking)
		}
	}

	if achievement.HiddenDescription != nil && *achievement.HiddenDescription && !achievedByUserChecking {
		achievementObject.Description = "Hidden"
	}

	return achievementObject, nil
}

func ConvertAchievementsToAchievementUserObjects(achievements []models.Achievement, userChecking *uuid.UUID) ([]models.AchievementUserObject, error) {
	achievementObjects := []models.AchievementUserObject{}

	for _, achievement := range achievements {

		achievementObject, err := ConvertAchievementToAchievementUserObject(achievement, userChecking)
		if err != nil {
			logger.Log.Info("Failed to convert achievement delegation to achievement object. Skipping...")
			continue
		}

		achievementObjects = append(achievementObjects, achievementObject)

	}

	return achievementObjects, nil
}

func GiveUserAnAchievement(userID uuid.UUID, achievementID uuid.UUID, achievementTime time.Time, optionalDelaySeconds int) error {
	achievement, err := database.GetAchievementByID(achievementID)
	if err != nil {
		logger.Log.Info("Failed to get achievement. Error: " + err.Error())
		return errors.New("Failed to get achievement.")
	}

	achievementDelegations, err := database.GetAchievementDelegationByAchievementIDAndUserID(userID, achievementID)
	if err != nil {
		logger.Log.Info("Failed to check achievement delegation. Error: " + err.Error())
		return errors.New("Failed to check achievement delegation.")
	} else if len(achievementDelegations) > 0 && (achievement.MultipleDelegations == nil || !*achievement.MultipleDelegations) {
		logger.Log.Debug("User already has achievement.")
		return nil
	}

	delegation := models.AchievementDelegation{
		UserID:        userID,
		AchievementID: achievementID,
		GivenAt:       achievementTime,
	}
	delegation.ID = uuid.New()

	_, err = database.RegisterAchievementDelegationInDB(delegation)
	if err != nil {
		logger.Log.Info("Failed to give achievement. Error: " + err.Error())
		return errors.New("Failed to give achievement.")
	}

	time.Sleep(time.Second * time.Duration(optionalDelaySeconds))

	err = PushNotificationsForAchievements(userID)
	if err != nil {
		logger.Log.Info("Failed to give achievement notification. Error: " + err.Error())
	}

	return nil
}

func GenerateAchievementsForWeek(weekResults models.WeekResults, targetUser *uuid.UUID) error {
	sundayDate, err := utilities.FindNextSunday(weekResults.WeekDate)
	if err != nil {
		logger.Log.Error("Failed to find next Sunday. Error: " + err.Error())
		return errors.New("Failed to find next Sunday.")
	}

	winnerUserIDs := []uuid.UUID{}
	loserUserIDs := []uuid.UUID{}

	for _, user := range weekResults.UserWeekResults {
		userObject, err := database.GetUserInformation(user.UserID)
		if err != nil {
			logger.Log.Error("Failed to get user object. Error: " + err.Error())
			return errors.New("Failed to get user object.")
		}

		// If a winner for the week
		if user.WeekCompletion >= 1.0 && !user.SickLeave {
			winnerUserIDs = append(winnerUserIDs, user.UserID)
		} else if user.WeekCompletion < 1.0 && !user.SickLeave {
			loserUserIDs = append(loserUserIDs, user.UserID)
		}

		// calculate new streak
		userStreak := user.CurrentStreak
		if user.FullWeekParticipation && user.WeekCompletion < 1 && !user.SickLeave {
			userStreak = 0
		} else if user.WeekCompletion >= 1 {
			userStreak++
		} else {
			userStreak = 0
		}

		// skip the rest if applicable
		if targetUser != nil && user.UserID != *targetUser {
			continue
		}

		if user.WeekCompletion > 1.0 && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("f7fad558-3e59-4812-9b13-4c30a91c04b9"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}

		if userStreak == 3 && user.WeekCompletion >= 1 && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user for three weeks
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("8875597e-d8f5-4514-b96f-c51ecce4eb1f"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}

		if userStreak == 10 && user.WeekCompletion >= 1 && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user for ten weeks
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("ca6a4692-153b-47a7-8444-457b906d0666"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}

		if userStreak == 15 && user.WeekCompletion >= 1 && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user for 15 weeks
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("2a84df89-9976-443b-a093-19f8d73b5eff"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}

		if userStreak == 20 && user.WeekCompletion >= 1 && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user for 20 weeks
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("09da2ab1-393d-4c43-a1d0-daa45520b49f"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		} else {
			logger.Log.Trace("20 weeks achievement not given. " + strconv.Itoa(userStreak) + " " + strconv.FormatFloat(user.WeekCompletion, 'f', -1, 64))
		}

		week, err := GetExerciseDaysForWeekUsingUserID(weekResults.WeekDate, user.UserID)
		if err != nil {
			logger.Log.Warn("Failed to get week exercises for user '" + user.UserID.String() + "'. Returning. Error: " + err.Error())
			return errors.New("Failed to get week exercises for user.")
		}

		everyday := true
		weekday := false
		weekend := false
		exerciseSum := 0
		now := time.Now()

		for _, day := range week.Days {
			logger.Log.Trace("doing day from week: " + day.Date.String())

			// ensure time is midnight
			day.Date = utilities.SetClockToMinimum(day.Date)

			// get date values for calculation
			dayDate := day.Date.Day()
			dayMonth := day.Date.Month()
			dayWeekday := day.Date.Weekday()

			// get different dates for calculations
			christmasDate := time.Date(now.Year(), 12, 24, 0, 0, 0, 0, time.Local)
			logger.Log.Trace("Christmas date: " + christmasDate.String())

			dayLastSundayAdvent, err := utilities.FindEarlierSunday(christmasDate)
			if err != nil {
				logger.Log.Error("Failed to get previous sunday for date. Error: " + err.Error())
				return errors.New("Failed to get previous sunday for date.")
			}
			logger.Log.Trace("Advent (4) date: " + dayLastSundayAdvent.String())

			dayThirdSundayAdvent := dayLastSundayAdvent.AddDate(0, 0, -7)
			logger.Log.Trace("Advent (3) date: " + dayThirdSundayAdvent.String())

			daySecondSundayAdvent := dayLastSundayAdvent.AddDate(0, 0, -14)
			logger.Log.Trace("Advent (2) date: " + daySecondSundayAdvent.String())

			dayFirstSundayAdvent := dayLastSundayAdvent.AddDate(0, 0, -21)
			logger.Log.Trace("Advent (1) date: " + dayFirstSundayAdvent.String())

			if dayDate == 17 && dayMonth == 5 && day.ExerciseInterval > 0 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for 17. of may
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("ab0b1bf0-c57b-469f-a6ba-5d195f1b896d"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.Date.Equal(dayFirstSundayAdvent) && day.ExerciseInterval > 0 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for first advent
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("5276382c-fdae-410b-a298-5107a3ff3089"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.Date.Equal(daySecondSundayAdvent) && day.ExerciseInterval > 0 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for second advent
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("6c991ba6-d0ae-4022-9410-6558e376ec5e"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.Date.Equal(dayThirdSundayAdvent) && day.ExerciseInterval > 0 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for third advent
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("7ef923b5-21aa-4478-a658-68078f499620"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.Date.Equal(dayLastSundayAdvent) && day.ExerciseInterval > 0 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for last advent
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("720b036c-7d24-418f-88e6-a0e84147efda"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if dayDate == 24 && dayMonth == 12 && day.ExerciseInterval > 0 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for 24 dec
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("c4a131a6-2aa6-49fb-98e5-fa797152a9a4"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if userObject.BirthDate != nil &&
				(dayDate == userObject.BirthDate.Day() &&
					dayMonth == userObject.BirthDate.Month() &&
					day.ExerciseInterval > 0) &&
				(targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("5e0f5605-b3e5-4350-a408-1c9f5b5a99a4"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if len(day.Note) > 59 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user for long note
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("ae27d8bf-dfc8-4be1-b7a9-01183b375ebf"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.ExerciseInterval > 1 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("51c48b42-4429-4b82-8fb2-d2bb2bfe907a"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			if day.ExerciseInterval == 3 && (targetUser == nil || *targetUser == user.UserID) {

				// Give achievement to user
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("c92178b4-753a-4624-a7f6-ae5afd0a9ca3"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}

			}

			// If Monday, give Monday achievement
			if dayWeekday == 1 && day.ExerciseInterval >= 1 && (targetUser == nil || *targetUser == user.UserID) {
				// Give achievement to user
				err := GiveUserAnAchievement(user.UserID, uuid.MustParse("47f04b1f-4e19-40fe-ace3-3afa18378751"), day.Date, 0)
				if err != nil {
					logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
				}
			}

			if day.ExerciseInterval == 0 {
				everyday = false
			}

			if dayWeekday > 0 && dayWeekday < 6 && day.ExerciseInterval > 0 {
				weekday = true
			}

			if (dayWeekday == 0 || dayWeekday == 6) && day.ExerciseInterval > 0 {
				weekend = true
			}

			exerciseSum += day.ExerciseInterval

		}

		if everyday && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user for exercising everyday
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("a8c62293-6090-4b16-a070-ad65404836ae"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}

		// If exercise occurred on a weekend, and not a weekday
		if !weekday && weekend && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user for only exercising on the weekend
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("31fa2681-eec7-43e4-bc69-35dee352eaee"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}

		// If the sum of exercise is more than 7
		if exerciseSum > 7 && (targetUser == nil || *targetUser == user.UserID) {

			// Give achievement to user
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("e7ee36d4-f39e-40a3-af92-2f7e1f707d07"), sundayDate, 0)
			if err != nil {
				logger.Log.Warn("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}

		}
	}

	// If there is one winner, give achievement
	if len(winnerUserIDs) == 1 && len(loserUserIDs) > 0 && (targetUser == nil || *targetUser == winnerUserIDs[0]) {
		// Give achievement to user
		err := GiveUserAnAchievement(winnerUserIDs[0], uuid.MustParse("6cc0f1b0-c894-4b12-a9ed-569cdfde3b16"), sundayDate, 0)
		if err != nil {
			logger.Log.Warn("Failed to give achievement for user '" + winnerUserIDs[0].String() + "'. Ignoring. Error: " + err.Error())
		}
	}

	return nil

}

func GenerateAchievementsForSeason(seasonResults []models.WeekResults, targetUser *uuid.UUID) error {
	type UserTally struct {
		UserID     uuid.UUID
		LoseAmount int
		WinAmount  int
		SickAmount int
		LoseStreak int
		WinStreak  int
		SickStreak int
	}

	if len(seasonResults) == 0 {
		return errors.New("Empty season, returning...")
	}

	// First week is the last week
	seasonSunday, err := utilities.FindNextSunday(seasonResults[0].WeekDate)
	if err != nil {
		return errors.New("Failed to find Sunday for last week of the season.")
	}

	// Reverse array
	reversedArray := []models.WeekResults{}
	for i := (len(seasonResults) - 1); i >= 0; i-- {
		reversedArray = append(reversedArray, seasonResults[i])
	}
	seasonResults = reversedArray

	// User array
	userTally := []UserTally{}

	for _, weekResults := range seasonResults {

		for _, weekResult := range weekResults.UserWeekResults {

			if weekResult.SickLeave {

				found := false
				foundIndex := 0
				for index, user := range userTally {

					if user.UserID == weekResult.UserID {
						found = true
						foundIndex = index
						break
					}

				}

				if !found {
					userTally = append(userTally, UserTally{
						UserID:     weekResult.UserID,
						LoseAmount: 0,
						LoseStreak: 0,
						WinAmount:  0,
						SickAmount: 1,
						WinStreak:  0,
						SickStreak: 1,
					})
				} else {
					userTally[foundIndex].WinStreak = 0
					userTally[foundIndex].LoseStreak = 0
					userTally[foundIndex].SickStreak += 1
					userTally[foundIndex].SickAmount += 1
				}

			} else if weekResult.WeekCompletion >= 1.0 {

				found := false
				foundIndex := 0
				for index, user := range userTally {

					if user.UserID == weekResult.UserID {
						found = true
						foundIndex = index
						break
					}

				}

				if !found {
					userTally = append(userTally, UserTally{
						UserID:     weekResult.UserID,
						LoseAmount: 0,
						LoseStreak: 0,
						WinAmount:  1,
						SickAmount: 0,
						WinStreak:  1,
						SickStreak: 0,
					})
				} else {

					// If week is won, and user has lost more than one week in a row
					if userTally[foundIndex].LoseStreak > 1 && (targetUser == nil || *targetUser == userTally[foundIndex].UserID) {

						// Give achievement to user
						err := GiveUserAnAchievement(userTally[foundIndex].UserID, uuid.MustParse("38524a0a-f0b6-4cbf-b221-05ebfa0797f7"), seasonSunday, 0)
						if err != nil {
							logger.Log.Info("Failed to give achievement for user '" + userTally[foundIndex].UserID.String() + "'. Ignoring. Error: " + err.Error())
						}

					}

					// If week is won, and user has been sick one week or more
					if userTally[foundIndex].SickStreak > 0 && (targetUser == nil || *targetUser == userTally[foundIndex].UserID) {

						// Give achievement to user
						err := GiveUserAnAchievement(userTally[foundIndex].UserID, uuid.MustParse("b342cd1b-1812-4384-967f-51d2be772eab"), seasonSunday, 0)
						if err != nil {
							logger.Log.Info("Failed to give achievement for user '" + userTally[foundIndex].UserID.String() + "'. Ignoring. Error: " + err.Error())
						}

					}

					userTally[foundIndex].WinAmount += 1
					userTally[foundIndex].WinStreak += 1
					userTally[foundIndex].LoseStreak = 0
					userTally[foundIndex].SickStreak = 0
				}

			} else {

				found := false
				foundIndex := 0
				for index, user := range userTally {

					if user.UserID == weekResult.UserID {
						found = true
						foundIndex = index
						break
					}

				}

				if !found {
					userTally = append(userTally, UserTally{
						UserID:     weekResult.UserID,
						LoseAmount: 1,
						LoseStreak: 1,
						WinAmount:  0,
						SickAmount: 0,
						WinStreak:  0,
						SickStreak: 0,
					})
				} else {
					userTally[foundIndex].LoseAmount += 1
					userTally[foundIndex].LoseStreak += 1
					userTally[foundIndex].WinStreak = 0
					userTally[foundIndex].SickStreak = 0
				}

			}

		}

	}

	for _, user := range userTally {

		// If win amount is more than zero, and lose amount is zero
		if user.LoseAmount == 0 && user.WinAmount > 0 && (targetUser == nil || *targetUser == user.UserID) {
			// Give achievement to user
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("01dc9c4b-cf65-4d3c-9596-1417b67bd86f"), seasonSunday, 0)
			if err != nil {
				logger.Log.Info("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}
		}

		// If sick leave amount is zero
		if user.SickAmount == 0 && (targetUser == nil || *targetUser == user.UserID) {
			// Give achievement to user
			err := GiveUserAnAchievement(user.UserID, uuid.MustParse("b566e486-d476-40f1-a9f2-28035bb43f37"), seasonSunday, 0)
			if err != nil {
				logger.Log.Info("Failed to give achievement for user '" + user.UserID.String() + "'. Ignoring. Error: " + err.Error())
			}
		}

	}

	return nil
}

func APIGiveUserAnAchievement(context *gin.Context) {
	// Create user request
	var userIDString = context.Param("user_id")
	var achievementDelegationCreationRequest models.AchievementDelegationCreationRequest

	// Parse creation request
	if err := context.ShouldBindJSON(&achievementDelegationCreationRequest); err != nil {
		logger.Log.Info("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		logger.Log.Info("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	givenAt := time.Now()
	if achievementDelegationCreationRequest.GivenAt != nil {
		givenAt = *achievementDelegationCreationRequest.GivenAt
	}

	err = GiveUserAnAchievement(userID, achievementDelegationCreationRequest.AchievementID, givenAt, 0)
	if err != nil {
		logger.Log.Info("Failed to give achievement to user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to give achievement to user."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Achievement given."})
}
