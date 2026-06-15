package controllers

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSafeImageFilePath(t *testing.T) {
	base := "/var/app/images"

	good, err := safeImageFilePath(base, "550e8400-e29b-41d4-a716-446655440000.jpg")
	if err != nil {
		t.Fatalf("unexpected error for valid filename: %v", err)
	}
	if want := filepath.Join(base, "550e8400-e29b-41d4-a716-446655440000.jpg"); good != want {
		t.Errorf("got %q, want %q", good, want)
	}

	// Names that resolve outside the base directory must be rejected.
	for _, bad := range []string{"../secret.jpg", "../../etc/passwd", "sub/../../escape.jpg"} {
		if _, err := safeImageFilePath(base, bad); err == nil {
			t.Errorf("expected error for %q, got nil", bad)
		}
	}
}

// makeTestPNG builds a solid-color PNG of the given size for use as test input.
func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
	return buf.Bytes()
}

// makeTestJPEG builds a solid-color JPEG of the given size for use as test input.
func makeTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, nil); err != nil {
		t.Fatalf("failed to encode test JPEG: %v", err)
	}
	return buf.Bytes()
}

func TestImageBytesToBase64Prefix(t *testing.T) {
	cases := []struct {
		name   string
		bytes  []byte
		prefix string
	}{
		{"png", makeTestPNG(t, 8, 8), "data:image/png;base64,"},
		{"jpeg", makeTestJPEG(t, 8, 8), "data:image/jpeg;base64,"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := ImageBytesToBase64(tc.bytes)
			if err != nil {
				t.Fatalf("ImageBytesToBase64 returned error: %v", err)
			}
			if !strings.HasPrefix(encoded, tc.prefix) {
				t.Errorf("expected prefix %q, got %q", tc.prefix, encoded[:min(len(encoded), 40)])
			}
		})
	}
}

func TestBase64RoundTrip(t *testing.T) {
	original := makeTestPNG(t, 16, 16)

	encoded, err := ImageBytesToBase64(original)
	if err != nil {
		t.Fatalf("ImageBytesToBase64 returned error: %v", err)
	}

	decoded, mimeType, err := Base64ToImageBytes(encoded)
	if err != nil {
		t.Fatalf("Base64ToImageBytes returned error: %v", err)
	}

	if mimeType != "image/png" {
		t.Errorf("expected mime type image/png, got %q", mimeType)
	}

	if !bytes.Equal(original, decoded) {
		t.Errorf("round-tripped bytes differ from original (len %d vs %d)", len(original), len(decoded))
	}
}

func TestBase64ToImageBytesErrors(t *testing.T) {
	cases := map[string]string{
		"missing base64 marker": "data:image/png," + "notbase64",
		"invalid base64 data":   "data:image/png;base64,not valid base64!!!",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Base64ToImageBytes(input); err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}

func TestResizeImageDownscalesWithinBounds(t *testing.T) {
	// A 1000x500 source resized into a 250x250 box should scale down by the
	// limiting dimension (width here) to 250x125, preserving aspect ratio.
	src := makeTestPNG(t, 1000, 500)

	out, err := ResizeImage(250, 250, src)
	if err != nil {
		t.Fatalf("ResizeImage returned error: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}

	b := img.Bounds()
	if b.Dx() > 250 || b.Dy() > 250 {
		t.Errorf("resized image %dx%d exceeds 250x250 box", b.Dx(), b.Dy())
	}
	if b.Dx() != 250 || b.Dy() != 125 {
		t.Errorf("expected 250x125, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestResizeImageDoesNotUpscale(t *testing.T) {
	// Source smaller than the box must be left at its original dimensions.
	src := makeTestPNG(t, 100, 80)

	out, err := ResizeImage(250, 250, src)
	if err != nil {
		t.Fatalf("ResizeImage returned error: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}

	b := img.Bounds()
	if b.Dx() != 100 || b.Dy() != 80 {
		t.Errorf("expected unchanged 100x80, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestDetectImageMimeType(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{"jpeg", makeTestJPEG(t, 8, 8), "image/jpeg"},
		{"png", makeTestPNG(t, 8, 8), "image/png"},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24"></svg>`), "image/svg+xml"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectImageMimeType(tc.bytes); got != tc.want {
				t.Errorf("detectImageMimeType(%s) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestResizeImageRejectsNonImage(t *testing.T) {
	if _, err := ResizeImage(250, 250, []byte("this is not an image")); err == nil {
		t.Error("expected error when resizing non-image bytes, got nil")
	}
}

func TestLoadResizedImageCached(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/photo.jpg"

	if err := os.WriteFile(filePath, makeTestJPEG(t, 1000, 500), 0644); err != nil {
		t.Fatalf("failed to write test image: %v", err)
	}

	first, err := loadResizedImageCached(filePath, 250, 250)
	if err != nil {
		t.Fatalf("loadResizedImageCached returned error: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(first))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 250 || b.Dy() != 125 {
		t.Errorf("expected 250x125, got %dx%d", b.Dx(), b.Dy())
	}

	// Second call with the file unchanged should return the same cached bytes.
	second, err := loadResizedImageCached(filePath, 250, 250)
	if err != nil {
		t.Fatalf("loadResizedImageCached (cached) returned error: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("expected identical bytes from the cache on the second call")
	}

	// Rewrite the file with different dimensions and bump its modtime; the cache must
	// invalidate and return the newly resized image.
	if err := os.WriteFile(filePath, makeTestJPEG(t, 400, 400), 0644); err != nil {
		t.Fatalf("failed to overwrite test image: %v", err)
	}
	newModTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filePath, newModTime, newModTime); err != nil {
		t.Fatalf("failed to bump modtime: %v", err)
	}

	third, err := loadResizedImageCached(filePath, 250, 250)
	if err != nil {
		t.Fatalf("loadResizedImageCached (after change) returned error: %v", err)
	}
	img, _, err = image.Decode(bytes.NewReader(third))
	if err != nil {
		t.Fatalf("failed to decode re-resized image: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 250 || b.Dy() != 250 {
		t.Errorf("expected 250x250 after file change, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestLoadResizedImageCachedMissingFile(t *testing.T) {
	if _, err := loadResizedImageCached(t.TempDir()+"/nope.jpg", 250, 250); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
