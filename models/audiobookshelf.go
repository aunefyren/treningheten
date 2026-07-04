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
	ID            string  `json:"id"`
	LibraryItemID string  `json:"libraryItemId"`
	DisplayTitle  string  `json:"displayTitle"`
	DisplayAuthor string  `json:"displayAuthor"`
	MediaType     string  `json:"mediaType"`
	Duration      float64 `json:"duration"`
	TimeListening float64 `json:"timeListening"`
	StartedAt     int64   `json:"startedAt"`
	UpdatedAt     int64   `json:"updatedAt"`
}

// AudiobookshelfConnectRequest is the account-page connect payload: the self-hosted
// server URL and a per-user API token from the ABS account settings.
type AudiobookshelfConnectRequest struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
}
