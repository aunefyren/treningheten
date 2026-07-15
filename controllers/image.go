package controllers

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

// Resized images are cached in memory keyed by source path + target size, so the
// expensive Lanczos3 resize runs once per (image, size) rather than on every request.
// The cache self-invalidates on the source file's modification time, so a re-uploaded
// profile photo is picked up without any explicit invalidation call.
type cachedResizedImage struct {
	bytes   []byte
	modTime time.Time
}

var imageCacheMutex sync.RWMutex
var imageCache = map[string]cachedResizedImage{}

// loadResizedImageCached returns the resized bytes for filePath at the given max
// dimensions, serving from the in-memory cache when the file is unchanged. It returns an
// error if the file does not exist (callers fall back to a default).
func loadResizedImageCached(filePath string, maxWidth uint, maxHeight uint) ([]byte, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("%s|%dx%d", filePath, maxWidth, maxHeight)

	imageCacheMutex.RLock()
	entry, found := imageCache[cacheKey]
	imageCacheMutex.RUnlock()
	if found && entry.modTime.Equal(info.ModTime()) {
		return entry.bytes, nil
	}

	imageBytes, err := LoadImageFile(filePath)
	if err != nil {
		return nil, err
	}

	resizedBytes, err := ResizeImage(maxWidth, maxHeight, imageBytes)
	if err != nil {
		return nil, err
	}

	imageCacheMutex.Lock()
	imageCache[cacheKey] = cachedResizedImage{bytes: resizedBytes, modTime: info.ModTime()}
	imageCacheMutex.Unlock()

	return resizedBytes, nil
}

// safeImageFilePath joins baseDir and fileName and verifies the result stays within
// baseDir, so a request parameter can never traverse out of the image directory (defence
// in depth — callers already restrict fileName to a parsed UUID).
func safeImageFilePath(baseDir string, fileName string) (string, error) {
	cleanBase := filepath.Clean(baseDir)
	joined := filepath.Join(cleanBase, fileName)
	if joined != cleanBase && !strings.HasPrefix(joined, cleanBase+string(os.PathSeparator)) {
		return "", errors.New("Resolved path escapes the image directory.")
	}
	return joined, nil
}

// serveImageBytes writes raw image bytes for direct use in an <img> tag. The MIME type is
// detected from the content (resized photos are JPEG; the profile default is an SVG).
//
// When cacheable, it sets `private, max-age=300` + an ETag and honours If-None-Match for cheap
// revalidation. The default placeholder must be served with cacheable=false (`no-store`): a
// "no photo yet" response cached under the real image URL would otherwise be served from the
// browser cache for up to 5 minutes and mask a photo uploaded later — the stale-default bug that
// previously forced callers to append cache-busting query strings.
func serveImageBytes(context *gin.Context, imageBytes []byte, cacheable bool) {
	if !cacheable {
		context.Header("Cache-Control", "no-store")
		context.Data(http.StatusOK, detectImageMimeType(imageBytes), imageBytes)
		return
	}

	etag := fmt.Sprintf("\"%x\"", md5.Sum(imageBytes))
	context.Header("Cache-Control", "private, max-age=300")
	context.Header("ETag", etag)

	if context.GetHeader("If-None-Match") == etag {
		context.Status(http.StatusNotModified)
		return
	}

	context.Data(http.StatusOK, detectImageMimeType(imageBytes), imageBytes)
}

// detectImageMimeType returns the MIME type for image bytes. It special-cases SVG, which
// http.DetectContentType reports as XML/text — that would stop a browser rendering the
// default profile placeholder in an <img> tag.
func detectImageMimeType(imageBytes []byte) string {
	sniff := imageBytes
	if len(sniff) > 1024 {
		sniff = sniff[:1024]
	}
	if bytes.Contains(sniff, []byte("<svg")) {
		return "image/svg+xml"
	}
	return http.DetectContentType(imageBytes)
}

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

	// Build the path from the parsed UUID's canonical string (not the raw request
	// parameter) and confirm it stays within the image directory, so the filename can only
	// ever be a valid UUID inside that directory — no path traversal. This also matches how
	// the file is written on upload.
	filePath, err := safeImageFilePath(profile_image_path, userID.String()+".jpg")
	if err != nil {
		logger.Log.Info("Failed to resolve profile image path. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to resolve image path."})
		context.Abort()
		return
	}

	imageBytes, err := loadResizedImageCached(filePath, imageWidth, imageHeight)
	servedDefault := false
	if err != nil {
		// No profile image on disk: serve the (unresized) default placeholder.
		imageBytes, err = LoadDefaultProfileImage()
		if err != nil {
			logger.Log.Info("Failed to load default profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load default profile image."})
			context.Abort()
			return
		}
		servedDefault = true
	}

	// Reply with the raw image bytes so it can be loaded directly into an <img> tag. The real
	// photo is cacheable; the default placeholder is not, so it can't mask a later upload.
	serveImageBytes(context, imageBytes, !servedDefault)

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

	// Build the path from the parsed UUID's canonical string and confirm it stays within
	// the image directory — no path traversal.
	filePath, err := safeImageFilePath(achievements_image_path, achievementID.String()+".jpg")
	if err != nil {
		logger.Log.Info("Failed to resolve achievement image path. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to resolve image path."})
		context.Abort()
		return
	}

	imageBytes, err := loadResizedImageCached(filePath, imageWidth, imageHeight)
	if err != nil {
		logger.Log.Info("Failed to find achievement image. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find achievement image."})
		context.Abort()
		return
	}

	// Reply with the raw image bytes so it can be loaded directly into an <img> tag. Achievement
	// images always exist here (missing → error above), so they are always cacheable.
	serveImageBytes(context, imageBytes, true)

}
