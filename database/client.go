package database

import (
	"database/sql"
	"encoding/json"
	"errors"
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

		connStrDb := postgresDSN(dbIP, dbPort, dbUsername, dbPassword, dbName, dbSSL, timezone)
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

		// clientFoundRows: MySQL otherwise reports an UPDATE's affected rows as the rows it
		// *changed*, so re-saving an unchanged value (within the same millisecond of
		// updated_at) reads as 0 rows and fails every "RowsAffected != 1" check. With it,
		// MySQL counts matched rows like SQLite and Postgres. See docs/conventions.md.
		connStrDb := dbUsername + ":" + dbPassword + "@tcp(" + dbIP + ":" + strconv.Itoa(dbPort) + ")/" + dbName + "?parseTime=True&loc=Local&charset=utf8mb4&clientFoundRows=true"

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

// postgresDSN builds a libpq key/value connection string. Values are single-quoted with
// backslash escaping, so a password containing spaces or quotes stays one value.
func postgresDSN(host string, port int, username string, password string, dbName string, ssl bool, timezone string) string {
	sslMode := "disable"
	if ssl {
		sslMode = "require"
	}

	quote := func(value string) string {
		value = strings.ReplaceAll(value, `\`, `\\`)
		return "'" + strings.ReplaceAll(value, "'", `\'`) + "'"
	}

	return "host=" + quote(host) +
		" port=" + strconv.Itoa(port) +
		" user=" + quote(username) +
		" password=" + quote(password) +
		" dbname=" + quote(dbName) +
		" sslmode=" + sslMode +
		" TimeZone=" + quote(timezone)
}

// CreateTable creates the MySQL database when it doesn't exist yet. It connects without
// a database name (a MySQL DSN, not the key=value form Postgres uses — the old version
// built the Postgres form, passed the username as the password and panicked, so this
// path never worked).
func CreateTable(dbUsername string, dbPassword string, dbIP string, dbPort int, dbName string) error {
	dsn := dbUsername + ":" + dbPassword + "@tcp(" + dbIP + ":" + strconv.Itoa(dbPort) + ")/?charset=utf8mb4"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return errors.New("failed to open MySQL connection: " + err.Error())
	}
	defer db.Close()

	if _, err := db.Exec("CREATE DATABASE `" + strings.ReplaceAll(dbName, "`", "``") + "`"); err != nil {
		return errors.New("failed to create database: " + err.Error())
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
	Instance.AutoMigrate(&models.UserSportMaxHeartrate{})
	Instance.AutoMigrate(&models.IntegrationStatus{})
	Instance.AutoMigrate(&models.BestEffort{})

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
	migrateStravaSkipHevyToSourcePriority()
	backfillObservedMaxHeartrate()
	backfillStreamDerivedData()
	mergeDuplicateExerciseDays()
	capListenedTimeToPlaySpan()
	backfillActionCategories()
	backfillHevyOperationNotes()

	logger.Log.Info("Database migration completed.")
}

// backfillStreamDerivedData brings every stream-bearing activity up to
// models.StreamDerivedVersion: it recomputes the rollup columns (heart rate, cadence,
// temperature, elevation gain, average power) and the best efforts from the stored stream,
// then stamps the version. Self-limiting: once every such operation carries the current
// version the count is zero and nothing is read. A stream that fails to parse is stamped too,
// so it can't keep the scan alive. Going forward, the Strava sync writes all of it on import.
func backfillStreamDerivedData() {
	var pending int64
	if err := Instance.Table("operations").
		Joins("JOIN operation_sets ON operation_sets.operation_id = operations.id").
		Where("operation_sets.strava_streams IS NOT NULL").
		Where("operations.stream_derived_version < ?", models.StreamDerivedVersion).
		Count(&pending).Error; err != nil {
		logger.Log.Warn("Failed to count operations needing stream-derived backfill. Error: " + err.Error())
		return
	}
	if pending == 0 {
		return
	}

	// One row per stream-bearing set; the first stream seen for an operation wins, mirroring the
	// runtime's "first set with streams" selection.
	rows, err := Instance.Table("operation_sets").
		Select("operations.id AS operation_id, exercise_days.user_id AS user_id, COALESCE(actions.strava_name, '') AS strava_name, operation_sets.strava_streams AS streams").
		Joins("JOIN operations ON operations.id = operation_sets.operation_id").
		Joins("LEFT JOIN actions ON actions.id = operations.action_id").
		Joins("LEFT JOIN exercises ON exercises.id = operations.exercise_id").
		Joins("LEFT JOIN exercise_days ON exercise_days.id = exercises.exercise_day_id").
		Where("operation_sets.strava_streams IS NOT NULL").
		Where("operations.stream_derived_version < ?", models.StreamDerivedVersion).
		Rows()
	if err != nil {
		logger.Log.Warn("Failed to scan streams for stream-derived backfill. Error: " + err.Error())
		return
	}
	defer rows.Close()

	type derived struct {
		userID  *uuid.UUID
		parsed  bool
		rollup  models.OperationStreamRollup
		efforts []models.BestEffortResult
	}
	results := map[uuid.UUID]derived{}
	for rows.Next() {
		var opIDStr string
		var userIDStr *string
		var stravaName string
		var blob string
		if err := rows.Scan(&opIDStr, &userIDStr, &stravaName, &blob); err != nil {
			continue
		}
		opID, err := uuid.Parse(opIDStr)
		if err != nil {
			continue
		}
		if _, done := results[opID]; done {
			continue
		}
		result := derived{}
		if userIDStr != nil {
			if userID, err := uuid.Parse(*userIDStr); err == nil {
				result.userID = &userID
			}
		}
		var s models.StravaStreamsJSON
		if err := json.Unmarshal([]byte(blob), &s); err == nil {
			result.parsed = true
			result.rollup = models.ComputeStreamRollup(&s.StravaActivityStreams)
			result.efforts = models.ComputeBestEfforts(&s.StravaActivityStreams, stravaName)
		}
		results[opID] = result
	}
	rows.Close()

	updated := 0
	for opID, result := range results {
		fields := map[string]interface{}{"stream_derived_version": models.StreamDerivedVersion}
		if result.parsed {
			fields["avg_heartrate"] = result.rollup.AvgHeartrate
			fields["max_heartrate"] = result.rollup.MaxHeartrate
			fields["avg_cadence"] = result.rollup.AvgCadence
			fields["temp_c"] = result.rollup.TempC
			fields["elevation_gain_m"] = result.rollup.ElevationGainM
			fields["avg_watts"] = result.rollup.AvgWatts
		}
		if err := Instance.Model(&models.Operation{}).Where("id = ?", opID).Updates(fields).Error; err != nil {
			logger.Log.Warn("Failed to write stream-derived data for operation " + opID.String() + ". Error: " + err.Error())
			continue
		}
		if result.parsed && result.userID != nil {
			if err := ReplaceBestEffortsForOperation(opID, *result.userID, result.efforts); err != nil {
				logger.Log.Warn("Failed to write best efforts for operation " + opID.String() + ". Error: " + err.Error())
			}
		}
		updated++
	}
	logger.Log.Info("Backfilled stream-derived data for " + strconv.Itoa(updated) + " operations.")
}

// backfillObservedMaxHeartrate is a one-time backfill: it seeds each existing user's
// all-time observed max heart rate from their already-stored activity streams, so HR zones
// are well anchored from the first boot rather than only after future syncs. Self-limiting:
// new users are created with a concrete 0 (see RegisterUserInDB), so a NULL means "legacy,
// not yet computed" — once every NULL is filled the scan is skipped on later boots. Going
// forward, BumpObservedMaxHeartrate keeps the value current on each sync.
func backfillObservedMaxHeartrate() {
	var nullCount int64
	if err := Instance.Model(&models.User{}).Where("observed_max_heartrate IS NULL").Count(&nullCount).Error; err != nil {
		logger.Log.Warn("Failed to count users needing observed-max backfill. Error: " + err.Error())
		return
	}
	if nullCount == 0 {
		return
	}

	// Walk every stored stream once, joining up to the owning user, tracking each user's
	// peak plausible HR. Streams are JSON in a longtext column, so the max is computed in Go.
	rows, err := Instance.Table("operation_sets").
		Select("exercise_days.user_id AS user_id, operation_sets.strava_streams AS streams").
		Joins("JOIN operations on operations.id = operation_sets.operation_id").
		Joins("JOIN exercises on exercises.id = operations.exercise_id").
		Joins("JOIN exercise_days on exercise_days.id = exercises.exercise_day_id").
		Where("operation_sets.strava_streams IS NOT NULL AND exercise_days.user_id IS NOT NULL").
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
	if err := Instance.Model(&models.User{}).Where("observed_max_heartrate IS NULL").Update("observed_max_heartrate", 0).Error; err != nil {
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

// migrateStravaSkipHevyToSourcePriority is a one-time backfill: the per-user "skip Strava
// activities already in Hevy" flag is replaced by the general ActivitySourcePriority ranking.
// Each user with the flag on gets "hevy,strava" (unless they already have a ranking), which
// keeps Hevy winning. Self-limiting: the flag is cleared once migrated and nothing sets it
// any more, so later boots find no rows.
func migrateStravaSkipHevyToSourcePriority() {
	if !Instance.Migrator().HasColumn(&models.User{}, "strava_skip_hevy_duplicates") {
		return
	}

	var skipHevyUserIDs []uuid.UUID
	if err := Instance.Model(&models.User{}).Where("strava_skip_hevy_duplicates = ?", true).Pluck("id", &skipHevyUserIDs).Error; err != nil {
		logger.Log.Warn("Failed to read strava_skip_hevy_duplicates for source-priority migration. Error: " + err.Error())
		return
	}
	if len(skipHevyUserIDs) == 0 {
		return
	}

	migrated := Instance.Model(&models.User{}).
		Where("strava_skip_hevy_duplicates = ?", true).
		Where("activity_source_priority IS NULL").
		Update("activity_source_priority", models.IntegrationProviderHevy+","+models.IntegrationProviderStrava)
	if migrated.Error != nil {
		logger.Log.Warn("Failed to migrate strava_skip_hevy_duplicates to source priority. Error: " + migrated.Error.Error())
		return
	}
	if err := Instance.Model(&models.User{}).Where("strava_skip_hevy_duplicates = ?", true).Update("strava_skip_hevy_duplicates", false).Error; err != nil {
		logger.Log.Warn("Failed to clear migrated strava_skip_hevy_duplicates flags. Error: " + err.Error())
	}
	logger.Log.Info("Migrated " + strconv.FormatInt(migrated.RowsAffected, 10) + " users' 'skip Strava duplicates of Hevy' preference to a source priority.")
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
