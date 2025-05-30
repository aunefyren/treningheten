package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/logger"
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

var profile_image_path, _ = filepath.Abs("./images/profiles")
var achievements_image_path, _ = filepath.Abs("./web/assets/achievements")
var default_profile_image_path, _ = filepath.Abs("./web/assets/user.svg")
var default_max_image_height = 1000
var default_max_image_width = 1000
var default_max_thumbnail_height = 250
var default_max_thumbnail_width = 250

func APIGetUserProfileImage(context *gin.Context) {

	// Create user request
	var userIDString = context.Param("user_id")
	var thumbnail = context.Query("thumbnail")
	var imageWidth uint
	var imageHeight uint

	if thumbnail == "true" {
		imageWidth = uint(default_max_thumbnail_width)
		imageHeight = uint(default_max_thumbnail_height)
	} else {
		imageWidth = uint(default_max_image_width)
		imageHeight = uint(default_max_image_height)
	}

	// Parse user id
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		logger.Log.Info("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	// Check if user exists
	_, err = database.GetUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to find user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user."})
		context.Abort()
		return
	}

	var filePath = profile_image_path + "/" + userIDString + ".jpg"

	imageBytes, err := LoadImageFile(filePath)
	resize := true
	if err != nil {
		// Debug line
		// logger.Log.Info("Failed to find profile image. Loading default.")

		imageBytes, err = LoadDefaultProfileImage()
		if err != nil {
			logger.Log.Info("Failed to load default profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load default profile image."})
			context.Abort()
			return
		}
		resize = false
	}

	if resize {
		imageBytes, err = ResizeImage(imageWidth, imageHeight, imageBytes)
		if err != nil {
			logger.Log.Info("Failed to resize image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize image."})
			context.Abort()
			return
		}
	}

	base64, err := ImageBytesToBase64(imageBytes)
	if err != nil {
		logger.Log.Info("Failed to convert image file to Base64. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert image file to Base64."})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"image": base64, "message": "Picture retrieved."})

}

func LoadImageFile(filePath string) ([]byte, error) {

	// Read the entire file into a byte slice
	imageBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		// Debug line
		// logger.Log.Info("Failed to read file. Returning.")

		return nil, errors.New("Failed to read file.")
	}

	return imageBytes, nil

}

func SaveImageFile(filePath string, fileName string, imageFile image.Image) error {

	err := os.MkdirAll(filePath, 0755)
	if err != nil {
		logger.Log.Info("Failed to create directory for image. Error: " + err.Error())
		return errors.New("Failed to create directory for image.")
	}

	file, err := os.Create(filePath + "/" + fileName)
	if err != nil {
		logger.Log.Info("Failed to create file for image. Error: " + err.Error())
		return errors.New("Failed to create file for image.")
	}
	defer file.Close()
	if err = jpeg.Encode(file, imageFile, nil); err != nil {
		logger.Log.Info("Failed to encode file for image. Error: " + err.Error())
		return errors.New("Failed to encode file for image.")
	}

	return nil

}

func ImageBytesToBase64(image []byte) (string, error) {

	var base64Encoding string

	// Determine the content type of the image file
	mimeType := http.DetectContentType(image)

	// Prepend the appropriate URI scheme header depending
	// on the MIME type
	switch mimeType {
	case "image/jpeg":
		base64Encoding += "data:image/jpeg;base64,"
	case "image/png":
		base64Encoding += "data:image/png;base64,"
	case "image/svg+xml":
		base64Encoding += "data:image/svg+xml;base64,"
	default:
		base64Encoding += "data:image/svg+xml;base64,"
	}

	// Append the base64 encoded output
	base64Encoding += base64.StdEncoding.EncodeToString(image)

	return base64Encoding, nil

}

func Base64ToImageBytes(base64String string) ([]byte, string, error) {

	var imageBytes []byte
	var b64Data string
	var mimeType string

	b64DataArray := strings.Split(base64String, "base64,")

	if len(b64DataArray) != 2 {
		return nil, "", errors.New("Base64 string does not contain mime type.")
	} else {
		b64Data = b64DataArray[1]
		mimeType = b64DataArray[0]
	}

	mimeType = strings.Replace(mimeType, "data:", "", -1)
	mimeType = strings.Replace(mimeType, ";", "", -1)

	// Append the base64 encoded output
	imageBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		logger.Log.Info("Failed to convert Base64 string to byte array. Returning. Error: " + err.Error())
		return nil, "", errors.New("Invalid Base64 string.")
	}

	return imageBytes, mimeType, nil

}

func LoadDefaultProfileImage() ([]byte, error) {

	imageBytes, err := LoadImageFile(default_profile_image_path)
	if err != nil {
		logger.Log.Info("Failed to load default profile image. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to load default profile image.")
	}

	return imageBytes, nil

}

func ResizeImage(maxWidth uint, maxHeight uint, imageBytes []byte) ([]byte, error) {

	// decode jpeg into image.Image
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		logger.Log.Info("Failed to convert bytes to image object. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to convert bytes to image object.")
	}

	// resize to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	resizedImage := resize.Thumbnail(maxWidth, maxHeight, img, resize.Lanczos3)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resizedImage, nil)
	if err != nil {
		logger.Log.Info("Failed to convert resized image file to bytes. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to convert resized image file to bytes.")
	}
	resizedImageBytes := buf.Bytes()

	return resizedImageBytes, nil
}

func UpdateUserProfileImage(userID uuid.UUID, base64String string) error {

	imageBytes, mimeType, err := Base64ToImageBytes(base64String)
	if err != nil {
		logger.Log.Info("Failed to convert Base64 String to bytes. Error: " + err.Error())
		return errors.New("Invalid Base64 string.")
	}

	if len(imageBytes) > 10000000 {
		return errors.New("Image is too large.")
	}

	if len(imageBytes) < 10000 {
		return errors.New("Image is too small.")
	}

	var imageObject image.Image

	if mimeType == "image/jpeg" {
		imageObject, err = jpeg.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			logger.Log.Info("Failed to create image from byte array. Returning. Error: " + err.Error())
			return errors.New("Failed to create image from, byte array.")
		}
	} else if mimeType == "image/png" {
		imageObject, err = png.Decode(bytes.NewReader(imageBytes))
		if err != nil {
			logger.Log.Info("Failed to create image from byte array. Returning. Error: " + err.Error())
			return errors.New("Failed to create image from, byte array.")
		}
	} else {
		logger.Log.Info("Invalid mime type for image. Type: " + mimeType)
		return errors.New("Invalid image type.")
	}

	userIDString := userID.String()

	err = SaveImageFile(profile_image_path, userIDString+".jpg", imageObject)
	if err != nil {
		logger.Log.Info("Failed to save image to disk. Returning. Error: " + err.Error())
		return errors.New("Failed to save image to disk.")
	}

	return nil

}

func APIGetAchievementsImage(context *gin.Context) {

	// Create achievement request
	var achievementIDString = context.Param("achievement_id")
	var thumbnail = context.Query("thumbnail")
	var imageWidth uint
	var imageHeight uint

	if thumbnail == "true" {
		imageWidth = uint(default_max_thumbnail_width)
		imageHeight = uint(default_max_thumbnail_height)
	} else {
		imageWidth = uint(default_max_image_width)
		imageHeight = uint(default_max_image_height)
	}

	// Parse achievement id
	achievementID, err := uuid.Parse(achievementIDString)
	if err != nil {
		logger.Log.Info("Failed to parse achievement ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse achievement ID."})
		context.Abort()
		return
	}

	// Check if achievement exists
	_, err = database.GetAchievementByID(achievementID)
	if err != nil {
		logger.Log.Info("Failed to find achievement. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find achievement."})
		context.Abort()
		return
	}

	var filePath = achievements_image_path + "/" + achievementIDString + ".jpg"

	imageBytes, err := LoadImageFile(filePath)
	resize := true
	if err != nil {
		logger.Log.Info("Failed to find achievement image. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find achievement image."})
		context.Abort()
		return
	}

	if resize {
		imageBytes, err = ResizeImage(imageWidth, imageHeight, imageBytes)
		if err != nil {
			logger.Log.Info("Failed to resize image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize image."})
			context.Abort()
			return
		}
	}

	base64, err := ImageBytesToBase64(imageBytes)
	if err != nil {
		logger.Log.Info("Failed to convert image file to Base64. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert image file to Base64."})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"image": base64, "message": "Picture retrieved."})

}
