package controllers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	spotifyTokenURL          = "https://accounts.spotify.com/api/token"
	spotifyRecentlyPlayedURL = "https://api.spotify.com/v1/me/player/recently-played"
	// spotifyTokenSkew refreshes a little before the real expiry so an in-flight
	// request never races the deadline.
	spotifyTokenSkew = 60 * time.Second
)

// spotifyEnabled reports whether Spotify is usable: the tenant media flag, the
// provider flag, and the registered-app credentials must all be present.
func spotifyEnabled() bool {
	s := files.ConfigFile.Media.Spotify
	return mediaEnabled() && s.Enabled && s.ClientID != "" && s.ClientSecret != "" && s.RedirectURI != ""
}

func requireSpotifyEnabled(context *gin.Context) bool {
	if !spotifyEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Spotify integration is not enabled."})
		context.Abort()
		return false
	}
	return true
}

// spotifyTokenRequest performs a form-encoded POST to the Spotify token endpoint
// with HTTP Basic auth (client id/secret), used for both the code exchange and the
// refresh grant.
func spotifyTokenRequest(form url.Values) (models.SpotifyTokenResponse, error) {
	token := models.SpotifyTokenResponse{}

	req, err := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return token, errors.New("Spotify token request generation threw error.")
	}
	basic := base64.StdEncoding.EncodeToString([]byte(files.ConfigFile.Media.Spotify.ClientID + ":" + files.ConfigFile.Media.Spotify.ClientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basic)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("Spotify token request threw error. Error: " + err.Error())
		return token, errors.New("Spotify token request threw error.")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("Spotify token endpoint returned non-200. Status: " + strconv.Itoa(resp.StatusCode) + " Body: " + string(body))
		return token, errors.New("Spotify rejected the token request.")
	}

	if err := json.Unmarshal(body, &token); err != nil {
		logger.Log.Error("Failed to parse Spotify token response. Error: " + err.Error())
		return token, errors.New("Failed to parse Spotify token response.")
	}

	return token, nil
}

// APISpotifyCallback completes the OAuth flow: it exchanges the authorization code
// (relayed by the /oauth page) for access + refresh tokens and stores the connection.
func APISpotifyCallback(context *gin.Context) {
	if !requireSpotifyEnabled(context) {
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	var request models.SpotifyCallbackRequest
	if err := context.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Code) == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Missing Spotify authorization code."})
		context.Abort()
		return
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(request.Code))
	form.Set("redirect_uri", files.ConfigFile.Media.Spotify.RedirectURI)

	tokens, err := spotifyTokenRequest(form)
	if err != nil {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to connect Spotify."})
		context.Abort()
		return
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Spotify returned an incomplete token set."})
		context.Abort()
		return
	}

	connection, err := storeSpotifyTokens(userID, tokens)
	if err != nil {
		logger.Log.Info("Failed to store Spotify connection. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store Spotify connection."})
		context.Abort()
		return
	}

	object := ConvertMediaConnectionToObject(connection)
	context.JSON(http.StatusOK, gin.H{"message": "Spotify connected.", "connection": object})
}

// storeSpotifyTokens encrypts the access + refresh tokens and upserts the user's
// Spotify connection, recording the access-token expiry.
func storeSpotifyTokens(userID uuid.UUID, tokens models.SpotifyTokenResponse) (models.MediaConnection, error) {
	access, err := utilities.EncryptString(tokens.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return models.MediaConnection{}, errors.New("failed to encrypt Spotify access token: " + err.Error())
	}
	refresh, err := utilities.EncryptString(tokens.RefreshToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return models.MediaConnection{}, errors.New("failed to encrypt Spotify refresh token: " + err.Error())
	}
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	existing, err := database.GetMediaConnectionForUserProvider(userID, models.MediaProviderSpotify)
	if err != nil {
		return models.MediaConnection{}, err
	}

	if existing != nil {
		existing.Enabled = true
		existing.AccessToken = &access
		existing.RefreshToken = &refresh
		existing.TokenExpiresAt = &expiresAt
		return database.UpdateMediaConnectionInDB(*existing)
	}

	connection := models.MediaConnection{
		Enabled:        true,
		UserID:         userID,
		Provider:       models.MediaProviderSpotify,
		AccessToken:    &access,
		RefreshToken:   &refresh,
		TokenExpiresAt: &expiresAt,
	}
	connection.ID = uuid.New()
	return database.CreateMediaConnectionInDB(connection)
}

// spotifyEnsureToken returns a valid access token, transparently refreshing it (and
// persisting the new token + expiry) when the stored one is expired or near expiry.
func spotifyEnsureToken(connection *models.MediaConnection) (string, error) {
	if connection.AccessToken == nil {
		return "", errors.New("no Spotify access token stored")
	}

	access, err := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return "", errors.New("failed to decrypt Spotify access token")
	}

	if connection.TokenExpiresAt != nil && time.Now().Before(connection.TokenExpiresAt.Add(-spotifyTokenSkew)) {
		return access, nil
	}

	if connection.RefreshToken == nil {
		return "", errors.New("no Spotify refresh token stored")
	}
	refresh, err := utilities.DecryptString(*connection.RefreshToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return "", errors.New("failed to decrypt Spotify refresh token")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refresh)

	tokens, err := spotifyTokenRequest(form)
	if err != nil {
		return "", err
	}
	if tokens.AccessToken == "" {
		return "", errors.New("Spotify refresh returned no access token")
	}

	encrypted, err := utilities.EncryptString(tokens.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return "", errors.New("failed to encrypt refreshed Spotify token")
	}
	connection.AccessToken = &encrypted
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	connection.TokenExpiresAt = &expiresAt
	// Spotify occasionally rotates the refresh token; persist it when it does.
	if tokens.RefreshToken != "" {
		if encR, encErr := utilities.EncryptString(tokens.RefreshToken, files.ConfigFile.Media.TokenKey); encErr == nil {
			connection.RefreshToken = &encR
		}
	}
	if _, err := database.UpdateMediaConnectionInDB(*connection); err != nil {
		logger.Log.Warn("Failed to persist refreshed Spotify token. Error: " + err.Error())
	}

	return tokens.AccessToken, nil
}

// spotifyFetchRecentlyPlayed returns the user's last ~50 plays (Spotify caps history
// at the most recent ~24h, so the sync must run promptly after the activity).
func spotifyFetchRecentlyPlayed(token string) ([]models.SpotifyPlayHistory, error) {
	req, err := http.NewRequest("GET", spotifyRecentlyPlayedURL+"?limit=50", nil)
	if err != nil {
		return nil, errors.New("Spotify history request generation threw error.")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("Spotify history request threw error. Error: " + err.Error())
		return nil, errors.New("Spotify history request threw error.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("Spotify history returned non-200. Status: " + strconv.Itoa(resp.StatusCode))
		return nil, errors.New("Spotify history returned non-200 status.")
	}

	history := models.SpotifyRecentlyPlayed{}
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		logger.Log.Error("Failed to parse Spotify history. Error: " + err.Error())
		return nil, errors.New("Failed to parse Spotify history.")
	}

	return history.Items, nil
}

// buildSpotifyPlaybackForWindow maps recently-played items into provider-neutral
// play events and defers window matching to the shared playbackForWindow.
func buildSpotifyPlaybackForWindow(items []models.SpotifyPlayHistory, start, end time.Time) []models.MediaPlayback {
	events := []mediaPlayEvent{}

	for _, item := range items {
		playedAt, err := time.Parse(time.RFC3339, item.PlayedAt)
		if err != nil {
			continue
		}

		names := []string{}
		for _, artist := range item.Track.Artists {
			if strings.TrimSpace(artist.Name) != "" {
				names = append(names, artist.Name)
			}
		}

		var lengthSeconds int64
		if item.Track.DurationMs > 0 {
			lengthSeconds = item.Track.DurationMs / 1000
		}

		events = append(events, mediaPlayEvent{
			title:          item.Track.Name,
			artist:         strings.Join(names, ", "),
			album:          item.Track.Album.Name,
			providerItemID: item.Track.ID,
			artworkURL:     spotifySmallestImage(item.Track.Album.Images),
			startedAt:      playedAt.UTC(),
			trackLengthSec: lengthSeconds,
		})
	}

	return playbackForWindow(events, start, end)
}

// spotifySmallestImage picks the smallest album image (the list is ordered largest
// first), which is plenty for the rail thumbnail and saves bandwidth.
func spotifySmallestImage(images []models.SpotifyImage) string {
	if len(images) == 0 {
		return ""
	}
	return images[len(images)-1].URL
}

// SpotifySyncOperationForUser pulls the Spotify listening history overlapping one
// operation's window and stores it (delete-and-replace per provider), stamping the
// pull guard regardless of outcome.
func SpotifySyncOperationForUser(user models.User, operation models.Operation) error {
	connection, err := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderSpotify)
	if err != nil {
		return err
	}
	if connection == nil || connection.AccessToken == nil {
		return database.SetOperationMediaRetrievedAt(operation.ID, time.Now())
	}

	token, err := spotifyEnsureToken(connection)
	if err != nil {
		return err
	}

	exercise, err := database.GetExerciseByIDAndUserID(operation.ExerciseID, user.ID)
	if err != nil {
		return err
	} else if exercise == nil {
		return errors.New("exercise not found for media sync")
	}

	start, end := activityWindowForExercise(*exercise)

	items, err := spotifyFetchRecentlyPlayed(token)
	if err != nil {
		return err
	}

	playback := buildSpotifyPlaybackForWindow(items, start, end)

	if err := database.ReplaceMediaPlaybackForOperationProvider(operation.ID, models.MediaProviderSpotify, playback); err != nil {
		return err
	}
	if err := database.SetOperationMediaRetrievedAt(operation.ID, time.Now()); err != nil {
		return err
	}

	now := time.Now()
	connection.LastSyncedAt = &now
	if _, err := database.UpdateMediaConnectionInDB(*connection); err != nil {
		logger.Log.Warn("Failed to update Spotify connection last-synced time. Error: " + err.Error())
	}

	logger.Log.Info("Synced " + strconv.Itoa(len(playback)) + " Spotify playback rows for operation " + operation.ID.String())
	return nil
}
