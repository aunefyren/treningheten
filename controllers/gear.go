package controllers

import (
	"net/http"
	"strings"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// gearTypes is the accepted gear-type vocabulary.
var gearTypes = map[string]bool{"shoe": true, "bike": true, "other": true}

func validGearType(gearType string) bool {
	return gearTypes[strings.ToLower(strings.TrimSpace(gearType))]
}

// ConvertGearToGearObject flattens a Gear into its read shape. Distance (km) is
// the caller's responsibility — it is the roll-up from GetGearDistanceTotalsForUser,
// or zero when an operation embeds its gear's identity only.
func ConvertGearToGearObject(gear models.Gear, distance float64) models.GearObject {
	gearObject := models.GearObject{}
	gearObject.ID = gear.ID
	gearObject.CreatedAt = gear.CreatedAt
	gearObject.UpdatedAt = gear.UpdatedAt
	gearObject.DeletedAt = gear.DeletedAt
	gearObject.Enabled = gear.Enabled
	gearObject.User = gear.UserID
	gearObject.Name = gear.Name
	gearObject.Type = gear.Type
	gearObject.Brand = gear.Brand
	gearObject.Model = gear.Model
	gearObject.Nickname = gear.Nickname
	gearObject.Retired = gear.Retired
	gearObject.IsPrimary = gear.IsPrimary
	gearObject.StravaGearID = gear.StravaGearID
	gearObject.Distance = distance
	return gearObject
}

// BuildGearObjectsForUser returns a user's gear enriched with the computed
// distance total (km) for each item.
func BuildGearObjectsForUser(userID uuid.UUID) (gearObjects []models.GearObject, err error) {
	gearObjects = []models.GearObject{}

	gear, err := database.GetGearForUser(userID)
	if err != nil {
		return gearObjects, err
	}

	totals, err := database.GetGearDistanceTotalsForUser(userID)
	if err != nil {
		return gearObjects, err
	}

	for _, item := range gear {
		gearObjects = append(gearObjects, ConvertGearToGearObject(item, totals[item.ID]))
	}

	return gearObjects, nil
}

func APIGetGearForUser(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	gearObjects, err := BuildGearObjectsForUser(userID)
	if err != nil {
		logger.Log.Error("Failed to get gear for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get gear for user."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Gear retrieved.", "gear": gearObjects})
}

func APICreateGear(context *gin.Context) {
	var gearCreationRequest models.GearCreationRequest

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	if err := context.ShouldBindJSON(&gearCreationRequest); err != nil {
		logger.Log.Error("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	gearCreationRequest.Name = strings.TrimSpace(gearCreationRequest.Name)
	if gearCreationRequest.Name == "" {
		logger.Log.Warn("Gear creation request had an empty name.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Gear must have a name."})
		context.Abort()
		return
	}

	if gearCreationRequest.Type == "" {
		gearCreationRequest.Type = "shoe"
	} else if !validGearType(gearCreationRequest.Type) {
		logger.Log.Warn("Gear creation request had an invalid type.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Gear type must be shoe, bike or other."})
		context.Abort()
		return
	}

	gear := models.Gear{}
	gear.ID = uuid.New()
	gear.UserID = userID
	gear.Name = gearCreationRequest.Name
	gear.Type = strings.ToLower(gearCreationRequest.Type)
	gear.Brand = gearCreationRequest.Brand
	gear.Model = gearCreationRequest.Model
	gear.Nickname = gearCreationRequest.Nickname

	gear, err = database.CreateGearInDB(gear)
	if err != nil {
		logger.Log.Error("Failed to create gear in DB. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create gear in DB."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Gear created.", "gear": ConvertGearToGearObject(gear, 0)})
}

func APIUpdateGear(context *gin.Context) {
	var gearUpdateRequest models.GearUpdateRequest

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	gearID, err := uuid.Parse(context.Param("gear_id"))
	if err != nil {
		logger.Log.Error("Failed to verify gear ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify gear ID."})
		context.Abort()
		return
	}

	if err := context.ShouldBindJSON(&gearUpdateRequest); err != nil {
		logger.Log.Error("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	gear, err := database.GetGearByIDAndUserID(gearID, userID)
	if err != nil {
		logger.Log.Error("Failed to get gear. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get gear."})
		context.Abort()
		return
	} else if gear == nil {
		logger.Log.Warn("Gear not found for user.")
		context.JSON(http.StatusNotFound, gin.H{"error": "Gear not found."})
		context.Abort()
		return
	}

	// Identity fields are read-only for Strava-sourced gear (Strava owns them).
	isStravaGear := gear.StravaGearID != nil
	if isStravaGear && (gearUpdateRequest.Name != nil || gearUpdateRequest.Type != nil || gearUpdateRequest.Brand != nil || gearUpdateRequest.Model != nil) {
		logger.Log.Warn("Attempt to edit identity fields of Strava-sourced gear.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Name, type and brand are managed by Strava for synced gear."})
		context.Abort()
		return
	}

	if gearUpdateRequest.Name != nil {
		name := strings.TrimSpace(*gearUpdateRequest.Name)
		if name == "" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Gear must have a name."})
			context.Abort()
			return
		}
		gear.Name = name
	}
	if gearUpdateRequest.Type != nil {
		if !validGearType(*gearUpdateRequest.Type) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Gear type must be shoe, bike or other."})
			context.Abort()
			return
		}
		gear.Type = strings.ToLower(*gearUpdateRequest.Type)
	}
	if gearUpdateRequest.Brand != nil {
		gear.Brand = gearUpdateRequest.Brand
	}
	if gearUpdateRequest.Model != nil {
		gear.Model = gearUpdateRequest.Model
	}
	if gearUpdateRequest.Nickname != nil {
		gear.Nickname = gearUpdateRequest.Nickname
	}
	if gearUpdateRequest.Retired != nil {
		gear.Retired = *gearUpdateRequest.Retired
	}
	if gearUpdateRequest.IsPrimary != nil {
		gear.IsPrimary = *gearUpdateRequest.IsPrimary
	}

	updatedGear, err := database.UpdateGearInDB(*gear)
	if err != nil {
		logger.Log.Error("Failed to update gear in DB. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gear in DB."})
		context.Abort()
		return
	}

	// Only one gear can be primary; demote the rest.
	if updatedGear.IsPrimary {
		if err := database.UnsetPrimaryGearForUser(userID, updatedGear.ID); err != nil {
			logger.Log.Error("Failed to clear other primary gear. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gear in DB."})
			context.Abort()
			return
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Gear updated.", "gear": ConvertGearToGearObject(updatedGear, 0)})
}

func APIDeleteGear(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	gearID, err := uuid.Parse(context.Param("gear_id"))
	if err != nil {
		logger.Log.Error("Failed to verify gear ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify gear ID."})
		context.Abort()
		return
	}

	gear, err := database.GetGearByIDAndUserID(gearID, userID)
	if err != nil {
		logger.Log.Error("Failed to get gear. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get gear."})
		context.Abort()
		return
	} else if gear == nil {
		logger.Log.Warn("Gear not found for user.")
		context.JSON(http.StatusNotFound, gin.H{"error": "Gear not found."})
		context.Abort()
		return
	}

	gear.Enabled = false

	_, err = database.UpdateGearInDB(*gear)
	if err != nil {
		logger.Log.Error("Failed to delete gear in DB. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete gear in DB."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Gear deleted."})
}

// APISetGearForExercise assigns (or clears, when gear_id is null) gear on every
// operation of a session — the exercise-level selector in the builder.
func APISetGearForExercise(context *gin.Context) {
	var request struct {
		GearID *uuid.UUID `json:"gear_id"`
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	exerciseID, err := uuid.Parse(context.Param("exercise_id"))
	if err != nil {
		logger.Log.Error("Failed to verify exercise ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise ID."})
		context.Abort()
		return
	}

	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Confirm the exercise belongs to the user.
	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		logger.Log.Error("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	} else if exercise == nil {
		logger.Log.Warn("Exercise not found for user.")
		context.JSON(http.StatusNotFound, gin.H{"error": "Exercise not found."})
		context.Abort()
		return
	}

	// Confirm the gear belongs to the user when one is supplied.
	if request.GearID != nil {
		gear, err := database.GetGearByIDAndUserID(*request.GearID, userID)
		if err != nil {
			logger.Log.Error("Failed to get gear. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get gear."})
			context.Abort()
			return
		} else if gear == nil {
			logger.Log.Warn("Gear not found for user.")
			context.JSON(http.StatusBadRequest, gin.H{"error": "Gear not found."})
			context.Abort()
			return
		}
	}

	operations, err := database.GetOperationsByExerciseID(exerciseID)
	if err != nil {
		logger.Log.Error("Failed to get operations for exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operations for exercise."})
		context.Abort()
		return
	}

	for _, operation := range operations {
		operation.GearID = request.GearID
		_, err = database.UpdateOperationInDB(operation)
		if err != nil {
			logger.Log.Error("Failed to update operation gear in DB. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation gear in DB."})
			context.Abort()
			return
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Gear assigned to session."})
}

// resolveStravaGearForUser finds or creates the user's gear row for a Strava
// gear id, fetching the gear's identity (name/brand/model/type) lazily from
// Strava the first time it is seen. Returns nil when the activity has no gear.
func resolveStravaGearForUser(stravaGearID *string, userID uuid.UUID, token string) (*models.Gear, error) {
	if stravaGearID == nil || strings.TrimSpace(*stravaGearID) == "" {
		return nil, nil
	}

	gearID := strings.TrimSpace(*stravaGearID)

	existing, err := database.GetGearByStravaGearIDAndUserID(gearID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// New Strava gear — fetch its detail once to name it. A failure here must not
	// block the import, so fall back to the id as the name.
	gear := models.Gear{}
	gear.ID = uuid.New()
	gear.UserID = userID
	gear.StravaGearID = &gearID
	gear.Name = gearID
	gear.Type = stravaGearTypeFromID(gearID)

	detail, detailErr := StravaGetGear(token, gearID)
	if detailErr != nil {
		logger.Log.Warn("Failed to fetch Strava gear detail, importing id only. Error: " + detailErr.Error())
	} else {
		if strings.TrimSpace(detail.Name) != "" {
			gear.Name = strings.TrimSpace(detail.Name)
		}
		if strings.TrimSpace(detail.BrandName) != "" {
			brand := strings.TrimSpace(detail.BrandName)
			gear.Brand = &brand
		}
		if strings.TrimSpace(detail.ModelName) != "" {
			model := strings.TrimSpace(detail.ModelName)
			gear.Model = &model
		}
		gear.Retired = detail.Retired
		if detail.Primary {
			gear.IsPrimary = true
		}
	}

	created, err := database.CreateGearInDB(gear)
	if err != nil {
		return nil, err
	}

	// A Strava "primary" gear becomes the local primary too; demote the rest.
	if created.IsPrimary {
		if err := database.UnsetPrimaryGearForUser(userID, created.ID); err != nil {
			logger.Log.Warn("Failed to clear other primary gear after Strava import. Error: " + err.Error())
		}
	}

	return &created, nil
}

// stravaGearTypeFromID maps Strava's gear-id prefix to a gear type: "b" = bike,
// "g" = shoe; anything else is "other".
func stravaGearTypeFromID(stravaGearID string) string {
	if strings.HasPrefix(stravaGearID, "b") {
		return "bike"
	} else if strings.HasPrefix(stravaGearID, "g") {
		return "shoe"
	}
	return "other"
}
