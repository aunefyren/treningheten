package models

import "time"

type StravaAuthorizeRequestReply struct {
	TokenType    string `json:"token_type"`
	ExpiresAt    int    `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	Athlete      struct {
		ID            int         `json:"id"`
		Username      interface{} `json:"username"`
		ResourceState int         `json:"resource_state"`
		Firstname     string      `json:"firstname"`
		Lastname      string      `json:"lastname"`
		Bio           string      `json:"bio"`
		City          string      `json:"city"`
		State         string      `json:"state"`
		Country       string      `json:"country"`
		Sex           string      `json:"sex"`
		Premium       bool        `json:"premium"`
		Summit        bool        `json:"summit"`
		CreatedAt     time.Time   `json:"created_at"`
		UpdatedAt     time.Time   `json:"updated_at"`
		BadgeTypeID   int         `json:"badge_type_id"`
		Weight        interface{} `json:"weight"`
		ProfileMedium string      `json:"profile_medium"`
		Profile       string      `json:"profile"`
		Friend        interface{} `json:"friend"`
		Follower      interface{} `json:"follower"`
	} `json:"athlete"`
}

type StravaReauthorizationRequestReply struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int    `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type StravaGetActivitiesRequestReply struct {
	ResourceState int `json:"resource_state"`
	Athlete       struct {
		ID            int `json:"id"`
		ResourceState int `json:"resource_state"`
	} `json:"athlete"`
	Name               string      `json:"name"`
	Distance           float64     `json:"distance"`
	MovingTime         int         `json:"moving_time"`
	ElapsedTime        int         `json:"elapsed_time"`
	TotalElevationGain float64     `json:"total_elevation_gain"`
	Type               string      `json:"type"`
	SportType          string      `json:"sport_type"`
	ID                 int64       `json:"id"`
	StartDate          time.Time   `json:"start_date"`
	StartDateLocal     time.Time   `json:"start_date_local"`
	Timezone           string      `json:"timezone"`
	UtcOffset          float64     `json:"utc_offset"`
	LocationCity       interface{} `json:"location_city"`
	LocationState      interface{} `json:"location_state"`
	LocationCountry    string      `json:"location_country"`
	AchievementCount   int         `json:"achievement_count"`
	KudosCount         int         `json:"kudos_count"`
	CommentCount       int         `json:"comment_count"`
	AthleteCount       int         `json:"athlete_count"`
	PhotoCount         int         `json:"photo_count"`
	Map                struct {
		ID              string `json:"id"`
		SummaryPolyline string `json:"summary_polyline"`
		ResourceState   int    `json:"resource_state"`
	} `json:"map"`
	Trainer                    bool        `json:"trainer"`
	Commute                    bool        `json:"commute"`
	Manual                     bool        `json:"manual"`
	Private                    bool        `json:"private"`
	Visibility                 string      `json:"visibility"`
	Flagged                    bool        `json:"flagged"`
	GearID                     interface{} `json:"gear_id"`
	StartLatlng                []float64   `json:"start_latlng"`
	EndLatlng                  []float64   `json:"end_latlng"`
	AverageSpeed               float64     `json:"average_speed"`
	MaxSpeed                   float64     `json:"max_speed"`
	HasHeartrate               bool        `json:"has_heartrate"`
	HeartrateOptOut            bool        `json:"heartrate_opt_out"`
	DisplayHideHeartrateOption bool        `json:"display_hide_heartrate_option"`
	ElevHigh                   float64     `json:"elev_high"`
	ElevLow                    float64     `json:"elev_low"`
	UploadID                   int64       `json:"upload_id"`
	UploadIDStr                string      `json:"upload_id_str"`
	ExternalID                 string      `json:"external_id"`
	FromAcceptedTag            bool        `json:"from_accepted_tag"`
	PrCount                    int         `json:"pr_count"`
	TotalPhotoCount            int         `json:"total_photo_count"`
	HasKudoed                  bool        `json:"has_kudoed"`
}
