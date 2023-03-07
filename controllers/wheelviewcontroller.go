package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"log"
	"strconv"
)

func ConvertWheelviewToWheelviewObject(wheelview models.Wheelview) (models.WheelviewObject, error) {

	wheelviewObject := models.WheelviewObject{}

	user, err := database.GetUserInformation(wheelview.User)
	if err != nil {
		log.Println("Failed to get user information for user '" + strconv.Itoa(wheelview.User) + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	}

	wheelviewObject.User = user

	debt, debtFound, err := database.GetDebtByDebtID(wheelview.Debt)
	if err != nil {
		log.Println("Failed to get debt for debt '" + strconv.Itoa(wheelview.Debt) + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	} else if !debtFound {
		log.Println("Failed to find debt for debt '" + strconv.Itoa(wheelview.Debt) + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	}

	debtObject, err := ConvertDebtToDebtObject(debt)
	if err != nil {
		log.Println("Failed to convert debt to debt onbject for debt '" + strconv.Itoa(wheelview.Debt) + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	}

	wheelviewObject.Debt = debtObject

	wheelviewObject.CreatedAt = wheelview.CreatedAt
	wheelviewObject.DeletedAt = wheelview.DeletedAt
	wheelviewObject.Enabled = wheelview.Enabled
	wheelviewObject.ID = wheelview.ID
	wheelviewObject.Model = wheelview.Model
	wheelviewObject.UpdatedAt = wheelview.UpdatedAt
	wheelviewObject.Viewed = wheelview.Viewed

	return wheelviewObject, err

}

func ConvertWheelviewsToWheelviewObjects(wheelviews []models.Wheelview) ([]models.WheelviewObject, error) {

	wheelviewObjects := []models.WheelviewObject{}

	for _, wheelview := range wheelviews {
		wheelviewObject, err := ConvertWheelviewToWheelviewObject(wheelview)
		if err != nil {
			log.Println("Failed to convert debt to debt onbject for debt '" + strconv.Itoa(wheelview.Debt) + "'. Returning. Error: " + err.Error())
			return []models.WheelviewObject{}, err
		}
		wheelviewObjects = append(wheelviewObjects, wheelviewObject)
	}

	return wheelviewObjects, nil
}
