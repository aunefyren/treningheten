package config

import (
	"aunefyren/treningheten/models"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/SherClockHolmes/webpush-go"
)

var treningheten_version_parameter = "{{RELEASE_TAG}}"
var config_path, _ = filepath.Abs("./files/config.json")

func GetConfig() (*models.ConfigStruct, error) {
	// Create config.json if it doesn't exist
	if _, err := os.Stat(config_path); errors.Is(err, os.ErrNotExist) {
		log.Println("Config file does not exist. Creating...")
		fmt.Println("Config file does not exist. Creating...")

		err := CreateConfigFile()
		if err != nil {
			return nil, err
		}
	}

	file, err := os.Open(config_path)
	if err != nil {
		log.Println("Get config file threw error trying to open the file.")
		fmt.Println("Get config file threw error trying to open the file.")
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := models.ConfigStruct{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Println("Get config file threw error trying to parse the file.")
		fmt.Println("Get config file threw error trying to parse the file.")
		return nil, err
	}

	anythingChanged := false

	if config.PrivateKey == "" {
		// Set new value
		newKey, err := GenerateSecureKey(64)
		if err != nil {
			return nil, errors.New("Failed to generate secure key. Error: " + err.Error())
		}
		config.PrivateKey = newKey
		anythingChanged = true
		log.Println("New private key set.")
	}

	if config.TreninghetenName == "" {
		// Set new value
		config.TreninghetenName = "Treningheten"
		anythingChanged = true
	}

	if config.TreninghetenEnvironment == "" {
		// Set new value
		config.TreninghetenEnvironment = "prod"
		anythingChanged = true
	}

	if config.Timezone == "" {
		// Set new value
		config.Timezone = "Europe/Paris"
		anythingChanged = true
	}

	if config.TreninghetenPort == 0 {
		// Set new value
		config.TreninghetenPort = 8080
		anythingChanged = true
	}

	if config.DBPort == 0 {
		// Set new value
		config.DBPort = 3306
		anythingChanged = true
	}

	if config.VAPIDPublicKey == "" || config.VAPIDSecretKey == "" {
		config, err = AddVapidKeysToConfig(config)
		if err != nil {
			log.Println("Failed to add Vapid keys to config. Error: " + err.Error())
			return &models.ConfigStruct{}, errors.New("Failed to add Vapid keys to config.")
		}
		anythingChanged = true
	}

	if anythingChanged {
		// Save new version of config json
		err = SaveConfig(&config)
		if err != nil {
			return nil, err
		}
	}

	config.TreninghetenVersion = treningheten_version_parameter

	// Return config object
	return &config, nil

}

// Creates empty config.json
func CreateConfigFile() error {
	var config models.ConfigStruct

	config.TreninghetenPort = 8080
	config.TreninghetenName = "Treningheten"
	config.TreninghetenEnvironment = "prod"
	config.DBPort = 3306
	config.SMTPEnabled = true
	config.TreninghetenVersion = treningheten_version_parameter

	config, err := AddVapidKeysToConfig(config)
	if err != nil {
		log.Println("Failed to add Vapid keys to config. Error: " + err.Error())
		return errors.New("Failed to add Vapid keys to config.")
	}

	privateKey, err := GenerateSecureKey(64)
	if err != nil {
		log.Println("Failed to generate private key. Error: " + err.Error())
		fmt.Println("Failed to generate private key. Error: " + err.Error())
		return err
	}
	config.PrivateKey = privateKey

	err = SaveConfig(&config)
	if err != nil {
		log.Println("Create config file threw error trying to save the file.")
		fmt.Println("Create config file threw error trying to save the file.")
		return err
	}

	return nil
}

// Saves the given config struct as config.json
func SaveConfig(config *models.ConfigStruct) error {

	err := os.MkdirAll("./files", os.ModePerm)
	if err != nil {
		log.Println("Failed to create directory for config. Error: " + err.Error())
		return errors.New("Failed to create directory for config.")
	}

	file, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(config_path, file, 0644)
	if err != nil {
		return err
	}

	return nil
}

func AddVapidKeysToConfig(config models.ConfigStruct) (models.ConfigStruct, error) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Println("Failed to create Vapid key pair. Error: " + err.Error())
		return models.ConfigStruct{}, errors.New("Failed to create Vapid key pair.")
	}

	config.VAPIDPublicKey = publicKey
	config.VAPIDSecretKey = privateKey

	return config, nil
}

func GetPrivateKey(epoch int) []byte {
	if epoch > 5 {
		log.Println("Failed to load private key. Exiting...")
		os.Exit(1)
	}

	configFile, err := GetConfig()
	if err != nil {
		log.Println("Failed to load config for private key. Exiting...")
		os.Exit(1)
	}

	secretKey, err := base64.StdEncoding.DecodeString(configFile.PrivateKey)
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
	configFile, err := GetConfig()
	if err != nil {
		log.Println("Failed to load config for private key. Exiting...")
		os.Exit(1)
	}
	configFile.PrivateKey, err = GenerateSecureKey(64)
	if err != nil {
		log.Println("Failed to generate new secret key. Exiting...")
		os.Exit(1)
	}
	SaveConfig(configFile)
	if err != nil {
		log.Println("Failed to save new config. Exiting...")
		os.Exit(1)
	}
	log.Println("New private key set.")
}
