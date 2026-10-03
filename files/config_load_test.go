package files

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aunefyren/treningheten/models"
)

// withTempConfig points the config file at a temp directory (the working directory is
// moved there too, since SaveConfig creates ./config) and restores the global config.
func withTempConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	t.Chdir(dir)

	previousPath, previousConfig := configFilePath, ConfigFile
	configFilePath = filepath.Join(dir, "config", "config.json")
	t.Cleanup(func() { configFilePath, ConfigFile = previousPath, previousConfig })
	return configFilePath
}

func writeConfig(t *testing.T, path string, config any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readConfig(t *testing.T, path string) models.ConfigStruct {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config models.ConfigStruct
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	return config
}

func TestLoadConfigCreatesAFreshConfigWithSecrets(t *testing.T) {
	path := withTempConfig(t)

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	saved := readConfig(t, path)
	if saved.DBType != "sqlite" || saved.DBLocation != "config/data.db" || saved.TreninghetenPort != 8080 {
		t.Errorf("fresh defaults = %s/%s/%d", saved.DBType, saved.DBLocation, saved.TreninghetenPort)
	}
	for name, secret := range map[string]string{
		"private key": saved.PrivateKey, "VAPID public": saved.VAPIDPublicKey, "VAPID secret": saved.VAPIDSecretKey,
		"Strava token key": saved.StravaTokenKey, "Hevy token key": saved.HevyTokenKey,
		"media token key": saved.Media.TokenKey, "Plex client id": saved.Media.Plex.ClientIdentifier,
	} {
		if secret == "" {
			t.Errorf("%s was not generated and persisted", name)
		}
	}
	if saved.MCPEnabled == nil || !*saved.MCPEnabled {
		t.Error("MCP should default on")
	}
	if saved.Media.AllowPrivateTargets == nil || !*saved.Media.AllowPrivateTargets {
		t.Error("private media targets should default on")
	}
	if !saved.SMTPEnabled {
		t.Error("SMTP should default on")
	}
	if ConfigFile.TreninghetenVersion != treninghetenVersionParameter {
		t.Errorf("version = %q, want the build parameter", ConfigFile.TreninghetenVersion)
	}
}

func TestLoadConfigFillsGapsButKeepsExistingValues(t *testing.T) {
	path := withTempConfig(t)

	disabled := false
	writeConfig(t, path, models.ConfigStruct{
		PrivateKey:           "existing-private-key",
		TreninghetenName:     "My Gym",
		DBType:               "sqlite",
		TreninghetenLogLevel: "not-a-level",
		MCPEnabled:           &disabled,
		Media:                models.MediaSettings{TokenKey: "existing-media-key"},
	})

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	saved := readConfig(t, path)

	if saved.PrivateKey != "existing-private-key" || saved.TreninghetenName != "My Gym" || saved.Media.TokenKey != "existing-media-key" {
		t.Error("an existing value was overwritten")
	}
	if saved.MCPEnabled == nil || *saved.MCPEnabled {
		t.Error("an explicit mcp_enabled=false was flipped back on")
	}
	if saved.DBLocation != "config/data.db" || saved.Timezone == "" || saved.TreninghetenEnvironment != "production" || saved.DBPort != 3306 {
		t.Errorf("missing defaults not filled: %+v", saved)
	}
	if saved.StravaTokenKey == "" || saved.HevyTokenKey == "" || saved.VAPIDSecretKey == "" {
		t.Error("missing secrets were not generated")
	}
	if saved.TreninghetenLogLevel != "info" {
		t.Errorf("invalid log level = %q, want the info fallback", saved.TreninghetenLogLevel)
	}
}

func TestLoadConfigDBType(t *testing.T) {
	tests := []struct {
		dbType  string
		want    string
		wantErr bool
	}{
		{"", "mysql", false},
		{"mysql", "mysql", false},
		{"postgres", "postgres", false},
		{"Postgres", "Postgres", false},
		{"sqlite", "sqlite", false},
		{"oracle", "", true},
	}
	for _, tt := range tests {
		t.Run("db_type "+tt.dbType, func(t *testing.T) {
			path := withTempConfig(t)
			writeConfig(t, path, models.ConfigStruct{DBType: tt.dbType, TreninghetenLogLevel: "debug"})

			err := LoadConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadConfig accepted an unsupported db_type")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if saved := readConfig(t, path); saved.DBType != tt.want {
				t.Errorf("db_type = %q, want %q", saved.DBType, tt.want)
			}
		})
	}
}

func TestLoadConfigRejectsInvalidJSON(t *testing.T) {
	path := withTempConfig(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadConfig(); err == nil {
		t.Error("expected an error for a corrupt config file")
	}
}

func TestGetPrivateKeyDecodesOrRegenerates(t *testing.T) {
	path := withTempConfig(t)

	key, err := GenerateSecureKey(64)
	if err != nil {
		t.Fatal(err)
	}
	ConfigFile.PrivateKey = key
	decoded := GetPrivateKey(0)
	if len(decoded) != 64 {
		t.Errorf("decoded key length = %d, want 64", len(decoded))
	}

	// A key that isn't valid base64 is replaced with a fresh one, which is persisted.
	ConfigFile.PrivateKey = "%%% not base64 %%%"
	regenerated := GetPrivateKey(0)
	if len(regenerated) != 64 {
		t.Errorf("regenerated key length = %d, want 64", len(regenerated))
	}
	if saved := readConfig(t, path); saved.PrivateKey != base64.StdEncoding.EncodeToString(regenerated) {
		t.Error("the regenerated key was not saved")
	}
}

func TestGenerateSecureKeyIsRandom(t *testing.T) {
	first, _ := GenerateSecureKey(32)
	second, _ := GenerateSecureKey(32)
	if first == second {
		t.Error("two generated keys are identical")
	}
	if raw, err := base64.StdEncoding.DecodeString(first); err != nil || len(raw) != 32 {
		t.Errorf("key is not 32 bytes of base64: %v", err)
	}
}

// Saving into a directory that can't be written must report the failure rather than
// silently lose the generated secrets.
func TestSaveAndCreateConfigReportUnwritableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := withTempConfig(t)
	root := filepath.Dir(filepath.Dir(dir))
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "config"), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "config"), 0o700) })

	if err := SaveConfig(); err == nil {
		t.Error("SaveConfig succeeded into a read-only directory")
	}
	if err := CreateConfigFile(); err == nil {
		t.Error("CreateConfigFile succeeded into a read-only directory")
	}
	if err := LoadConfig(); err == nil {
		t.Error("LoadConfig succeeded though it could not create the file")
	}
}
