package database

import (
	"aunefyren/treningheten/models"
)

// Create new subscription for a user
func CreateSubscriptionInDB(subscription models.Subscription) (uint, error) {
	record := Instance.Create(&subscription)
	if record.Error != nil {
		return 0, record.Error
	}
	return subscription.ID, nil
}

// Update an exercise in the database
func GetAllSubscriptionsForUserByUserID(userID int) ([]models.Subscription, error) {

	var subscriptionStruct []models.Subscription

	subscriptionRecord := Instance.Where("`subscriptions`.enabled = ?", 1).Where("`subscriptions`.user = ?", userID).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return []models.Subscription{}, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return []models.Subscription{}, nil
	}

	return subscriptionStruct, nil

}
