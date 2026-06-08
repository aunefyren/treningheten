package controllers

import (
	"math"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

const (
	mcpDefaultStreamMaxPoints = 2000
	mcpHardStreamMaxPoints    = 5000
)

// assembleWorkoutStreams returns the processed Strava sensor data for one activity
// (operation) owned by the user. The summary header always describes the whole
// workout; the series is the raw sensor data restricted to [fromSeconds, toSeconds]
// and downsampled to stay within maxPoints (auto-fit when no resolution is given).
// resolution (seconds between samples; 1 = full fidelity) lets the caller zoom a
// narrow window back to full resolution.
func assembleWorkoutStreams(userID uuid.UUID, activityID uuid.UUID, fromSeconds int, toSeconds int, resolution int, maxPoints int) (models.MCPWorkoutStreams, error) {
	sets, err := database.GetOperationSetsByOperationIDAndUserID(activityID, userID)
	if err != nil {
		return models.MCPWorkoutStreams{}, err
	}

	var streams *models.StravaActivityStreams
	for i := range sets {
		if sets[i].StravaStreams != nil {
			streams = &sets[i].StravaStreams.StravaActivityStreams
			break
		}
	}
	if streams == nil {
		return models.MCPWorkoutStreams{
			HasStreams: false,
			Message:    "This activity has no Strava sensor streams. Streams exist only for GPS/sensor activities imported from Strava (e.g. runs and rides); strength and manually-logged workouts have none.",
		}, nil
	}

	// time stream gives seconds-from-start per index; synthesize 1 Hz if absent.
	n := streamLength(streams)
	times := make([]int, n)
	if streams.Time != nil && len(streams.Time.Data) == n {
		copy(times, streams.Time.Data)
	} else {
		for i := range times {
			times[i] = i
		}
	}

	out := models.MCPWorkoutStreams{HasStreams: true}
	if n > 0 {
		out.DurationSeconds = int64(times[n-1])
	}
	out.Available = streamNames(streams)
	out.HasGPS = streams.LatLng != nil && len(streams.LatLng.Data) > 0

	// --- whole-workout summary header ---
	if streams.Heartrate != nil {
		out.Heartrate = intStat(streams.Heartrate.Data, true)
	}
	if streams.Cadence != nil {
		out.Cadence = intStat(streams.Cadence.Data, true)
	}
	if streams.Temp != nil {
		out.Temperature = intStat(streams.Temp.Data, false)
	}
	if streams.Altitude != nil {
		out.Elevation = elevationStat(streams.Altitude.Data)
	}
	if streams.VelocitySmooth != nil {
		out.Speed = speedStat(streams.VelocitySmooth.Data)
	}
	if streams.Watts != nil {
		out.Power = powerStat(streams.Watts.Data, times)
	}

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

func streamLength(s *models.StravaActivityStreams) int {
	n := 0
	consider := func(l int) {
		if l > n {
			n = l
		}
	}
	if s.Time != nil {
		consider(len(s.Time.Data))
	}
	if s.Heartrate != nil {
		consider(len(s.Heartrate.Data))
	}
	if s.Altitude != nil {
		consider(len(s.Altitude.Data))
	}
	if s.VelocitySmooth != nil {
		consider(len(s.VelocitySmooth.Data))
	}
	if s.Cadence != nil {
		consider(len(s.Cadence.Data))
	}
	if s.Watts != nil {
		consider(len(s.Watts.Data))
	}
	if s.Temp != nil {
		consider(len(s.Temp.Data))
	}
	if s.LatLng != nil {
		consider(len(s.LatLng.Data))
	}
	return n
}

func streamNames(s *models.StravaActivityStreams) []string {
	names := []string{}
	if s.Heartrate != nil && len(s.Heartrate.Data) > 0 {
		names = append(names, "heartrate")
	}
	if s.Altitude != nil && len(s.Altitude.Data) > 0 {
		names = append(names, "altitude")
	}
	if s.VelocitySmooth != nil && len(s.VelocitySmooth.Data) > 0 {
		names = append(names, "velocity_smooth")
	}
	if s.Cadence != nil && len(s.Cadence.Data) > 0 {
		names = append(names, "cadence")
	}
	if s.Watts != nil && len(s.Watts.Data) > 0 {
		names = append(names, "watts")
	}
	if s.Temp != nil && len(s.Temp.Data) > 0 {
		names = append(names, "temp")
	}
	if s.LatLng != nil && len(s.LatLng.Data) > 0 {
		names = append(names, "latlng")
	}
	return names
}

// intStat summarizes an int channel. When filterZero is set, zero/negative samples
// are ignored (heart rate and cadence record 0 while paused/stopped).
func intStat(data []int, filterZero bool) *models.MCPStreamStat {
	sum, count := 0, 0
	min, max := math.MaxInt, math.MinInt
	for _, v := range data {
		if filterZero && v <= 0 {
			continue
		}
		sum += v
		count++
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if count == 0 {
		return nil
	}
	return &models.MCPStreamStat{
		Avg: round1(float64(sum) / float64(count)),
		Min: float64(min),
		Max: float64(max),
	}
}

func elevationStat(data []float64) *models.MCPElevationStat {
	if len(data) == 0 {
		return nil
	}
	min, max := math.Inf(1), math.Inf(-1)
	gain := 0.0
	prev := data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		if v > prev {
			gain += v - prev
		}
		prev = v
	}
	return &models.MCPElevationStat{
		GainM: round1(gain),
		MinM:  round1(min),
		MaxM:  round1(max),
	}
}

func speedStat(data []float64) *models.MCPSpeedStat {
	sum, count := 0.0, 0
	max := 0.0
	for _, v := range data {
		if v <= 0 {
			continue
		}
		sum += v
		count++
		if v > max {
			max = v
		}
	}
	if count == 0 {
		return nil
	}
	avgMs := sum / float64(count)
	avgKmh := avgMs * 3.6
	pace := 0.0
	if avgKmh > 0 {
		pace = 60.0 / avgKmh
	}
	return &models.MCPSpeedStat{
		AvgKmh:       round1(avgKmh),
		MaxKmh:       round1(max * 3.6),
		AvgPaceMinKm: round2(pace),
	}
}

// powerStat summarizes watts and integrates work (kJ) using the time deltas so it is
// correct even when the series is not 1 Hz.
func powerStat(data []int, times []int) *models.MCPPowerStat {
	sum, count, max := 0, 0, 0
	work := 0.0
	for i, v := range data {
		sum += v
		count++
		if v > max {
			max = v
		}
		dt := 1
		if i > 0 && i < len(times) {
			dt = times[i] - times[i-1]
			if dt < 0 {
				dt = 0
			}
		}
		work += float64(v) * float64(dt)
	}
	if count == 0 {
		return nil
	}
	return &models.MCPPowerStat{
		AvgW:   round1(float64(sum) / float64(count)),
		MaxW:   float64(max),
		WorkKj: round1(work / 1000.0),
	}
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

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
