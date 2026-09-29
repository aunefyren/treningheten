package auth

import (
	"database/sql"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// withAuthDB installs an in-memory, migrated database for the refresh-token tests.
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
	previous := database.Instance
	database.Instance = gormDB
	database.Migrate()
	t.Cleanup(func() {
		database.Instance = previous
		_ = sqlDB.Close()
	})
}

func makeAuthTestUser(t *testing.T, admin bool) models.User {
	t.Helper()
	user := models.User{FirstName: "A", LastName: "B", Email: uuid.NewString() + "@auth.test", Password: "x", Enabled: true, Verified: true, Admin: &admin}
	user.ID = uuid.New()
	created, err := database.RegisterUserInDB(user)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func TestRefreshTokenRotationAndReuseDetection(t *testing.T) {
	withSigningKey(t, "")
	withAuthDB(t)
	user := makeAuthTestUser(t, false)

	issued, err := IssueTokenSet(user.ID, false, models.ScopeAPI, "client-a")
	if err != nil {
		t.Fatalf("IssueTokenSet: %v", err)
	}
	if issued.TokenType != "Bearer" || issued.RefreshToken == "" || issued.ExpiresIn <= 0 {
		t.Fatalf("token set = %+v", issued)
	}
	if claims, err := ParseToken(issued.AccessToken); err != nil || claims.UserID != user.ID {
		t.Fatalf("issued access token: %+v %v", claims, err)
	}

	// Refresh tokens are bound to their client.
	if _, err := RefreshTokenSet(issued.RefreshToken, "client-b"); err == nil {
		t.Error("a refresh token was accepted from another client")
	}

	rotated, err := RefreshTokenSet(issued.RefreshToken, "client-a")
	if err != nil {
		t.Fatalf("RefreshTokenSet: %v", err)
	}
	if rotated.RefreshToken == issued.RefreshToken {
		t.Error("the refresh token was not rotated")
	}

	// Replaying the rotated-out token is reuse: refused, and the whole chain dies with it.
	if _, err := RefreshTokenSet(issued.RefreshToken, "client-a"); err == nil {
		t.Error("a rotated-out refresh token was accepted")
	}
	if _, err := RefreshTokenSet(rotated.RefreshToken, "client-a"); err == nil {
		t.Error("the chain survived reuse of an earlier token")
	}

	if _, err := RefreshTokenSet("never-issued", ""); err == nil {
		t.Error("an unknown refresh token was accepted")
	}
}

func TestRefreshReDerivesAdminAndRejectsExpired(t *testing.T) {
	withSigningKey(t, "")
	withAuthDB(t)
	user := makeAuthTestUser(t, true)

	issued, err := IssueTokenSet(user.ID, true, models.ScopeAPI+" "+models.ScopeAdmin, "web")
	if err != nil {
		t.Fatal(err)
	}

	// Demote the user; the refreshed access token must drop the admin claim.
	notAdmin := false
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("admin", &notAdmin)
	refreshed, err := RefreshTokenSet(issued.RefreshToken, "")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(refreshed.AccessToken)
	if err != nil || claims.Admin {
		t.Errorf("admin claim after demotion = %v (err %v), want false", claims.Admin, err)
	}
	if err := ValidateToken(refreshed.AccessToken, true); err == nil {
		t.Error("a demoted user's token passed admin validation")
	}

	// An expired refresh token is refused.
	expired, err := IssueTokenSet(user.ID, false, models.ScopeAPI, "web")
	if err != nil {
		t.Fatal(err)
	}
	database.Instance.Model(&models.OAuthRefreshToken{}).Where("token_hash = ?", HashToken(expired.RefreshToken)).
		Update("expires_at", time.Now().Add(-time.Hour))
	if _, err := RefreshTokenSet(expired.RefreshToken, "web"); err == nil {
		t.Error("an expired refresh token was accepted")
	}
}

func TestValidateTokenAdmin(t *testing.T) {
	withSigningKey(t, "")
	withAuthDB(t)
	admin := makeAuthTestUser(t, true)
	member := makeAuthTestUser(t, false)

	adminToken, _, _ := GenerateAccessToken(admin.ID, true, ScopeForUser(true), "web")
	memberToken, _, _ := GenerateAccessToken(member.ID, false, ScopeForUser(false), "web")
	// A token claiming admin for someone who isn't one in the database.
	forged, _, _ := GenerateAccessToken(member.ID, true, ScopeForUser(true), "web")
	ghostToken, _, _ := GenerateAccessToken(uuid.New(), true, ScopeForUser(true), "web")

	if err := ValidateToken(adminToken, true); err != nil {
		t.Errorf("admin token: %v", err)
	}
	if err := ValidateToken(memberToken, false); err != nil {
		t.Errorf("member token, non-admin check: %v", err)
	}
	for name, token := range map[string]string{"member": memberToken, "claims admin but isn't": forged, "no such user": ghostToken, "garbage": "x.y.z"} {
		if err := ValidateToken(token, true); err == nil {
			t.Errorf("%s: passed admin validation", name)
		}
	}
}
