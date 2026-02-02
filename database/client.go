package database

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	files "github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

var Instance *gorm.DB
var dbError error

func Connect(dbType string, timezone string, dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string, dbSSL bool, dbLocation string) error {

	if strings.ToLower(dbType) == "postgres" {
		logger.Log.Debug("attempting to connect to postgres database")

		var sslString = "disable"
		if dbSSL {
			sslString = "enabled"
		}

		connStrDb := "host=" + dbIP + " user=" + dbUsername + " password=" + dbPassword + " dbname=" + dbName + " port=" + strconv.Itoa(dbPort) + " sslmode=" + sslString + " TimeZone=" + timezone
		Instance, dbError = gorm.Open(postgres.New(postgres.Config{
			DSN:                  connStrDb,
			PreferSimpleProtocol: true,
		}), &gorm.Config{
			PrepareStmt: true,
		})
		if dbError != nil {
			logger.Log.Error("failed to connect to database. error: " + dbError.Error())
			return errors.New("failed to connect to database")
		}
	} else if strings.ToLower(dbType) == "sqlite" {
		logger.Log.Debug("attempting to connect to sqlite database")

		_, err := os.Stat(files.ConfigFile.DBLocation)
		if errors.Is(err, fs.ErrNotExist) {
			err = InitializeSQLiteDB()
			if err != nil {
				return errors.New("failed to initialize SQLite file")
			}
		} else if err != nil {
			return errors.New("failed to verify SQLite file")
		}

		dbSQL, err := sql.Open("sqlite", "file:"+files.ConfigFile.DBLocation+"?_pragma=busy_timeout(5000)")
		if err != nil {
			logger.Log.Error("failed to open database. error: " + err.Error())
			return errors.New("failed to open database")
		}

		Instance, dbError = gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
		if dbError != nil {
			logger.Log.Error("failed to connect to database. error: " + dbError.Error())
			return errors.New("failed to connect to database")
		}
	} else if strings.ToLower(dbType) == "mysql" {
		logger.Log.Debug("attempting to connect to mysql database")

		connStrDb := dbUsername + ":" + dbPassword + "@tcp(" + dbIP + ":" + strconv.Itoa(dbPort) + ")/" + dbName + "?parseTime=True&loc=Local&charset=utf8mb4"

		// Connect to DB without DB Name
		Instance, dbError = gorm.Open(mysql.Open(connStrDb), &gorm.Config{})
		if dbError != nil {

			if strings.Contains(dbError.Error(), "Unknown database '"+dbName+"'") {
				err := CreateTable(dbUsername, dbPassword, dbIP, dbPort, dbName)
				if err != nil {
					return err
				} else {
					Instance, dbError = gorm.Open(mysql.Open(connStrDb), &gorm.Config{})
					if dbError != nil {
						return dbError
					}
				}
			} else {
				logger.Log.Error("failed to connect to database. error: " + dbError.Error())
				return errors.New("failed to connect to database")
			}
		}
	} else {
		return errors.New("database type not recognized")
	}

	return nil
}

func CreateTable(dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string) error {
	url := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=disable TimeZone=%s", dbIP, strconv.Itoa(dbPort), dbUsername, dbUsername, "local")
	db, err := sql.Open("mysql", url)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
	if err != nil {
		panic(err)
	}

	return nil
}

func Migrate() {
	Instance.AutoMigrate(&models.User{})
	Instance.AutoMigrate(&models.Invite{})
	Instance.AutoMigrate(&models.Season{})
	Instance.AutoMigrate(&models.Goal{})
	Instance.AutoMigrate(&models.ExerciseDay{})
	Instance.AutoMigrate(&models.Exercise{})
	Instance.AutoMigrate(&models.Sickleave{})
	Instance.AutoMigrate(&models.News{})
	Instance.AutoMigrate(&models.Debt{})
	Instance.AutoMigrate(&models.Prize{})
	Instance.AutoMigrate(&models.Wheelview{})
	Instance.AutoMigrate(&models.Achievement{})
	Instance.AutoMigrate(&models.AchievementDelegation{})
	Instance.AutoMigrate(&models.Subscription{})
	Instance.AutoMigrate(&models.Operation{})
	Instance.AutoMigrate(&models.OperationSet{})
	Instance.AutoMigrate(&models.Action{})
	Instance.AutoMigrate(&models.WeightValue{})

	logger.Log.Info("Database migration completed.")
}

func InitializeSQLiteDB() error {
	logger.Log.Info("initializing new SQLite file at: " + files.ConfigFile.DBLocation)
	_, err := os.Create(files.ConfigFile.DBLocation)
	if err != nil {
		logger.Log.Error("failed to create DB file. error: " + err.Error())
		return errors.New("failed to create DB file")
	}
	return nil
}
