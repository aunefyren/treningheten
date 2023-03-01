package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterInvite(context *gin.Context) {

	invite, err := database.GenrateRandomInvite()
	if err != nil {
		fmt.Println("Failed to create new invite. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new invite."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Code created", "invitation": invite})
}

func APIGetAllInvites(context *gin.Context) {

	invites, err := database.GetAllEnabledInvites()
	if err != nil {
		fmt.Println("Failed to get invites from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get invites from database."})
		context.Abort()
		return
	}

	inviteObjects, err := ConvertInvitesToInviteObjects(invites)
	if err != nil {
		fmt.Println("Failed to process invites. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process invites."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Invites retrieved", "invites": inviteObjects})
}

func ConvertInviteToInviteObject(invite models.Invite) (models.InviteObject, error) {

	inviteObject := models.InviteObject{}

	if invite.InviteRecipient == 0 {
		inviteObject.User = models.User{}
	} else {
		user, err := database.GetUserInformation(invite.InviteRecipient)
		if err != nil {
			fmt.Println("Failed to get user information for user '" + strconv.Itoa(int(invite.InviteRecipient)) + "'. Returning. Error: " + err.Error())
			return models.InviteObject{}, err
		}
		inviteObject.User = user
	}

	inviteObject.ID = invite.ID
	inviteObject.CreatedAt = invite.CreatedAt
	inviteObject.DeletedAt = invite.DeletedAt
	inviteObject.UpdatedAt = invite.UpdatedAt
	inviteObject.InviteCode = invite.InviteCode
	inviteObject.InviteUsed = invite.InviteUsed
	inviteObject.InviteEnabled = invite.InviteEnabled

	return inviteObject, nil

}

func ConvertInvitesToInviteObjects(invites []models.Invite) ([]models.InviteObject, error) {

	inviteObjects := []models.InviteObject{}

	for _, invite := range invites {
		inviteObject, err := ConvertInviteToInviteObject(invite)
		if err != nil {
			fmt.Println("Failed convert invite '" + strconv.Itoa(int(invite.ID)) + "' to season object. Returning. Error: " + err.Error())
			return []models.InviteObject{}, err
		}
		inviteObjects = append(inviteObjects, inviteObject)
	}

	return inviteObjects, nil

}
