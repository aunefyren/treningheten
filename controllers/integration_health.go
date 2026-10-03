package controllers

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/database"
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
type integrationError struct {
	kind    error
	message string
}

func (e *integrationError) Error() string { return e.message }
func (e *integrationError) Unwrap() error { return e.kind }

// integrationAuthError marks a failure as the provider rejecting the stored credential.
func integrationAuthError(message string) error {
	return &integrationError{kind: ErrIntegrationAuth, message: message}
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

// integrationDisplayName is the provider name as the user knows it.
func integrationDisplayName(provider string) string {
	switch provider {
	case models.IntegrationProviderPlex:
		return "Plex"
	default:
		return provider
	}
}

// integrationAlertBody is the push notification text for a newly broken connection.
func integrationAlertBody(provider string, status string) string {
	name := integrationDisplayName(provider)
	if status == models.IntegrationStatusAuthFailed {
		return "Your " + name + " connection has stopped working. Reconnect it on your account page to keep your history in sync."
	}
	return "Your " + name + " server hasn't responded for a day. Your history will catch up once it's reachable again."
}

// recordIntegrationFailure notes a failed provider call. Errors that aren't health
// signals are ignored.
func recordIntegrationFailure(userID uuid.UUID, provider string, failure error) {
	recordIntegrationFailureSince(userID, provider, failure, time.Now())
}

// recordIntegrationFailureSince notes a failed provider call whose run of failures began
// at since (now, unless a recovery backfill failed part-way and must keep the start of
// the gap it was filling). The user is notified once per breakage: when the connection
// first turns bad, and again only if it gets worse (unavailable → auth failed).
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
	previousStatus := status.Status
	previousFailingSince := status.FailingSince

	if status.FailingSince == nil || since.Before(*status.FailingSince) {
		status.FailingSince = &since
	}

	now := time.Now()
	switch failureStatus {
	case models.IntegrationStatusAuthFailed:
		status.Status = models.IntegrationStatusAuthFailed
	case models.IntegrationStatusUnavailable:
		// A rejected credential stays the headline even if the server also goes quiet:
		// reconnecting is what the user has to do either way.
		if status.Status != models.IntegrationStatusAuthFailed && now.Sub(*status.FailingSince) >= integrationUnavailableGrace {
			status.Status = models.IntegrationStatusUnavailable
		}
	}

	notify := status.Status != models.IntegrationStatusOK && (status.NotifiedAt == nil || status.Status != previousStatus)
	if notify {
		status.NotifiedAt = &now
	}

	// Most failures repeat a known state; don't rewrite the row every hour for them.
	if existing != nil && !notify && status.Status == previousStatus && previousFailingSince != nil && previousFailingSince.Equal(*status.FailingSince) {
		return
	}

	if status.Status != previousStatus {
		logger.Log.Warn("Integration '" + provider + "' for user " + userID.String() + " is now '" + status.Status + "'. Error: " + failure.Error())
	}

	if _, err := database.SaveIntegrationStatus(status); err != nil {
		logger.Log.Warn("Failed to save integration status. Error: " + err.Error())
		return
	}

	if notify {
		if err := PushNotificationsForAccountAlert(userID, integrationAlertBody(provider, status.Status)); err != nil {
			logger.Log.Warn("Failed to notify user about integration status. Error: " + err.Error())
		}
	}
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
	case models.IntegrationProviderPlex:
		plexBackfillSince(user, since)
	}
}

// IntegrationHealthCheckForAllUsers is the daily cron job: it makes one cheap
// authenticated call per connection, so a broken one is noticed even when the user
// hasn't logged a workout that would have tripped over it.
func IntegrationHealthCheckForAllUsers() {
	if plexEnabled() {
		PlexHealthCheckForAllUsers()
	}

	logger.Log.Info("Integration health check task finished.")
}

// integrationStatusForUser returns the status to show for a connection: "ok" unless
// the connection is known to be broken, with the time it started failing. A transient
// failure still inside its grace period reads as ok.
func integrationStatusForUser(userID uuid.UUID, provider string) (string, *time.Time) {
	existing, err := database.GetIntegrationStatus(userID, provider)
	if err != nil {
		logger.Log.Warn("Failed to get integration status. Error: " + err.Error())
		return models.IntegrationStatusOK, nil
	} else if existing == nil || existing.Status == models.IntegrationStatusOK {
		return models.IntegrationStatusOK, nil
	}
	return existing.Status, existing.FailingSince
}
