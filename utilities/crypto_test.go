package utilities

import (
	"encoding/base64"
	"strings"
	"testing"
)

// testKey is a valid 32-byte AES-256 key, base64-encoded as the config expects.
func testKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := testKey()
	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty", ""},
		{"ascii", "hello world"},
		{"strava refresh token", "a1b2c3d4e5f60718293a4b5c6d7e8f90"},
		{"unicode", "trening 🏋️ æøå"},
		{"long", strings.Repeat("x", 4096)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := EncryptString(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptString() error: %v", err)
			}

			got, err := DecryptString(ciphertext, key)
			if err != nil {
				t.Fatalf("DecryptString() error: %v", err)
			}
			if got != tt.plaintext {
				t.Errorf("round trip = %q, want %q", got, tt.plaintext)
			}
		})
	}
}

// The output must contain no ':' so it is safe inside the prefixed Strava code field
// ("r:<ciphertext>"), and it must be valid base64.
func TestEncryptStringOutputIsColonFreeBase64(t *testing.T) {
	key := testKey()
	ciphertext, err := EncryptString("a:b:c refresh token", key)
	if err != nil {
		t.Fatalf("EncryptString() error: %v", err)
	}
	if strings.Contains(ciphertext, ":") {
		t.Errorf("ciphertext contains ':': %q", ciphertext)
	}
	if _, err := base64.StdEncoding.DecodeString(ciphertext); err != nil {
		t.Errorf("ciphertext is not valid base64: %v", err)
	}
}

// GCM uses a random nonce, so encrypting the same plaintext twice yields different
// ciphertexts (both of which still decrypt correctly).
func TestEncryptStringUsesRandomNonce(t *testing.T) {
	key := testKey()
	a, err := EncryptString("same", key)
	if err != nil {
		t.Fatalf("EncryptString() error: %v", err)
	}
	b, err := EncryptString("same", key)
	if err != nil {
		t.Fatalf("EncryptString() error: %v", err)
	}
	if a == b {
		t.Errorf("expected distinct ciphertexts for repeated encryption, got identical %q", a)
	}
}

func TestEncryptStringInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"not base64", "not-base64-!!!"},
		{"wrong length", base64.StdEncoding.EncodeToString([]byte("too short"))},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := EncryptString("data", tt.key); err == nil {
				t.Errorf("EncryptString() with %s key: expected error, got nil", tt.name)
			}
		})
	}
}

func TestDecryptStringErrors(t *testing.T) {
	key := testKey()
	valid, err := EncryptString("payload", key)
	if err != nil {
		t.Fatalf("setup EncryptString() error: %v", err)
	}

	t.Run("invalid base64", func(t *testing.T) {
		if _, err := DecryptString("not base64 !!!", key); err == nil {
			t.Error("expected error for invalid base64, got nil")
		}
	})

	t.Run("ciphertext too short", func(t *testing.T) {
		short := base64.StdEncoding.EncodeToString([]byte("tiny"))
		if _, err := DecryptString(short, key); err == nil {
			t.Error("expected error for short ciphertext, got nil")
		}
	})

	t.Run("legacy plaintext value", func(t *testing.T) {
		// A pre-encryption plaintext token is not valid ciphertext; callers rely on
		// this error to fall back to treating the stored value as plaintext.
		if _, err := DecryptString("r1a2b3c4plaintextrefreshtoken", key); err == nil {
			t.Error("expected error decrypting legacy plaintext, got nil")
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		otherKey := base64.StdEncoding.EncodeToString(make([]byte, 32)) // all-zero key
		if _, err := DecryptString(valid, otherKey); err == nil {
			t.Error("expected error decrypting with wrong key, got nil")
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		raw, _ := base64.StdEncoding.DecodeString(valid)
		raw[len(raw)-1] ^= 0xFF // flip bits in the last byte
		tampered := base64.StdEncoding.EncodeToString(raw)
		if _, err := DecryptString(tampered, key); err == nil {
			t.Error("expected error for tampered ciphertext (GCM auth), got nil")
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		if _, err := DecryptString(valid, "not-base64-!!!"); err == nil {
			t.Error("expected error for invalid key, got nil")
		}
	})
}
