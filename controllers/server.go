package controllers

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetServerInfo(context *gin.Context) {

	config, err := config.GetConfig()
	if err != nil {
		log.Info("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	serverInfo := models.ServerInfoReply{
		Timezone:            config.Timezone,
		TreninghetenVersion: config.TreninghetenVersion,
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
