package controllers

import (
	"aunefyren/treningheten/files"
	"aunefyren/treningheten/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetServerInfo(context *gin.Context) {
	serverInfo := models.ServerInfoReply{
		Timezone:            files.ConfigFile.Timezone,
		TreninghetenVersion: files.ConfigFile.TreninghetenVersion,
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
