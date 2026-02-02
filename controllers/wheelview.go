package controllers

import (
	"errors"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"
)

func ConvertWheelviewToWheelviewObject(wheelview models.Wheelview) (models.WheelviewObject, error) {

	wheelviewObject := models.WheelviewObject{}

	user, err := database.GetUserInformation(wheelview.UserID)
	if err != nil {
		logger.Log.Info("Failed to get user information for user '" + wheelview.UserID.String() + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	}

	wheelviewObject.User = user

	debt, debtFound, err := database.GetDebtByDebtID(wheelview.DebtID)
	if err != nil {
		logger.Log.Info("Failed to get debt for debt '" + wheelview.DebtID.String() + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	} else if !debtFound {
		logger.Log.Info("Failed to find debt for debt '" + wheelview.DebtID.String() + "'. Returning.")
		return models.WheelviewObject{}, errors.New("Failed to find debt for debt '" + wheelview.DebtID.String() + "'.")
	}

	debtObject, err := ConvertDebtToDebtObject(debt)
	if err != nil {
		logger.Log.Info("Failed to convert debt to debt object for debt '" + wheelview.DebtID.String() + "'. Returning. Error: " + err.Error())
		return models.WheelviewObject{}, err
	}

	wheelviewObject.Debt = debtObject

	wheelviewObject.CreatedAt = wheelview.CreatedAt
	wheelviewObject.DeletedAt = wheelview.DeletedAt
	wheelviewObject.Enabled = wheelview.Enabled
	wheelviewObject.ID = wheelview.ID
	wheelviewObject.UpdatedAt = wheelview.UpdatedAt
	wheelviewObject.Viewed = wheelview.Viewed

	return wheelviewObject, err

}

func ConvertWheelviewsToWheelviewObjects(wheelviews []models.Wheelview) ([]models.WheelviewObject, error) {

	wheelviewObjects := []models.WheelviewObject{}

	for _, wheelview := range wheelviews {
		wheelviewObject, err := ConvertWheelviewToWheelviewObject(wheelview)
		if err != nil {
			logger.Log.Info("Failed to convert debt to debt object for debt '" + wheelview.Debt.ID.String() + "'. Returning. Error: " + err.Error())
			return []models.WheelviewObject{}, err
		}
		wheelviewObjects = append(wheelviewObjects, wheelviewObject)
	}

	return wheelviewObjects, nil
}
