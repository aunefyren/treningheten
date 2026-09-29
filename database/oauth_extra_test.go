package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"
)

// Public is tagged default:true, so a plain GORM Create dropped Public=false and stored
// confidential clients as public — the token endpoint then never checked their secret.
func TestCreateOAuthClientStoresConfidentialClientsAsConfidential(t *testing.T) {
	newTestDB(t)

	hash := "bcrypt-hash"
	client := models.OAuthClient{ClientID: "confidential-client", ClientName: "c", RedirectURIs: "https://a/cb",
		GrantTypes: "authorization_code", ResponseTypes: "code", Scope: "api", TokenEndpointAuthMethod: "client_secret_basic",
		ClientSecretHash: &hash, Public: false}
	if err := CreateOAuthClient(&client); err != nil {
		t.Fatal(err)
	}
	stored, err := GetOAuthClientByClientID("confidential-client")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Public {
		t.Error("a confidential client was stored as public")
	}
}
