package database

import (
	"aunefyren/treningheten/models"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Instance *gorm.DB
var dbError error

func Connect(dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string) error {

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
			return dbError
		}
	}

	log.Println("Connected to database.")
	fmt.Println("Connected to database.")

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

	log.Println("Database Migration Completed!")
}
