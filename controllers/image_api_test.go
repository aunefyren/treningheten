package controllers

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// noisyImage returns an image random enough that its encoding clears the 10 kB minimum
// upload size.
func noisyImage(size int) image.Image {
	generator := rand.New(rand.NewSource(1))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			img.Set(x, y, color.RGBA{uint8(generator.Intn(256)), uint8(generator.Intn(256)), uint8(generator.Intn(256)), 255})
		}
	}
	return img
}

func encodedImage(t *testing.T, format string, img image.Image) string {
	t.Helper()
	var buffer bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&buffer, img)
	} else {
		err = jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 95})
	}
	if err != nil {
		t.Fatal(err)
	}
	return "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

// withImageDirs points the image directories at temp dirs and the default placeholder at
// the repo's real SVG (package paths resolve from the controllers/ test directory).
func withImageDirs(t *testing.T) (profiles string, achievements string) {
	t.Helper()
	profiles, achievements = t.TempDir(), t.TempDir()
	placeholder, err := filepath.Abs("../web/assets/user.svg")
	if err != nil {
		t.Fatal(err)
	}
	previous := []string{profile_image_path, achievements_image_path, default_profile_image_path}
	profile_image_path, achievements_image_path, default_profile_image_path = profiles, achievements, placeholder
	t.Cleanup(func() {
		profile_image_path, achievements_image_path, default_profile_image_path = previous[0], previous[1], previous[2]
	})
	return profiles, achievements
}

func TestProfileImageUploadAndServe(t *testing.T) {
	h := newAPIHarness(t)
	withImageDirs(t)
	user, token := h.user("photo@image.test", false)
	imagePath := "/api/auth/users/" + user.ID.String() + "/image"

	// No upload yet: the SVG placeholder, never cached (it must not mask a later upload).
	placeholder := h.do("GET", imagePath, token, nil)
	if placeholder.Code != http.StatusOK || placeholder.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("placeholder: status %d, type %q", placeholder.Code, placeholder.Header().Get("Content-Type"))
	}
	if placeholder.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("placeholder Cache-Control = %q, want no-store", placeholder.Header().Get("Cache-Control"))
	}

	// Upload validation.
	for name, upload := range map[string]string{
		"not base64":   "data:image/jpeg;base64,%%%",
		"no mime type": "plainbase64",
		"too small":    encodedImage(t, "jpeg", noisyImage(8)),
		"wrong type":   "data:image/gif;base64," + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("g"), 20000)),
		"corrupt jpeg": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("j"), 20000)),
	} {
		if err := UpdateUserProfileImage(user.ID, upload); err == nil {
			t.Errorf("%s: upload accepted", name)
		}
	}
	if err := UpdateUserProfileImage(user.ID, encodedImage(t, "png", noisyImage(120))); err != nil {
		t.Fatalf("png upload: %v", err)
	}
	if err := UpdateUserProfileImage(user.ID, encodedImage(t, "jpeg", noisyImage(1200))); err != nil {
		t.Fatalf("jpeg upload: %v", err)
	}

	photo := h.do("GET", imagePath, token, nil)
	if photo.Code != http.StatusOK || photo.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("photo: status %d, type %q", photo.Code, photo.Header().Get("Content-Type"))
	}
	decoded, err := jpeg.DecodeConfig(bytes.NewReader(photo.Body.Bytes()))
	if err != nil || decoded.Width > 1000 {
		t.Errorf("served photo width = %d (err %v), want resized to at most 1000", decoded.Width, err)
	}
	etag := photo.Header().Get("ETag")
	if etag == "" {
		t.Fatal("photo has no ETag")
	}

	// Revalidation and the thumbnail size.
	request := httptest.NewRequest("GET", imagePath, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("If-None-Match", etag)
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotModified {
		t.Errorf("If-None-Match: status = %d, want 304", recorder.Code)
	}
	thumbnail := h.do("GET", imagePath+"?thumbnail=true", token, nil)
	if thumb, _ := jpeg.DecodeConfig(bytes.NewReader(thumbnail.Body.Bytes())); thumb.Width > 250 {
		t.Errorf("thumbnail width = %d, want at most 250", thumb.Width)
	}

	// Loaded by <img src> with only the cookie.
	cookieRequest := httptest.NewRequest("GET", imagePath, nil)
	cookieRequest.AddCookie(&http.Cookie{Name: "treningheten", Value: token})
	cookieRecorder := httptest.NewRecorder()
	h.router.ServeHTTP(cookieRecorder, cookieRequest)
	if cookieRecorder.Code != http.StatusOK {
		t.Errorf("cookie-authenticated image: status = %d", cookieRecorder.Code)
	}

	h.expect(http.StatusBadRequest, "GET", "/api/auth/users/nope/image", token, nil)
}

func TestAchievementImages(t *testing.T) {
	h := newAPIHarness(t)
	_, achievements := withImageDirs(t)
	_, token := h.user("badge@image.test", false)

	achievementID := "7f2d49ad-d056-415e-aa80-0ada6db7cc00"
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, noisyImage(300), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(achievements, achievementID+".jpg"), buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	served := h.do("GET", "/api/auth/achievements/"+achievementID+"/image?thumbnail=true", token, nil)
	if served.Code != http.StatusOK || served.Header().Get("Cache-Control") != "private, max-age=300" {
		t.Errorf("achievement image: status %d, cache %q", served.Code, served.Header().Get("Cache-Control"))
	}
	// Second read is served from the resize cache.
	if again := h.do("GET", "/api/auth/achievements/"+achievementID+"/image?thumbnail=true", token, nil); !bytes.Equal(again.Body.Bytes(), served.Body.Bytes()) {
		t.Error("cached image differs from the first render")
	}

	h.expect(http.StatusBadRequest, "GET", "/api/auth/achievements/nope/image", token, nil)
	// A catalog achievement without an image file.
	h.expect(http.StatusBadRequest, "GET", "/api/auth/achievements/a8c62293-6090-4b16-a070-ad65404836ae/image", token, nil)
}

func TestImageBytesToBase64RoundTrip(t *testing.T) {
	var buffer bytes.Buffer
	_ = png.Encode(&buffer, noisyImage(4))
	encoded, err := ImageBytesToBase64(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	decoded, mimeType, err := Base64ToImageBytes(encoded)
	if err != nil || mimeType != "image/png" || !bytes.Equal(decoded, buffer.Bytes()) {
		t.Errorf("round trip: mime %q, err %v", mimeType, err)
	}
	if svg, _ := ImageBytesToBase64([]byte("<svg></svg>")); svg[:26] != "data:image/svg+xml;base64," {
		t.Errorf("unknown bytes prefix = %q", svg[:26])
	}
}
