// Package testdb opens an isolated, empty database for one test.
//
// By default it is an in-memory SQLite database. Setting TRENINGHETEN_TEST_POSTGRES_DSN or
// TRENINGHETEN_TEST_MYSQL_DSN runs the same tests against a real server instead: each test
// gets its own Postgres schema or MySQL database, dropped again on cleanup. The GORM
// settings mirror database.Connect for each backend, so the suite exercises the dialect
// the app will actually use.
package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

const (
	PostgresDSNEnv = "TRENINGHETEN_TEST_POSTGRES_DSN"
	MySQLDSNEnv    = "TRENINGHETEN_TEST_MYSQL_DSN"
	// TimezoneEnv sets the zone server-backed tests run in (default Europe/Paris, the
	// config default).
	TimezoneEnv = "TRENINGHETEN_TEST_TIMEZONE"
)

var pinZoneOnce sync.Once
var pinnedZone *time.Location

// serverZone pins time.Local to the test zone and returns it. It mirrors main.go, which
// sets time.Local from the configured timezone, and database.Connect, which hands the
// same zone to the server session. Whole-day lookups bound dates with strings, which
// Postgres and MySQL read in the session zone, so app and session must agree.
func serverZone(t testing.TB) *time.Location {
	pinZoneOnce.Do(func() {
		name := os.Getenv(TimezoneEnv)
		if name == "" {
			name = "Europe/Paris"
		}
		location, err := time.LoadLocation(name)
		if err != nil {
			panic(fmt.Sprintf("invalid %s %q: %v", TimezoneEnv, name, err))
		}
		time.Local = location
		pinnedZone = location
	})
	return pinnedZone
}

// Backend names the database the tests run against: "sqlite", "postgres" or "mysql".
func Backend() string {
	switch {
	case os.Getenv(PostgresDSNEnv) != "":
		return "postgres"
	case os.Getenv(MySQLDSNEnv) != "":
		return "mysql"
	}
	return "sqlite"
}

// Open returns a fresh database for t. config may be nil; foreign keys are never created
// on migrate, as in production.
func Open(t testing.TB, config *gorm.Config) *gorm.DB {
	t.Helper()

	if config == nil {
		config = &gorm.Config{}
	}
	config.DisableForeignKeyConstraintWhenMigrating = true

	switch Backend() {
	case "postgres":
		return openPostgres(t, os.Getenv(PostgresDSNEnv), config)
	case "mysql":
		return openMySQL(t, os.Getenv(MySQLDSNEnv), config)
	}
	return openSQLite(t, config)
}

func openSQLite(t testing.TB, config *gorm.Config) *gorm.DB {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	// One connection keeps the :memory: database alive and isolated for the test.
	sqlDB.SetMaxOpenConns(1)

	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, config)
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func openPostgres(t testing.TB, dsn string, config *gorm.Config) *gorm.DB {
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open postgres: %v", err)
	}
	schema := isolatedName()
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Mirrors database.Connect's postgres branch.
	config.PrepareStmt = true
	dsn = withParam(withParam(dsn, "search_path", schema), "TimeZone", serverZone(t).String())
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), config)
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		_ = admin.Close()
	})
	return db
}

// withParam adds a run-time parameter to a URL or key/value Postgres DSN.
func withParam(dsn string, key string, value string) string {
	if !strings.Contains(dsn, "://") {
		return dsn + " " + key + "=" + value
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&" + key + "=" + value
	}
	return dsn + "?" + key + "=" + value
}

func openMySQL(t testing.TB, dsn string, config *gorm.Config) *gorm.DB {
	parsed, err := gomysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("invalid %s: %v", MySQLDSNEnv, err)
	}
	parsed.DBName = ""
	admin, err := sql.Open("mysql", parsed.FormatDSN())
	if err != nil {
		t.Fatalf("failed to open mysql: %v", err)
	}
	name := isolatedName()
	if _, err := admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4"); err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Mirrors database.Connect's mysql branch (parseTime, local time, utf8mb4).
	parsed.DBName = name
	parsed.ParseTime = true
	parsed.Loc = serverZone(t)
	db, err := gorm.Open(mysql.Open(parsed.FormatDSN()), config)
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		_, _ = admin.Exec("DROP DATABASE " + name)
		_ = admin.Close()
	})
	return db
}

func isolatedName() string {
	return fmt.Sprintf("t_%s", strings.ReplaceAll(uuid.NewString(), "-", ""))
}
