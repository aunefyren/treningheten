package utilities

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// v1Dump is a trimmed MySQL dump in the pre-UUID (v1) shape: integer ids and FK columns
// named after the entity (`user`, `season`, …).
const v1Dump = "CREATE TABLE `goals` (\n" +
	"  `id` bigint(20) UNSIGNED NOT NULL,\n" +
	"  `season` bigint(20) UNSIGNED DEFAULT NULL,\n" +
	"  `user` bigint(20) DEFAULT NULL\n" +
	");\n" +
	"\n" +
	"INSERT INTO `users` (`id`, `created_at`, `updated_at`, `deleted_at`, `first_name`) VALUES\n" +
	"(1, '2023-01-01', '2023-01-01', NULL, 'Ann, the first'),\n" +
	"(2, '2023-01-01', '2023-01-01', NULL, 'Bob');\n" +
	"\n" +
	"INSERT INTO `prizes` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`) VALUES\n" +
	"(1, '2023-01-01', '2023-01-01', NULL, 'Pizza');\n" +
	"\n" +
	"INSERT INTO `seasons` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`, `description`, `start`, `end`, `enabled`, `prize`) VALUES\n" +
	"(1, '2023-01-01', '2023-01-01', NULL, 'S1', 'D', '2023-01-02', '2023-03-05', 1, 1);\n" +
	"\n" +
	"INSERT INTO `goals` (`id`, `created_at`, `updated_at`, `deleted_at`, `season`, `competing`, `exercise_interval`, `user`) VALUES\n" +
	"(1, '2023-01-01', '2023-01-01', NULL, 1, 1, 3, 2);\n" +
	"\n" +
	"INSERT INTO `achievements` (`id`, `created_at`, `updated_at`, `deleted_at`, `name`) VALUES\n" +
	"(5, '2023-01-01', '2023-01-01', NULL, 'Joined');\n" +
	"\n" +
	"INSERT INTO `achievement_delegations` (`id`, `created_at`, `updated_at`, `deleted_at`, `enabled`, `user`, `achievement`) VALUES\n" +
	"(1, '2023-01-01', '2023-01-01', NULL, 1, 1, 5);\n" +
	"\n" +
	"INSERT INTO `invites` (`id`, `created_at`, `updated_at`, `deleted_at`, `invite_code`, `invite_used`, `invite_recipient`) VALUES\n" +
	"(1, '2023-01-01', '2023-01-01', NULL, 'CODE', 1, NULL);\n" +
	"\n" +
	"ALTER TABLE `users`\n" +
	"  ADD PRIMARY KEY (`id`);\n" +
	"THIS LINE IS AFTER THE ALTER AND MUST BE DROPPED\n"

var quotedUUID = regexp.MustCompile(`'([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})'`)

// valueLine returns the migrated VALUES line containing marker, split into its values.
func valueLine(t *testing.T, sql string, marker string) []string {
	t.Helper()
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(line, "(") && strings.Contains(line, marker) {
			line = strings.TrimPrefix(line, "(")
			line = strings.TrimSuffix(strings.TrimSuffix(line, ");"), "),")
			return strings.Split(line, ", ")
		}
	}
	t.Fatalf("no value line containing %q in:\n%s", marker, sql)
	return nil
}

func TestMigrateSQLRemapsIDsConsistently(t *testing.T) {
	migrated, err := MigrateSQL(bufio.NewScanner(strings.NewReader(v1Dump)))
	if err != nil {
		t.Fatalf("MigrateSQL: %v", err)
	}

	// Column types and FK column names are rewritten.
	for _, want := range []string{"`id` varchar(100) NOT NULL", "`season_id` varchar(100)", "`user_id` varchar(100)",
		"`code`, `used`, `recipient_id`", "`achievement_id`)", "`prize_id`)"} {
		if !strings.Contains(migrated, want) {
			t.Errorf("migrated SQL is missing %q", want)
		}
	}
	if strings.Contains(migrated, "bigint") {
		t.Error("bigint columns survived the migration")
	}

	// Every row id becomes a UUID, and every FK points at the UUID its target row got.
	bob := valueLine(t, migrated, "'Bob'")
	ann := valueLine(t, migrated, "'Ann, the first'") // the quoted comma must not split the value
	prize := valueLine(t, migrated, "'Pizza'")
	season := valueLine(t, migrated, "'S1'")
	goal := valueLine(t, migrated, "3, ")
	delegation := valueLine(t, migrated, "NULL, 1, ")

	for name, row := range map[string][]string{"bob": bob, "ann": ann, "prize": prize, "season": season} {
		if !quotedUUID.MatchString(row[0]) {
			t.Errorf("%s id = %s, want a quoted UUID", name, row[0])
		}
	}
	if !strings.Contains(migrated, ", NULL, 'Ann, the first'),") {
		t.Error("the quoted name containing a comma was not kept whole")
	}
	if season[9] != prize[0] {
		t.Errorf("season prize = %s, want the prize's UUID %s", season[9], prize[0])
	}
	if goal[4] != season[0] || goal[7] != bob[0] {
		t.Errorf("goal season/user = %s/%s, want %s/%s", goal[4], goal[7], season[0], bob[0])
	}
	if delegation[5] != ann[0] {
		t.Errorf("delegation user = %s, want ann's UUID %s", delegation[5], ann[0])
	}
	// Achievements keep their fixed, well-known UUIDs.
	if delegation[6] != "'bb964360-6413-47c2-8400-ee87b40365a7'" {
		t.Errorf("delegation achievement = %s, want achievement 5's fixed UUID", delegation[6])
	}
	if invite := valueLine(t, migrated, "'CODE'"); invite[6] != "NULL" {
		t.Errorf("invite recipient = %s, want NULL kept", invite[6])
	}

	// Processing stops at the first ALTER; keys are re-added per table, then COMMIT.
	if strings.Contains(migrated, "MUST BE DROPPED") {
		t.Error("lines after the first ALTER TABLE were kept")
	}
	for _, table := range []string{"users", "prizes", "seasons", "goals", "achievements", "achievement_delegations", "invites"} {
		if !strings.Contains(migrated, "ALTER TABLE `"+table+"`\n\tADD PRIMARY KEY (`id`),") {
			t.Errorf("no primary key re-added for %s", table)
		}
	}
	if !strings.HasSuffix(migrated, "\nCOMMIT;") {
		t.Error("migrated SQL does not end with COMMIT")
	}
}

func TestChangeColumnsPerTable(t *testing.T) {
	tests := []struct{ table, in, want string }{
		{"debts", "`season`, `loser`, `winner`", "`season_id`, `loser_id`, `winner_id`"},
		{"exercises", "`exercise_day`", "`exercise_day_id`"},
		{"exercise_days", "`goal`", "`goal_id`"},
		{"wishlist_memberships", "`group`, `wishlist`", "`group_id`, `wishlist_id`"},
		{"sickleaves", "`goal`, `sickleave_used`", "`goal_id`, `used`"},
		{"subscriptions", "`user`", "`user_id`"},
		{"wheelviews", "`user`, `debt`", "`user_id`, `debt_id`"},
		{"news", "`user` bigint(20)", "`user` varchar(100)"},
	}
	for _, test := range tests {
		if got := ChangeColumns(test.in, test.table); got != test.want {
			t.Errorf("ChangeColumns(%q, %s) = %q, want %q", test.in, test.table, got, test.want)
		}
	}
}

func TestReplaceValuesSecondRunRemapsForeignKeys(t *testing.T) {
	maps := []IDMap{
		{TableName: "users", ID: "7", UUID: "u7"},
		{TableName: "seasons", ID: "3", UUID: "s3"},
		{TableName: "goals", ID: "4", UUID: "g4"},
		{TableName: "exercise_days", ID: "9", UUID: "d9"},
		{TableName: "debts", ID: "2", UUID: "b2"},
	}
	tests := []struct{ table, line, want string }{
		{"debts", "('x', 'c', 'u', NULL, 1, 3, 7, 7),", "('x', 'c', 'u', NULL, 1, 's3', 'u7', 'u7'),"},
		{"exercises", "('x', 'c', 'u', NULL, 1, 1, 'n', 9);", "('x', 'c', 'u', NULL, 1, 1, 'n', 'd9');"},
		{"exercise_days", "('x', 'c', 'u', NULL, 1, 1, 'n', 4);", "('x', 'c', 'u', NULL, 1, 1, 'n', 'g4');"},
		{"sickleaves", "('x', 'c', 'u', NULL, 1, 4);", "('x', 'c', 'u', NULL, 1, 'g4');"},
		{"subscriptions", "('x', 'c', 'u', NULL, 1, 7);", "('x', 'c', 'u', NULL, 1, 'u7');"},
		{"wheelviews", "('x', 'c', 'u', NULL, 7, 2);", "('x', 'c', 'u', NULL, 'u7', 'b2');"},
		{"news", "('x', 'c');", "('x', 'c');"},
	}
	for _, test := range tests {
		if got, _ := ReplaceValues(test.line, test.table, maps, true); got != test.want {
			t.Errorf("ReplaceValues(%s) = %q, want %q", test.table, got, test.want)
		}
	}
}

func TestMatchIDToUUID(t *testing.T) {
	maps := []IDMap{{TableName: "users", ID: "1", UUID: "abc"}}
	if got := MatchIDToUUID(maps, "users", "NULL"); got != "NULL" {
		t.Errorf("NULL = %q, want NULL", got)
	}
	if got := MatchIDToUUID(maps, "users", "1"); got != "'abc'" {
		t.Errorf("known id = %q, want 'abc'", got)
	}
	// An id with no mapping (a dangling FK) still gets a fresh UUID rather than an int.
	if got := MatchIDToUUID(maps, "seasons", "1"); !quotedUUID.MatchString(got) {
		t.Errorf("unknown id = %q, want a fresh quoted UUID", got)
	}
}

func TestMigrateDBToV2ReadsAndWritesTheFilesDirectory(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "files", "db.sql"), []byte(v1Dump), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(directory)

	MigrateDBToV2()

	written, err := os.ReadFile(filepath.Join(directory, "files", "db_modified_sql_file.sql"))
	if err != nil {
		t.Fatalf("no migrated file written: %v", err)
	}
	if !strings.Contains(string(written), "`season_id`") {
		t.Error("migrated file does not contain the rewritten dump")
	}
}
