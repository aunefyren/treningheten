package database

import "time"

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
