package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpPrincipalKey struct{}

const mcpDefaultLimit = 20

// --- Tool argument and output types (schemas are inferred from these) ---

type mcpEmptyArgs struct{}

type mcpListWeightsArgs struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of entries to return (default 20)"`
}

type mcpListExercisesArgs struct {
	Action      string `json:"action,omitempty" jsonschema:"filter to activities whose exercise type matches this text (case-insensitive substring), e.g. Run or Bicycling"`
	Query       string `json:"query,omitempty" jsonschema:"free-text search over the activity/session/day notes and the exercise type name (case-insensitive substring)"`
	From        string `json:"from,omitempty" jsonschema:"only activities on or after this date; accepts YYYY-MM-DD or an RFC3339 timestamp"`
	To          string `json:"to,omitempty" jsonschema:"only activities on or before this date; accepts YYYY-MM-DD or an RFC3339 timestamp"`
	HasDistance bool   `json:"has_distance,omitempty" jsonschema:"when true, only activities that recorded a distance (runs, rides, etc.)"`
	Sort        string `json:"sort,omitempty" jsonschema:"sort key: date (default), distance, duration, weight or reps"`
	Order       string `json:"order,omitempty" jsonschema:"sort direction: desc (default, newest/largest first) or asc"`
	Limit       int    `json:"limit,omitempty" jsonschema:"maximum number of activities to return (default 20, max 100)"`
	Offset      int    `json:"offset,omitempty" jsonschema:"number of activities to skip from the start, for pagination (default 0)"`
}

type mcpWorkoutArgs struct {
	ActivityID string   `json:"activity_id" jsonschema:"the id of an activity from list_activities"`
	Include    []string `json:"include,omitempty" jsonschema:"processed sensor blocks to attach for a stream-backed (has_streams=true) activity, without pulling the raw time-series. For a full analysis pass request [\"segments\",\"zones\",\"analysis\"]. Blocks: 'segments' (per-km/mile splits with pace, HR, cadence and elevation gain — the single highest-value block for run/ride analysis), 'zones' (heart-rate zone breakdown), 'analysis' (aerobic decoupling, first/second-half splits, pace consistency, walk/stop breaks with count and timing, HR-by-gradient), 'elevation' (gain/loss/min/max and biggest climb), 'route' (GPS path summary), 'profile' (altitude-over-distance). Ignored for activities without streams"`
}

type mcpListSeasonsArgs struct {
	ActiveOnly bool `json:"active_only,omitempty" jsonschema:"when true, return only seasons currently within their start-end window (ongoing)"`
}

type mcpSeasonArgs struct {
	SeasonID string `json:"season_id" jsonschema:"the season id"`
}

type mcpAchievementArgs struct {
	AchievementID string `json:"achievement_id" jsonschema:"the achievement id"`
}

type mcpListDelegationsArgs struct {
	Limit         int    `json:"limit,omitempty" jsonschema:"maximum number of awards to return (default 20)"`
	Offset        int    `json:"offset,omitempty" jsonschema:"number of awards to skip from the start, for pagination (default 0)"`
	AchievementID string `json:"achievement_id,omitempty" jsonschema:"filter to awards of a single achievement"`
}

type mcpDelegationArgs struct {
	DelegationID string `json:"delegation_id" jsonschema:"the achievement delegation (award) id"`
}

type mcpSeasonsOutput struct {
	Seasons []models.MCPSeason `json:"seasons"`
}

type mcpAchievementsOutput struct {
	Achievements []models.MCPAchievement `json:"achievements"`
}

type mcpDelegationsOutput struct {
	Total       int                               `json:"total" jsonschema:"total matching awards before limit/offset are applied"`
	Delegations []models.MCPAchievementDelegation `json:"delegations"`
}

type mcpWorkoutStreamsArgs struct {
	ActivityID  string `json:"activity_id" jsonschema:"the id of an activity from list_activities (must have has_streams=true)"`
	FromSeconds int    `json:"from_seconds,omitempty" jsonschema:"start of the time window, in seconds from workout start (default 0)"`
	ToSeconds   int    `json:"to_seconds,omitempty" jsonschema:"end of the time window, in seconds from workout start (default end of workout)"`
	Resolution  int    `json:"resolution,omitempty" jsonschema:"desired spacing between returned samples in seconds; 1 = full fidelity. Omit to auto-fit the whole window to max_points"`
	MaxPoints   int    `json:"max_points,omitempty" jsonschema:"hard cap on the number of series points returned (default 2000, max 5000)"`
}

type mcpWorkoutSoundtrackArgs struct {
	ActivityID string `json:"activity_id" jsonschema:"the id of an activity from list_activities (must have has_soundtrack=true)"`
}

type mcpWeightsOutput struct {
	Weights []models.MCPWeight `json:"weights"`
}

type mcpLatestWeightOutput struct {
	Weight *models.MCPWeight `json:"weight" jsonschema:"the most recent weight entry, or null if none"`
}

type mcpActivitiesOutput struct {
	Total      int64                       `json:"total" jsonschema:"total matching activities before limit/offset are applied"`
	HasMore    bool                        `json:"has_more" jsonschema:"whether more activities remain beyond this page"`
	Activities []models.MCPActivitySummary `json:"activities"`
}

type mcpActivityOutput struct {
	Activity models.MCPActivity `json:"activity"`
}

// MCPHandler authenticates the caller, then serves the MCP Streamable HTTP
// endpoint. The authenticated principal is passed to the per-request server via
// the request context.
func MCPHandler() gin.HandlerFunc {
	streamHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		principal, _ := r.Context().Value(mcpPrincipalKey{}).(middlewares.Principal)
		return buildMCPServer(principal.UserID)
	}, nil)

	return func(c *gin.Context) {
		if files.ConfigFile.MCPEnabled == nil || !*files.ConfigFile.MCPEnabled {
			// MCP server disabled via config; behave as if the endpoint does not exist.
			c.JSON(http.StatusNotFound, gin.H{"error": "MCP server is disabled."})
			return
		}

		principal, err := middlewares.Authenticate(c.GetHeader("Authorization"))
		if err != nil {
			// 401 with a WWW-Authenticate challenge so MCP clients discover the OAuth flow.
			middlewares.BearerChallenge(c, http.StatusUnauthorized, "invalid_token", "authentication required")
			return
		}
		if !auth.ScopeCanRead(principal.Scope) {
			middlewares.BearerChallenge(c, http.StatusForbidden, "insufficient_scope", "token lacks API read access")
			return
		}

		ctx := context.WithValue(c.Request.Context(), mcpPrincipalKey{}, principal)
		c.Request = c.Request.WithContext(ctx)
		streamHandler.ServeHTTP(c.Writer, c.Request)
	}
}

// buildMCPServer constructs an MCP server whose tools operate on a single user.
func buildMCPServer(userID uuid.UUID) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "treningheten",
		Version: files.ConfigFile.TreninghetenVersion,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "whoami",
		Description: "Get the authenticated user's profile (name, email, admin status, member since).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, models.MCPProfile, error) {
		profile, err := assembleUserProfile(userID)
		if err != nil {
			return nil, models.MCPProfile{}, err
		}
		return nil, profile, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_weights",
		Description: "List the user's recorded body-weight entries, newest first.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListWeightsArgs) (*mcp.CallToolResult, mcpWeightsOutput, error) {
		weights, err := assembleUserWeights(userID, limitOrDefault(args.Limit))
		if err != nil {
			return nil, mcpWeightsOutput{}, err
		}
		return nil, mcpWeightsOutput{Weights: weights}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_latest_weight",
		Description: "Get the user's most recent body-weight entry.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, mcpLatestWeightOutput, error) {
		weights, err := assembleUserWeights(userID, 1)
		if err != nil {
			return nil, mcpLatestWeightOutput{}, err
		}
		out := mcpLatestWeightOutput{}
		if len(weights) > 0 {
			out.Weight = &weights[0]
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_activities",
		Description: "Search the user's logged exercise activities and return a slim, ranked list — the way to find relevant activities without pulling everything. " +
			"Each result is one activity with its id, date, exercise type, source (strava/hevy/manual), note, aggregated metrics (distance, duration, reps, top weight, set count), has_streams and counts_toward_goal. " +
			"Filter by action (exercise type), free-text query (notes + type name), from/to date range and has_distance; sort by date (default), distance, duration, weight or reps in asc/desc order; paginate with limit (default 20, max 100) and offset. The response reports total and has_more. " +
			"For per-set detail, tags, description and soundtrack, drill into one result with get_activity by its id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListExercisesArgs) (*mcp.CallToolResult, mcpActivitiesOutput, error) {
		filter, err := activityFeedFilterFromSearchArgs(args)
		if err != nil {
			return nil, mcpActivitiesOutput{}, err
		}
		activities, total, hasMore, err := assembleActivitySearch(userID, filter)
		if err != nil {
			return nil, mcpActivitiesOutput{}, err
		}
		return nil, mcpActivitiesOutput{Total: total, HasMore: hasMore, Activities: activities}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_statistics",
		Description: "Get a summary of the user's training over three rolling windows (trailing ~1 month, trailing 12 months, and all time, relative to now). Each window reports activity count, total distance (km) and total time (seconds). Counts include ALL exercise types (runs, rides, strength, etc.); distance and time totals only include activities that record them, so they may cover fewer activities than the count. Also returns personal day and ISO-week activity streaks (current + best); these are independent of seasons and goals.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, models.MCPStatistics, error) {
		stats, err := assembleUserStatistics(userID)
		if err != nil {
			return nil, models.MCPStatistics{}, err
		}
		return nil, stats, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_activity",
		Description: "Get the detail of a single activity by its id (action, type, source, tags, note/description, duration and per-set distance/time/reps/weight). Use after list_activities to drill into one activity. " +
			"IMPORTANT for stream-backed activities (has_streams=true): to analyse the workout, pass include — e.g. include:[\"segments\",\"zones\",\"analysis\"] — to attach per-km/mile splits, heart-rate zones and derived metrics (aerobic decoupling, split halves, pace consistency, walk/stop breaks, HR-by-gradient), plus optional 'elevation', 'route' and 'profile'. This is the summary-first way to get splits and drift WITHOUT the raw time-series; reach for get_activity_streams only when you need the raw second-by-second series. Called without include on a stream-backed activity, the response carries an analysis_hint naming what to request.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpWorkoutArgs) (*mcp.CallToolResult, mcpActivityOutput, error) {
		activityID, err := uuid.Parse(args.ActivityID)
		if err != nil {
			return nil, mcpActivityOutput{}, fmt.Errorf("invalid activity_id: %w", err)
		}
		activity, err := assembleSingleActivity(userID, activityID, args.Include)
		if err != nil {
			return nil, mcpActivityOutput{}, err
		}
		return nil, mcpActivityOutput{Activity: activity}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_activity_streams",
		Description: "Get the high-resolution Strava sensor data for one activity. IMPORTANT: streams exist ONLY for GPS/sensor activities imported from Strava (runs, rides, etc.) — strength and manually-logged workouts return has_streams=false. " +
			"The data is recorded second-by-second from the athlete's device, so it must be processed to be meaningful: this tool returns (1) a whole-workout summary header (heart rate, speed/pace, elevation gain, power, cadence, temperature) and (2) a 'series' of time-aligned samples (t_seconds plus the available channels: heartrate_bpm, altitude_m, speed_kmh, cadence_rpm, watts, temperature_c, latlng). " +
			"By default the whole workout is returned, auto-downsampled to max_points (~2000). To inspect a moment at full fidelity, call again with from_seconds/to_seconds narrowing the window and resolution=1. Check series.sampled_every_seconds and series.total_points_in_window to know how much detail you have. " +
				"Stability over time: the summary, segments, elevation and route are derived purely from the immutable recorded stream, so they are identical whenever the activity is viewed. Only the HR zones reflect the athlete's current settings, and their age-based anchor uses the activity's own date (not today), so historical activities stay accurate as the athlete ages.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpWorkoutStreamsArgs) (*mcp.CallToolResult, models.MCPWorkoutStreams, error) {
		activityID, err := uuid.Parse(args.ActivityID)
		if err != nil {
			return nil, models.MCPWorkoutStreams{}, fmt.Errorf("invalid activity_id: %w", err)
		}
		streams, err := assembleWorkoutStreams(userID, activityID, args.FromSeconds, args.ToSeconds, args.Resolution, args.MaxPoints)
		if err != nil {
			return nil, models.MCPWorkoutStreams{}, err
		}
		return nil, streams, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_activity_soundtrack",
		Description: "Get the listening history (music, podcasts, audiobooks) matched to one session by an activity id. " +
			"Fetched on demand like streams, because it can be long. The soundtrack is a SESSION-level fact: any activity id from the same session returns the same tracks, and activities with has_soundtrack=false (or on servers without media integration) return has_soundtrack=false with an explanatory message. " +
			"Each track has its type, title, artist/album, provider (plex/spotify/audiobookshelf) and absolute start/end times — so it can be lined up against get_activity_streams to relate what was playing to the athlete's effort.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpWorkoutSoundtrackArgs) (*mcp.CallToolResult, models.MCPWorkoutSoundtrack, error) {
		activityID, err := uuid.Parse(args.ActivityID)
		if err != nil {
			return nil, models.MCPWorkoutSoundtrack{}, fmt.Errorf("invalid activity_id: %w", err)
		}
		soundtrack, err := assembleWorkoutSoundtrack(userID, activityID)
		if err != nil {
			return nil, models.MCPWorkoutSoundtrack{}, err
		}
		return nil, soundtrack, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_seasons",
		Description: "List the seasons the authenticated user has joined, newest first. Each includes the season's dates and status plus the user's own weekly goal, competing flag and remaining sick-leave. Set active_only to return only seasons currently in progress.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListSeasonsArgs) (*mcp.CallToolResult, mcpSeasonsOutput, error) {
		seasons, err := assembleUserSeasons(userID, args.ActiveOnly)
		if err != nil {
			return nil, mcpSeasonsOutput{}, err
		}
		return nil, mcpSeasonsOutput{Seasons: seasons}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_season",
		Description: "Get one season by id with the authenticated user's personal data (their weekly goal, competing flag and remaining sick-leave). The goal fields are absent and joined=false if the user has not joined that season.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpSeasonArgs) (*mcp.CallToolResult, models.MCPSeason, error) {
		seasonID, err := uuid.Parse(args.SeasonID)
		if err != nil {
			return nil, models.MCPSeason{}, fmt.Errorf("invalid season_id: %w", err)
		}
		season, err := assembleSeason(userID, seasonID)
		if err != nil {
			return nil, models.MCPSeason{}, err
		}
		return nil, season, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_achievements",
		Description: "List the achievement catalog with, for each, whether the authenticated user has earned it, how many times, and when last. Hidden achievements show their description as 'Hidden' until the user earns them.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpEmptyArgs) (*mcp.CallToolResult, mcpAchievementsOutput, error) {
		achievements, err := assembleUserAchievements(userID)
		if err != nil {
			return nil, mcpAchievementsOutput{}, err
		}
		return nil, mcpAchievementsOutput{Achievements: achievements}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_achievement",
		Description: "Get one achievement by id with the authenticated user's earned state (earned, times earned, last earned at).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpAchievementArgs) (*mcp.CallToolResult, models.MCPAchievement, error) {
		achievementID, err := uuid.Parse(args.AchievementID)
		if err != nil {
			return nil, models.MCPAchievement{}, fmt.Errorf("invalid achievement_id: %w", err)
		}
		achievement, err := assembleAchievement(userID, achievementID)
		if err != nil {
			return nil, models.MCPAchievement{}, err
		}
		return nil, achievement, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_achievement_delegations",
		Description: "List the authenticated user's achievement awards (delegations) — each is one time an achievement was granted to them — newest first. This can be long, so it is paginated (limit/offset) and can be filtered to a single achievement_id. The response includes the total match count.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpListDelegationsArgs) (*mcp.CallToolResult, mcpDelegationsOutput, error) {
		var achievementFilter *uuid.UUID
		if args.AchievementID != "" {
			parsed, err := uuid.Parse(args.AchievementID)
			if err != nil {
				return nil, mcpDelegationsOutput{}, fmt.Errorf("invalid achievement_id: %w", err)
			}
			achievementFilter = &parsed
		}

		total, delegations, err := assembleUserDelegations(userID, achievementFilter, limitOrDefault(args.Limit), args.Offset)
		if err != nil {
			return nil, mcpDelegationsOutput{}, err
		}
		return nil, mcpDelegationsOutput{Total: total, Delegations: delegations}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_achievement_delegation",
		Description: "Get one of the authenticated user's achievement awards (delegations) by id, including which achievement it is and when it was given.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args mcpDelegationArgs) (*mcp.CallToolResult, models.MCPAchievementDelegation, error) {
		delegationID, err := uuid.Parse(args.DelegationID)
		if err != nil {
			return nil, models.MCPAchievementDelegation{}, fmt.Errorf("invalid delegation_id: %w", err)
		}
		delegation, found, err := assembleDelegation(userID, delegationID)
		if err != nil {
			return nil, models.MCPAchievementDelegation{}, err
		}
		if !found {
			return nil, models.MCPAchievementDelegation{}, fmt.Errorf("delegation not found or not owned by you")
		}
		return nil, delegation, nil
	})

	return server
}

func limitOrDefault(limit int) int {
	if limit <= 0 {
		return mcpDefaultLimit
	}
	return limit
}

// activityFeedFilterFromSearchArgs validates the list_activities search arguments into an
// ActivityFeedFilter, reusing the same date parsing and sort/order whitelists as the web
// /exercises feed (parseActivityFeedTime, activityFeedSorts). A bad date, sort or order is a
// client error. Limit defaults to 20 and is capped at 100; a negative offset falls back to 0.
func activityFeedFilterFromSearchArgs(args mcpListExercisesArgs) (models.ActivityFeedFilter, error) {
	filter := models.ActivityFeedFilter{
		ActionName:  strings.TrimSpace(args.Action),
		Query:       strings.TrimSpace(args.Query),
		HasDistance: args.HasDistance,
		Sort:        "date",
		Order:       "desc",
		Limit:       limitOrDefault(args.Limit),
		Offset:      args.Offset,
	}

	if value := strings.TrimSpace(args.From); value != "" {
		parsed, err := parseActivityFeedTime(value)
		if err != nil {
			return filter, fmt.Errorf("invalid from date: %w", err)
		}
		filter.Start = &parsed
	}
	if value := strings.TrimSpace(args.To); value != "" {
		parsed, err := parseActivityFeedTime(value)
		if err != nil {
			return filter, fmt.Errorf("invalid to date: %w", err)
		}
		filter.End = &parsed
	}

	if value := strings.ToLower(strings.TrimSpace(args.Sort)); value != "" {
		if !activityFeedSorts[value] {
			return filter, fmt.Errorf("invalid sort: %s", value)
		}
		filter.Sort = value
	}
	if value := strings.ToLower(strings.TrimSpace(args.Order)); value != "" {
		if value != "asc" && value != "desc" {
			return filter, fmt.Errorf("invalid order: %s", value)
		}
		filter.Order = value
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return filter, nil
}
