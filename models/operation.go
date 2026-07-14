package models

import (
	"time"

	"github.com/google/uuid"
)

type Operation struct {
	GormModel
	Enabled      bool       `json:"enabled" gorm:"not null; default: true"`
	ExerciseID   uuid.UUID  `json:"" gorm:"type:varchar(100);"`
	Exercise     Exercise   `json:"exercise" gorm:"not null;"`
	ActionID     *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Action       *Action    `json:"action" gorm:""`
	GearID       *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Gear         *Gear      `json:"gear" gorm:""`
	Type         string     `json:"type" gorm:"not null; default: lifting"`
	WeightUnit   string     `json:"weight_unit" gorm:"not null; default: kg"`
	DistanceUnit string     `json:"distance_unit" gorm:"not null; default: km"`
	Equipment    *string    `json:"equipment" gorm:""`
	Note         *string    `json:"note" gorm:"default: null;"`
	Description  *string    `json:"description" gorm:"type:longtext;default: null;"`
	Tags         TagList    `json:"tags" gorm:"type:longtext;default: null;"`
	Duration     *int64     `json:"duration"`
}

type OperationCreationRequest struct {
	ExerciseID   uuid.UUID  `json:"exercise_id"`
	Action       *uuid.UUID `json:"action"`
	Type         string     `json:"type"`
	WeightUnit   string     `json:"weight_unit"`
	DistanceUnit string     `json:"distance_unit"`
	Equipment    *string    `json:"equipment"`
}

type OperationUpdateRequest struct {
	Action       string `json:"action"`
	Type         string `json:"type"`
	WeightUnit   string `json:"weight_unit"`
	DistanceUnit string `json:"distance_unit"`
	Equipment    string `json:"equipment"`
	// Tags and Description are pointers so an omitted field (normal operation edits)
	// leaves the stored value untouched, while an explicit value replaces it.
	Tags        *[]string `json:"tags"`
	Description *string   `json:"description"`
}

type OperationObject struct {
	GormModel
	Enabled       bool                 `json:"enabled"`
	Exercise      uuid.UUID            `json:"exercise"`
	OperationSets []OperationSetObject `json:"operation_sets"`
	Action        *Action              `json:"action"`
	Type          string               `json:"type"`
	WeightUnit    string               `json:"weight_unit"`
	DistanceUnit  string               `json:"distance_unit"`
	Equipment     *string              `json:"equipment"`
	Gear          *GearObject          `json:"gear"`
	StravaID      *string              `json:"strava_id"`
	Note          *string              `json:"note"`
	Description   *string              `json:"description"`
	Tags          []string             `json:"tags"`
	Duration      *int64               `json:"duration"`
}

type OperationSet struct {
	GormModel
	Enabled               bool               `json:"enabled" gorm:"not null; default: true;"`
	OperationID           uuid.UUID          `json:"" gorm:"type:varchar(100);"`
	Operation             Operation          `json:"operation" gorm:"not null"`
	Repetitions           *float64           `json:"repetitions" gorm:"default: null"`
	Weight                *float64           `json:"weight" gorm:"default: null"`
	Distance              *float64           `json:"distance" gorm:"default: null"`
	Time                  *int64             `json:"time" gorm:"default: null"`
	MovingTime            *int64             `json:"moving_time" gorm:"default: null"`
	StravaID              *string            `json:"strava_id" gorm:"default: null;"`
	StravaStreams         *StravaStreamsJSON `json:"strava_streams" gorm:"type:longtext;default: null;"`
	StravaDataRetrievedAt *time.Time         `json:"strava_data_retrieved_at" gorm:"default: null;"`
	// StravaDetailRetrievedAt guards the detailed-activity fetch (description), which
	// is not in the list-sync payload, so the hourly sync only re-fetches when stale.
	StravaDetailRetrievedAt *time.Time `json:"strava_detail_retrieved_at" gorm:"default: null;"`
}

type OperationSetCreationRequest struct {
	OperationID uuid.UUID `json:"operation_id"`
	Repetitions *float64  `json:"repetitions"`
	Weight      *float64  `json:"weight"`
	Distance    *float64  `json:"distance"`
	Time        *int64    `json:"time"`
}

type OperationSetUpdateRequest struct {
	Repetitions *float64 `json:"repetitions"`
	Weight      *float64 `json:"weight"`
	Distance    *float64 `json:"distance"`
	MovingTime  *int64   `json:"moving_time"`
}

type OperationSetObject struct {
	GormModel
	Enabled               bool               `json:"enabled"`
	Operation             uuid.UUID          `json:"operation"`
	Repetitions           *float64           `json:"repetitions"`
	Weight                *float64           `json:"weight"`
	Distance              *float64           `json:"distance"`
	Time                  *int64             `json:"time"`
	MovingTime            *int64             `json:"moving_time"`
	StravaID              *string            `json:"strava_id"`
	StravaStreams         *StravaStreamsJSON `json:"strava_streams"`
	StravaDataRetrievedAt *time.Time         `json:"strava_data_retrieved_at"`
}

type Action struct {
	GormModel
	Enabled        bool    `json:"enabled" gorm:"not null; default: true;"`
	Name           string  `json:"name"`
	NorwegianName  string  `json:"norwegian_name"`
	Description    string  `json:"description"`
	Type           string  `json:"type"`
	BodyPart       string  `json:"body_part"`
	StravaName     string  `json:"strava_name"`
	HevyTemplateID *string `json:"hevy_template_id" gorm:"default: null"`
	PastTenseVerb  *string `json:"past_tense_verb"`
	HasLogo        bool    `json:"has_logo" gorm:"not null; default: false;"`
}

type ActionCreationRequest struct {
	Name          string `json:"name"`
	NorwegianName string `json:"norwegian_name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	BodyPart      string `json:"body_part"`
}

type ActionStatistics struct {
	Action     Action                 `json:"action"`
	Statistics StatisticsCompilation  `json:"statistics"`
	Operations []OperationObject      `json:"operations"`
	Media      *ActionMediaStatistics `json:"media"`
}

// ActionMediaStatistics is the aggregated soundtrack overlay for an action's
// statistics period. It is nil unless media is enabled and at least one matched
// session had playback rows, so the frontend renders the block only when relevant.
// The soundtrack is a session-level fact (keyed to the Exercise, not the operation),
// so these figures aggregate over the distinct sessions the matched operations belong
// to — a session's tracks are counted once regardless of how many of its operations
// matched. Spoken audio (podcasts/audiobooks) is folded into SpokenTime rather than
// the song-centric track/artist tallies. The time fields hold a plain seconds count
// (int64), per the repo duration convention.
type ActionMediaStatistics struct {
	Songs         int             `json:"songs"`
	UniqueArtists int             `json:"unique_artists"`
	ListeningTime int64           `json:"listening_time"`
	SpokenTime    int64           `json:"spoken_time"`
	TopTrack      *MediaCountItem `json:"top_track"`
	TopArtist     *MediaCountItem `json:"top_artist"`
	// Spoken-audio detail, split so podcasts and audiobooks each get their own figures
	// rather than being folded into the single SpokenTime lump. PodcastEpisodes counts
	// distinct episodes and Audiobooks counts distinct books; TopPodcast is grouped by
	// show (its Count is that show's episode count) and TopAudiobook by book (Artist is
	// the author). Any of these may be nil/zero when no such media was matched.
	PodcastTime     int64           `json:"podcast_time"`
	AudiobookTime   int64           `json:"audiobook_time"`
	PodcastEpisodes int             `json:"podcast_episodes"`
	Audiobooks      int             `json:"audiobooks"`
	TopPodcast      *MediaCountItem `json:"top_podcast"`
	TopAudiobook    *MediaCountItem `json:"top_audiobook"`
}

// MediaCountItem is a most-played tally (a track, artist, podcast, or audiobook) with
// its play count. Artwork is the cover image when the provider supplied one (songs from
// Spotify/Plex usually do; ABS spoken items usually do not), and is empty otherwise.
type MediaCountItem struct {
	Title   string `json:"title"`
	Artist  string `json:"artist"`
	Count   int    `json:"count"`
	Artwork string `json:"artwork,omitempty"`
}

type StatisticsCompilation struct {
	Sums     StatisticsSumCompilation     `json:"sums"`
	Averages StatisticsAverageCompilation `json:"averages"`
	Tops     StatisticsTopCompilation     `json:"tops"`
}

type StatisticsSumCompilation struct {
	Distance   float64 `json:"distance"`
	Time       int64   `json:"time"`
	Repetition float64 `json:"repetition"`
	Weight     float64 `json:"weight"`
	Operations int64   `json:"operations"`
}

type StatisticsAverageCompilation struct {
	Distance   float64 `json:"distance"`
	Time       int64   `json:"time"`
	Repetition float64 `json:"repetition"`
	Weight     float64 `json:"weight"`
}

type StatisticsTopCompilation struct {
	Distance   *OperationObject `json:"distance"`
	Time       *OperationObject `json:"time"`
	Repetition *OperationObject `json:"repetition"`
	Weight     *OperationObject `json:"weight"`
}
