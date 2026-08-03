package auth

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// TestMain stubs logger.Log so the config helpers can log without nil-panicking.
func TestMain(m *testing.M) {
	if logger.Log == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger.Log = l
	}
	m.Run()
}

// withSigningKey installs a valid signing key (and optionally an external URL, which turns
// on issuer/audience binding) for the duration of a test, restoring the previous config
// after. The key must be valid base64: GetPrivateKey resets and *persists* a fresh key to
// the config file when it fails to decode, which a test must never trigger.
func withSigningKey(t *testing.T, externalURL string) {
	t.Helper()

	prevKey := files.ConfigFile.PrivateKey
	prevURL := files.ConfigFile.TreninghetenExternalURL

	files.ConfigFile.PrivateKey = base64.StdEncoding.EncodeToString([]byte("test-signing-key-please-ignore"))
	files.ConfigFile.TreninghetenExternalURL = externalURL

	t.Cleanup(func() {
		files.ConfigFile.PrivateKey = prevKey
		files.ConfigFile.TreninghetenExternalURL = prevURL
	})
}

// signClaims signs an arbitrary claim set with the currently configured key, so a test can
// mint tokens GenerateAccessToken would never produce (expired, not-yet-valid, no claims).
func signClaims(t *testing.T, claims *JWTClaim) string {
	t.Helper()
	token, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("failed to sign claims: %v", err)
	}
	return token
}

// baseClaims returns a claim set that parses cleanly, for a test to then break one field of.
func baseClaims(userID uuid.UUID) *JWTClaim {
	now := time.Now()
	claims := &JWTClaim{
		UserID: userID,
		Scope:  models.ScopeAPI,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	if issuer := OAuthIssuer(); issuer != "" {
		claims.Issuer = issuer
		claims.Audience = jwt.ClaimStrings{issuer}
	}
	return claims
}

// TestGenerateAccessTokenRoundTrip covers that a freshly minted token parses back into the
// claims it was built from, including the audience binding applied when an external URL is
// configured.
func TestGenerateAccessTokenRoundTrip(t *testing.T) {
	withSigningKey(t, "https://treningheten.example/")

	userID := uuid.New()
	token, expiresAt, err := GenerateAccessToken(userID, true, ScopeForUser(true), "treningheten-web")
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	if time.Until(expiresAt) > AccessTokenTTL+time.Minute || time.Until(expiresAt) < AccessTokenTTL-time.Minute {
		t.Errorf("expiry %v is not ~%v out", expiresAt, AccessTokenTTL)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("user id = %v, want %v", claims.UserID, userID)
	}
	if !claims.Admin {
		t.Errorf("admin claim = false, want true")
	}
	if claims.Scope != ScopeForUser(true) {
		t.Errorf("scope = %q, want %q", claims.Scope, ScopeForUser(true))
	}
	if claims.ClientID != "treningheten-web" {
		t.Errorf("client id = %q, want %q", claims.ClientID, "treningheten-web")
	}
	// The trailing slash on the configured URL must be trimmed before it becomes the issuer.
	if claims.Issuer != "https://treningheten.example" {
		t.Errorf("issuer = %q, want %q", claims.Issuer, "https://treningheten.example")
	}
}

// TestParseTokenRejections covers every way a presented token must be refused. Each case is
// a token that is well-formed apart from the one property under test.
func TestParseTokenRejections(t *testing.T) {
	withSigningKey(t, "")
	userID := uuid.New()

	expired := baseClaims(userID)
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))

	notYetValid := baseClaims(userID)
	notYetValid.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour))
	notYetValid.ExpiresAt = jwt.NewNumericDate(time.Now().Add(2 * time.Hour))

	noExpiry := baseClaims(userID)
	noExpiry.ExpiresAt = nil

	noNotBefore := baseClaims(userID)
	noNotBefore.NotBefore = nil

	tests := []struct {
		name  string
		token string
	}{
		{"expired", signClaims(t, expired)},
		{"not yet valid", signClaims(t, notYetValid)},
		{"missing expiry claim", signClaims(t, noExpiry)},
		{"missing not-before claim", signClaims(t, noNotBefore)},
		{"empty string", ""},
		{"not a jwt", "this-is-not-a-token"},
		{"personal access token", GeneratePATToken()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseToken(test.token); err == nil {
				t.Errorf("ParseToken accepted a %s token", test.name)
			}
		})
	}
}

// TestParseTokenRejectsForeignSignature covers that a token signed with a different key is
// refused — the check that stops anyone who can craft claims from minting sessions.
func TestParseTokenRejectsForeignSignature(t *testing.T) {
	withSigningKey(t, "")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, baseClaims(uuid.New()))
	foreign, err := token.SignedString([]byte("a-completely-different-key"))
	if err != nil {
		t.Fatalf("failed to sign with foreign key: %v", err)
	}

	if _, err := ParseToken(foreign); err == nil {
		t.Errorf("ParseToken accepted a token signed with a foreign key")
	}
}

// TestParseTokenRejectsUnsignedToken covers the alg-confusion guard: a token declaring
// "alg: none" carries no signature at all, so accepting it would let anyone mint an admin
// session. ParseToken must insist on HMAC.
func TestParseTokenRejectsUnsignedToken(t *testing.T) {
	withSigningKey(t, "")

	claims := baseClaims(uuid.New())
	claims.Admin = true
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build unsigned token: %v", err)
	}

	if _, err := ParseToken(unsigned); err == nil {
		t.Errorf("ParseToken accepted an unsigned (alg=none) token")
	}
}

// TestParseTokenAudienceBinding covers that a token is only accepted by the resource it was
// issued for, and that the check is skipped when no external URL is configured.
func TestParseTokenAudienceBinding(t *testing.T) {
	// Mint against one issuer...
	withSigningKey(t, "https://treningheten.example")
	token, _, err := GenerateAccessToken(uuid.New(), false, models.ScopeAPI, "treningheten-web")
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	if _, err := ParseToken(token); err != nil {
		t.Fatalf("ParseToken rejected a token for its own audience: %v", err)
	}

	// ...and present it to another. The signing key is shared here, so the audience is the
	// only thing standing between the two resources.
	files.ConfigFile.TreninghetenExternalURL = "https://someone-else.example"
	if _, err := ParseToken(token); err == nil {
		t.Errorf("ParseToken accepted a token minted for a different audience")
	}

	// With no external URL configured there is nothing to bind to, so it parses again.
	files.ConfigFile.TreninghetenExternalURL = ""
	if _, err := ParseToken(token); err != nil {
		t.Errorf("ParseToken rejected a token while audience binding was off: %v", err)
	}
}

// TestValidateTokenAdminClaim covers the claim half of the admin check. The DB half (the
// user still being an admin) is enforced separately in ValidateToken and needs a database.
func TestValidateTokenAdminClaim(t *testing.T) {
	withSigningKey(t, "")

	token, _, err := GenerateAccessToken(uuid.New(), false, models.ScopeAPI, "treningheten-web")
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	if err := ValidateToken(token, false); err != nil {
		t.Errorf("ValidateToken rejected a valid non-admin token: %v", err)
	}
	if err := ValidateToken(token, true); err == nil {
		t.Errorf("ValidateToken accepted a non-admin token for an admin route")
	}
}

// TestHashToken covers the storage hash used for opaque tokens: stable, hex-encoded, and
// never equal to the token it came from.
func TestHashToken(t *testing.T) {
	token := GeneratePATToken()

	hash := HashToken(token)
	if hash != HashToken(token) {
		t.Errorf("HashToken is not stable across calls")
	}
	if hash == token {
		t.Errorf("HashToken returned the token itself")
	}
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64 hex characters", len(hash))
	}
	if strings.ContainsAny(hash, "GHIJKLMNOPQRSTUVWXYZ") {
		t.Errorf("hash %q is not lowercase hex", hash)
	}
	if HashToken(token) == HashToken(token+"x") {
		t.Errorf("distinct tokens hashed to the same value")
	}
}

// TestGeneratePATToken covers the properties the auth middleware relies on: the prefix it
// dispatches on, and that tokens don't repeat.
func TestGeneratePATToken(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		token := GeneratePATToken()

		if !strings.HasPrefix(token, models.PATPrefix) {
			t.Fatalf("token %q lacks the %q prefix the middleware dispatches on", token, models.PATPrefix)
		}
		if len(strings.TrimPrefix(token, models.PATPrefix)) < 32 {
			t.Errorf("token %q has too little entropy after the prefix", token)
		}
		if seen[token] {
			t.Fatalf("GeneratePATToken repeated a token")
		}
		seen[token] = true
	}
}

// TestAudienceContains covers the audience matching helper, including that it matches whole
// entries rather than prefixes.
func TestAudienceContains(t *testing.T) {
	audience := jwt.ClaimStrings{"https://a.example", "https://b.example"}

	if !audienceContains(audience, "https://b.example") {
		t.Errorf("audienceContains missed a present entry")
	}
	if audienceContains(audience, "https://a.example/extra") {
		t.Errorf("audienceContains matched a longer value")
	}
	if audienceContains(audience, "https://a.exa") {
		t.Errorf("audienceContains matched a prefix")
	}
	if audienceContains(nil, "https://a.example") {
		t.Errorf("audienceContains matched against an empty audience")
	}
}
