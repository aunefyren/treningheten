package controllers

import (
	"math"
	"net/http"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// fiveZone* define the classic five-zone model, shared by its %-of-max and heart-rate-reserve
// variants.
var (
	fiveZoneBounds = []float64{0.60, 0.70, 0.80, 0.90}
	fiveZoneCodes  = []string{"Z1", "Z2", "Z3", "Z4", "Z5"}
	fiveZoneNames  = []string{"Recovery", "Endurance", "Tempo", "Threshold", "Anaerobic"}
)

// hrZoneSystems is every selectable heart-rate zone model, in the order /account lists
// them. The first entry is the default. A new model is a new entry here plus a key
// constant in models; nothing else needs to know about it.
var hrZoneSystems = []models.HRZoneSystem{
	{
		Key:         models.HRZoneSystemPercentMax,
		Label:       "Five zones (% of max)",
		Description: "The common five-zone model, with edges at 60, 70, 80 and 90% of your maximum heart rate.",
		Bounds:      fiveZoneBounds,
		Codes:       fiveZoneCodes,
		Names:       fiveZoneNames,
	},
	{
		Key:         models.HRZoneSystemReserve,
		Label:       "Heart-rate reserve (Karvonen)",
		Description: "The same five zones, placed on the range between your resting and maximum heart rate. Needs a resting heart rate.",
		UsesResting: true,
		Bounds:      fiveZoneBounds,
		Codes:       fiveZoneCodes,
		Names:       fiveZoneNames,
	},
	{
		// Olympiatoppen's I-scale (olt-skala.nif.no), I-1 to I-5 by % of max HR. I-6–I-8
		// are anaerobic/sprint/strength work that heart rate can't classify, so they are
		// left out. I-1 nominally starts at 55%; time below that is counted as I-1.
		Key:         models.HRZoneSystemOlympiatoppen,
		Label:       "Olympiatoppen I-scale",
		Description: "Olympiatoppen's intensity scale, I-1 to I-5, with edges at 72, 82, 87 and 92% of your maximum heart rate.",
		Bounds:      []float64{0.72, 0.82, 0.87, 0.92},
		Codes:       []string{"I-1", "I-2", "I-3", "I-4", "I-5"},
		Names:       []string{"Easy", "Moderate", "Threshold", "Hard", "Very hard"},
	},
}

// hrZoneSystemByKey looks a zone system up by its stored key.
func hrZoneSystemByKey(key string) (models.HRZoneSystem, bool) {
	for _, system := range hrZoneSystems {
		if system.Key == key {
			return system, true
		}
	}
	return models.HRZoneSystem{}, false
}

// defaultHRZoneSystem is the plain %-of-max model, used when nothing else applies.
func defaultHRZoneSystem() models.HRZoneSystem {
	return hrZoneSystems[0]
}

// hrAnchor is everything the zone model needs from the athlete: the maximum HR (0 means
// "use the activity's own peak"), the resting HR (only used by reserve systems), where the
// max came from ("sport_max"/"max"/"observed_max"/"age", or "observed" once resolved to the
// peak) and the zone system. It is the single seam for per-activity anchors: the per-sport
// max is resolved into MaxBpm here, and the zone maths never needs to know.
type hrAnchor struct {
	MaxBpm  int
	RestBpm int
	Basis   string
	System  models.HRZoneSystem
}

// resolveUserHR turns a user's configured heart-rate settings into an hrAnchor for one
// activity. sportMax is the user's explicit max for the activity's sport (0 = none, see
// sportMaxHeartrate). Max precedence: explicit settings win — the sport's own max, then the
// global one — then the all-time max observed across their activities (real data beats a
// formula), then the age-based estimate. A zero max means "use this activity's observed
// peak" — resolved inside computeHRZones.
func resolveUserHR(user models.User, sportMax int, now time.Time) hrAnchor {
	anchor := hrAnchor{System: effectiveHRZoneSystem(user)}
	if sportMax > 0 {
		anchor.MaxBpm, anchor.Basis = sportMax, "sport_max"
	} else if user.MaxHeartrate != nil && *user.MaxHeartrate > 0 {
		anchor.MaxBpm, anchor.Basis = *user.MaxHeartrate, "max"
	} else if user.ObservedMaxHeartrate != nil && *user.ObservedMaxHeartrate > 0 {
		anchor.MaxBpm, anchor.Basis = *user.ObservedMaxHeartrate, "observed_max"
	} else if age := hrMaxFromBirthDate(user.BirthDate, now); age > 0 {
		anchor.MaxBpm, anchor.Basis = age, "age"
	}
	if user.RestingHeartrate != nil && *user.RestingHeartrate > 0 {
		anchor.RestBpm = *user.RestingHeartrate
	}
	return anchor
}

// sportMaxHeartrate picks the user's explicit max for an activity's sport out of the map
// built by loadSportMaxHeartrates, or 0 when the activity has no action or the sport has no
// max set.
func sportMaxHeartrate(sportMax map[uuid.UUID]int, actionID *uuid.UUID) int {
	if actionID == nil {
		return 0
	}
	return sportMax[*actionID]
}

// loadSportMaxHeartrates returns the user's per-sport max heart rates keyed by action.
func loadSportMaxHeartrates(userID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := database.GetSportMaxHeartratesForUserID(userID)
	if err != nil {
		return nil, err
	}
	sportMax := map[uuid.UUID]int{}
	for _, row := range rows {
		sportMax[row.ActionID] = row.MaxHeartrate
	}
	return sportMax, nil
}

// effectiveHRZoneSystem is the user's chosen zone system. Users who never chose one (or
// hold a key that no longer exists) keep the behaviour from before systems were
// selectable: reserve zones when a resting HR is set, plain % of max otherwise.
func effectiveHRZoneSystem(user models.User) models.HRZoneSystem {
	if user.HRZoneSystem != nil {
		if system, ok := hrZoneSystemByKey(*user.HRZoneSystem); ok {
			return system
		}
	}
	if user.RestingHeartrate != nil && *user.RestingHeartrate > 0 {
		system, _ := hrZoneSystemByKey(models.HRZoneSystemReserve)
		return system
	}
	return defaultHRZoneSystem()
}

// computeHRZones buckets heart-rate time into the anchor's zone system. Boundaries are
// fractions of MaxBpm, or of heart-rate reserve (Karvonen: rest + f·(max−rest)) for a
// system that uses resting HR. When MaxBpm<=0 nothing is configured, so the zones anchor
// to the activity's own peak and the basis becomes "observed". A reserve system without a
// usable resting HR (unset, or not below the max) falls back to the default %-of-max
// system. It returns the zones and the anchor actually applied, or nil when there is no
// heart-rate data.
func computeHRZones(streams *models.StravaActivityStreams, times []int, anchor hrAnchor) ([]models.StreamHRZone, hrAnchor) {
	if streams.Heartrate == nil || len(streams.Heartrate.Data) == 0 {
		return nil, hrAnchor{}
	}
	hr := streams.Heartrate.Data

	if anchor.MaxBpm <= 0 {
		anchor.Basis = "observed"
		for _, v := range hr {
			if v > anchor.MaxBpm {
				anchor.MaxBpm = v
			}
		}
	}
	if anchor.MaxBpm <= 0 {
		return nil, hrAnchor{}
	}

	if len(anchor.System.Bounds) == 0 {
		anchor.System = defaultHRZoneSystem()
	}
	if anchor.System.UsesResting && (anchor.RestBpm <= 0 || anchor.RestBpm >= anchor.MaxBpm) {
		anchor.System = defaultHRZoneSystem()
	}
	if !anchor.System.UsesResting {
		anchor.RestBpm = 0
	}

	bounds := hrZoneBpmBounds(anchor)
	seconds := make([]int64, len(bounds)+1)
	var total int64
	for i, v := range hr {
		if v <= 0 {
			continue
		}
		dt := int64(1)
		if i > 0 && i < len(times) {
			d := int64(times[i] - times[i-1])
			if d > 0 {
				dt = d
			} else {
				dt = 0
			}
		}
		seconds[hrZoneIndex(v, bounds)] += dt
		total += dt
	}
	if total == 0 {
		return nil, hrAnchor{}
	}

	zones := make([]models.StreamHRZone, len(seconds))
	for i := range zones {
		minBpm := anchor.RestBpm // 0 unless on heart-rate reserve
		if i > 0 {
			minBpm = bounds[i-1]
		}
		maxBpm := 0 // open-ended top zone
		if i < len(bounds) {
			maxBpm = bounds[i]
		}
		zones[i] = models.StreamHRZone{
			Zone:    i + 1,
			Code:    anchor.System.Codes[i],
			Name:    anchor.System.Names[i],
			MinBpm:  minBpm,
			MaxBpm:  maxBpm,
			Seconds: seconds[i],
			Percent: round1(float64(seconds[i]) / float64(total) * 100),
		}
	}
	return zones, anchor
}

// hrZoneBpmBounds converts the system's fractional zone edges to bpm, rounded once so the
// displayed bounds and the bucketing agree.
func hrZoneBpmBounds(anchor hrAnchor) []int {
	bounds := make([]int, len(anchor.System.Bounds))
	for i, f := range anchor.System.Bounds {
		if anchor.System.UsesResting {
			bounds[i] = int(math.Round(float64(anchor.RestBpm) + f*float64(anchor.MaxBpm-anchor.RestBpm)))
		} else {
			bounds[i] = int(math.Round(f * float64(anchor.MaxBpm)))
		}
	}
	return bounds
}

// hrZoneIndex maps a heart rate to a 0-based zone using the rounded boundaries.
func hrZoneIndex(v int, bounds []int) int {
	for i, b := range bounds {
		if v < b {
			return i
		}
	}
	return len(bounds) // top (open-ended) zone
}

// hrMaxFromBirthDate returns an age-based maximum heart rate (220 - age) or 0 when the
// birth date is unknown, in which case zones fall back to the observed peak.
func hrMaxFromBirthDate(birthDate *time.Time, now time.Time) int {
	if birthDate == nil {
		return 0
	}
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	if age <= 0 || age > 120 {
		return 0
	}
	return 220 - age
}

// APIGetHRZoneSystems lists the selectable heart-rate zone systems, so /account can offer
// them and preview each zone's bpm range without duplicating the definitions.
func APIGetHRZoneSystems(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"hr_zone_systems": hrZoneSystems, "message": "Heart-rate zone systems retrieved."})
}

// validateSportMaxHeartrates checks the per-sport maxes of a training profile: each must name
// one of the activity-type sports (sports, from database.GetActivityTypeActions) at most once,
// sit in the same range as the global max, and stay above the resting HR so reserve zones
// keep a usable range. Returns a user-facing message, or "" when valid.
func validateSportMaxHeartrates(entries []models.SportMaxHeartrate, restingHeartrate *int, sports []models.Action) string {
	known := map[uuid.UUID]bool{}
	for _, sport := range sports {
		known[sport.ID] = true
	}
	seen := map[uuid.UUID]bool{}
	for _, entry := range entries {
		if !known[entry.ActionID] {
			return "Unknown sport for a sport-specific maximum heart rate."
		}
		if seen[entry.ActionID] {
			return "Each sport can only have one maximum heart rate."
		}
		seen[entry.ActionID] = true
		if entry.MaxHeartrate < 100 || entry.MaxHeartrate > 240 {
			return "Sport-specific maximum heart rates must be between 100 and 240 bpm."
		}
		if restingHeartrate != nil && *restingHeartrate >= entry.MaxHeartrate {
			return "Your resting heart rate must be below every sport-specific maximum."
		}
	}
	return ""
}

// mergeSportHeartrates builds the /account view of per-sport heart rate: one entry per
// activity-type sport, carrying the user's explicit max (if set) and the highest heart rate
// seen in that sport (if any).
func mergeSportHeartrates(sports []models.Action, rows []models.UserSportMaxHeartrate, observed map[uuid.UUID]int) []models.SportHeartrateObject {
	stored := map[uuid.UUID]int{}
	for _, row := range rows {
		stored[row.ActionID] = row.MaxHeartrate
	}

	objects := []models.SportHeartrateObject{}
	for _, sport := range sports {
		object := models.SportHeartrateObject{ActionID: sport.ID, ActionName: sport.Name}
		if value, ok := stored[sport.ID]; ok {
			object.MaxHeartrate = &value
		}
		if value, ok := observed[sport.ID]; ok {
			object.ObservedMaxHeartrate = &value
		}
		objects = append(objects, object)
	}
	return objects
}

// APIGetSportHeartrates lists the authenticated user's per-sport heart-rate settings for
// /account: every activity-type sport with its explicit max and its observed max.
func APIGetSportHeartrates(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user from header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user from header."})
		context.Abort()
		return
	}

	sports, err := database.GetActivityTypeActions()
	if err != nil {
		logger.Log.Error("Failed to get activity types. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sport heart rates."})
		context.Abort()
		return
	}
	rows, err := database.GetSportMaxHeartratesForUserID(userID)
	if err != nil {
		logger.Log.Error("Failed to get sport max heart rates. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sport heart rates."})
		context.Abort()
		return
	}
	observed, err := database.GetObservedMaxHeartratesByActionForUserID(userID)
	if err != nil {
		logger.Log.Error("Failed to get observed max heart rates per sport. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sport heart rates."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"sport_heartrates": mergeSportHeartrates(sports, rows, observed), "message": "Sport heart rates retrieved."})
}
