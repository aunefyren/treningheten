package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIAdminGetUsers lists every user, disabled ones included, so the admin page can
// re-enable them.
func APIAdminGetUsers(context *gin.Context) {
	users, err := database.GetAllUsersIncludingDisabled()
	if err != nil {
		logger.Log.Info("Failed to get users. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users."})
		context.Abort()
		return
	}

	adminUsers := make([]models.AdminUser, 0, len(users))
	for _, user := range users {
		adminUsers = append(adminUsers, database.ToAdminUser(user))
	}

	context.JSON(http.StatusOK, gin.H{"users": adminUsers, "message": "Users retrieved."})
}

// APIAdminSetUserEnabled enables or disables a user. The auth middleware checks the flag
// on every request, so disabling cuts off sessions, PATs, OAuth and MCP immediately.
// Disabling also takes the user out of every unfinished season so they stop counting in
// weekly results and debts; their finished seasons are left as history. Re-enabling
// restores access only — the user rejoins seasons themselves.
func APIAdminSetUserEnabled(context *gin.Context) {
	var request models.UserEnabledRequest
	if err := context.ShouldBindJSON(&request); err != nil || request.Enabled == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	userID, err := uuid.Parse(context.Param("user_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	adminID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID from token. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID from token."})
		context.Abort()
		return
	} else if adminID == userID && !*request.Enabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You cannot disable your own account."})
		context.Abort()
		return
	}

	user, err := database.GetUserByIDIncludingDisabled(userID)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	} else if user == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "User not found."})
		context.Abort()
		return
	}

	err = database.SetUserEnabled(userID, *request.Enabled)
	if err != nil {
		logger.Log.Info("Failed to update user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user."})
		context.Abort()
		return
	}
	user.Enabled = *request.Enabled

	message := "User enabled."
	if !*request.Enabled {
		// Runs on every disable, not just a state change, so a retry after a failure here
		// still finishes the job.
		goalsDisabled, err := database.DisableGoalsForUserInUnfinishedSeasons(userID, time.Now())
		if err != nil {
			logger.Log.Info("Failed to remove disabled user from seasons. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "User disabled, but failed to remove them from seasons. Try again."})
			context.Abort()
			return
		}
		message = "User disabled and removed from " + strconv.FormatInt(goalsDisabled, 10) + " season(s)."
	}

	context.JSON(http.StatusOK, gin.H{"user": database.ToAdminUser(*user), "message": message})
}
