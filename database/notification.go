package database

import (
	"errors"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Create new subscription for a user
func CreateSubscriptionInDB(subscription models.Subscription) (uuid.UUID, error) {
	record := Instance.Create(&subscription)
	if record.Error != nil {
		return uuid.UUID{}, record.Error
	}
	return subscription.ID, nil
}

// Get all subscriptions for user by user ID
func GetAllSubscriptionsForUserByUserID(userID uuid.UUID) ([]models.Subscription, error) {

	var subscriptionStruct []models.Subscription

	subscriptionRecord := Instance.Where("subscriptions.enabled = ?", true).Where("subscriptions.user_id = ?", userID).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return []models.Subscription{}, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return []models.Subscription{}, nil
	}

	return subscriptionStruct, nil

}

// Get subscription for user by user ID and endpoint
func GetAllSubscriptionForUserByUserIDAndEndpoint(userID uuid.UUID, endpoint string) (models.Subscription, bool, error) {

	var subscriptionStruct models.Subscription

	subscriptionRecord := Instance.Where("subscriptions.enabled = ?", true).Where("subscriptions.user_id = ?", userID).Where("subscriptions.endpoint = ?", endpoint).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return models.Subscription{}, false, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return models.Subscription{}, false, nil
	}

	return subscriptionStruct, true, nil

}

// Get subscription for achievements by user ID
func GetAllSubscriptionsForAchievementsForUserID(userID uuid.UUID) ([]models.Subscription, bool, error) {

	var subscriptionStruct []models.Subscription

	subscriptionRecord := Instance.Where("subscriptions.enabled = ?", true).Where("subscriptions.achievement_alert = ?", true).Where("subscriptions.user_id = ?", userID).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return []models.Subscription{}, false, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return []models.Subscription{}, false, nil
	}

	return subscriptionStruct, true, nil

}

// GetAllSubscriptionsForAccountAlertsForUserID returns the user's enabled subscriptions
// that accept account notices (e.g. a broken integration connection).
func GetAllSubscriptionsForAccountAlertsForUserID(userID uuid.UUID) ([]models.Subscription, error) {
	subscriptions := []models.Subscription{}

	record := Instance.Where("subscriptions.enabled = ?", true).
		Where("subscriptions.account_alert = ?", true).
		Where("subscriptions.user_id = ?", userID).
		Find(&subscriptions)
	if record.Error != nil {
		return []models.Subscription{}, record.Error
	}

	return subscriptions, nil
}

// Get subscriptions for news
func GetAllSubscriptionsForNews() ([]models.Subscription, bool, error) {

	var subscriptionStruct []models.Subscription

	subscriptionRecord := Instance.Where("subscriptions.enabled = ?", true).Where("subscriptions.news_alert = ?", true).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return []models.Subscription{}, false, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return []models.Subscription{}, false, nil
	}

	return subscriptionStruct, true, nil

}

// Get subscriptions for sunday alerts
func GetAllSubscriptionsForSundayAlerts() ([]models.Subscription, bool, error) {

	var subscriptionStruct []models.Subscription

	subscriptionRecord := Instance.Where("subscriptions.enabled = ?", true).Where("subscriptions.sunday_alert = ?", true).Find(&subscriptionStruct)
	if subscriptionRecord.Error != nil {
		return []models.Subscription{}, false, subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected == 0 {
		return []models.Subscription{}, false, nil
	}

	return subscriptionStruct, true, nil

}

// Update an exercise in the database
// accountAlert is optional: nil leaves the stored value alone.
func UpdateSubscriptionForUserByUserIDAndEndpoint(userID uuid.UUID, endpoint string, reminder bool, achievement bool, news bool, accountAlert *bool) (err error) {

	err = nil

	err = UpdateSubscriptionSundayReminderByEndpointAndUserID(userID, endpoint, reminder)
	if err != nil {
		return err
	}

	err = UpdateSubscriptionAchievementByEndpointAndUserID(userID, endpoint, achievement)
	if err != nil {
		return err
	}

	err = UpdateSubscriptionNewsByEndpointAndUserID(userID, endpoint, news)
	if err != nil {
		return err
	}

	if accountAlert != nil {
		err = UpdateSubscriptionAccountAlertByEndpointAndUserID(userID, endpoint, *accountAlert)
		if err != nil {
			return err
		}
	}

	return err

}

func UpdateSubscriptionSundayReminderByEndpointAndUserID(userID uuid.UUID, endpoint string, reminder bool) (err error) {

	var subscriptionStruct models.Subscription
	err = nil

	subscriptionRecord := Instance.Model(subscriptionStruct).Where("subscriptions.enabled = ?", true).Where("subscriptions.user_id = ?", userID).Where("subscriptions.endpoint = ?", endpoint).Update("sunday_alert", reminder)
	if subscriptionRecord.Error != nil {
		return subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected != 1 {
		return errors.New("Failed to update sunday reminder value in update.")
	}

	return err

}

func UpdateSubscriptionAchievementByEndpointAndUserID(userID uuid.UUID, endpoint string, achievement bool) (err error) {

	var subscriptionStruct models.Subscription
	err = nil

	subscriptionRecord := Instance.Model(subscriptionStruct).Where("subscriptions.enabled = ?", true).Where("subscriptions.user_id = ?", userID).Where("subscriptions.endpoint = ?", endpoint).Update("achievement_alert", achievement)
	if subscriptionRecord.Error != nil {
		return subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected != 1 {
		return errors.New("Failed to update achievement value in update.")
	}

	return err

}

func UpdateSubscriptionNewsByEndpointAndUserID(userID uuid.UUID, endpoint string, news bool) (err error) {

	var subscriptionStruct models.Subscription
	err = nil

	subscriptionRecord := Instance.Model(subscriptionStruct).Where("subscriptions.enabled = ?", true).Where("subscriptions.user_id = ?", userID).Where("subscriptions.endpoint = ?", endpoint).Update("news_alert", news)
	if subscriptionRecord.Error != nil {
		return subscriptionRecord.Error
	} else if subscriptionRecord.RowsAffected != 1 {
		return errors.New("Failed to update news value in update.")
	}

	return err

}

// UpdateSubscriptionAccountAlertByEndpointAndUserID sets the account-notice opt-in. It
// is also how a new subscription stores false: the column defaults to true, so GORM
// drops a false from the insert (see docs/data-conventions.md).
func UpdateSubscriptionAccountAlertByEndpointAndUserID(userID uuid.UUID, endpoint string, accountAlert bool) error {
	record := Instance.Model(&models.Subscription{}).
		Where("subscriptions.enabled = ?", true).
		Where("subscriptions.user_id = ?", userID).
		Where("subscriptions.endpoint = ?", endpoint).
		Update("account_alert", accountAlert)
	if record.Error != nil {
		return record.Error
	} else if record.RowsAffected != 1 {
		return errors.New("Failed to update account alert value in update.")
	}

	return nil
}

func UpdateSubscription(subscription models.Subscription) (models.Subscription, error) {
	record := Instance.Save(&subscription)
	if record.Error != nil {
		return subscription, record.Error
	}
	return subscription, nil
}
