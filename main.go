package main

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/controllers"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	newPath := filepath.Join(".", "files")
	err := os.MkdirAll(newPath, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create 'files' directory. Error: " + err.Error())

		os.Exit(1)
	}
	fmt.Println("Directory 'files' valid.")

	// Create and define file for logging
	logFile, err := os.OpenFile("files/treningheten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Failed to load log file. Error: " + err.Error())

		os.Exit(1)
	}

	// Set log file as log destination
	log.SetOutput(logFile)
	log.Println("Log file set.")

	var mw io.Writer

	out := os.Stdout
	mw = io.MultiWriter(out, logFile)

	// Get pipe reader and writer | writes to pipe writer come out pipe reader
	_, w, _ := os.Pipe()

	// Replace stdout,stderr with pipe writer | all writes to stdout, stderr will go through pipe instead (log.print, log)
	os.Stdout = w
	os.Stderr = w

	// writes with log.Print should also write to mw
	log.SetOutput(mw)

	// Load config file
	configFile, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to load configuration file. Error: " + err.Error())
		os.Exit(1)
	}
	log.Println("Configuration file loaded.")

	// Set GIN mode
	if configFile.TreninghetenEnvironment != "test" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Change the config to respect flags
	configFile, generateInvite, err := parseFlags(configFile)
	if err != nil {
		log.Println("Failed to parse input flags. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Flags parsed.")

	// Set time zone from config if it is not empty
	if configFile.Timezone != "" {
		loc, err := time.LoadLocation(configFile.Timezone)
		if err != nil {
			log.Println("Failed to set time zone from config. Error: " + err.Error())
			log.Println("Removing value...")

			configFile.Timezone = ""
			err = config.SaveConfig(configFile)
			if err != nil {
				log.Println("Failed to set new time zone in the config. Error: " + err.Error())

				os.Exit(1)
			}

		} else {
			time.Local = loc
		}
	}
	log.Println("Timezone set.")

	// Initialize Database
	log.Println("Connecting to database...")

	err = database.Connect(configFile.DBUsername, configFile.DBPassword, configFile.DBIP, configFile.DBPort, configFile.DBName)
	if err != nil {
		log.Println("Failed to connect to database. Error: " + err.Error())

		os.Exit(1)
	}
	database.Migrate()

	log.Println("Database connected.")

	err = controllers.ValidateAchievements()
	if err != nil {
		log.Println("Failed to validate achievements. Error: " + err.Error())
		os.Exit(1)
	}

	if generateInvite {
		invite, err := database.GenerateRandomInvite()
		if err != nil {
			log.Println("Failed to generate random invitation code. Error: " + err.Error())

			os.Exit(1)
		}
		log.Println("Generated new invite code. Code: " + invite)
	}

	// Create task scheduler for sunday reminders
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		log.Println("Sunday reminder task executing.")
		controllers.SendSundayReminders()
	}, "0 0 18 * * 7")

	if err != nil {
		log.Println("Sunday reminder task was not scheduled successfully.")
	}

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		log.Println("Generating results for last week.")
		controllers.ProcessLastWeek()
	}, "0 0 8 * * 1")

	if err != nil {
		log.Println("Generating results for last week task was not scheduled successfully.")
	}

	if configFile.StravaEnabled {
		_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
			log.Println("Strava sync task executing.")
			controllers.StravaSyncWeekForAllUsers()
		}, "0 0 * * * *")

		if err != nil {
			log.Println("Strava sync task was not scheduled successfully. Error: " + err.Error())
		}
	}

	// Initialize Router
	router := initRouter()

	log.Println("Router initialized.")

	log.Fatal(router.Run(":" + strconv.Itoa(configFile.TreninghetenPort)))
}

func initRouter() *gin.Engine {
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

			auth.GET("/operation-sets", controllers.APIGetOperationSets)
			auth.POST("/operation-sets", controllers.APICreateOperationSetForUser)
			auth.PUT("/operation-sets/:operation_set_id", controllers.APIUpdateOperationSet)
			auth.DELETE("/operation-sets/:operation_set_id", controllers.APIDeleteOperationSet)

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

	// Static endpoint for different directories
	router.Static("/assets", "./web/assets")
	router.Static("/css", "./web/css")
	router.Static("/js", "./web/js")
	router.Static("/json", "./web/json")

	// Static endpoint for homepage
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "frontpage.html", nil)
	})

	// Static endpoint for selecting your group
	router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	// Static endpoint for selecting your group
	router.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})

	// Static endpoint for your own account
	router.GET("/account", func(c *gin.Context) {
		c.HTML(http.StatusOK, "account.html", nil)
	})

	// Static endpoint for other accounts
	router.GET("/users/:user_id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "user.html", nil)
	})

	// Static endpoint for seeing news
	router.GET("/news", func(c *gin.Context) {
		c.HTML(http.StatusOK, "news.html", nil)
	})

	// Static endpoint for seeing seasons
	router.GET("/seasons", func(c *gin.Context) {
		c.HTML(http.StatusOK, "seasons.html", nil)
	})

	// Static endpoint for seeing exercises
	router.GET("/exercises", func(c *gin.Context) {
		c.HTML(http.StatusOK, "exercises.html", nil)
	})

	// Static endpoint for seeing achievements
	router.GET("/achievements", func(c *gin.Context) {
		c.HTML(http.StatusOK, "achievements.html", nil)
	})

	// Static endpoint for admin functions
	router.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})

	// Static endpoint for wheel spinning
	router.GET("/wheel", func(c *gin.Context) {
		c.HTML(http.StatusOK, "wheel.html", nil)
	})

	// Static endpoint for season countdown
	router.GET("/countdown", func(c *gin.Context) {
		c.HTML(http.StatusOK, "countdown.html", nil)
	})

	// Static endpoint for account verification
	router.GET("/verify", func(c *gin.Context) {
		c.HTML(http.StatusOK, "verify.html", nil)
	})

	// Static endpoint for account verification
	router.GET("/oauth", func(c *gin.Context) {
		c.HTML(http.StatusOK, "oauth.html", nil)
	})

	// Static endpoint for season sign up
	router.GET("/registergoal", func(c *gin.Context) {
		c.HTML(http.StatusOK, "registergoal.html", nil)
	})

	// Static endpoint for editing exercise log
	router.GET("/exercises/:exercise_id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "exercise.html", nil)
	})

	// Static endpoint for service-worker
	router.GET("/service-worker.js", func(c *gin.Context) {
		JSfile, err := os.ReadFile("./web/js/service-worker.js")
		if err != nil {
			log.Println("Reading service-worker threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/javascript", JSfile)
	})

	// Static endpoint for manifest
	router.GET("/manifest.json", func(c *gin.Context) {
		JSONfile, err := os.ReadFile("./web/json/manifest.json")
		if err != nil {
			log.Println("Reading manifest threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/json", JSONfile)
	})

	// Static endpoint for robots.txt
	router.GET("/robots.txt", func(c *gin.Context) {
		TXTfile, err := os.ReadFile("./web/txt/robots.txt")
		if err != nil {
			log.Println("Reading manifest threw error trying to open the file. Error: " + err.Error())
		}
		c.Data(http.StatusOK, "text/plain", TXTfile)
	})

	return router
}

func parseFlags(configFile *models.ConfigStruct) (*models.ConfigStruct, bool, error) {
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

	// Save the new config
	err := config.SaveConfig(configFile)
	if err != nil {
		return &models.ConfigStruct{}, false, err
	}

	return configFile, generateInviteBool, nil
}
