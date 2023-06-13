package controllers

import (
	"aunefyren/treningheten/database"
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

var profile_image_path, _ = filepath.Abs("./images/profiles")
var default_profile_image_path, _ = filepath.Abs("./images/profiles/default.svg")
var default_max_image_height = 1000
var default_max_image_width = 1000
var default_max_thumbnail_height = 100
var default_max_thumbnail_width = 100

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
	userID, err := strconv.Atoi(userIDString)
	if err != nil {
		log.Println("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	// Check if user exists
	_, err = database.GetUserInformation(userID)
	if err != nil {
		log.Println("Failed to find user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user."})
		context.Abort()
		return
	}

	var filePath = profile_image_path + "/" + userIDString + ".jpg"

	imageBytes, err := LoadImageFile(filePath)
	resize := true
	if err != nil {
		log.Println("Failed to find profile image. Loading default.")
		imageBytes, err = LoadDefaultProfileImage()
		if err != nil {
			log.Println("Failed to load default profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load default profile image."})
			context.Abort()
			return
		}
		resize = false
	}

	if resize {
		imageBytes, err = ResizeImage(imageWidth, imageHeight, imageBytes)
		if err != nil {
			log.Println("Failed to resize image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resize image."})
			context.Abort()
			return
		}
	}

	base64, err := ImageBytesToBase64(imageBytes)
	if err != nil {
		log.Println("Failed to convert image file to Base64. Error: " + err.Error())
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
		log.Println("Failed to read file. Returning.")
		return nil, errors.New("Failed to read file.")
	}

	return imageBytes, nil

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

func LoadDefaultProfileImage() ([]byte, error) {

	imageBytes, err := LoadImageFile(default_profile_image_path)
	if err != nil {
		log.Println("Failed to load default profile image. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to load default profile image.")
	}

	return imageBytes, nil

}

func ResizeImage(maxWidth uint, maxHeight uint, imageBytes []byte) ([]byte, error) {

	// decode jpeg into image.Image
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		log.Println("Failed to convert bytes to image object. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to convert bytes to image object.")
	}

	// resize to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	resizedImage := resize.Thumbnail(maxWidth, maxHeight, img, resize.Lanczos3)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resizedImage, nil)
	if err != nil {
		log.Println("Failed to convert resized image file to bytes. Error: " + err.Error() + ". Returning.")
		return nil, errors.New("Failed to convert resized image file to bytes.")
	}
	resizedImageBytes := buf.Bytes()

	return resizedImageBytes, nil
}
