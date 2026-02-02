package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/files"
	"aunefyren/treningheten/logger"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func PushNotificationToSubscriptions(notificationType string, notificationBody string, notificationTitle string, subscriptions []models.Subscription, notificationAdditionalData *string) (int, error) {
	vapidSettings, err := GetVAPIDSettings()
	if err != nil {
		logger.Log.Info("Failed to get VAPID settings. Error: " + err.Error())
		return 0, errors.New("Failed to get VAPID settings.")
	}

	notificationSum := 0

	notificationAdditionalDataString := "null"
	if notificationAdditionalData != nil {
		notificationAdditionalDataString = "\"" + *notificationAdditionalData + "\""
	}

	notificationData := `
		{
			"title": "` + notificationTitle + `",
			"body": "` + notificationBody + `",
			"additional_data": ` + notificationAdditionalDataString + `,
			"category": "` + notificationType + `"
		}
	`

	dataBuffer := &bytes.Buffer{}
	if err = json.Compact(dataBuffer, []byte(notificationData)); err != nil {
		logger.Log.Info("Failed to compact JSON data. Error: " + err.Error())
		return 0, errors.New("Failed to compact JSON data.")
	}

	for _, subscription := range subscriptions {
		// Decode subscription
		s := &webpush.Subscription{
			Endpoint: subscription.Endpoint,
		}

		s.Keys.Auth = subscription.Auth
		s.Keys.P256dh = subscription.P256Dh

		// Send Notification
		response, err := webpush.SendNotification(dataBuffer.Bytes(), s, &webpush.Options{
			Subscriber:      vapidSettings.VAPIDContact,
			VAPIDPublicKey:  vapidSettings.VAPIDPublicKey,
			VAPIDPrivateKey: vapidSettings.VAPIDSecretKey,
			TTL:             30,
			RecordSize:      2048,
		})

		if err != nil {
			logger.Log.Info("Failed to push notification. Error: " + err.Error())
			return notificationSum, errors.New("Failed to push notification.")
		}

		logger.Log.Info("Pushed notification, got status code: " + string(response.Status))

		// Disable "gone" endpoints
		if response.StatusCode == 410 {
			subscription.Enabled = false
			_, err := database.UpdateSubscription(subscription)
			if err != nil {
				logger.Log.Info("Failed to disable subscription. Error: " + err.Error())
			}
		}

		notificationSum += 1
	}

	return notificationSum, nil
}

func GetVAPIDSettings() (models.VAPIDSettings, error) {
	vapidSettings := models.VAPIDSettings{}

	vapidSettings.VAPIDContact = files.ConfigFile.VAPIDContact
	vapidSettings.VAPIDPublicKey = files.ConfigFile.VAPIDPublicKey
	vapidSettings.VAPIDSecretKey = files.ConfigFile.VAPIDSecretKey

	return vapidSettings, nil

}

func APISubscribeToNotification(context *gin.Context) {

	// Create season request
	var subscriptionRequest models.SubscriptionCreationRequest
	var subscription models.Subscription

	if err := context.ShouldBindJSON(&subscriptionRequest); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get User ID from token. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get User ID from token."})
		context.Abort()
		return
	}

	subscription.Endpoint = subscriptionRequest.Subscription.Endpoint
	subscription.ExpirationTime = subscriptionRequest.Subscription.ExpirationTime
	subscription.Auth = subscriptionRequest.Subscription.Keys.Auth
	subscription.P256Dh = subscriptionRequest.Subscription.Keys.P256Dh
	subscription.SundayAlert = subscriptionRequest.Settings.SundayAlert
	subscription.AchievementAlert = subscriptionRequest.Settings.AchievementAlert
	subscription.NewsAlert = subscriptionRequest.Settings.NewsAlert
	subscription.UserID = userID
	subscription.ID = uuid.New()

	_, err = database.CreateSubscriptionInDB(subscription)
	if err != nil {
		logger.Log.Info("Failed to create subscription in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription in database."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Subscription created."})

}

func APIPushNotificationToAllDevicesForUser(context *gin.Context) {

	// Create notification request
	var notificationRequest models.NotificationCreationRequest

	if err := context.ShouldBindJSON(&notificationRequest); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	subscriptions, err := database.GetAllSubscriptionsForUserByUserID(notificationRequest.UserID)
	if err != nil {
		logger.Log.Info("Failed to get subscriptions for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscriptions for user."})
		context.Abort()
		return
	}

	pushedAmount, err := PushNotificationToSubscriptions(notificationRequest.Category, notificationRequest.Body, notificationRequest.Title, subscriptions, notificationRequest.AdditionalData)
	if err != nil {
		logger.Log.Info("Failed to push notification. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to push notification."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Notification(s) pushed.", "amount": pushedAmount})

}

func APIGetSubscriptionForEndpoint(context *gin.Context) {

	// Create notification request
	var subscriptionRequest models.SubscriptionGetRequest

	if err := context.ShouldBindJSON(&subscriptionRequest); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get User ID from token. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get User ID from token."})
		context.Abort()
		return
	}

	subscription, subFound, err := database.GetAllSubscriptionForUserByUserIDAndEndpoint(userID, subscriptionRequest.Endpoint)
	if err != nil {
		logger.Log.Info("Failed to get subscription from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscription from database."})
		context.Abort()
		return
	} else if !subFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "No subscription found."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Subscription found.", "subscription": subscription})

}

func APIUpdateSubscriptionForEndpoint(context *gin.Context) {

	// Create notification request
	var subscriptionUpdateRequest models.SubscriptionUpdateRequest

	if err := context.ShouldBindJSON(&subscriptionUpdateRequest); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get User ID from token. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get User ID from token."})
		context.Abort()
		return
	}

	err = database.UpdateSubscriptionForUserByUserIDAndEndpoint(userID, subscriptionUpdateRequest.Endpoint, subscriptionUpdateRequest.SundayAlert, subscriptionUpdateRequest.AchievementAlert, subscriptionUpdateRequest.NewsAlert)
	if err != nil {
		logger.Log.Info("Failed to update subscription in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription in database."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Subscription updated."})

}

func PushNotificationsForAchievements(userID uuid.UUID) (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, subscriptionsFound, err := database.GetAllSubscriptionsForAchievementsForUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if !subscriptionsFound {
		logger.Log.Info("No subscriptions found for achievements.")
		return nil
	}

	title := "Treningheten"
	body := "You just got a new achievement 🏆"
	category := "achievement"

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, nil)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}

func PushNotificationsForNews() (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, subscriptionsFound, err := database.GetAllSubscriptionsForNews()
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if !subscriptionsFound {
		logger.Log.Info("No subscriptions found for achievements.")
		return nil
	}

	title := "Treningheten"
	body := "A news post was just published 📰"
	category := "news"

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, nil)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}

func PushNotificationsForSundayAlerts() (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, subscriptionsFound, err := database.GetAllSubscriptionsForSundayAlerts()
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if !subscriptionsFound {
		logger.Log.Info("No subscriptions found for achievements.")
		return nil
	}

	title := "Treningheten"
	body := "It's Sunday, remember to log your workouts 🔔"
	category := "alert"

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, nil)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}

func PushNotificationsForWeekLost(userId uuid.UUID) (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, err := database.GetAllSubscriptionsForUserByUserID(userId)
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if len(subscriptions) == 0 {
		logger.Log.Info("No subscriptions found for user.")
		return nil
	}

	title := "Treningheten"
	body := "You didn't hit your goal this week 😢"
	category := "alert"

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, nil)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}

func PushNotificationsForWheelSpin(userId uuid.UUID, debt models.Debt) (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, err := database.GetAllSubscriptionsForUserByUserID(userId)
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if len(subscriptions) == 0 {
		logger.Log.Info("No subscriptions found for user.")
		return nil
	}

	title := "Treningheten"
	body := "You have a wheel to spin 🎡"
	category := "debt"
	additionalDataString := debt.ID.String()

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, &additionalDataString)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}

func PushNotificationsForWheelSpinCheck(userId uuid.UUID, debt models.Debt) (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, err := database.GetAllSubscriptionsForUserByUserID(userId)
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if len(subscriptions) == 0 {
		logger.Log.Info("No subscriptions found for user.")
		return nil
	}

	title := "Treningheten"
	body := "Someone spun the wheel, check if you won 🏆"
	category := "debt"
	additionalDataString := debt.ID.String()

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, &additionalDataString)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}

func PushNotificationsForWheelSpinWin(userId uuid.UUID, debt models.Debt) (err error) {
	err = nil

	// Return if in test environment
	if strings.ToLower(files.ConfigFile.TreninghetenEnvironment) == "test" {
		return nil
	}

	subscriptions, err := database.GetAllSubscriptionsForUserByUserID(userId)
	if err != nil {
		logger.Log.Info("Failed to get subscriptions from database. Error: " + err.Error())
		return errors.New("Failed to get subscriptions from database.")
	} else if len(subscriptions) == 0 {
		logger.Log.Info("No subscriptions found for user.")
		return nil
	}

	title := "Treningheten"
	body := "Someone didn't hit their goal, and you won 🏆"
	category := "debt"
	additionalDataString := debt.ID.String()

	_, err = PushNotificationToSubscriptions(category, body, title, subscriptions, &additionalDataString)
	if err != nil {
		logger.Log.Info("Failed to push notification(s). Error: " + err.Error())
		return errors.New("Failed to push notification(s).")
	}

	return nil

}
