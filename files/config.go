package files

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/sirupsen/logrus"
)

var (
	treninghetenVersionParameter = "{{RELEASE_TAG}}"
	configFilePath, _            = filepath.Abs("./config/config.json")
	ConfigFile                   = models.ConfigStruct{}
)

func LoadConfig() (err error) {
	// Create config.json if it doesn't exist
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Println("config file does not exist. creating...")

		err := CreateConfigFile()
		if err != nil {
			return err
		}
	}

	file, err := os.Open(configFilePath)
	if err != nil {
		fmt.Println("get config file threw error trying to open the file")
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)

	err = decoder.Decode(&ConfigFile)
	if err != nil {
		fmt.Println("get config file threw error trying to parse the file")
		return err
	}

	anythingChanged := false

	if ConfigFile.PrivateKey == "" {
		// Set new value
		newKey, err := GenerateSecureKey(64)
		if err != nil {
			return errors.New("failed to generate secure key. error: " + err.Error())
		}
		ConfigFile.PrivateKey = newKey
		anythingChanged = true
		fmt.Println("new private key set")
	}

	if ConfigFile.TreninghetenName == "" {
		// Set new value
		ConfigFile.TreninghetenName = "Treningheten"
		anythingChanged = true
	}

	if ConfigFile.TreninghetenEnvironment == "" {
		// Set new value
		ConfigFile.TreninghetenEnvironment = "production"
		anythingChanged = true
	}

	if ConfigFile.Timezone == "" {
		// Set new value
		ConfigFile.Timezone = "Europe/Paris"
		anythingChanged = true
	}

	if ConfigFile.DBType == "" || (strings.ToLower(ConfigFile.DBType) != "mysql" && strings.ToLower(ConfigFile.DBType) != "sqlite") {
		// Set new value
		ConfigFile.DBType = "mysql"
		anythingChanged = true
	}

	if (strings.ToLower(ConfigFile.DBType) == "sqlite") && ConfigFile.DBLocation == "" {
		// Set new value
		ConfigFile.DBLocation = "config/data.db"
		anythingChanged = true
	}

	if ConfigFile.TreninghetenPort == 0 {
		// Set new value
		ConfigFile.TreninghetenPort = 8080
		anythingChanged = true
	}

	if ConfigFile.DBPort == 0 {
		// Set new value
		ConfigFile.DBPort = 3306
		anythingChanged = true
	}

	if ConfigFile.VAPIDPublicKey == "" || ConfigFile.VAPIDSecretKey == "" {
		err = AddVapidKeysToConfig()
		if err != nil {
			return errors.New("failed to add Vapid keys to config")
		}
		anythingChanged = true
	}

	if ConfigFile.TreninghetenLogLevel == "" {
		level := logrus.InfoLevel
		ConfigFile.TreninghetenLogLevel = level.String()
		anythingChanged = true
	} else {
		parsedLogLevel, err := logrus.ParseLevel(ConfigFile.TreninghetenLogLevel)
		if err != nil {
			level := logrus.InfoLevel
			ConfigFile.TreninghetenLogLevel = level.String()
			anythingChanged = true
		} else {
			logrus.SetLevel(parsedLogLevel)
		}
	}

	if anythingChanged {
		// Save new version of config json
		err = SaveConfig()
		if err != nil {
			return err
		}
	}

	ConfigFile.TreninghetenVersion = treninghetenVersionParameter

	// Return nil error
	return nil
}

// Creates empty config.json
func CreateConfigFile() error {
	ConfigFile = models.ConfigStruct{}

	ConfigFile.TreninghetenPort = 8080
	ConfigFile.TreninghetenName = "Treningheten"
	ConfigFile.TreninghetenEnvironment = "production"
	ConfigFile.DBPort = 3306
	ConfigFile.DBType = "sqlite"
	ConfigFile.DBLocation = "config/data.db"
	ConfigFile.SMTPEnabled = true
	ConfigFile.TreninghetenVersion = treninghetenVersionParameter

	err := AddVapidKeysToConfig()
	if err != nil {
		fmt.Println("failed to add Vapid keys to config. error: " + err.Error())
		return errors.New("failed to add Vapid keys to config")
	}

	level := logrus.InfoLevel
	ConfigFile.TreninghetenLogLevel = level.String()

	privateKey, err := GenerateSecureKey(64)
	if err != nil {
		fmt.Println("failed to generate private key. error: " + err.Error())
		return err
	}
	ConfigFile.PrivateKey = privateKey

	err = SaveConfig()
	if err != nil {
		fmt.Println("create config file threw error trying to save the file")
		return err
	}

	return nil
}

// Saves the active config struct as config.json
func SaveConfig() error {
	err := os.MkdirAll("./config", os.ModePerm)
	if err != nil {
		fmt.Println("failed to create directory for config. error: " + err.Error())
		return errors.New("failed to create directory for config")
	}

	file, err := json.MarshalIndent(ConfigFile, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(configFilePath, file, 0644)
	if err != nil {
		return err
	}

	return nil
}

func AddVapidKeysToConfig() error {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fmt.Println("Failed to create Vapid key pair. Error: " + err.Error())
		return errors.New("Failed to create Vapid key pair.")
	}

	ConfigFile.VAPIDPublicKey = publicKey
	ConfigFile.VAPIDSecretKey = privateKey

	return nil
}

func GetPrivateKey(epoch int) []byte {
	if epoch > 5 {
		logger.Log.Info("Failed to load private key. Exiting...")
		os.Exit(1)
	}

	secretKey, err := base64.StdEncoding.DecodeString(ConfigFile.PrivateKey)
	if err != nil {
		ResetSecureKey()
		return GetPrivateKey(epoch + 1)
	}

	return secretKey
}

// GenerateSecureKey creates a cryptographically secure random key of the given length (in bytes).
func GenerateSecureKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	// Encode to Base64 to make it easy to store
	return base64.StdEncoding.EncodeToString(key), nil
}

func ResetSecureKey() {
	privateKey, err := GenerateSecureKey(64)
	if err != nil {
		fmt.Println("failed to generate new secret key. exiting...")
		os.Exit(1)
	}
	ConfigFile.PrivateKey = privateKey
	SaveConfig()
	if err != nil {
		fmt.Println("failed to save new config. exiting...")
		os.Exit(1)
	}
	fmt.Println("new private key set")
}
