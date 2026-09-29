package middlewares

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// withAuthDB installs an in-memory, migrated database and a valid signing key.
func withAuthDB(t *testing.T) {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	gormDB, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	previousDB, previousConfig := database.Instance, files.ConfigFile
	database.Instance = gormDB
	database.Migrate()
	files.ConfigFile.PrivateKey = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("m"), 64))
	files.ConfigFile.TreninghetenExternalURL = ""
	files.ConfigFile.SMTPEnabled = false

	t.Cleanup(func() {
		database.Instance, files.ConfigFile = previousDB, previousConfig
		_ = sqlDB.Close()
	})
}

type authUser struct {
	admin, enabled, verified bool
}

func makeAuthUser(t *testing.T, email string, spec authUser) models.User {
	t.Helper()
	user := models.User{FirstName: "M", LastName: "W", Email: email, Password: "x", Admin: &spec.admin, Enabled: spec.enabled, Verified: spec.verified}
	user.ID = uuid.New()
	created, err := database.RegisterUserInDB(user)
	if err != nil {
		t.Fatal(err)
	}
	// RegisterUserInDB may apply defaults; force the flags under test.
	database.Instance.Model(&models.User{}).Where("id = ?", created.ID).Updates(map[string]any{"enabled": spec.enabled, "verified": spec.verified})
	return created
}

func jwtFor(t *testing.T, user models.User, scope string) string {
	t.Helper()
	token, _, err := auth.GenerateAccessToken(user.ID, user.Admin != nil && *user.Admin, scope, "test")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func patFor(t *testing.T, user models.User, scope string, expires time.Time, revoked bool) string {
	t.Helper()
	plain := auth.GeneratePATToken()
	pat := models.PersonalAccessToken{UserID: user.ID, Name: "t", TokenHash: auth.HashToken(plain), Scope: scope, ExpiresAt: expires}
	if revoked {
		now := time.Now()
		pat.RevokedAt = &now
	}
	if err := database.CreatePersonalAccessToken(&pat); err != nil {
		t.Fatal(err)
	}
	return plain
}

func authRouter() *gin.Engine {
	router := gin.New()
	ok := func(context *gin.Context) {
		userID, err := GetAuthUsername(context.GetHeader("Authorization"))
		if err != nil {
			userID, _ = ImageRequestUserID(context)
		}
		context.String(http.StatusOK, userID.String())
	}
	router.GET("/user", Auth(false), ok)
	router.POST("/user", Auth(false), ok)
	router.GET("/admin", Auth(true), ok)
	router.GET("/image", AuthImageReadOnly(), ok)
	return router
}

func send(router *gin.Engine, method, path, token string, cookie string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "treningheten", Value: cookie})
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAuthMiddlewareDecisions(t *testing.T) {
	withAuthDB(t)
	router := authRouter()

	admin := makeAuthUser(t, "admin@mw.test", authUser{admin: true, enabled: true, verified: true})
	member := makeAuthUser(t, "member@mw.test", authUser{enabled: true, verified: true})
	disabled := makeAuthUser(t, "disabled@mw.test", authUser{enabled: false, verified: true})
	unverified := makeAuthUser(t, "unverified@mw.test", authUser{enabled: true, verified: false})
	demoted := makeAuthUser(t, "demoted@mw.test", authUser{enabled: true, verified: true})

	adminJWT := jwtFor(t, admin, auth.ScopeForUser(true))
	memberJWT := jwtFor(t, member, auth.ScopeForUser(false))
	// A token minted with the admin scope for someone who is no longer an admin in the DB.
	demotedJWT := jwtFor(t, demoted, auth.ScopeForUser(true))

	future := time.Now().Add(24 * time.Hour)
	readPAT := patFor(t, member, models.ScopeAPIRead, future, false)
	writePAT := patFor(t, member, models.ScopeAPIWrite, future, false)
	expiredPAT := patFor(t, member, models.ScopeAPIWrite, time.Now().Add(-time.Hour), false)
	revokedPAT := patFor(t, member, models.ScopeAPIWrite, future, true)

	tests := []struct {
		name, method, path, token, cookie string
		want                              int
	}{
		{"no token", "GET", "/user", "", "", http.StatusUnauthorized},
		{"garbage token", "GET", "/user", "not-a-jwt", "", http.StatusUnauthorized},
		{"member JWT", "GET", "/user", memberJWT, "", http.StatusOK},
		{"member JWT write", "POST", "/user", memberJWT, "", http.StatusOK},
		{"member on admin route", "GET", "/admin", memberJWT, "", http.StatusForbidden},
		{"admin on admin route", "GET", "/admin", adminJWT, "", http.StatusOK},
		{"admin scope but not admin in DB", "GET", "/admin", demotedJWT, "", http.StatusForbidden},
		{"disabled account", "GET", "/user", jwtFor(t, disabled, auth.ScopeForUser(false)), "", http.StatusForbidden},
		{"read PAT read", "GET", "/user", readPAT, "", http.StatusOK},
		{"read PAT write", "POST", "/user", readPAT, "", http.StatusForbidden},
		{"write PAT write", "POST", "/user", writePAT, "", http.StatusOK},
		{"expired PAT", "GET", "/user", expiredPAT, "", http.StatusUnauthorized},
		{"revoked PAT", "GET", "/user", revokedPAT, "", http.StatusUnauthorized},
		{"unknown PAT", "GET", "/user", models.PATPrefix + "nope", "", http.StatusUnauthorized},
		{"image via header", "GET", "/image", memberJWT, "", http.StatusOK},
		{"image via cookie", "GET", "/image", "", memberJWT, http.StatusOK},
		{"image without credentials", "GET", "/image", "", "", http.StatusUnauthorized},
		{"image with bad cookie", "GET", "/image", "", "junk", http.StatusUnauthorized},
		{"image for disabled account", "GET", "/image", "", jwtFor(t, disabled, auth.ScopeForUser(false)), http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := send(router, test.method, test.path, test.token, test.cookie).Code; got != test.want {
				t.Errorf("status = %d, want %d", got, test.want)
			}
		})
	}

	// The handler sees the right user, including through the cookie path.
	if body := send(router, "GET", "/image", "", memberJWT).Body.String(); body != member.ID.String() {
		t.Errorf("image request user = %q, want %s", body, member.ID)
	}

	// With SMTP on, an unverified account is held back (and gets a verification code).
	files.ConfigFile.SMTPEnabled = true
	unverifiedJWT := jwtFor(t, unverified, auth.ScopeForUser(false))
	if got := send(router, "GET", "/user", unverifiedJWT, "").Code; got != http.StatusForbidden {
		t.Errorf("unverified with SMTP: status = %d, want 403", got)
	}
	if hasCode, _ := database.VerifyUserHasVerificationCode(unverified.ID); !hasCode {
		t.Error("no verification code was generated for the unverified user")
	}
	if _, err := Authenticate("Bearer " + unverifiedJWT); err == nil {
		t.Error("Authenticate accepted an unverified account with SMTP on")
	}
	if got := send(router, "GET", "/user", memberJWT, "").Code; got != http.StatusOK {
		t.Errorf("verified with SMTP: status = %d, want 200", got)
	}

	// PAT usage is recorded.
	pat, _ := database.GetPersonalAccessTokenByHash(auth.HashToken(readPAT))
	if pat.LastUsedAt == nil {
		t.Error("PAT last_used_at was not recorded")
	}
}

func TestTokenHelpers(t *testing.T) {
	withAuthDB(t)
	member := makeAuthUser(t, "helpers@mw.test", authUser{enabled: true, verified: true})
	token := jwtFor(t, member, auth.ScopeForUser(false))

	if _, err := GetAuthUsername(""); err == nil {
		t.Error("GetAuthUsername accepted an empty header")
	}
	if _, err := GetAuthUsername("Bearer junk"); err == nil {
		t.Error("GetAuthUsername accepted junk")
	}
	if id, err := GetAuthUsername("Bearer " + token); err != nil || id != member.ID {
		t.Errorf("GetAuthUsername = %v, %v", id, err)
	}

	claims, err := GetTokenClaims("Bearer " + token)
	if err != nil || claims.UserID != member.ID {
		t.Errorf("GetTokenClaims = %+v, %v", claims, err)
	}
	if _, err := GetTokenClaims(""); err == nil {
		t.Error("GetTokenClaims accepted an empty header")
	}
	if _, err := GetTokenClaims("Bearer junk"); err == nil {
		t.Error("GetTokenClaims accepted junk")
	}
	if _, err := Authenticate(""); err == nil {
		t.Error("Authenticate accepted an empty header")
	}
	ghost := models.User{}
	ghost.ID = uuid.New()
	if _, err := Authenticate("Bearer " + jwtFor(t, ghost, auth.ScopeForUser(false))); err == nil {
		t.Error("Authenticate accepted a token for a user that doesn't exist")
	}
}
