package controllers

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"errors"
	"log"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

func PushNotification(notficationType string, notificationBody string, notficationTitle string, subscriptions []models.Subscription) (int, error) {

	vapidSettings, err := GetVAPIDSettings()
	if err != nil {
		log.Println("Failed to get VAPID settings. Error: " + err.Error())
		return 0, errors.New("Failed to get VAPID settings.")
	}

	notificationSum := 0

	notificationData := `
		{
			"title": "` + notficationTitle + `",
			"body": "` + notificationBody + `",
			"category": "` + notficationType + `"
		}
	`

	for _, subscription := range subscriptions {

		// Decode subscription
		s := &webpush.Subscription{
			Endpoint: subscription.Endpoint,
		}

		s.Keys.Auth = subscription.Auth
		s.Keys.P256dh = subscription.P256Dh

		// Send Notification
		response, err := webpush.SendNotification([]byte(notificationData), s, &webpush.Options{
			Subscriber:      vapidSettings.VAPIDContact,
			VAPIDPublicKey:  vapidSettings.VAPIDPublicKey,
			VAPIDPrivateKey: vapidSettings.VAPIDSecretKey,
			TTL:             30,
		})

		if err != nil {
			log.Println("Failed to push notification. Error: " + err.Error())
			return notificationSum, errors.New("Failed to push notification.")
		}

		log.Println("Pushed notification, got status code: " + string(response.Status))

		notificationSum += 1

	}

	return notificationSum, nil

}

func GetVAPIDSettings() (models.VAPIDSettings, error) {

	vapidSettings := models.VAPIDSettings{}

	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config file. Error: " + err.Error())
		return vapidSettings, errors.New("Failed to get config file.")
	}

	vapidSettings.VAPIDContact = config.VAPIDContact
	vapidSettings.VAPIDPublicKey = config.VAPIDPublicKey
	vapidSettings.VAPIDSecretKey = config.VAPIDSecretKey

	return vapidSettings, nil

}

func APISubscribeToNotification(context *gin.Context) {

	// Create season request
	var subscriptionRequest models.SubscriptionCreationRequest
	var subscription models.Subscription

	if err := context.ShouldBindJSON(&subscriptionRequest); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get User ID from token. Error: " + err.Error())
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
	subscription.User = userID

	_, err = database.CreateSubscriptionInDB(subscription)
	if err != nil {
		log.Println("Failed to create subscription in database. Error: " + err.Error())
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
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	subscriptions, err := database.GetAllSubscriptionsForUserByUserID(int(notificationRequest.UserID))
	if err != nil {
		log.Println("Failed to get subscriptions for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscriptions for user."})
		context.Abort()
		return
	}

	pushedAmount, err := PushNotification(notificationRequest.Category, notificationRequest.Body, notificationRequest.Title, subscriptions)
	if err != nil {
		log.Println("Failed to push notification. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to push notification."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Notification(s) pushed.", "amount": pushedAmount})

}
