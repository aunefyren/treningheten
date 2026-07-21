package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	files "github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
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
			// Manage relationships at the application layer, not with DB-level foreign
			// keys. AutoMigrate otherwise aborts table creation (MySQL errno 150) when a
			// new child table's FK column collation differs from an older parent table's
			// id column — which silently left gear/media_* tables uncreated. GORM still
			// uses the struct foreignKey/references tags for joins and preloads.
			DisableForeignKeyConstraintWhenMigrating: true,
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

		Instance, dbError = gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{
			// See the postgres branch: relationships are enforced at the application
			// layer, so AutoMigrate never creates DB-level foreign keys.
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if dbError != nil {
			logger.Log.Error("failed to connect to database. error: " + dbError.Error())
			return errors.New("failed to connect to database")
		}
	} else if strings.ToLower(dbType) == "mysql" {
		logger.Log.Debug("attempting to connect to mysql database")

		connStrDb := dbUsername + ":" + dbPassword + "@tcp(" + dbIP + ":" + strconv.Itoa(dbPort) + ")/" + dbName + "?parseTime=True&loc=Local&charset=utf8mb4"

		// Connect to DB without DB Name
		Instance, dbError = gorm.Open(mysql.Open(connStrDb), &gorm.Config{
			// See the postgres branch: relationships are enforced at the application
			// layer, so AutoMigrate never creates DB-level foreign keys (avoids MySQL
			// errno 150 on collation-mismatched parent/child id columns).
			DisableForeignKeyConstraintWhenMigrating: true,
		})
		if dbError != nil {

			if strings.Contains(dbError.Error(), "Unknown database '"+dbName+"'") {
				err := CreateTable(dbUsername, dbPassword, dbIP, dbPort, dbName)
				if err != nil {
					return err
				} else {
					Instance, dbError = gorm.Open(mysql.Open(connStrDb), &gorm.Config{
						// See the postgres branch: relationships are enforced at the application
						// layer, so AutoMigrate never creates DB-level foreign keys (avoids MySQL
						// errno 150 on collation-mismatched parent/child id columns).
						DisableForeignKeyConstraintWhenMigrating: true,
					})
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
	Instance.AutoMigrate(&models.Gear{})
	Instance.AutoMigrate(&models.WeightValue{})
	Instance.AutoMigrate(&models.OAuthClient{})
	Instance.AutoMigrate(&models.OAuthAuthorizationCode{})
	Instance.AutoMigrate(&models.OAuthRefreshToken{})
	Instance.AutoMigrate(&models.PersonalAccessToken{})
	Instance.AutoMigrate(&models.MediaConnection{})
	Instance.AutoMigrate(&models.MediaPlayback{})
	Instance.AutoMigrate(&models.UserActivityGoalSetting{})

	// One-time cleanup: MediaPlayback moved from per-operation to per-session. The
	// AutoMigrate above adds the NOT NULL exercise_id column, which backfills existing
	// (legacy per-operation) rows with an empty string — a value that references no
	// exercise. These rows are cheap, derived, and duplicated per operation, so drop any
	// row whose exercise_id points at no real exercise and let the next sync repopulate
	// per session. Self-limiting: once gone, this is a no-op on later boots.
	cleanup := Instance.Unscoped().
		Where("exercise_id IS NULL OR exercise_id = '' OR exercise_id NOT IN (SELECT id FROM exercises)").
		Delete(&models.MediaPlayback{})
	if cleanup.Error != nil {
		logger.Log.Warn("Failed to clean up legacy per-operation media playback rows. Error: " + cleanup.Error.Error())
	} else if cleanup.RowsAffected > 0 {
		logger.Log.Info("Removed " + strconv.FormatInt(cleanup.RowsAffected, 10) + " legacy per-operation media playback rows (will re-sync per session).")
	}

	// Drop the now-dead operation_id column. AutoMigrate never drops columns, so the old
	// per-operation column lingers — still NOT NULL and still carrying its FK to
	// operations. Session-level inserts don't set it, so it defaults to '' and every
	// insert fails that FK (errno 1452). Drop the FK first (MySQL won't drop a column a
	// FK references), then the column. No-op once gone / on fresh installs.
	migrator := Instance.Migrator()
	if migrator.HasColumn(&models.MediaPlayback{}, "operation_id") {
		if migrator.HasConstraint(&models.MediaPlayback{}, "fk_media_playbacks_operation") {
			if err := migrator.DropConstraint(&models.MediaPlayback{}, "fk_media_playbacks_operation"); err != nil {
				logger.Log.Warn("Failed to drop legacy media_playbacks operation foreign key. Error: " + err.Error())
			}
		}
		if err := migrator.DropColumn(&models.MediaPlayback{}, "operation_id"); err != nil {
			logger.Log.Warn("Failed to drop legacy media_playbacks operation_id column. Error: " + err.Error())
		} else {
			logger.Log.Info("Dropped legacy media_playbacks.operation_id column.")
		}
	}

	migrateStravaIgnoreWalksToGoalSettings()
	backfillObservedMaxHeartrate()

	logger.Log.Info("Database migration completed.")
}

// backfillObservedMaxHeartrate is a one-time backfill: it seeds each existing user's
// all-time observed max heart rate from their already-stored activity streams, so HR zones
// are well anchored from the first boot rather than only after future syncs. Self-limiting:
// new users are created with a concrete 0 (see RegisterUserInDB), so a NULL means "legacy,
// not yet computed" — once every NULL is filled the scan is skipped on later boots. Going
// forward, BumpObservedMaxHeartrate keeps the value current on each sync.
func backfillObservedMaxHeartrate() {
	var nullCount int64
	if err := Instance.Model(&models.User{}).Where("`observed_max_heartrate` IS NULL").Count(&nullCount).Error; err != nil {
		logger.Log.Warn("Failed to count users needing observed-max backfill. Error: " + err.Error())
		return
	}
	if nullCount == 0 {
		return
	}

	// Walk every stored stream once, joining up to the owning user, tracking each user's
	// peak plausible HR. Streams are JSON in a longtext column, so the max is computed in Go.
	rows, err := Instance.Table("operation_sets").
		Select("`exercise_days`.user_id AS user_id, `operation_sets`.strava_streams AS streams").
		Joins("JOIN operations on `operations`.id = `operation_sets`.operation_id").
		Joins("JOIN exercises on `exercises`.id = `operations`.exercise_id").
		Joins("JOIN exercise_days on `exercise_days`.id = `exercises`.exercise_day_id").
		Where("`operation_sets`.strava_streams IS NOT NULL AND `exercise_days`.user_id IS NOT NULL").
		Rows()
	if err != nil {
		logger.Log.Warn("Failed to scan streams for observed-max backfill. Error: " + err.Error())
		return
	}
	defer rows.Close()

	maxByUser := map[uuid.UUID]int{}
	for rows.Next() {
		var uidStr string
		var blob string
		if err := rows.Scan(&uidStr, &blob); err != nil {
			continue
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			continue
		}
		var s models.StravaStreamsJSON
		if err := json.Unmarshal([]byte(blob), &s); err != nil {
			continue
		}
		if peak := models.ObservedMaxHeartrate(&s.StravaActivityStreams); peak > maxByUser[uid] {
			maxByUser[uid] = peak
		}
	}

	// Mark every legacy row processed (0), then raise the ones we found a peak for.
	if err := Instance.Model(&models.User{}).Where("`observed_max_heartrate` IS NULL").Update("observed_max_heartrate", 0).Error; err != nil {
		logger.Log.Warn("Failed to mark users processed in observed-max backfill. Error: " + err.Error())
		return
	}
	updated := 0
	for uid, peak := range maxByUser {
		if peak <= 0 {
			continue
		}
		if err := BumpObservedMaxHeartrate(uid, peak); err == nil {
			updated++
		}
	}
	logger.Log.Info("Backfilled observed max heart rate for " + strconv.Itoa(updated) + " users.")
}

// migrateStravaIgnoreWalksToGoalSettings is a one-time backfill: the per-user Strava "ignore
// walks" flag (which skipped walk imports entirely) is replaced by per-activity-type goal
// settings (models.UserActivityGoalSetting). Each user who currently has walks ignored gets an
// explicit "Walking → doesn't count" setting, so their behaviour is preserved now that walks
// import (with streams/media) rather than being dropped. Self-limiting: the flag is cleared per
// migrated user and the column now defaults to false, so re-boots and newly created users don't
// re-trigger it.
func migrateStravaIgnoreWalksToGoalSettings() {
	if !Instance.Migrator().HasColumn(&models.User{}, "strava_walks") {
		return
	}

	var ignoreWalkUserIDs []uuid.UUID
	if err := Instance.Model(&models.User{}).Where("strava_walks = ?", true).Pluck("id", &ignoreWalkUserIDs).Error; err != nil {
		logger.Log.Warn("Failed to read strava_walks for goal-setting migration. Error: " + err.Error())
		return
	}
	if len(ignoreWalkUserIDs) == 0 {
		return
	}

	walking, actionErr := GetActionByStravaName("Walk")
	if actionErr != nil || walking == nil {
		logger.Log.Warn("Skipping strava_walks migration: could not resolve the Walking action.")
		return
	}

	migrated := 0
	for _, userID := range ignoreWalkUserIDs {
		if err := UpsertActivityGoalSettingInDB(userID, walking.ID, false); err != nil {
			logger.Log.Warn("Failed to migrate strava_walks for user " + userID.String() + ". Error: " + err.Error())
			continue
		}
		migrated++
	}
	if err := Instance.Model(&models.User{}).Where("strava_walks = ?", true).Update("strava_walks", false).Error; err != nil {
		logger.Log.Warn("Failed to clear migrated strava_walks flags. Error: " + err.Error())
	}
	logger.Log.Info("Migrated " + strconv.Itoa(migrated) + " users' 'ignore walks' preference to Walking goal settings.")
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
