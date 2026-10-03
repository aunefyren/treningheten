package controllers

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

// TestUpdateUserProfileImageClassifiesErrors covers the split the account handler relies
// on: anything wrong with the upload itself is an *InvalidProfileImageError (answered 400
// with its message), while failing to store a valid image is not (answered 500).
func TestUpdateUserProfileImageClassifiesErrors(t *testing.T) {
	garbage := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 20000))
	validJPEG := encodedImage(t, "jpeg", noisyImage(120))

	tests := []struct {
		name        string
		upload      string
		brokenDisk  bool
		wantInvalid bool
		wantMessage string
	}{
		{name: "no mime prefix", upload: "bm90IGFuIGltYWdl", wantInvalid: true, wantMessage: "Invalid Base64 string."},
		{name: "too small", upload: "data:image/jpeg;base64,bm90IGFuIGltYWdl", wantInvalid: true, wantMessage: "Image is too small."},
		{name: "unsupported type", upload: "data:image/gif;base64," + garbage, wantInvalid: true, wantMessage: "Invalid image type."},
		{name: "undecodable jpeg", upload: "data:image/jpeg;base64," + garbage, wantInvalid: true, wantMessage: "The image could not be read."},
		{name: "undecodable png", upload: "data:image/png;base64," + garbage, wantInvalid: true, wantMessage: "The image could not be read."},
		{name: "valid image, storage fails", upload: validJPEG, brokenDisk: true, wantInvalid: false},
		{name: "valid image is stored", upload: validJPEG},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles, _ := withImageDirs(t)
			if test.brokenDisk {
				// A regular file where the directory should be: MkdirAll can't create under it.
				blocker := filepath.Join(profiles, "blocker")
				if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
					t.Fatal(err)
				}
				profile_image_path = filepath.Join(blocker, "nested")
			}

			err := UpdateUserProfileImage(uuid.New(), test.upload)

			var invalid *InvalidProfileImageError
			isInvalid := errors.As(err, &invalid)
			switch {
			case test.wantInvalid && !isInvalid:
				t.Fatalf("error = %v, want an InvalidProfileImageError", err)
			case test.wantInvalid && invalid.Message != test.wantMessage:
				t.Errorf("message = %q, want %q", invalid.Message, test.wantMessage)
			case !test.wantInvalid && isInvalid:
				t.Errorf("a storage-side failure was classified as an invalid upload: %v", err)
			case test.brokenDisk && err == nil:
				t.Error("storing under a regular file succeeded")
			case !test.wantInvalid && !test.brokenDisk && err != nil:
				t.Errorf("valid upload returned error: %v", err)
			}
		})
	}
}
