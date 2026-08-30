package files

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// config.json holds the JWT signing key, the database and SMTP passwords, the media
// token key and the VAPID private key, so it must never be group- or world-readable.
func TestSaveConfigRestrictsPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not meaningful on Windows")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	previousPath := configFilePath
	configFilePath = filepath.Join(dir, "config", "config.json")
	t.Cleanup(func() { configFilePath = previousPath })

	if err := SaveConfig(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	fileInfo, err := os.Stat(configFilePath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	if mode := fileInfo.Mode().Perm(); mode != configFileMode {
		t.Fatalf("expected config file mode %o, got %o", configFileMode, mode)
	}

	dirInfo, err := os.Stat(filepath.Join(dir, "config"))
	if err != nil {
		t.Fatalf("failed to stat config directory: %v", err)
	}
	if mode := dirInfo.Mode().Perm(); mode != configDirMode {
		t.Fatalf("expected config directory mode %o, got %o", configDirMode, mode)
	}
}

// Installs that predate the tightening already have a 0644 config.json, and
// os.WriteFile does not change the mode of a file that already exists — so the save
// path has to chmod explicitly or those secrets stay world-readable forever.
func TestSaveConfigTightensExistingLoosePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not meaningful on Windows")
	}

	dir := t.TempDir()
	t.Chdir(dir)

	previousPath := configFilePath
	configFilePath = filepath.Join(dir, "config", "config.json")
	t.Cleanup(func() { configFilePath = previousPath })

	// Recreate the old, loose layout.
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0777); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configFilePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write legacy config file: %v", err)
	}
	if err := os.Chmod(filepath.Join(dir, "config"), 0777); err != nil {
		t.Fatalf("failed to loosen config directory: %v", err)
	}

	if err := SaveConfig(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	fileInfo, err := os.Stat(configFilePath)
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	if mode := fileInfo.Mode().Perm(); mode != configFileMode {
		t.Fatalf("expected an existing config file to be tightened to %o, got %o", configFileMode, mode)
	}

	dirInfo, err := os.Stat(filepath.Join(dir, "config"))
	if err != nil {
		t.Fatalf("failed to stat config directory: %v", err)
	}
	if mode := dirInfo.Mode().Perm(); mode != configDirMode {
		t.Fatalf("expected an existing config directory to be tightened to %o, got %o", configDirMode, mode)
	}
}
