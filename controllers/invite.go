package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RegisterInvite(context *gin.Context) {

	invite, err := database.GenerateRandomInvite()
	if err != nil {
		logger.Log.Info("Failed to create new invite. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new invite."})
		context.Abort()
		return
	}

	invites, err := database.GetAllEnabledInvites()
	if err != nil {
		logger.Log.Info("Failed to get invites from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get invites from database."})
		context.Abort()
		return
	}

	inviteObjects, err := ConvertInvitesToInviteObjects(invites)
	if err != nil {
		logger.Log.Info("Failed to process invites. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process invites."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Invite created.", "invitation": invite, "invites": inviteObjects})
}

func APIDeleteInvite(context *gin.Context) {

	// Get ID
	var inviteID = context.Param("invite_id")

	// Parse group id
	inviteIDInt, err := uuid.Parse(inviteID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	invite, err := database.GetInviteByID(inviteIDInt)
	if err != nil {
		logger.Log.Info("Failed to find invite. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find invite."})
		context.Abort()
		return
	}

	if invite.Used {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invite already used."})
		context.Abort()
		return
	}

	err = database.DeleteInviteByID(inviteIDInt)
	if err != nil {
		logger.Log.Info("Failed to delete invite. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete invite."})
		context.Abort()
		return
	}

	invites, err := database.GetAllEnabledInvites()
	if err != nil {
		logger.Log.Info("Failed to get invites from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get invites from database."})
		context.Abort()
		return
	}

	inviteObjects, err := ConvertInvitesToInviteObjects(invites)
	if err != nil {
		logger.Log.Info("Failed to process invites. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process invites."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Invite deleted.", "invites": inviteObjects})
}

func APIGetAllInvites(context *gin.Context) {

	invites, err := database.GetAllEnabledInvites()
	if err != nil {
		logger.Log.Info("Failed to get invites from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get invites from database."})
		context.Abort()
		return
	}

	inviteObjects, err := ConvertInvitesToInviteObjects(invites)
	if err != nil {
		logger.Log.Info("Failed to process invites. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process invites."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Invites retrieved.", "invites": inviteObjects})
}

func ConvertInviteToInviteObject(invite models.Invite) (models.InviteObject, error) {

	inviteObject := models.InviteObject{}

	if invite.RecipientID == nil {
		inviteObject.Recipient = nil
	} else {
		user, err := database.GetUserInformation(*invite.RecipientID)
		if err != nil {
			logger.Log.Info("Failed to get user information for user '" + invite.Recipient.ID.String() + "'. Returning. Error: " + err.Error())
			return models.InviteObject{}, err
		}
		inviteObject.Recipient = &user
	}

	inviteObject.ID = invite.ID
	inviteObject.CreatedAt = invite.CreatedAt
	inviteObject.DeletedAt = invite.DeletedAt
	inviteObject.UpdatedAt = invite.UpdatedAt
	inviteObject.Code = invite.Code
	inviteObject.Used = invite.Used
	inviteObject.Enabled = &invite.Enabled

	return inviteObject, nil

}

func ConvertInvitesToInviteObjects(invites []models.Invite) ([]models.InviteObject, error) {

	inviteObjects := []models.InviteObject{}

	for _, invite := range invites {
		inviteObject, err := ConvertInviteToInviteObject(invite)
		if err != nil {
			logger.Log.Info("Failed convert invite '" + invite.ID.String() + "' to season object. Returning. Error: " + err.Error())
			return []models.InviteObject{}, err
		}
		inviteObjects = append(inviteObjects, inviteObject)
	}

	return inviteObjects, nil

}
