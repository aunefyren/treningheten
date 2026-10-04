package models

// Audiobookshelf (ABS) DTOs for identifying the connected user and pulling listening
// history. Only the fields Treningheten uses are modelled. ABS is self-hosted; a
// connection is a server URL + a per-user API token (from the ABS account settings),
// and history comes from the user-scoped /api/me/listening-sessions endpoint — so no
// privacy fail-closed scoping is needed (the token is inherently one user's). See
// docs/media.md.

// AudiobookshelfUser is the GET /api/me reply, used to validate the token at connect
// time and record the ABS user id.
type AudiobookshelfUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// AudiobookshelfListeningSessionsResponse is the GET /api/me/listening-sessions reply
// (paginated, most-recent first).
type AudiobookshelfListeningSessionsResponse struct {
	Total        int                           `json:"total"`
	NumPages     int                           `json:"numPages"`
	Page         int                           `json:"page"`
	ItemsPerPage int                           `json:"itemsPerPage"`
	Sessions     []AudiobookshelfListenSession `json:"sessions"`
}

// AudiobookshelfListenSession is one continuous-listen session — coarser than a
// per-track scrobble, but ideal for an audiobook/podcast workout (one item = one
// rail node). StartedAt/UpdatedAt are epoch milliseconds; TimeListening/Duration are
// seconds. MediaType is "book" or "podcast" — ABS is the first provider that natively
// distinguishes the two.
type AudiobookshelfListenSession struct {
	// ID identifies this listening session — ABS opens a new one on a device switch,
	// app restart, or a resume after an unclean close, so several can cover one listen.
	ID string `json:"id"`
	// LibraryItemID is the *show* for a podcast, not the episode; EpisodeID is the
	// per-episode identity and is empty for books. Keying on LibraryItemID alone would
	// treat every episode of a series as the same item.
	LibraryItemID string  `json:"libraryItemId"`
	EpisodeID     string  `json:"episodeId"`
	DisplayTitle  string  `json:"displayTitle"`
	DisplayAuthor string  `json:"displayAuthor"`
	MediaType     string  `json:"mediaType"`
	Duration      float64 `json:"duration"`
	// TimeListening is null from some clients (the ABS iOS app sends it on only some
	// sessions), which decodes as 0 — fall back to CurrentTime − StartTime.
	TimeListening float64 `json:"timeListening"`
	// StartTime/CurrentTime are the playback positions within the item (seconds) where
	// the session began and where it last was. A session that resumes a listen begins
	// where the previous one stopped, on every client and across device switches.
	StartTime   float64 `json:"startTime"`
	CurrentTime float64 `json:"currentTime"`
	StartedAt   int64   `json:"startedAt"`
	// UpdatedAt is when the session was last touched, which is not always when listening
	// stopped: a session can hold a pause, and some clients close a paused session only
	// when playback resumes — hours later.
	UpdatedAt int64 `json:"updatedAt"`
}

// AudiobookshelfConnectRequest is the account-page connect payload: the self-hosted
// server URL and a per-user API token from the ABS account settings.
type AudiobookshelfConnectRequest struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
}
