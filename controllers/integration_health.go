package controllers

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Integration health: noticing that a user's connection to an external service has
// broken, telling them once, and re-pulling what was missed once it works again. See
// docs/integration-health.md.

// The two kinds of provider failure that say something about the connection itself. A
// provider client wraps its errors with integrationAuthError / integrationUnavailableError
// and callers test for them with errors.Is; any other error (a database write, a parse
// of our own data) is not a health signal and is ignored by the tracker.
var (
	ErrIntegrationAuth        = errors.New("The provider rejected the stored credentials.")
	ErrIntegrationUnavailable = errors.New("The provider could not be reached.")
)

const (
	// integrationUnavailableGrace is how long a provider may keep failing before it is
	// reported as unavailable. Self-hosted servers go offline for updates and power cuts,
	// so a short outage should neither alarm the user nor be reported at all.
	integrationUnavailableGrace = 24 * time.Hour
	// integrationBackfillLimit bounds how far back a recovery re-pulls, so a connection
	// that was broken for a year doesn't re-walk the whole history in one go.
	integrationBackfillLimit = 90 * 24 * time.Hour
)

// startIntegrationRecovery runs a recovery backfill. It is goSafely in production — a
// backfill can re-pull months of sessions, which must not hold up a request — and a var
// solely so tests can run it inline; nothing in the app reassigns it.
var startIntegrationRecovery = goSafely

// integrationError keeps the provider's own message while marking what kind of failure
// it is, so errors.Is(err, ErrIntegrationAuth) works without rewording every message.
// cause keeps a provider's own sentinel reachable too (errors.Is(err,
// ErrSpotifyForbidden)); reason refines the status (models.IntegrationReason*).
type integrationError struct {
	kind    error
	cause   error
	reason  string
	message string
}

func (e *integrationError) Error() string { return e.message }

func (e *integrationError) Unwrap() []error {
	if e.cause != nil {
		return []error{e.kind, e.cause}
	}
	return []error{e.kind}
}

// integrationAuthError marks a failure as the provider rejecting the stored credential.
func integrationAuthError(message string) error {
	return &integrationError{kind: ErrIntegrationAuth, message: message}
}

// integrationAuthErrorFor marks a provider sentinel (or a refined reason) as a rejected
// credential, keeping the sentinel's message and identity.
func integrationAuthErrorFor(cause error, reason string) error {
	return &integrationError{kind: ErrIntegrationAuth, cause: cause, reason: reason, message: cause.Error()}
}

// integrationUnavailableError marks a failure as the provider not answering (network
// error, timeout, 5xx, garbage reply).
func integrationUnavailableError(message string) error {
	return &integrationError{kind: ErrIntegrationUnavailable, message: message}
}

// integrationErrorStatus maps a failure onto the status it implies, or "" when the error
// is not a health signal.
func integrationErrorStatus(failure error) string {
	switch {
	case errors.Is(failure, ErrIntegrationAuth):
		return models.IntegrationStatusAuthFailed
	case errors.Is(failure, ErrIntegrationUnavailable):
		return models.IntegrationStatusUnavailable
	default:
		return ""
	}
}

// integrationErrorReason returns the reason a failure carries, or "".
func integrationErrorReason(failure error) string {
	var tagged *integrationError
	if errors.As(failure, &tagged) {
		return tagged.reason
	}
	return ""
}

// integrationDisplayName is the provider name as the user knows it.
func integrationDisplayName(provider string) string {
	switch provider {
	case models.IntegrationProviderPlex:
		return "Plex"
	case models.IntegrationProviderSpotify:
		return "Spotify"
	case models.IntegrationProviderAudiobookshelf:
		return "Audiobookshelf"
	case models.IntegrationProviderStrava:
		return "Strava"
	case models.IntegrationProviderHevy:
		return "Hevy"
	default:
		return provider
	}
}

// integrationAlertBody is the push notification text for a newly broken connection.
func integrationAlertBody(provider string, status string, reason string) string {
	name := integrationDisplayName(provider)
	switch {
	case reason == models.IntegrationReasonNotAllowlisted:
		return "Your " + name + " account isn't allowed to use this app yet. Ask the admin to add it so your history can sync."
	case reason == models.IntegrationReasonSetupIncomplete:
		return "Your " + name + " connection isn't finished. Open your account page to complete it so your history can sync."
	case status == models.IntegrationStatusAuthFailed:
		return "Your " + name + " connection has stopped working. Reconnect it on your account page to keep your history in sync."
	default:
		return name + " hasn't responded for a day. Your history will catch up once it's reachable again."
	}
}

// recordIntegrationFailure notes a failed provider call. Errors that aren't health
// signals are ignored.
func recordIntegrationFailure(userID uuid.UUID, provider string, failure error) {
	recordIntegrationFailureSince(userID, provider, failure, time.Now())
}

// recordIntegrationFailureSince notes a failed provider call whose run of failures began
// at since (now, unless a recovery backfill failed part-way and must keep the start of
// the gap it was filling). The user is notified once per breakage: when the connection
// first turns bad, and again only if what they have to do changes (unavailable → auth
// failed, or a different reason).
func recordIntegrationFailureSince(userID uuid.UUID, provider string, failure error, since time.Time) {
	failureStatus := integrationErrorStatus(failure)
	if failureStatus == "" {
		return
	}

	existing, err := database.GetIntegrationStatus(userID, provider)
	if err != nil {
		logger.Log.Warn("Failed to get integration status. Error: " + err.Error())
		return
	}

	status := models.IntegrationStatus{UserID: userID, Provider: provider, Status: models.IntegrationStatusOK}
	if existing != nil {
		status = *existing
	}
	previousStatus, previousReason := status.Status, status.Reason
	previousFailingSince := status.FailingSince

	if status.FailingSince == nil || since.Before(*status.FailingSince) {
		status.FailingSince = &since
	}

	now := time.Now()
	switch failureStatus {
	case models.IntegrationStatusAuthFailed:
		status.Status = models.IntegrationStatusAuthFailed
		status.Reason = integrationErrorReason(failure)
	case models.IntegrationStatusUnavailable:
		// A rejected credential stays the headline even if the server also goes quiet:
		// reconnecting is what the user has to do either way.
		if status.Status != models.IntegrationStatusAuthFailed && now.Sub(*status.FailingSince) >= integrationUnavailableGrace {
			status.Status = models.IntegrationStatusUnavailable
			status.Reason = integrationErrorReason(failure)
		}
	}

	changed := status.Status != previousStatus || status.Reason != previousReason
	notify := status.Status != models.IntegrationStatusOK && (status.NotifiedAt == nil || changed)
	if notify {
		status.NotifiedAt = &now
	}

	// Most failures repeat a known state; don't rewrite the row every hour for them.
	if existing != nil && !notify && !changed && previousFailingSince != nil && previousFailingSince.Equal(*status.FailingSince) {
		return
	}

	if changed {
		logger.Log.Warn("Integration '" + provider + "' for user " + userID.String() + " is now '" + status.Status + "'. Error: " + failure.Error())
	}

	if _, err := database.SaveIntegrationStatus(status); err != nil {
		logger.Log.Warn("Failed to save integration status. Error: " + err.Error())
		return
	}

	if notify {
		if err := PushNotificationsForAccountAlert(userID, integrationAlertBody(provider, status.Status, status.Reason)); err != nil {
			logger.Log.Warn("Failed to notify user about integration status. Error: " + err.Error())
		}
	}
}

// recordIntegrationOutcome records a provider call's result: a failure (when it is a
// health signal) or a success.
func recordIntegrationOutcome(userID uuid.UUID, provider string, failure error) {
	if failure != nil {
		recordIntegrationFailure(userID, provider, failure)
		return
	}
	recordIntegrationSuccess(userID, provider)
}

// recordIntegrationSuccess notes a working provider call. When the connection had been
// failing, the status is cleared and whatever was missed in the meantime is re-pulled.
func recordIntegrationSuccess(userID uuid.UUID, provider string) {
	since, ok := clearIntegrationStatus(userID, provider)
	if !ok {
		return
	}

	logger.Log.Info("Integration '" + provider + "' for user " + userID.String() + " recovered; re-pulling since " + since.UTC().Format(time.RFC3339) + ".")
	startIntegrationRecovery("integration recovery", func() { recoverIntegration(userID, provider, since) })
}

// clearIntegrationStatus removes a (user, provider) health row and returns when its run
// of failures began, or false when there was nothing to clear. Disconnect and reconnect
// call it too: a fresh credential starts with a clean slate.
func clearIntegrationStatus(userID uuid.UUID, provider string) (time.Time, bool) {
	existing, err := database.GetIntegrationStatus(userID, provider)
	if err != nil {
		logger.Log.Warn("Failed to get integration status. Error: " + err.Error())
		return time.Time{}, false
	} else if existing == nil {
		return time.Time{}, false
	}

	if err := database.DeleteIntegrationStatus(userID, provider); err != nil {
		logger.Log.Warn("Failed to clear integration status. Error: " + err.Error())
		return time.Time{}, false
	}

	if existing.FailingSince == nil {
		return time.Time{}, false
	}
	return *existing.FailingSince, true
}

// resumeIntegrationAfterReconnect clears a broken connection's status once the user has
// supplied a fresh credential, and re-pulls the gap in the background.
func resumeIntegrationAfterReconnect(userID uuid.UUID, provider string) {
	since, ok := clearIntegrationStatus(userID, provider)
	if !ok {
		return
	}

	logger.Log.Info("Integration '" + provider + "' for user " + userID.String() + " reconnected; re-pulling since " + since.UTC().Format(time.RFC3339) + ".")
	startIntegrationRecovery("integration recovery", func() { recoverIntegration(userID, provider, since) })
}

// recoverIntegration re-pulls a provider's data from the start of a failure run.
func recoverIntegration(userID uuid.UUID, provider string, since time.Time) {
	if limit := time.Now().Add(-integrationBackfillLimit); since.Before(limit) {
		since = limit
	}

	// The provider syncs run as the user against their own connection: the real row.
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Warn("Integration recovery could not load user. Error: " + err.Error())
		return
	}

	switch provider {
	case models.IntegrationProviderPlex, models.IntegrationProviderSpotify, models.IntegrationProviderAudiobookshelf:
		mediaBackfillSince(user, provider, since)
	case models.IntegrationProviderStrava:
		stravaBackfillSince(user, since)
	case models.IntegrationProviderHevy:
		// Nothing to do: the events sync only advances HevyLastSync after a run that
		// succeeded, so the first good run after a breakage already covers the gap.
	}
}

// IntegrationHealthCheckForAllUsers is the daily cron job: it makes one cheap
// authenticated call per media connection, so a broken one is noticed even when the user
// hasn't logged a workout that would have tripped over it. Strava and Hevy need no check
// of their own — their hourly syncs make the same call anyway.
func IntegrationHealthCheckForAllUsers() {
	for _, provider := range []string{models.MediaProviderPlex, models.MediaProviderSpotify, models.MediaProviderAudiobookshelf} {
		if mediaProviderEnabled(provider) {
			MediaHealthCheckForProvider(provider)
		}
	}

	logger.Log.Info("Integration health check task finished.")
}

// integrationHealthForUser returns the health to show for a connection: "ok" unless the
// connection is known to be broken. A transient failure still inside its grace period
// reads as ok.
func integrationHealthForUser(userID uuid.UUID, provider string) models.IntegrationHealthObject {
	healthy := models.IntegrationHealthObject{Status: models.IntegrationStatusOK}

	existing, err := database.GetIntegrationStatus(userID, provider)
	if err != nil {
		logger.Log.Warn("Failed to get integration status. Error: " + err.Error())
		return healthy
	} else if existing == nil || existing.Status == models.IntegrationStatusOK {
		return healthy
	}

	return models.IntegrationHealthObject{Status: existing.Status, StatusReason: existing.Reason, FailingSince: existing.FailingSince}
}

// userIntegrationHealth is the health of the user's own Strava / Hevy connections, for
// the account page. Only connected, enabled integrations are included.
func userIntegrationHealth(user models.User) map[string]models.IntegrationHealthObject {
	health := map[string]models.IntegrationHealthObject{}
	if files.ConfigFile.StravaEnabled && user.StravaCode != nil && *user.StravaCode != "" {
		health[models.IntegrationProviderStrava] = integrationHealthForUser(user.ID, models.IntegrationProviderStrava)
	}
	if files.ConfigFile.HevyEnabled && user.HevyAPIKey != nil && *user.HevyAPIKey != "" {
		health[models.IntegrationProviderHevy] = integrationHealthForUser(user.ID, models.IntegrationProviderHevy)
	}
	return health
}
