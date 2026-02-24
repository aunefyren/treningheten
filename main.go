package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	textTemplate "text/template"

	"github.com/aunefyren/treningheten/controllers"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"strconv"
	"time"

	_ "time/tzdata"

	"codnect.io/chrono"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	utilities.PrintASCII()

	// Create files directory
	newPath := filepath.Join(".", "config")
	err := os.MkdirAll(newPath, os.ModePerm)
	if err != nil {
		fmt.Println("failed to create 'files' directory. error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("directory 'files' valid")

	// Load config file
	err = files.LoadConfig()
	if err != nil {
		fmt.Println("failed to load configuration file. error: " + err.Error())
		os.Exit(1)
	}
	fmt.Println("configuration file loaded")

	// Create and define file for logging
	logger.InitLogger(files.ConfigFile.TreninghetenLogLevel)

	// Set GIN mode
	if files.ConfigFile.TreninghetenEnvironment != "test" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Change the config to respect flags
	generateInvite := false
	files.ConfigFile, generateInvite, err = parseFlags(files.ConfigFile)
	if err != nil {
		logger.Log.Fatal("failed to parse input flags. error: " + err.Error())
		os.Exit(1)
	}
	logger.Log.Info("flags parsed")

	// save new version of config
	err = files.SaveConfig()
	if err != nil {
		logger.Log.Error("failed to save new config. error: " + err.Error())
		os.Exit(1)
	}

	// Set time zone from config if it is not empty
	if files.ConfigFile.Timezone != "" {
		loc, err := time.LoadLocation(files.ConfigFile.Timezone)
		if err != nil {
			logger.Log.Info("failed to set time zone from config. error: " + err.Error())
			logger.Log.Info("removing value...")

			files.ConfigFile.Timezone = ""
			err = files.SaveConfig()
			if err != nil {
				logger.Log.Fatal("failed to set new time zone in the config. error: " + err.Error())
				os.Exit(1)
			}

		} else {
			time.Local = loc
		}
	}
	logger.Log.Info("timezone set")

	// Initialize Database
	logger.Log.Info("connecting to database...")

	err = database.Connect(
		files.ConfigFile.DBType,
		files.ConfigFile.Timezone,
		files.ConfigFile.DBUsername,
		files.ConfigFile.DBPassword,
		files.ConfigFile.DBIP,
		files.ConfigFile.DBPort,
		files.ConfigFile.DBName,
		files.ConfigFile.DBSSL,
		files.ConfigFile.DBLocation)
	if err != nil {
		logger.Log.Fatal("failed to connect to database. error: " + err.Error())
		os.Exit(1)
	}
	database.Migrate()

	logger.Log.Info("database connected")

	err = controllers.ValidateAchievements()
	if err != nil {
		logger.Log.Info("failed to validate achievements. error: " + err.Error())
		os.Exit(1)
	}

	if generateInvite {
		invite, err := database.GenerateRandomInvite()
		if err != nil {
			logger.Log.Fatal("failed to generate random invitation code. error: " + err.Error())
			os.Exit(1)
		}
		logger.Log.Info("generated new invite code. code: " + invite)
	}

	// Create task scheduler for sunday reminders
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		logger.Log.Info("sunday reminder task executing")
		controllers.SendSundayReminders()
	}, "0 0 18 * * 7")

	if err != nil {
		logger.Log.Info("sunday reminder task was not scheduled successfully")
	}

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		logger.Log.Info("generating results for last week")
		controllers.ProcessLastWeek()
	}, "0 0 8 * * 1")

	if err != nil {
		logger.Log.Info("generating results for last week task was not scheduled successfully")
	}

	if files.ConfigFile.StravaEnabled {
		_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
			logger.Log.Info("strava sync task executing")
			controllers.StravaSyncWeekForAllUsers()
		}, "0 0 * * * *")

		if err != nil {
			logger.Log.Info("strava sync task was not scheduled successfully. error: " + err.Error())
		}
	}

	// Initialize Router
	router := initRouter(files.ConfigFile)

	logger.Log.Info("Router initialized.")

	log.Fatal(router.Run(":" + strconv.Itoa(files.ConfigFile.TreninghetenPort)))
}

func initRouter(configFile models.ConfigStruct) *gin.Engine {
	router := gin.Default()

	router.LoadHTMLGlob("web/*/*.html")

	// API endpoint
	api := router.Group("/api")
	{
		open := api.Group("/open")
		{
			open.POST("/tokens/register", controllers.GenerateToken)

			open.POST("/users", controllers.RegisterUser)
			open.POST("/users/reset", controllers.APIResetPassword)
			open.GET("/users/reset/:resetCode", controllers.APIVerifyResetCode)
			open.POST("/users/password", controllers.APIChangePassword)

			open.POST("/users/verify/:code", controllers.VerifyUser)
			open.POST("/users/verification", controllers.SendUserVerificationCode)
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.POST("/tokens/validate", controllers.ValidateToken)

			auth.GET("/seasons", controllers.APIGetSeasons)
			auth.GET("/seasons/:season_id", controllers.APIGetSeason)
			auth.GET("/seasons/:season_id/weeks", controllers.APIGetSeasonWeeks)
			auth.GET("/seasons/:season_id/weeks-personal", controllers.APIGetSeasonWeeksPersonal)
			auth.GET("/seasons/get-on-going", controllers.APIGetOngoingSeasons)
			auth.GET("/seasons/:season_id/leaderboard", controllers.APIGetCurrentSeasonLeaderboard)
			auth.GET("/seasons/:season_id/activities", controllers.APIGetCurrentSeasonActivities)

			auth.POST("/goals", controllers.APIRegisterGoalToSeason)
			auth.DELETE("/goals/:goal_id", controllers.APIDeleteGoalToSeason)
			auth.GET("/goals", controllers.APIGetGoals)

			auth.GET("/exercise-days", controllers.APIGetExerciseDays)
			auth.GET("/exercise-days/:exercise_day_id", controllers.APIGetExerciseDay)
			auth.POST("/exercise-days/:exercise_day_id", controllers.APIUpdateExerciseDay)

			auth.POST("/exercises/week", controllers.APIRegisterWeek)
			auth.GET("/exercises/week", controllers.APIGetWeek)
			auth.POST("/exercises", controllers.APICreateExercise)
			auth.PUT("/exercises/:exercise_id", controllers.APIUpdateExercise)
			auth.POST("/exercises/:exercise_id/strava-divide", controllers.APIStravaDivide)
			auth.POST("/exercises/strava-combine", controllers.APIStravaCombine)

			auth.GET("/operations", controllers.APIGetOperationsForUser)
			auth.POST("/operations", controllers.APICreateOperationForUser)
			auth.GET("/operations/:operation_id", controllers.APIGetOperation)
			auth.PUT("/operations/:operation_id", controllers.APIUpdateOperation)
			auth.DELETE("/operations/:operation_id", controllers.APIDeleteOperation)

			auth.GET("/actions", controllers.APIGetActions)
			auth.POST("/actions", controllers.APICreateAction)
			auth.GET("/actions/:action_id/statistics", controllers.APIGetActionStatistics)

			auth.GET("/operation-sets", controllers.APIGetOperationSets)
			auth.POST("/operation-sets", controllers.APICreateOperationSetForUser)
			auth.PUT("/operation-sets/:operation_set_id", controllers.APIUpdateOperationSet)
			auth.DELETE("/operation-sets/:operation_set_id", controllers.APIDeleteOperationSet)

			auth.GET("/weights", controllers.APIGetWeightsForUser)
			auth.GET("/weights/:weight_id", controllers.APIGetWeightForUser)
			auth.POST("/weights", controllers.APICreateWeightForUser)
			auth.DELETE("/weights/:weight_id", controllers.APIDeleteWeightForUser)

			auth.POST("/sickleave/:season_id", controllers.APIRegisterSickleave)

			auth.GET("/news", controllers.GetNews)
			auth.GET("/news/:news_id", controllers.GetNewsPost)

			auth.GET("/users/:user_id", controllers.GetUser)
			auth.POST("/users/:user_id/strava", controllers.APISetStravaCode)
			auth.POST("/users/:user_id/strava-sync", controllers.APISyncStravaForUser)
			auth.GET("/users/:user_id/image", controllers.APIGetUserProfileImage)
			auth.GET("/users", controllers.GetUsers)
			auth.POST("/users/:user_id", controllers.UpdateUser)
			auth.PATCH("/users/:user_id", controllers.APIPartialUpdateUser)

			auth.GET("/debts/unchosen", controllers.APIGetUnchosenDebt)
			auth.GET("/debts/:debt_id", controllers.APIGetDebt)
			auth.POST("/debts/:debt_id/choose", controllers.APIChooseWinnerForDebt)
			auth.GET("/debts", controllers.APIGetDebtOverview)
			auth.POST("/debts/:debt_id/received", controllers.APISetPrizeReceived)

			auth.GET("/achievements", controllers.APIGetAchievements)
			auth.GET("/achievements/:achievement_id/image", controllers.APIGetAchievementsImage)

			auth.POST("/notifications/subscribe", controllers.APISubscribeToNotification)
			auth.POST("/notifications/subscription", controllers.APIGetSubscriptionForEndpoint)
			auth.POST("/notifications/subscription/update", controllers.APIUpdateSubscriptionForEndpoint)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/invites", controllers.RegisterInvite)
			admin.GET("/invites", controllers.APIGetAllInvites)
			admin.DELETE("/invites/:invite_id", controllers.APIDeleteInvite)

			admin.POST("/seasons", controllers.APIRegisterSeason)

			admin.POST("/news", controllers.RegisterNewsPost)
			admin.DELETE("/news/:news_id", controllers.DeleteNewsPost)

			admin.GET("/server-info", controllers.APIGetServerInfo)

			admin.GET("/exercise-days", controllers.APIAdminGetExerciseDays)

			admin.POST("/debts", controllers.APIGenerateDebtForWeek)

			admin.POST("/users/:user_id/achievement-delegations", controllers.APIGiveUserAnAchievement)

			admin.GET("/prizes", controllers.APIGetPrizes)
			admin.POST("/prizes", controllers.APIRegisterPrize)

			admin.POST("/notifications/push/all-devices", controllers.APIPushNotificationToAllDevicesForUser)

			admin.POST("/exercises/correlate", controllers.APICorrelateAllExercises)
		}

	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		// AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Access-Control-Allow-Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc:  func(origin string) bool { return true },
		MaxAge:           12 * time.Hour,
	}))

	// load HTML blobs
	router.LoadHTMLGlob("./web/*/*.html")

	// Static endpoint for different directories
	router.Static("/assets", "./web/assets")
	router.Static("/css", "./web/css")

	// Create template for HTML variables
	templateData := gin.H{
		"appName":        configFile.TreninghetenName,
		"appDescription": configFile.TreninghetenName,
	}

	// endpoint handler building for JS
	router, err := registerTemplatedStaticFilesForDirectory(router, "/js", true, "./web/js", templateData)
	if err != nil {
		logger.Log.Error("failed to build JS paths. error: " + err.Error())
	}

	// endpoint handler building for HTML
	router, err = registerTemplatedStaticFilesForDirectory(router, "", false, "./web/html", templateData)
	if err != nil {
		logger.Log.Error("failed to build HTML paths. error: " + err.Error())
	}

	// endpoint handler building for JSON
	router, err = registerTemplatedStaticFilesForDirectory(router, "/json", true, "./web/json", templateData)
	if err != nil {
		logger.Log.Error("failed to build JSON paths. error: " + err.Error())
	}

	// endpoint handler building for TXT
	router, err = registerTemplatedStaticFilesForDirectory(router, "/txt", true, "./web/txt", templateData)
	if err != nil {
		logger.Log.Error("failed to build TXT paths. error: " + err.Error())
	}

	return router

	return router
}

func parseFlags(configFile models.ConfigStruct) (models.ConfigStruct, bool, error) {
	// Define flag variables with the configuration file as default values
	var port = flag.Int("port", configFile.TreninghetenPort, "The port Treningheten is listening on.")
	var externalURL = flag.String("externalurl", configFile.TreninghetenExternalURL, "The URL others would use to access Treningheten.")
	var timezone = flag.String("timezone", configFile.Timezone, "The timezone Treningheten is running in.")

	// Timezone flags
	var dbPort = flag.Int("dbport", configFile.DBPort, "The port the database is listening on.")
	var dbUsername = flag.String("dbusername", configFile.DBUsername, "The username used to interact with the database.")
	var dbPassword = flag.String("dbpassword", configFile.DBPassword, "The password used to interact with the database.")
	var dbName = flag.String("dbname", configFile.DBName, "The database table used within the database.")
	var dbIP = flag.String("dbip", configFile.DBIP, "The IP address used to reach the database.")

	// SMTP flags
	var smtpDisabled = flag.String("disablesmtp", "false", "Disables user verification using e-mail.")
	var smtpHost = flag.String("smtphost", configFile.SMTPHost, "The SMTP server which sends e-mail.")
	var smtpPort = flag.Int("smtpport", configFile.SMTPPort, "The SMTP server port.")
	var smtpUsername = flag.String("smtpusername", configFile.SMTPUsername, "The username used to verify against the SMTP server.")
	var smtpPassword = flag.String("smtppassword", configFile.SMTPPassword, "The password used to verify against the SMTP server.")
	var smtpFrom = flag.String("smtpfrom", configFile.SMTPFrom, "The sender address when sending e-mail from Treningheten.")

	// Generate invite flag
	var generateInvite = flag.String("generateinvite", "false", "If an invite code should be automatically generate on startup.")

	// Parse the flags from input
	flag.Parse()

	// Respect the flag if config is empty
	if port != nil {
		configFile.TreninghetenPort = *port
	}

	// Respect the flag if config is empty
	if externalURL == nil {
		configFile.TreninghetenExternalURL = *externalURL
	}

	// Respect the flag if config is empty
	if timezone == nil {
		configFile.Timezone = *timezone
	}

	// Respect the flag if config is empty
	if dbPort != nil {
		configFile.DBPort = *dbPort
	}

	// Respect the flag if config is empty
	if dbUsername != nil {
		configFile.DBUsername = *dbUsername
	}

	// Respect the flag if config is empty
	if dbPassword != nil {
		configFile.DBPassword = *dbPassword
	}

	// Respect the flag if config is empty
	if dbName != nil {
		configFile.DBName = *dbName
	}

	// Respect the flag if config is empty
	if dbIP != nil {
		configFile.DBIP = *dbIP
	}

	// Respect the flag if string is true
	if smtpDisabled != nil && strings.ToLower(*smtpDisabled) == "true" {
		configFile.SMTPEnabled = false
	} else {
		configFile.SMTPEnabled = true
	}

	// Respect the flag if config is empty
	if smtpHost != nil {
		configFile.SMTPHost = *smtpHost
	}

	// Respect the flag if config is empty
	if smtpPort != nil {
		configFile.SMTPPort = *smtpPort
	}

	// Respect the flag if config is empty
	if smtpUsername != nil {
		configFile.SMTPUsername = *smtpUsername
	}

	// Respect the flag if config is empty
	if smtpPassword != nil {
		configFile.SMTPPassword = *smtpPassword
	}

	// Respect the flag if config is empty
	if smtpFrom != nil {
		configFile.SMTPFrom = *smtpFrom
	}

	// Respect the flag if string is true
	var generateInviteBool = false
	if generateInvite != nil && strings.ToLower(*generateInvite) == "true" {
		generateInviteBool = true
	}

	// Failsafe, if port is 0, set to default 8080
	if configFile.TreninghetenPort == 0 {
		configFile.TreninghetenPort = 8080
	}

	return configFile, generateInviteBool, nil
}

func registerTemplatedStaticFilesForDirectory(
	r *gin.Engine,
	urlPrefix string,
	keepFileExtension bool,
	fileDirectory string,
	templateData any,
) (
	newRouter *gin.Engine,
	err error,
) {
	type filePathTable struct {
		urlPath string
	}
	filePathTableList := map[string]*filePathTable{}
	filePathTableList["frontpage.html"] = &filePathTable{urlPath: "/"}
	filePathTableList["user.html"] = &filePathTable{urlPath: "/users/:user_id"}
	filePathTableList["manifest.json"] = &filePathTable{urlPath: "/manifest.json"}
	filePathTableList["service-worker.js"] = &filePathTable{urlPath: "/service-worker.js"}
	filePathTableList["robots.txt"] = &filePathTable{urlPath: "/robots.txt"}

	root := os.DirFS(fileDirectory)
	jsTemplates := MustLoadTemplates("./web/js/*.js")
	jsonTemplates := MustLoadTemplates("./web/json/*.json")
	txtTemplates := MustLoadTemplates("./web/txt/*.txt")

	foundFiles, err := fs.Glob(root, "*")

	if err != nil {
		logger.Log.Error("failed to load directory. error: " + err.Error())
		return
	}

	logger.Log.Debug("found " + strconv.Itoa(len(foundFiles)) + " files for endpoint mapping using path: " + fileDirectory)

	for _, file := range foundFiles {
		fileWithoutExtension := file
		extension := filepath.Ext(file)

		if !keepFileExtension && strings.Contains(file, ".") {
			fileWithoutExtension = strings.TrimSuffix(file, extension)
		}

		path := urlPrefix + "/" + fileWithoutExtension

		if filePathTableList[file] != nil {
			path = filePathTableList[file].urlPath
		}

		switch strings.ToLower(extension) {
		case ".html":
			r.GET(path, func(c *gin.Context) {
				c.HTML(http.StatusOK, file, templateData)
			})
			logger.Log.Debug("registered HTML '" + file + "' to path '" + path + "'")
		case ".js":
			r.GET(path, RenderJSTemplate(jsTemplates, file, templateData))
			logger.Log.Debug("registered JS '" + file + "' to path '" + path + "'")
		case ".json":
			r.GET(path, RenderJSONTemplate(jsonTemplates, file, templateData))
			logger.Log.Debug("registered JSON '" + file + "' to path '" + path + "'")
		case ".txt":
			r.GET(path, RenderTextTemplate(txtTemplates, file, templateData))
			logger.Log.Debug("registered TXT '" + file + "' to path '" + path + "'")
		}
	}

	return r, err
}

func MustLoadTemplates(glob string) *textTemplate.Template {
	t, err := textTemplate.ParseGlob(glob)
	if err != nil {
		logger.Log.Warn("failed to parse file: " + glob)
	}
	return t
}

func RenderJSTemplate(jsTemplates *textTemplate.Template, name string, data any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		if err := jsTemplates.ExecuteTemplate(&buf, name, data); err != nil {
			c.String(http.StatusInternalServerError, "template error: %v", err)
			return
		}

		c.Data(http.StatusOK, "application/javascript; charset=utf-8", buf.Bytes())
	}
}

func RenderJSONTemplate(jsTemplates *textTemplate.Template, name string, data any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		if err := jsTemplates.ExecuteTemplate(&buf, name, data); err != nil {
			c.String(http.StatusInternalServerError, "template error: %v", err)
			return
		}

		c.Data(http.StatusOK, "application/json; charset=utf-8", buf.Bytes())
	}
}

func RenderTextTemplate(jsTemplates *textTemplate.Template, name string, data any) gin.HandlerFunc {
	return func(c *gin.Context) {
		var buf bytes.Buffer
		if err := jsTemplates.ExecuteTemplate(&buf, name, data); err != nil {
			c.String(http.StatusInternalServerError, "template error: %v", err)
			return
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", buf.Bytes())
	}
}
