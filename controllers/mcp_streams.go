package controllers

import (
	"math"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

const (
	mcpDefaultStreamMaxPoints = 2000
	mcpHardStreamMaxPoints    = 5000
)

// assembleWorkoutStreams returns the processed Strava sensor data for one activity
// (operation) owned by the user. The summary header (via SummarizeStreams) always
// describes the whole workout — including per-distance splits, a route overview and
// heart-rate zones; the series is the raw sensor data restricted to [fromSeconds,
// toSeconds] and downsampled to stay within maxPoints (auto-fit when no resolution is
// given). resolution (seconds between samples; 1 = full fidelity) lets the caller zoom a
// narrow window back to full resolution.
func assembleWorkoutStreams(userID uuid.UUID, activityID uuid.UUID, fromSeconds int, toSeconds int, resolution int, maxPoints int) (models.MCPWorkoutStreams, error) {
	streams, distanceUnit, hrMax, hrRest, hrBasis, err := loadActivityStreamContext(userID, activityID)
	if err != nil {
		return models.MCPWorkoutStreams{}, err
	}
	if streams == nil {
		return models.MCPWorkoutStreams{
			HasStreams: false,
			Message:    "This activity has no Strava sensor streams. Streams exist only for GPS/sensor activities imported from Strava (e.g. runs and rides); strength and manually-logged workouts have none.",
		}, nil
	}

	summary := SummarizeStreams(streams, distanceUnit, hrMax, hrRest, hrBasis)
	out := models.MCPWorkoutStreams{HasStreams: true}
	if summary != nil {
		out.StreamSummary = *summary
	}

	// time stream gives seconds-from-start per index; synthesize 1 Hz if absent.
	n := streamLength(streams)
	times := streamTimes(streams, n)

	// --- windowed, downsampled series ---
	from := fromSeconds
	if from < 0 {
		from = 0
	}
	to := toSeconds
	if to <= 0 && n > 0 {
		to = times[n-1]
	}
	if maxPoints <= 0 {
		maxPoints = mcpDefaultStreamMaxPoints
	}
	if maxPoints > mcpHardStreamMaxPoints {
		maxPoints = mcpHardStreamMaxPoints
	}

	idx, totalInWindow := selectStreamIndices(times, from, to, resolution, maxPoints)
	series := &models.MCPStreamSeries{
		ReturnedPoints:      len(idx),
		TotalPointsInWindow: totalInWindow,
		WindowFromSeconds:   from,
		WindowToSeconds:     to,
		SampledEverySeconds: averageSpacing(times, idx),
		TSeconds:            make([]int, 0, len(idx)),
	}
	for _, i := range idx {
		series.TSeconds = append(series.TSeconds, times[i])
		if streams.Heartrate != nil {
			series.HeartrateBpm = append(series.HeartrateBpm, intAt(streams.Heartrate.Data, i))
		}
		if streams.Altitude != nil {
			series.AltitudeM = append(series.AltitudeM, round1(floatAt(streams.Altitude.Data, i)))
		}
		if streams.VelocitySmooth != nil {
			series.SpeedKmh = append(series.SpeedKmh, round1(floatAt(streams.VelocitySmooth.Data, i)*3.6))
		}
		if streams.Cadence != nil {
			series.CadenceRpm = append(series.CadenceRpm, intAt(streams.Cadence.Data, i))
		}
		if streams.Watts != nil {
			series.Watts = append(series.Watts, intAt(streams.Watts.Data, i))
		}
		if streams.Temp != nil {
			series.TempC = append(series.TempC, intAt(streams.Temp.Data, i))
		}
		if streams.LatLng != nil && i < len(streams.LatLng.Data) {
			series.LatLng = append(series.LatLng, streams.LatLng.Data[i])
		}
	}
	out.Series = series

	return out, nil
}

// selectStreamIndices picks the sample indices whose time falls in [from, to],
// thinned to ~resolution-second spacing (when resolution > 1) and then uniformly
// strided down to at most maxPoints. Returns the chosen indices and how many raw
// samples were in the window before downsampling.
func selectStreamIndices(times []int, from int, to int, resolution int, maxPoints int) ([]int, int) {
	idx := []int{}
	for i, t := range times {
		if t >= from && t <= to {
			idx = append(idx, i)
		}
	}
	total := len(idx)

	if resolution > 1 && total > 0 {
		thinned := []int{idx[0]}
		lastT := times[idx[0]]
		for _, i := range idx[1:] {
			if times[i]-lastT >= resolution {
				thinned = append(thinned, i)
				lastT = times[i]
			}
		}
		idx = thinned
	}

	if maxPoints > 0 && len(idx) > maxPoints {
		stride := int(math.Ceil(float64(len(idx)) / float64(maxPoints)))
		strided := make([]int, 0, maxPoints)
		for j := 0; j < len(idx); j += stride {
			strided = append(strided, idx[j])
		}
		idx = strided
	}

	return idx, total
}

// loadActivityStreamContext gathers everything SummarizeStreams needs for one activity: the
// raw streams (nil when the activity has none), the distance unit (km vs mile splits) and the
// resolved HR-zone anchors. The athlete's age (for age-based zones) is taken from the
// activity's own date, not today, so an old activity stays historically accurate. Shared by
// get_activity_streams (which also builds the raw series) and get_activity (summary only).
func loadActivityStreamContext(userID uuid.UUID, activityID uuid.UUID) (streams *models.StravaActivityStreams, distanceUnit string, hrMax int, hrRest int, hrBasis string, err error) {
	sets, err := database.GetOperationSetsByOperationIDAndUserID(activityID, userID)
	if err != nil {
		return nil, "", 0, 0, "", err
	}
	for i := range sets {
		if sets[i].StravaStreams != nil {
			streams = &sets[i].StravaStreams.StravaActivityStreams
			break
		}
	}
	if streams == nil {
		return nil, "", 0, 0, "", nil
	}

	distanceUnit = "km"
	activityDate := time.Now()
	if operation, err := database.GetOperationByIDAndUserID(activityID, userID); err == nil {
		if operation.DistanceUnit != "" {
			distanceUnit = operation.DistanceUnit
		}
		if exercise, err := database.GetExerciseByIDAndUserID(operation.ExerciseID, userID); err == nil && exercise != nil {
			if day, err := database.GetExerciseDayByIDAndUserID(exercise.ExerciseDayID, userID); err == nil && day != nil {
				activityDate = day.Date
			}
		}
	}
	if user, err := database.GetUserInformation(userID); err == nil {
		hrMax, hrRest, hrBasis = resolveUserHR(user, activityDate)
	}
	return streams, distanceUnit, hrMax, hrRest, hrBasis, nil
}

// assembleActivityStreamSummary returns the processed StreamSummary for one activity, or nil
// when it has no Strava streams. It is the summary half of get_activity_streams without the
// raw series — get_activity attaches the requested blocks so a caller can read splits, zones
// and derived metrics without pulling thousands of samples.
func assembleActivityStreamSummary(userID uuid.UUID, activityID uuid.UUID) (*models.StreamSummary, error) {
	streams, distanceUnit, hrMax, hrRest, hrBasis, err := loadActivityStreamContext(userID, activityID)
	if err != nil {
		return nil, err
	}
	if streams == nil {
		return nil, nil
	}
	return SummarizeStreams(streams, distanceUnit, hrMax, hrRest, hrBasis), nil
}

// Recognized get_activity include tokens, mapping the caller-facing name to a summary block.
const (
	includeSegments  = "segments"
	includeZones     = "zones"
	includeElevation = "elevation"
	includeRoute     = "route"
	includeProfile   = "profile"
	includeAnalysis  = "analysis"
)

// filterStreamSummary returns a copy of full carrying only the blocks named in include (plus
// the always-cheap header: available channels, has_gps, duration and the per-channel stats),
// so a caller opts into the heavy derived blocks. An unrecognized token is ignored. Returns
// nil when full is nil or include selects nothing.
func filterStreamSummary(full *models.StreamSummary, include []string) *models.StreamSummary {
	if full == nil {
		return nil
	}
	// Header stats are cheap and small, so they always ride along once any block is requested.
	out := &models.StreamSummary{
		Available:       full.Available,
		HasGPS:          full.HasGPS,
		DurationSeconds: full.DurationSeconds,
		Heartrate:       full.Heartrate,
		Cadence:         full.Cadence,
		Speed:           full.Speed,
		Power:           full.Power,
		Temperature:     full.Temperature,
	}
	selected := false
	for _, token := range include {
		switch strings.ToLower(strings.TrimSpace(token)) {
		case includeSegments:
			out.Segments = full.Segments
		case includeZones:
			out.HRZones = full.HRZones
			out.HRMaxBasis = full.HRMaxBasis
			out.HRMaxBpm = full.HRMaxBpm
			out.HRRestBpm = full.HRRestBpm
		case includeElevation:
			out.Elevation = full.Elevation
		case includeRoute:
			out.Route = full.Route
		case includeProfile:
			out.ElevationProfile = full.ElevationProfile
		case includeAnalysis:
			out.Analysis = full.Analysis
		default:
			continue
		}
		selected = true
	}
	if !selected {
		return nil
	}
	return out
}

func averageSpacing(times []int, idx []int) int {
	if len(idx) < 2 {
		return 1
	}
	span := times[idx[len(idx)-1]] - times[idx[0]]
	spacing := int(math.Round(float64(span) / float64(len(idx)-1)))
	if spacing < 1 {
		spacing = 1
	}
	return spacing
}

func intAt(data []int, i int) int {
	if i < 0 || i >= len(data) {
		return 0
	}
	return data[i]
}

func floatAt(data []float64, i int) float64 {
	if i < 0 || i >= len(data) {
		return 0
	}
	return data[i]
}
