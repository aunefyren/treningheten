package database

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// instantParam renders an instant for comparing against a datetime column. SQLite stores
// times as text and compares text, so it gets the UTC text form (times are written in UTC
// there). MySQL and Postgres compare real datetimes: the drivers convert a time.Time to the
// session zone, which a pre-formatted UTC string would bypass — on MySQL (loc=Local) that
// shifted the bound by the server's UTC offset.
func instantParam(instant time.Time) any {
	if Instance.Dialector.Name() == "sqlite" {
		return instant.UTC().Format("2006-01-02 15:04:05")
	}
	return instant
}

// forUpdate makes a read inside a transaction take a row lock (SELECT … FOR UPDATE), so a
// concurrent transaction that reads the same row waits for this one to commit. SQLite has no
// row locks and rejects the clause; it already allows only one write transaction at a time,
// so the read runs plain there.
func forUpdate(tx *gorm.DB) *gorm.DB {
	if tx.Dialector.Name() == "sqlite" {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}
