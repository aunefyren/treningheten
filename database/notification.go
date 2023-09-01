package database

import (
	"aunefyren/treningheten/models"
	"errors"
)

// Create new subscription for a user
func CreateSubscriptionInDB(subscription models.Subscription) (uint, error) {
	record := Instance.Create(&subscription)
	if record.Error != nil {
		return 0, record.Error
	}
	return subscription.ID, nil
}

// Get all subscriptions for user by user ID
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

// Get subscription for user by user ID and endpoint
func GetAllSubscriptionForUserByUserIDAndEndpoint(userID int, endpoint string) (models.Subscription, bool, error) {

	var subscriptionStruct models.Subscription

	subscriptionRecord := Instance.Where("`subscriptions`.enabled = ?", 1).Where("`subscriptions`.user = ?", userID).Where("`subscriptions`.endpoint = ?", endpoint).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return models.Subscription{}, false, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return models.Subscription{}, false, nil
	}

	return subscriptionStruct, true, nil

}

// Update an exercise in the database
func UpdateSubscriptionForUserByUserIDAndEndpoint(userID int, endpoint string, reminder bool, achievement bool, news bool) error {

	var subscriptionStruct models.Subscription
	var reminderInt = 0
	var achievementInt = 0
	var newsInt = 0

	if reminder {
		reminderInt = 1
	}
	if achievement {
		achievementInt = 1
	}
	if news {
		newsInt = 1
	}

	subscriptionRecord := Instance.Model(subscriptionStruct).Where("`subscriptions`.enabled = ?", 1).Where("`subscriptions`.user = ?", userID).Where("`subscriptions`.endpoint = ?", endpoint).Update("sunday_alert", reminderInt)
	if subscriptionRecord.Error != nil {
		return subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected != 1 {
		return errors.New("Failed to update reminder value in update.")
	}

	subscriptionRecord = Instance.Model(subscriptionStruct).Where("`subscriptions`.enabled = ?", 1).Where("`subscriptions`.user = ?", userID).Where("`subscriptions`.endpoint = ?", endpoint).Update("achievement_alert", achievementInt)
	if subscriptionRecord.Error != nil {
		return subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected != 1 {
		return errors.New("Failed to update achievement value in update.")
	}

	subscriptionRecord = Instance.Model(subscriptionStruct).Where("`subscriptions`.enabled = ?", 1).Where("`subscriptions`.user = ?", userID).Where("`subscriptions`.endpoint = ?", endpoint).Update("news_alert", newsInt)
	if subscriptionRecord.Error != nil {
		return subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected != 1 {
		return errors.New("Failed to update news value in update.")
	}

	return nil

}
