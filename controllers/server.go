package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
)

func APIGetServerInfo(context *gin.Context) {
	config := files.ConfigFile

	// Only expose non-sensitive configuration. Secrets (passwords, private keys,
	// client secrets and API keys) are represented as booleans, never values.
	serverInfo := models.ServerInfoReply{
		Timezone:            config.Timezone,
		TreninghetenVersion: config.TreninghetenVersion,
		Name:                config.TreninghetenName,
		Description:         config.TreninghetenDescription,
		Environment:         config.TreninghetenEnvironment,
		ExternalURL:         config.TreninghetenExternalURL,
		LogLevel:            config.TreninghetenLogLevel,
		Port:                config.TreninghetenPort,
		Database: models.ServerInfoDatabase{
			Type:     config.DBType,
			Host:     config.DBIP,
			Port:     config.DBPort,
			Name:     config.DBName,
			Location: config.DBLocation,
			SSL:      config.DBSSL,
		},
		SMTP: models.ServerInfoSMTP{
			Enabled: config.SMTPEnabled,
			Host:    config.SMTPHost,
			Port:    config.SMTPPort,
			From:    config.SMTPFrom,
		},
		Strava: models.ServerInfoStrava{
			Enabled:     config.StravaEnabled,
			Configured:  config.StravaClientID != "" && config.StravaClientSecret != "",
			RedirectURI: config.StravaRedirectURI,
		},
		Hevy: models.ServerInfoHevy{
			Enabled: config.HevyEnabled,
		},
		AI: models.ServerInfoAI{
			Enabled:   config.Ollama.Enabled,
			URL:       config.Ollama.URL,
			Model:     config.Ollama.Model,
			APIKeySet: config.Ollama.APIKey != "",
		},
		Push: models.ServerInfoPush{
			Configured: config.VAPIDPublicKey != "" && config.VAPIDSecretKey != "",
			Contact:    config.VAPIDContact,
		},
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
