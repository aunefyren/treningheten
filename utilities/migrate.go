package utilities

import (
	"aunefyren/treningheten/logger"
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// TableInfo represents information about a table.
type IDMap struct {
	TableName string
	ID        string
	UUID      string
}

// ChangeIDsWithUUIDs replaces the IDs in the SQL data with UUIDs.
func MigrateSQL(sqlContent *bufio.Scanner) (modifiedSQL2 string, err error) {
	modifiedSQL := ""
	modifiedSQL2 = ""
	err = nil
	IDMaps := []IDMap{}
	Tables := []string{}

	newIDMAP := IDMap{
		TableName: "achievements",
		ID:        "1",
		UUID:      "7f2d49ad-d056-415e-aa80-0ada6db7cc00",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "2",
		UUID:      "a8c62293-6090-4b16-a070-ad65404836ae",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "3",
		UUID:      "ae27d8bf-dfc8-4be1-b7a9-01183b375ebf",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "4",
		UUID:      "d415fffc-ea99-4b27-8929-aeb02ae44da3",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "5",
		UUID:      "bb964360-6413-47c2-8400-ee87b40365a7",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "6",
		UUID:      "51c48b42-4429-4b82-8fb2-d2bb2bfe907a",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "7",
		UUID:      "f7fad558-3e59-4812-9b13-4c30a91c04b9",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "8",
		UUID:      "ab0b1bf0-c57b-469f-a6ba-5d195f1b896d",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "9",
		UUID:      "c4a131a6-2aa6-49fb-98e5-fa797152a9a4",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "10",
		UUID:      "420b020c-2cad-4898-bb94-d86dc0031203",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "11",
		UUID:      "31fa2681-eec7-43e4-bc69-35dee352eaee",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "12",
		UUID:      "e7ee36d4-f39e-40a3-af92-2f7e1f707d07",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "13",
		UUID:      "8875597e-d8f5-4514-b96f-c51ecce4eb1f",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "14",
		UUID:      "ca6a4692-153b-47a7-8444-457b906d0666",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "15",
		UUID:      "2a84df89-9976-443b-a093-19f8d73b5eff",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "16",
		UUID:      "01dc9c4b-cf65-4d3c-9596-1417b67bd86f",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "17",
		UUID:      "38524a0a-f0b6-4cbf-b221-05ebfa0797f7",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "18",
		UUID:      "b342cd1b-1812-4384-967f-51d2be772eab",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "19",
		UUID:      "c92178b4-753a-4624-a7f6-ae5afd0a9ca3",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "20",
		UUID:      "05a3579f-aa8d-4814-b28f-5824a2d904ec",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "21",
		UUID:      "5e0f5605-b3e5-4350-a408-1c9f5b5a99a4",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "22",
		UUID:      "96cf246b-5d16-4fc8-8887-d95815a89683",
	}
	IDMaps = append(IDMaps, newIDMAP)

	newIDMAP = IDMap{
		TableName: "achievements",
		ID:        "23",
		UUID:      "b566e486-d476-40f1-a9f2-28035bb43f37",
	}
	IDMaps = append(IDMaps, newIDMAP)

	// Position variables
	currentMode := "false"
	currentTable := ""

	createTableRegExString := `^CREATE TABLE \x60([\w_]{1,25})\x60 \((\n){0,1}`
	createTableRegEx := regexp.MustCompile(createTableRegExString)
	insertIntoRegExString := `^INSERT INTO \x60([\w_]{1,25})\x60 \([\x60\w, ]{1,}\) VALUES(\n){0,1}`
	insertIntoRegEx := regexp.MustCompile(insertIntoRegExString)
	emptyLineRegExString := `^$`
	emptyLineRegEx := regexp.MustCompile(emptyLineRegExString)
	valueLineRegExString := `^\(.{1,}\)[,;]{1,1}`
	valueLineRegEx := regexp.MustCompile(valueLineRegExString)
	alterTableRegExString := `^ALTER TABLE`
	alterTableRegEx := regexp.MustCompile(alterTableRegExString)

	// Process each table, but only replace ID's
	for sqlContent.Scan() {
		line := sqlContent.Text()
		modifiedLine := sqlContent.Text()

		if createTableRegEx.Match([]byte(line)) {
			currentMode = "create"
			matches := createTableRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if insertIntoRegEx.Match([]byte(line)) {
			currentMode = "insert"
			matches := insertIntoRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if currentMode == "insert" && emptyLineRegEx.Match([]byte(line)) {
			currentMode = "none"
			currentTable = "none"
		} else {
			// logger.Log.Info("No Regex matched: " + line)
		}

		if currentMode == "insert" && valueLineRegEx.Match([]byte(line)) {
			logger.Log.Info("INSERT MODE ON TABLE: " + currentTable)
			modifiedLine, IDMaps = ReplaceValues(modifiedLine, currentTable, IDMaps, false)
		} else if currentMode == "insert" && insertIntoRegEx.Match([]byte(line)) {
			modifiedLine = ChangeColumns(modifiedLine, currentTable)
		} else if currentMode == "create" {
			logger.Log.Info("CREATE MODE ON TABLE: " + currentTable)
			modifiedLine = ChangeColumns(modifiedLine, currentTable)
		}

		if alterTableRegEx.Match([]byte(line)) {
			break
		}

		modifiedSQL += modifiedLine + "\n"
	}

	for _, line := range strings.Split(strings.TrimSuffix(modifiedSQL, "\n"), "\n") {
		modifiedLine := line

		if createTableRegEx.Match([]byte(line)) {
			currentMode = "create"
			matches := createTableRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if insertIntoRegEx.Match([]byte(line)) {
			currentMode = "insert"
			matches := insertIntoRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if currentMode == "insert" && emptyLineRegEx.Match([]byte(line)) {
			currentMode = "none"
			currentTable = "none"
		} else {
			// logger.Log.Info("No Regex matched: " + line)
		}

		if currentMode == "insert" && valueLineRegEx.Match([]byte(line)) {
			logger.Log.Info("INSERT MODE ON TABLE: " + currentTable)
			modifiedLine, IDMaps = ReplaceValues(modifiedLine, currentTable, IDMaps, true)
		} else if currentMode == "create" {
			logger.Log.Info("CREATE MODE ON TABLE: " + currentTable)
			modifiedLine = ChangeColumns(modifiedLine, currentTable)
		}

		if alterTableRegEx.Match([]byte(line)) {
			break
		}

		modifiedSQL2 += modifiedLine + "\n"
	}

	for _, IDMap := range IDMaps {
		alreadyAdded := false
		for _, TableName := range Tables {
			if TableName == IDMap.TableName {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			Tables = append(Tables, IDMap.TableName)
		}
	}

	logger.Log.Info(len(Tables))

	for _, TableName := range Tables {
		modifiedSQL2 += "\n" +
			"ALTER TABLE `" + TableName + "`\n" +
			"	ADD PRIMARY KEY (`id`),\n" +
			"	ADD KEY `idx_" + TableName + "_deleted_at` (`deleted_at`);" +
			"\n"
	}

	modifiedSQL2 += "\nCOMMIT;"

	return
}

func ChangeColumns(line string, currentTable string) (newLine string) {
	newLine = line

	newLine = strings.ReplaceAll(newLine, "bigint(20) UNSIGNED", "varchar(100)")
	newLine = strings.ReplaceAll(newLine, "bigint(20)", "varchar(100)")

	switch currentTable {
	case "achievement_delegations":
		newLine = strings.ReplaceAll(newLine, "`user`", "`user_id`")
		newLine = strings.ReplaceAll(newLine, "`achievement`", "`achievement_id`")
	case "debts":
		newLine = strings.ReplaceAll(newLine, "`season`", "`season_id`")
		newLine = strings.ReplaceAll(newLine, "`loser`", "`loser_id`")
		newLine = strings.ReplaceAll(newLine, "`winner`", "`winner_id`")
	case "exercises":
		newLine = strings.ReplaceAll(newLine, "`exercise_day`", "`exercise_day_id`")
	case "exercise_days":
		newLine = strings.ReplaceAll(newLine, "`goal`", "`goal_id`")
	case "goals":
		newLine = strings.ReplaceAll(newLine, "`season`", "`season_id`")
		newLine = strings.ReplaceAll(newLine, "`user`", "`user_id`")
	case "invites":
		newLine = strings.ReplaceAll(newLine, "`invite_code`", "`code`")
		newLine = strings.ReplaceAll(newLine, "`invite_used`", "`used`")
		newLine = strings.ReplaceAll(newLine, "`invite_recipient`", "`recipient_id`")
		newLine = strings.ReplaceAll(newLine, "`invite_enabled`", "`enabled`")
	case "wishlist_memberships":
		newLine = strings.ReplaceAll(newLine, "`group`", "`group_id`")
		newLine = strings.ReplaceAll(newLine, "`wishlist`", "`wishlist_id`")
	case "seasons":
		newLine = strings.ReplaceAll(newLine, "`prize`", "`prize_id`")
	case "sickleaves":
		newLine = strings.ReplaceAll(newLine, "`goal`", "`goal_id`")
		newLine = strings.ReplaceAll(newLine, "`sickleave_used`", "`used`")
	case "subscriptions":
		newLine = strings.ReplaceAll(newLine, "`user`", "`user_id`")
	case "wheelviews":
		newLine = strings.ReplaceAll(newLine, "`user`", "`user_id`")
		newLine = strings.ReplaceAll(newLine, "`debt`", "`debt_id`")
	default:
		logger.Log.Info("No column updates on: " + currentTable)
	}

	return
}

func ReplaceValues(line string, currentTable string, IDMaps []IDMap, secondRun bool) (newLine string, UpdatedIDMaps []IDMap) {
	newLine = line
	UpdatedIDMaps = IDMaps

	startString := "("
	endString := ""

	newLine = strings.TrimPrefix(newLine, "(")
	if strings.HasSuffix(line, "),") {
		newLine = strings.TrimSuffix(newLine, "),")
		endString = "),"
	} else {
		newLine = strings.TrimSuffix(newLine, ");")
		endString = ");"
	}

	values := strings.Split(newLine, ", ")
	if len(values) == 0 {
		logger.Log.Info("Failed to split values for table: " + currentTable)
		return
	}

	finishedLoop := false
	sum := 1
	for sum < 1000 {
		for index, value := range values {
			if strings.HasPrefix(value, "'") && !strings.HasSuffix(value, "'") && index < len(values)+1 {
				newValues := []string{}
				for indexTwo, valueTwo := range values {
					if indexTwo == index+1 {
						newValues[indexTwo-1] += ", " + valueTwo
					} else {
						newValues = append(newValues, valueTwo)
					}

				}
				values = newValues
				break
			}
			if index+1 >= (len(values)) {
				finishedLoop = true
			}
		}
		if finishedLoop {
			break
		}
	}

	// Replace ID
	if !secondRun {
		currentID := values[0]
		newUUID := MatchIDToUUID(UpdatedIDMaps, currentTable, currentID)

		if newUUID == "NULL" || currentTable != "achievements" {
			newIDMap := IDMap{
				TableName: currentTable,
				ID:        currentID,
				UUID:      uuid.New().String(),
			}
			UpdatedIDMaps = append(UpdatedIDMaps, newIDMap)
			values[0] = "'" + newIDMap.UUID + "'"
		} else {
			values[0] = newUUID
		}
	} else {

		switch currentTable {
		case "achievement_delegations":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[5])
			values[5] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "achievements", values[6])
			values[6] = newUUID
		case "debts":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "seasons", values[5])
			values[5] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "users", values[6])
			values[6] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "users", values[7])
			values[7] = newUUID
		case "exercises":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "exercise_days", values[7])
			values[7] = newUUID
		case "exercise_days":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "goals", values[7])
			values[7] = newUUID
		case "goals":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "seasons", values[4])
			values[4] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "users", values[7])
			values[7] = newUUID
		case "invites":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[6])
			values[6] = newUUID
		case "seasons":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "prizes", values[9])
			values[9] = newUUID
		case "sickleaves":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "goals", values[5])
			values[5] = newUUID
		case "subscriptions":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[5])
			values[5] = newUUID
		case "wheelviews":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[4])
			values[4] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "debts", values[5])
			values[5] = newUUID
		default:
			logger.Log.Info("No column updates on: " + currentTable)
		}

	}

	newLineTwo := startString
	for index, value := range values {
		newLineTwo += value
		if index+1 < len(values) {
			newLineTwo += ", "
		}
	}
	newLineTwo += endString

	return newLineTwo, UpdatedIDMaps
}

func MatchIDToUUID(IDMaps []IDMap, currentTable string, ID string) string {
	if ID == "NULL" {
		return "NULL"
	}
	for _, IDMap := range IDMaps {
		if IDMap.TableName == currentTable && IDMap.ID == ID {
			return "'" + IDMap.UUID + "'"
		}
	}
	return "'" + uuid.New().String() + "'"
}

func MigrateDBToV2() {
	// Read SQL file content
	fileContent, err := os.Open("./files/db.sql")
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(fileContent)

	// Call the function to modify the SQL content
	modifiedSQL, err := MigrateSQL(scanner)
	if err != nil {
		panic(err)
	}

	// Write the modified content back to the file
	err = os.WriteFile("./files/db_modified_sql_file.sql", []byte(modifiedSQL), 0644)
	if err != nil {
		panic(err)
	}

	logger.Log.Info("Modification complete. Check './files/db_modified_sql_file.sql'")
}
