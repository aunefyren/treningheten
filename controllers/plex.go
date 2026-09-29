package controllers

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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

// The plex.tv endpoints are vars solely so tests can point them at an httptest server
// (like the Strava/Hevy/Spotify endpoints) — never reassign them at runtime.
var (
	plexPinsURL      = "https://plex.tv/api/v2/pins"
	plexUserURL      = "https://plex.tv/api/v2/user"
	plexResourcesURL = "https://plex.tv/api/v2/resources"
)

const (
	plexAuthAppURL = "https://app.plex.tv/auth"
	// plexProbeTimeout bounds each server-connection reachability check so an
	// unreachable address fails fast and the next candidate is tried.
	plexProbeTimeout = 5 * time.Second
)

// plexProduct is the X-Plex-Product header value — the install's display name, so
// the user sees a recognisable device entry in their Plex account.
func plexProduct() string {
	if files.ConfigFile.TreninghetenName != "" {
		return files.ConfigFile.TreninghetenName
	}
	return "Treningheten"
}

// plexRequest performs a plex.tv request with the standard Plex identity headers.
// When token is non-empty it is sent as X-Plex-Token to authenticate as the user.
func plexRequest(method, rawURL string, token string) ([]byte, int, error) {
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		logger.Log.Error("Plex request generation threw error. Error: " + err.Error())
		return nil, 0, errors.New("Plex request generation threw error.")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Product", plexProduct())
	req.Header.Set("X-Plex-Client-Identifier", files.ConfigFile.Media.Plex.ClientIdentifier)
	if token != "" {
		req.Header.Set("X-Plex-Token", token)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("Plex request threw error. Error: " + err.Error())
		return nil, 0, errors.New("Plex request threw error.")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read Plex reply body. Error: " + err.Error())
		return nil, resp.StatusCode, errors.New("Failed to read Plex reply body.")
	}

	return body, resp.StatusCode, nil
}

// plexTLSVerificationSkipped reports whether TLS verification has to be skipped for a
// given PMS host.
//
// It used to be skipped unconditionally, which meant the user's decrypted X-Plex-Token was
// carried to *public* hosts over a connection any on-path attacker could impersonate. The
// skip is now limited to the hosts that cannot present a publicly-verifiable certificate:
// a bare IP address, a loopback/mDNS name, and plex.direct (kept as-is so existing
// connections don't start failing). Every other hostname — anything reached across the
// internet, which is where the risk actually lives — is verified normally.
func plexTLSVerificationSkipped(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return false
	}

	if net.ParseIP(host) != nil {
		return true
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	// Plex hands out <ip-with-dashes>.<hash>.plex.direct addresses for direct connections.
	return host == "plex.direct" || strings.HasSuffix(host, ".plex.direct")
}

// plexServerClient is an HTTP client for talking directly to a user's PMS (history,
// library sections, artwork, reachability probes). The destination is user-supplied, so
// connections go through the media destination policy (see media_dial.go), and TLS
// verification is only skipped for the hosts that genuinely cannot be verified.
func plexServerClient(timeout time.Duration, target *url.URL) *http.Client {
	var tlsConfig *tls.Config
	if target != nil && plexTLSVerificationSkipped(target.Hostname()) {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return mediaHTTPClient(timeout, tlsConfig)
}

// A plex.tv PIN is claimable by whoever polls it with this install's client identifier —
// and that identifier is install-wide, shared by every user here. PIN ids are sequential,
// so without an owner record one user could poll for another's in-flight PIN and have that
// person's Plex token stored as their own connection. Creating a PIN therefore records who
// it belongs to, and the poll refuses anyone else.
//
// The record is deliberately in-memory (like the resized-image cache): a PIN is valid for
// minutes, the rows would be pure churn in the schema, and the worst case on a restart
// mid-flow is that the user clicks Connect again.
const plexPinTTL = 15 * time.Minute

type plexPinOwner struct {
	userID    uuid.UUID
	createdAt time.Time
}

var (
	plexPinOwnersMutex sync.Mutex
	plexPinOwners      = map[int64]plexPinOwner{}
)

// prunePlexPinsLocked drops expired records. Callers must hold plexPinOwnersMutex.
func prunePlexPinsLocked(now time.Time) {
	for id, owner := range plexPinOwners {
		if now.Sub(owner.createdAt) > plexPinTTL {
			delete(plexPinOwners, id)
		}
	}
}

// rememberPlexPin records the user a freshly created PIN belongs to.
func rememberPlexPin(pinID int64, userID uuid.UUID) {
	plexPinOwnersMutex.Lock()
	defer plexPinOwnersMutex.Unlock()

	now := time.Now()
	prunePlexPinsLocked(now)
	plexPinOwners[pinID] = plexPinOwner{userID: userID, createdAt: now}
}

// plexPinBelongsTo reports whether the PIN was created by this user and is still fresh.
func plexPinBelongsTo(pinID int64, userID uuid.UUID) bool {
	plexPinOwnersMutex.Lock()
	defer plexPinOwnersMutex.Unlock()

	now := time.Now()
	prunePlexPinsLocked(now)

	owner, found := plexPinOwners[pinID]
	return found && owner.userID == userID
}

// forgetPlexPin drops a record once the PIN has been redeemed.
func forgetPlexPin(pinID int64) {
	plexPinOwnersMutex.Lock()
	defer plexPinOwnersMutex.Unlock()

	delete(plexPinOwners, pinID)
}

// buildPlexAuthURL composes the browser URL the user opens to approve a PIN.
func buildPlexAuthURL(clientIdentifier, code, product string) string {
	v := url.Values{}
	v.Set("clientID", clientIdentifier)
	v.Set("code", code)
	v.Set("context[device][product]", product)
	// Plex reads these params from the URL fragment, not the query string.
	return plexAuthAppURL + "#?" + v.Encode()
}

// rankPlexServerConnections returns candidate server connection URIs, best first.
// Treningheten's server (not the browser) must reach the PMS, and there is no single
// correct address: it may run on the same LAN as Plex (local URI is fastest/only
// reachable) or remotely (needs the public/relay URI). So it ranks all connections
// and the caller probes them in order — preferring non-relay, then LOCAL (a
// co-located install is the common self-hosted case), then https, then owned.
func rankPlexServerConnections(resources []models.PlexResource) []string {
	type scored struct {
		uri   string
		score int
	}
	candidates := []scored{}

	for _, resource := range resources {
		if !strings.Contains(strings.ToLower(resource.Provides), "server") {
			continue
		}
		for _, conn := range resource.Connections {
			if conn.URI == "" {
				continue
			}
			score := 0
			if !conn.Relay {
				score += 8 // relay is the slow last resort
			}
			if conn.Local {
				score += 4 // prefer LAN — Treningheten is often co-located
			}
			if strings.EqualFold(conn.Protocol, "https") {
				score += 2
			}
			if resource.Owned {
				score += 1
			}
			candidates = append(candidates, scored{uri: conn.URI, score: score})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	uris := make([]string, 0, len(candidates))
	for _, c := range candidates {
		uris = append(uris, c.uri)
	}
	return uris
}

// probePlexServer reports whether a PMS connection URI is reachable, by hitting its
// unauthenticated /identity endpoint with a short timeout. A candidate the destination
// policy refuses simply fails to dial, so it is skipped like any other unreachable
// address and the next candidate is tried.
func probePlexServer(uri string, token string) bool {
	req, err := http.NewRequest("GET", strings.TrimRight(uri, "/")+"/identity", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", files.ConfigFile.Media.Plex.ClientIdentifier)

	resp, err := plexServerClient(plexProbeTimeout, req.URL).Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// selectReachablePlexServer probes ranked candidates and returns the first that
// responds, or "" when none are reachable from this host.
func selectReachablePlexServer(resources []models.PlexResource, token string) string {
	for _, uri := range rankPlexServerConnections(resources) {
		if probePlexServer(uri, token) {
			return uri
		}
	}
	return ""
}

// resolvePlexServerAccountID maps a plex.tv username to the SERVER-LOCAL account id
// used in history rows (the owner is usually 1). History must be scoped by this id,
// not the plex.tv global id — otherwise the server returns nothing (failing closed)
// or, if unscoped, every user's plays (a privacy leak). Returns "" when it can't be
// resolved, in which case the caller keeps history scoped (fail closed) rather than
// leaking other users' data.
func resolvePlexServerAccountID(serverURL, token string, candidates ...string) string {
	req, err := http.NewRequest("GET", strings.TrimRight(serverURL, "/")+"/accounts", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", files.ConfigFile.Media.Plex.ClientIdentifier)

	resp, err := plexServerClient(plexProbeTimeout, req.URL).Do(req)
	if err != nil {
		logger.Log.Warn("Failed to fetch Plex server accounts. Error: " + err.Error())
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Log.Warn("Plex server accounts returned non-200. Status: " + strconv.Itoa(resp.StatusCode))
		return ""
	}

	accounts := models.PlexServerAccountsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		logger.Log.Warn("Failed to parse Plex server accounts. Error: " + err.Error())
		return ""
	}

	for _, account := range accounts.MediaContainer.Account {
		if account.ID <= 0 {
			continue
		}
		name := strings.TrimSpace(account.Name)
		for _, candidate := range candidates {
			candidate = strings.TrimSpace(candidate)
			if candidate != "" && strings.EqualFold(name, candidate) {
				return strconv.FormatInt(account.ID, 10)
			}
		}
	}

	logger.Log.Warn("Could not match plex.tv identity to a server-local account; history will fail closed.")
	return ""
}

// plexFetchAccountAndServer resolves the authorized token into the server-local
// account id (for scoping history to this user) and a reachable server URL. A
// missing server is not fatal here — the connection is still stored so the user can
// retry discovery / pull later; the history pull surfaces it.
func plexFetchAccountAndServer(token string) (accountID string, serverURL *string, err error) {
	body, status, err := plexRequest("GET", plexUserURL, token)
	if err != nil {
		return "", nil, err
	}
	if status != http.StatusOK {
		logger.Log.Error("Plex user lookup returned non-200. Status: " + strconv.Itoa(status))
		return "", nil, errors.New("Plex user lookup failed.")
	}

	account := models.PlexAccount{}
	if err := json.Unmarshal(body, &account); err != nil {
		logger.Log.Error("Failed to parse Plex account. Error: " + err.Error())
		return "", nil, errors.New("Failed to parse Plex account.")
	}

	resBody, resStatus, err := plexRequest("GET", plexResourcesURL+"?includeHttps=1", token)
	if err != nil {
		logger.Log.Warn("Failed to fetch Plex resources; storing connection without a server URL. Error: " + err.Error())
		return "", nil, nil
	}
	if resStatus != http.StatusOK {
		logger.Log.Warn("Plex resources lookup returned non-200; storing connection without a server URL. Status: " + strconv.Itoa(resStatus))
		return "", nil, nil
	}

	resources := []models.PlexResource{}
	if err := json.Unmarshal(resBody, &resources); err != nil {
		logger.Log.Warn("Failed to parse Plex resources; storing connection without a server URL. Error: " + err.Error())
		return "", nil, nil
	}

	if serverURLValue := selectReachablePlexServer(resources, token); serverURLValue != "" {
		serverURL = &serverURLValue
		// Scope history to this user via the server-local account id.
		accountID = resolvePlexServerAccountID(serverURLValue, token, account.Username, account.Email)
	} else {
		logger.Log.Warn("No reachable Plex server connection found; storing connection without a server URL.")
	}

	return accountID, serverURL, nil
}

// APICreatePlexPin starts the plex.tv PIN flow and returns the code + browser auth
// URL for the frontend to open, plus the pin id the frontend polls.
func APICreatePlexPin(context *gin.Context) {
	if !requirePlexEnabled(context) {
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	body, status, err := plexRequest("POST", plexPinsURL+"?strong=true", "")
	if err != nil {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach Plex."})
		context.Abort()
		return
	}
	if status != http.StatusCreated && status != http.StatusOK {
		logger.Log.Error("Plex pin creation returned unexpected status: " + strconv.Itoa(status))
		context.JSON(http.StatusBadGateway, gin.H{"error": "Plex declined to create a PIN."})
		context.Abort()
		return
	}

	pin := models.PlexPin{}
	if err := json.Unmarshal(body, &pin); err != nil {
		logger.Log.Error("Failed to parse Plex pin. Error: " + err.Error())
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse Plex response."})
		context.Abort()
		return
	}

	// Bind the PIN to its creator so nobody else can redeem it (see plexPinOwners).
	rememberPlexPin(pin.ID, userID)

	response := models.PlexPinResponse{
		PinID:   pin.ID,
		Code:    pin.Code,
		AuthURL: buildPlexAuthURL(files.ConfigFile.Media.Plex.ClientIdentifier, pin.Code, plexProduct()),
	}

	context.JSON(http.StatusOK, gin.H{"message": "Plex PIN created.", "pin": response})
}

// APICheckPlexPin polls a PIN. While unapproved it returns authorized=false; once
// the user approves it, the account token is resolved, the server is discovered,
// the token is encrypted at rest, and the MediaConnection is upserted.
func APICheckPlexPin(context *gin.Context) {
	if !requirePlexEnabled(context) {
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	pinID := context.Param("pin_id")
	parsedPinID, convErr := strconv.ParseInt(pinID, 10, 64)
	if convErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PIN id."})
		context.Abort()
		return
	}

	// Only the user who created this PIN may redeem it. A PIN belonging to someone else is
	// reported exactly like one that never existed, so the endpoint can't be used to probe
	// which PIN ids are live.
	if !plexPinBelongsTo(parsedPinID, userID) {
		context.JSON(http.StatusNotFound, gin.H{"error": "Unknown PIN."})
		context.Abort()
		return
	}

	body, status, err := plexRequest("GET", plexPinsURL+"/"+pinID, "")
	if err != nil {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to reach Plex."})
		context.Abort()
		return
	}
	if status != http.StatusOK {
		logger.Log.Error("Plex pin check returned non-200. Status: " + strconv.Itoa(status))
		context.JSON(http.StatusBadGateway, gin.H{"error": "Plex declined the PIN check."})
		context.Abort()
		return
	}

	pin := models.PlexPin{}
	if err := json.Unmarshal(body, &pin); err != nil {
		logger.Log.Error("Failed to parse Plex pin. Error: " + err.Error())
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse Plex response."})
		context.Abort()
		return
	}

	// Not approved yet — the frontend keeps polling.
	if pin.AuthToken == nil || *pin.AuthToken == "" {
		context.JSON(http.StatusOK, gin.H{"message": "PIN not yet authorized.", "result": models.PlexPinCheckResponse{Authorized: false}})
		return
	}

	accountID, serverURL, err := plexFetchAccountAndServer(*pin.AuthToken)
	if err != nil {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to resolve Plex account."})
		context.Abort()
		return
	}

	connection, err := upsertMediaConnection(userID, models.MediaProviderPlex, *pin.AuthToken, serverURL, &accountID)
	if err != nil {
		logger.Log.Info("Failed to store Plex connection. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store Plex connection."})
		context.Abort()
		return
	}

	forgetPlexPin(parsedPinID)

	object := ConvertMediaConnectionToObject(connection)
	context.JSON(http.StatusOK, gin.H{"message": "Plex connected.", "result": models.PlexPinCheckResponse{Authorized: true, Connection: &object}})
}

// APISetPlexServerURL overrides the stored Plex server URL. Auto-discovery cannot
// see a reverse-proxy front (e.g. Cloudflare on :443, where the advertised :32400
// addresses are unreachable), so the user can set the working URL by hand. The URL
// is probed for reachability and the result is reported, but it is saved regardless
// so the user isn't blocked by a transient probe failure.
func APISetPlexServerURL(context *gin.Context) {
	if !requirePlexEnabled(context) {
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	var request models.PlexServerURLRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	serverURL, urlErr := validateMediaServerURL(request.ServerURL)
	if urlErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": urlErr.Error()})
		context.Abort()
		return
	}

	connection, err := database.GetMediaConnectionForUserProvider(userID, models.MediaProviderPlex)
	if err != nil {
		logger.Log.Info("Failed to get Plex connection. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Plex connection."})
		context.Abort()
		return
	}
	if connection == nil || connection.AccessToken == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Connect Plex before setting a server URL."})
		context.Abort()
		return
	}

	// Probe with the stored token to give the user feedback on the URL they entered,
	// and (re)resolve the server-local account id against this URL — required to scope
	// history to this user. This is the path that matters for reverse-proxy setups,
	// where the original connect couldn't reach the server to resolve it.
	reachable := false
	if token, decErr := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey); decErr == nil {
		reachable = probePlexServer(serverURL, token)

		if body, status, reqErr := plexRequest("GET", plexUserURL, token); reqErr == nil && status == http.StatusOK {
			account := models.PlexAccount{}
			if json.Unmarshal(body, &account) == nil {
				if id := resolvePlexServerAccountID(serverURL, token, account.Username, account.Email); id != "" {
					connection.AccountID = &id
				}
			}
		}
	}

	connection.ServerURL = &serverURL
	updated, err := database.UpdateMediaConnectionInDB(*connection)
	if err != nil {
		logger.Log.Info("Failed to update Plex server URL. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save server URL."})
		context.Abort()
		return
	}

	object := ConvertMediaConnectionToObject(updated)
	message := "Server URL saved."
	if !reachable {
		message = "Server URL saved, but it did not respond to a test request — double-check it if syncing fails."
	}
	context.JSON(http.StatusOK, gin.H{"message": message, "reachable": reachable, "connection": object})
}

// upsertMediaConnection creates or updates a user's connection for a provider,
// encrypting the access token at rest with Media.TokenKey (reusing Strava's scheme).
func upsertMediaConnection(userID uuid.UUID, provider, accessToken string, serverURL, accountID *string) (models.MediaConnection, error) {
	encrypted, err := utilities.EncryptString(accessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return models.MediaConnection{}, errors.New("failed to encrypt media access token: " + err.Error())
	}

	existing, err := database.GetMediaConnectionForUserProvider(userID, provider)
	if err != nil {
		return models.MediaConnection{}, err
	}

	if existing != nil {
		existing.Enabled = true
		existing.AccessToken = &encrypted
		existing.ServerURL = serverURL
		existing.AccountID = accountID
		return database.UpdateMediaConnectionInDB(*existing)
	}

	connection := models.MediaConnection{
		Enabled:     true,
		UserID:      userID,
		Provider:    provider,
		AccessToken: &encrypted,
		ServerURL:   serverURL,
		AccountID:   accountID,
	}
	connection.ID = uuid.New()
	return database.CreateMediaConnectionInDB(connection)
}

// plexArtworkThumbPath matches the shape of a real Plex cover-thumb path. The stored
// values come from Plex's own Thumb field (see plex_sync.go) and look like
// "/library/metadata/1/thumb/9" or "/library/parts/5/file.jpg". The character class has no
// "%", so a percent-encoded traversal cannot survive it either.
var plexArtworkThumbPath = regexp.MustCompile(`^/library/[A-Za-z0-9/._-]+$`)

// plexArtworkPathAllowed guards the artwork proxy against SSRF: only Plex library image
// paths may be fetched with the user's server token, never an arbitrary URL a caller
// crafts. A "/library/" prefix test is not enough on its own — Go's HTTP client does not
// remove dot-segments from a request path, so "/library/../status/sessions" would reach
// the PMS. A dot is legitimate in a thumb path (file extensions), so traversal is rejected
// by refusing ".." outright rather than by banning the character.
//
// It rejects rather than normalises: no legitimate thumb path needs cleaning, so anything
// path.Clean would rewrite is hostile and should 400 instead of being quietly turned into
// something valid.
func plexArtworkPathAllowed(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	return plexArtworkThumbPath.MatchString(path)
}

// APIGetPlexArtwork proxies a Plex cover-art thumbnail. Plex thumbs live on the user's
// own PMS and need the server token to fetch, so they can't be stored as a public URL
// (that would leak the credential). Instead the read layer points artwork_url at this
// endpoint, which fetches from the *requesting* user's Plex server with their decrypted
// token and streams the image back. Only /library/ paths are allowed (SSRF guard); it
// 404s when Plex is disabled, matching the other Plex endpoints. It lives under the
// image-auth group so an <img>/CSS background can load it via the session cookie.
func APIGetPlexArtwork(context *gin.Context) {
	if !requirePlexEnabled(context) {
		return
	}

	userID, err := middlewares.ImageRequestUserID(context)
	if err != nil {
		logger.Log.Info("Failed to authenticate Plex artwork request. Error: " + err.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to authenticate request."})
		context.Abort()
		return
	}

	path := context.Query("path")
	if !plexArtworkPathAllowed(path) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artwork path."})
		context.Abort()
		return
	}

	connection, err := database.GetMediaConnectionForUserProvider(userID, models.MediaProviderPlex)
	if err != nil || connection == nil || connection.AccessToken == nil || connection.ServerURL == nil || *connection.ServerURL == "" {
		context.JSON(http.StatusNotFound, gin.H{"error": "No Plex connection."})
		context.Abort()
		return
	}

	token, err := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		logger.Log.Warn("Failed to decrypt Plex token for artwork. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Plex credentials."})
		context.Abort()
		return
	}

	req, err := http.NewRequest("GET", strings.TrimRight(*connection.ServerURL, "/")+path, nil)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build artwork request."})
		context.Abort()
		return
	}
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", files.ConfigFile.Media.Plex.ClientIdentifier)

	resp, err := plexServerClient(15*time.Second, req.URL).Do(req)
	if err != nil {
		logger.Log.Warn("Plex artwork fetch threw error. Error: " + err.Error())
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch artwork."})
		context.Abort()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		context.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found."})
		context.Abort()
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	// Private (per-user token) but immutable for a day — thumbs don't change under a ratingKey.
	context.Header("Cache-Control", "private, max-age=86400")
	context.DataFromReader(http.StatusOK, resp.ContentLength, contentType, resp.Body, nil)
}
