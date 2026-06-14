package models

// AdminStatsReply holds aggregate, non-personal statistics shown on the admin
// panel. "Active user" is defined as an enabled user (`enabled = 1`).
type AdminStatsReply struct {
	TotalUsers int64 `json:"total_users"`

	UsersInSeasonNow    int64   `json:"users_in_season_now"`
	UsersInSeasonNowPct float64 `json:"users_in_season_now_pct"`

	UsersWithNotifications    int64   `json:"users_with_notifications"`
	UsersWithNotificationsPct float64 `json:"users_with_notifications_pct"`

	UsersWithStrava    int64   `json:"users_with_strava"`
	UsersWithStravaPct float64 `json:"users_with_strava_pct"`

	AchievementsTotal           int64   `json:"achievements_total"`
	AvgAchievementCompletionPct float64 `json:"avg_achievement_completion_pct"`
}

// AdminStatsCounts holds the raw counts gathered from the database before
// percentages are computed.
type AdminStatsCounts struct {
	TotalUsers             int64
	UsersInSeasonNow       int64
	UsersWithNotifications int64
	UsersWithStrava        int64
	AchievementsTotal      int64
	AchievementDelegations int64
}
