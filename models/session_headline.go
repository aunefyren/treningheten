package models

// SessionHeadline is the server's one description of what a workout session was, so the
// session header, the /exercises list and the front-page feed name a session the same way
// instead of each re-deriving it from the raw activity list (see controllers/session_headline.go
// for the rules).
//
//   - Title: the source's own title when the session came from Strava or Hevy ("Push day",
//     "Afternoon Run"), otherwise generated from the activities ("Strength training",
//     "Run + strength training").
//   - PastTenseVerb: the feed phrase ("did some strength training", "played some padel").
//   - Category: the dominant action category, or SessionCategoryMixed.
//   - Logos: action names whose logo the feed shows, de-duplicated; a strength session shows
//     the Weight Training logo once instead of one per lift.
type SessionHeadline struct {
	Title         string   `json:"title"`
	PastTenseVerb string   `json:"past_tense_verb"`
	Category      string   `json:"category"`
	Logos         []string `json:"logos"`
}

// SessionCategoryMixed is SessionHeadline.Category for a session with no dominant category.
const SessionCategoryMixed = "mixed"
