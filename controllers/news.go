package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/logger"
	"aunefyren/treningheten/models"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetNews(context *gin.Context) {

	// Get all enabled news
	newsPosts, err := database.GetNewsPosts()
	if err != nil {
		// If there is an error getting the list of news, return an internal server error
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return a response with all news posts
	context.JSON(http.StatusCreated, gin.H{"message": "News retrieved.", "news": newsPosts})
}

func GetNewsPost(context *gin.Context) {

	var newsID = context.Param("news_id")

	newsIDInt, err := uuid.Parse(newsID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get the news post by id
	newsPost, err := database.GetNewsPostByNewsID(newsIDInt)
	if err != nil {
		// If there is an error getting the news, return an internal server error
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return a response with all news posts
	context.JSON(http.StatusCreated, gin.H{"message": "News retrieved.", "news": newsPost})
}

func RegisterNewsPost(context *gin.Context) {
	// Create a new instance of the News and NewsCreationRequest models
	var news models.News
	var newsCreationRequest models.NewsCreationRequest

	// Bind the incoming request body to the NewsCreationRequest model
	if err := context.ShouldBindJSON(&newsCreationRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Copy the data from the NewsCreationRequest model to the News model
	news.Title = strings.TrimSpace(newsCreationRequest.Title)
	news.Body = strings.TrimSpace(newsCreationRequest.Body)

	// Verify that the News title is not empty and has at least 5 characters
	if len(news.Title) < 5 || news.Title == "" {
		// If the group name is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The title of the news post must be five or more letters."})
		context.Abort()
		return
	}

	if len(news.Body) < 5 || news.Body == "" {
		// If the News body is not valid, return a Bad Request response
		context.JSON(http.StatusBadRequest, gin.H{"error": "The body of the news post must be five or more letters."})
		context.Abort()
		return
	}

	news.Date = time.Now()
	news.ID = uuid.New()

	// Create the news post in the database
	newsRecord := database.Instance.Create(&news)
	if newsRecord.Error != nil {
		// If there is an error creating the news, return an Internal Server Error response
		context.JSON(http.StatusInternalServerError, gin.H{"error": newsRecord.Error.Error()})
		context.Abort()
		return
	}

	newsPosts, err := database.GetNewsPosts()
	if err != nil {
		// If there is an error getting the list of news, return an internal server error
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	err = PushNotificationsForNews()
	if err != nil {
		logger.Log.Info("Failed to push notifications for news post.")
	}

	// Return a response indicating that the group was created, along with the updated list of groups
	context.JSON(http.StatusCreated, gin.H{"message": "News post created.", "news": newsPosts})
}

func DeleteNewsPost(context *gin.Context) {

	// Bind news request and get news_id ID from URL parameter
	newsID := context.Param("news_id")

	// Parse news ID as integer
	newsIDInt, err := uuid.Parse(newsID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify that news post exists
	_, err = database.GetNewsPostByNewsID(newsIDInt)
	if err != nil {
		// If there is an error getting the news, return an internal server error
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Set the news post to disabled in the database
	err = database.DeleteNewsPost(newsIDInt)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get updated list of news
	newsPosts, err := database.GetNewsPosts()
	if err != nil {
		// If there is an error getting the list of news, return an internal server error
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "News post deleted.", "news": newsPosts})

}
