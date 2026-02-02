package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

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
