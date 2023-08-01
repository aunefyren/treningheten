package main

import (
	"aunefyren/treningheten/auth"
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/controllers"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"strconv"
	"time"

	_ "time/tzdata"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/procyon-projects/chrono"
	"github.com/thanhpk/randstr"
)

func main() {

	utilities.PrintASCII()

	// Create files directory
	newpath := filepath.Join(".", "files")
	err := os.MkdirAll(newpath, os.ModePerm)
	if err != nil {
		fmt.Println("Failed to create 'files' directory. Error: " + err.Error())

		os.Exit(1)
	}
	fmt.Println("Directory 'files' valid.")

	// Create and define file for logging
	Log, err := os.OpenFile("files/treningheten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Failed to load log file. Error: " + err.Error())

		os.Exit(1)
	}

	// Set log file as log destination
	log.SetOutput(Log)
	log.Println("Log file set.")
	fmt.Println("Log file set.")

	// Load config file
	Config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to load configuration file. Error: " + err.Error())
		fmt.Println("Failed to load configuration file. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Configuration file loaded.")
	fmt.Println("Configuration file loaded.")

	// Change the config to respect flags
	Config, generateInvite, err := parseFlags(Config)
	if err != nil {
		log.Println("Failed to parse input flags. Error: " + err.Error())
		fmt.Println("Failed to parse input flags. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Flags parsed.")
	fmt.Println("Flags parsed.")

	// Set time zone from config if it is not empty
	if Config.Timezone != "" {
		loc, err := time.LoadLocation(Config.Timezone)
		if err != nil {
			fmt.Println("Failed to set time zone from config. Error: " + err.Error())
			fmt.Println("Removing value...")

			log.Println("Failed to set time zone from config. Error: " + err.Error())
			log.Println("Removing value...")

			Config.Timezone = ""
			err = config.SaveConfig(Config)
			if err != nil {
				fmt.Println("Failed to set new time zone in the config. Error: " + err.Error())
				log.Println("Failed to set new time zone in the config. Error: " + err.Error())

				os.Exit(1)
			}

		} else {
			time.Local = loc
		}
	}
	log.Println("Timezone set.")
	fmt.Println("Timezone set.")

	if Config.PrivateKey == "" || len(Config.PrivateKey) < 16 {
		fmt.Println("Creating new private key.")
		log.Println("Creating new private key.")

		Config.PrivateKey = randstr.Hex(32)
		config.SaveConfig(Config)
	}

	err = auth.SetPrivateKey(Config.PrivateKey)
	if Config.PrivateKey == "" || len(Config.PrivateKey) < 16 {
		fmt.Println("Failed to set private key. Error: " + err.Error())
		log.Println("Failed to set private key. Error: " + err.Error())

		os.Exit(1)
	}
	log.Println("Private key set.")
	fmt.Println("Private key set.")

	// Initialize Database
	fmt.Println("Connecting to database...")
	log.Println("Connecting to database...")

	err = database.Connect(Config.DBUsername, Config.DBPassword, Config.DBIP, Config.DBPort, Config.DBName)
	if err != nil {
		fmt.Println("Failed to connect to database. Error: " + err.Error())
		log.Println("Failed to connect to database. Error: " + err.Error())

		os.Exit(1)
	}
	database.Migrate()

	log.Println("Database connected.")
	fmt.Println("Database connected.")

	if generateInvite {
		invite, err := database.GenrateRandomInvite()
		if err != nil {
			fmt.Println("Failed to generate random invitation code. Error: " + err.Error())
			log.Println("Failed to generate random invitation code. Error: " + err.Error())

			os.Exit(1)
		}
		fmt.Println("Generated new invite code. Code: " + invite)
		log.Println("Generated new invite code. Code: " + invite)
	}

	// Create task scheduler for sunday reminders
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		log.Println("Sunday reminder task executing.")
		controllers.SendSundayEmailReminder()
	}, "0 0 18 * * 7")

	if err != nil {
		log.Println("Sunday reminder task was not scheduled successfully.")
	}

	_, err = taskScheduler.ScheduleWithCron(func(ctx context.Context) {
		log.Println("Monday competition task executing.")
		controllers.GenerateLastWeeksDebt()
	}, "0 0 6 * * 1")

	if err != nil {
		log.Println("Monday competition task was not scheduled successfully.")
	}

	// Initialize Router
	router := initRouter()

	log.Println("Router initialized.")
	fmt.Println("Router initialized.")

	log.Fatal(router.Run(":" + strconv.Itoa(Config.TreninghetenPort)))

}

func initRouter() *gin.Engine {
	router := gin.Default()

	router.LoadHTMLGlob("web/*/*.html")

	// API endpoint
	api := router.Group("/api")
	{
		open := api.Group("/open")
		{
			open.POST("/token/register", controllers.GenerateToken)
			open.POST("/user/register", controllers.RegisterUser)
			open.POST("/user/reset", controllers.APIResetPassword)
			open.POST("/user/password", controllers.APIChangePassword)
		}

		auth := api.Group("/auth").Use(middlewares.Auth(false))
		{
			auth.POST("/token/validate", controllers.ValidateToken)

			auth.POST("/season", controllers.APIGetSeasons)
			auth.POST("/season/:season_id/leaderboard/", controllers.APIGetSeasonWeeks)
			auth.POST("/season/:season_id/leaderboard-personal", controllers.APIGetSeasonWeeksPersonal)
			auth.POST("/season/getongoing", controllers.APIGetOngoingSeason)
			auth.POST("/season/leaderboard", controllers.APIGetCurrentSeasonLeaderboard)

			auth.POST("/goal/register", controllers.APIRegisterGoalToSeason)
			auth.POST("/goal/delete", controllers.APIDeleteGoalToSeason)
			auth.POST("/goal", controllers.APIGetGoals)

			auth.POST("/exercise/update", controllers.APIRegisterWeek)
			auth.POST("/exercise/get", controllers.APIRGetWeek)
			auth.POST("/exercise/", controllers.APIRGetExercise)

			auth.POST("/sickleave/register", controllers.APIRegisterSickleave)

			auth.POST("/news/get", controllers.GetNews)
			auth.POST("/news/get/:news_id", controllers.GetNewsPost)

			open.POST("/user/verify/:code", controllers.VerifyUser)
			open.POST("/user/verification", controllers.SendUserVerificationCode)
			auth.POST("/user/get/:user_id", controllers.GetUser)
			auth.POST("/user/get/:user_id/image", controllers.APIGetUserProfileImage)
			auth.POST("/user/get", controllers.GetUsers)
			auth.POST("/user/update", controllers.UpdateUser)

			auth.POST("/debt/unchosen", controllers.APIGetUnchosenDebt)
			auth.POST("/debt/:debt_id", controllers.APIGetDebt)
			auth.POST("/debt/:debt_id/choose", controllers.APIChooseWinnerForDebt)
			auth.POST("/debt", controllers.APIGetDebtOverview)
			auth.POST("/debt/:debt_id/received", controllers.APISetPrizeReceived)
		}

		admin := api.Group("/admin").Use(middlewares.Auth(true))
		{
			admin.POST("/invite/register", controllers.RegisterInvite)
			admin.POST("/invite/get", controllers.APIGetAllInvites)
			admin.POST("/invite/:invite_id/delete", controllers.APIDeleteInvite)

			admin.POST("/season/register", controllers.APIRegisterSeason)

			admin.POST("/news/register", controllers.RegisterNewsPost)
			admin.POST("/news/:news_id/delete", controllers.DeleteNewsPost)

			admin.POST("/server-info", controllers.APIGetServerInfo)

			admin.POST("/debt/generate", controllers.APIGenerateDebtForWeek)

			admin.POST("/prize", controllers.APIGetPrizes)
			admin.POST("/prize/register", controllers.APIRegisterPrize)
		}

	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		// AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST"},
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
	router.GET("/user/:user_id", func(c *gin.Context) {
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

	// Static endpoint for admin functions
	router.GET("/admin", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})

	// Static endpoint for wheel spinning
	router.GET("/wheel", func(c *gin.Context) {
		c.HTML(http.StatusOK, "wheel.html", nil)
	})

	return router
}

func parseFlags(Config *models.ConfigStruct) (*models.ConfigStruct, bool, error) {

	// Define flag variables with the configuration file as default values
	var port int
	flag.IntVar(&port, "port", Config.TreninghetenPort, "The port Treningheten is listening on.")

	var externalURL string
	flag.StringVar(&externalURL, "externalurl", Config.TreninghetenExternalURL, "The URL others would use to access Treningheten.")

	var timezone string
	flag.StringVar(&timezone, "timezone", Config.Timezone, "The timezone Treningheten is running in.")

	var dbPort int
	flag.IntVar(&dbPort, "dbport", Config.DBPort, "The port the database is listening on.")

	var dbUsername string
	flag.StringVar(&dbUsername, "dbusername", Config.DBUsername, "The username used to interact with the database.")

	var dbPassword string
	flag.StringVar(&dbPassword, "dbpassword", Config.DBPassword, "The password used to interact with the database.")

	var dbName string
	flag.StringVar(&dbName, "dbname", Config.DBName, "The database table used within the database.")

	var dbIP string
	flag.StringVar(&dbIP, "dbip", Config.DBIP, "The IP address used to reach the database.")

	var smtpDisabled string
	flag.StringVar(&smtpDisabled, "disablesmtp", "false", "Disables user verification using e-mail.")

	var smtpHost string
	flag.StringVar(&smtpHost, "smtphost", Config.SMTPHost, "The SMTP server which sends e-mail.")

	var smtpPort int
	flag.IntVar(&smtpPort, "smtpport", Config.SMTPPort, "The SMTP server port.")

	var smtpUsername string
	flag.StringVar(&smtpUsername, "smtpusername", Config.SMTPUsername, "The username used to verify against the SMTP server.")

	var smtpPassword string
	flag.StringVar(&smtpPassword, "smtppassword", Config.SMTPPassword, "The password used to verify against the SMTP server.")

	var smtpFrom string
	flag.StringVar(&smtpFrom, "smtpfrom", Config.SMTPFrom, "The sender address when sending e-mail from Treningheten.")

	var generateInvite string
	var generateInviteBool bool
	flag.StringVar(&generateInvite, "generateinvite", "false", "If an invite code should be automatically generate on startup.")

	// Parse the flags from input
	flag.Parse()

	// Respect the flag if config is empty
	if Config.TreninghetenPort == 0 {
		Config.TreninghetenPort = port
	}

	// Respect the flag if config is empty
	if Config.TreninghetenExternalURL == "" {
		Config.TreninghetenExternalURL = externalURL
	}

	// Respect the flag if config is empty
	if Config.Timezone == "" {
		Config.Timezone = timezone
	}

	// Respect the flag if config is empty
	if Config.DBPort == 0 {
		Config.DBPort = dbPort
	}

	// Respect the flag if config is empty
	if Config.DBUsername == "" {
		Config.DBUsername = dbUsername
	}

	// Respect the flag if config is empty
	if Config.DBPassword == "" {
		Config.DBPassword = dbPassword
	}

	// Respect the flag if config is empty
	if Config.DBName == "" {
		Config.DBName = dbName
	}

	// Respect the flag if config is empty
	if Config.DBIP == "" {
		Config.DBIP = dbIP
	}

	// Respect the flag if string is true
	if strings.ToLower(smtpDisabled) == "true" {
		Config.SMTPEnabled = false
	}

	// Respect the flag if config is empty
	if Config.SMTPHost == "" {
		Config.SMTPHost = smtpHost
	}

	// Respect the flag if config is empty
	if Config.SMTPPort == 0 {
		Config.SMTPPort = smtpPort
	}

	// Respect the flag if config is empty
	if Config.SMTPUsername == "" {
		Config.SMTPUsername = smtpUsername
	}

	// Respect the flag if config is empty
	if Config.SMTPPassword == "" {
		Config.SMTPPassword = smtpPassword
	}

	// Respect the flag if config is empty
	if Config.SMTPFrom == "" {
		Config.SMTPFrom = smtpFrom
	}

	// Respect the flag if string is true
	if strings.ToLower(generateInvite) == "true" {
		generateInviteBool = true
	} else {
		generateInviteBool = false
	}

	// Failsafe, if port is 0, set to default 8080
	if Config.TreninghetenPort == 0 {
		Config.TreninghetenPort = 8080
	}

	// Save the new config
	err := config.SaveConfig(Config)
	if err != nil {
		return &models.ConfigStruct{}, false, err
	}

	return Config, generateInviteBool, nil

}
