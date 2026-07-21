package controllers

import (
	"math"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// Segment/route/zone tuning constants.
const (
	metersPerKm   = 1000.0
	metersPerMile = 1609.344
	// routeOverviewMaxPoints caps the down-sampled polyline on StreamRoute so the
	// overview stays small; the full trace is still available via the raw series.
	routeOverviewMaxPoints = 120
)

// hrZoneBounds are the upper fractions of HRmax for zones 1-4 (zone 5 is open-ended).
// Standard five-zone %-of-max model: recovery / endurance / tempo / threshold / anaerobic.
var hrZoneBounds = []float64{0.60, 0.70, 0.80, 0.90}
var hrZoneNames = []string{"Recovery", "Endurance", "Tempo", "Threshold", "Anaerobic"}

// SummarizeStreams turns one activity's raw Strava sensor streams into the shared,
// presentation-ready StreamSummary consumed by both the MCP get_activity_streams tool
// and the /exercises detail page. It is pure: distanceUnit selects km vs mile splits;
// hrMax anchors the HR zones (0 falls back to the activity's own peak); hrRest (when set
// and below hrMax) switches the zones to heart-rate reserve (Karvonen); hrBasis labels
// where hrMax came from ("max"/"age"). Returns nil when there are no streams.
func SummarizeStreams(streams *models.StravaActivityStreams, distanceUnit string, hrMax int, hrRest int, hrBasis string) *models.StreamSummary {
	if streams == nil {
		return nil
	}

	n := streamLength(streams)
	times := streamTimes(streams, n)

	out := &models.StreamSummary{}
	out.Available = streamNames(streams)
	out.HasGPS = streams.LatLng != nil && len(streams.LatLng.Data) > 0
	if n > 0 {
		out.DurationSeconds = int64(times[n-1])
	}

	// --- whole-workout header ---
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

	// --- derived views ---
	cumMeters := cumulativeDistanceMeters(streams, times, n)
	out.Segments = computeSegments(streams, times, cumMeters, distanceUnit)
	out.Route = computeRoute(streams, cumMeters)
	out.ElevationProfile = computeElevationProfile(streams, cumMeters)
	if out.Elevation != nil {
		out.Elevation.BiggestClimb = computeBiggestClimb(streams, cumMeters)
	}
	out.HRZones, out.HRMaxBasis, out.HRMaxBpm = computeHRZones(streams, times, hrMax, hrRest, hrBasis)
	if out.HRMaxBasis == "reserve" {
		out.HRRestBpm = hrRest
	}
	out.Analysis = computeAnalysis(streams, times, cumMeters, out.Segments)

	return out
}

// Analysis tuning constants.
const (
	// breakSpeedMS is the speed at or below which a sample counts as stopped/very slow, and
	// minBreakSeconds is how long that must be sustained to count as a break (so a single
	// slow GPS sample doesn't register). gradientWindowMeters smooths per-sample gradient
	// over a short distance so sensor noise doesn't scatter samples across bands.
	breakSpeedMS         = 0.5
	minBreakSeconds      = 8
	gradientWindowMeters = 30.0
)

// gradientBandBounds are the upper percent-grade edges of the descent→climb bands; the label
// list has one more entry than the bounds for the open-ended top (steep climb) band.
var gradientBandBounds = []float64{-5, -2, 2, 5}
var gradientBandLabels = []string{"steep descent", "descent", "flat", "climb", "steep climb"}

// computeAnalysis derives the second-order "coach" metrics from the raw stream. Each block is
// independent and returns nil/empty when its required channels are missing, so the whole
// StreamAnalysis is nil only when nothing at all could be computed.
func computeAnalysis(streams *models.StravaActivityStreams, times []int, cumMeters []float64, segments []models.StreamSegment) *models.StreamAnalysis {
	analysis := &models.StreamAnalysis{}
	populated := false

	if d := computeDecoupling(streams, times, cumMeters); d != nil {
		analysis.DecouplingPct = d
		populated = true
	}
	if s := computeSplitHalves(streams, times, cumMeters); s != nil {
		analysis.SplitHalves = s
		populated = true
	}
	if p := computePaceStdDev(segments); p != nil {
		analysis.PaceStdDevSeconds = p
		populated = true
	}
	if b := computeBreaks(streams, times, cumMeters); b != nil {
		analysis.Breaks = b
		populated = true
	}
	if g := computeHRByGradient(streams, times, cumMeters); g != nil {
		analysis.HRByGradient = g
		populated = true
	}

	if !populated {
		return nil
	}
	return analysis
}

// midpointIndex returns the first sample index at or past the midpoint of elapsed time, used
// to split an activity into first and second halves by time.
func midpointIndex(times []int) int {
	n := len(times)
	if n < 2 {
		return 0
	}
	mid := (times[0] + times[n-1]) / 2
	for i, t := range times {
		if t >= mid {
			return i
		}
	}
	return n - 1
}

// computeDecoupling measures aerobic decoupling: the percent drop in speed-per-heartbeat
// efficiency from the first half to the second half. Needs heart rate and a distance signal.
// Returns nil when either half lacks a valid speed or heart rate.
func computeDecoupling(streams *models.StravaActivityStreams, times []int, cumMeters []float64) *float64 {
	if streams.Heartrate == nil || cumMeters == nil {
		return nil
	}
	n := len(times)
	if n < 4 || len(cumMeters) < n {
		return nil
	}
	mid := midpointIndex(times)
	if mid <= 0 || mid >= n-1 {
		return nil
	}
	ratio := func(from, to int) (float64, bool) {
		dt := times[to] - times[from]
		dist := cumMeters[to] - cumMeters[from]
		if dt <= 0 || dist <= 0 {
			return 0, false
		}
		hr := avgIntOver(streams.Heartrate, from, to, true)
		if hr == nil || *hr <= 0 {
			return 0, false
		}
		speed := dist / float64(dt)
		return speed / float64(*hr), true
	}
	first, ok1 := ratio(0, mid)
	second, ok2 := ratio(mid, n-1)
	if !ok1 || !ok2 || first <= 0 {
		return nil
	}
	dec := round1((first - second) / first * 100)
	return &dec
}

// computeSplitHalves reports each half's elapsed time, distance, average heart rate and pace,
// split at the midpoint of elapsed time. Returns nil without heart rate or a distance signal.
func computeSplitHalves(streams *models.StravaActivityStreams, times []int, cumMeters []float64) *models.StreamSplitHalves {
	n := len(times)
	if n < 4 {
		return nil
	}
	mid := midpointIndex(times)
	if mid <= 0 || mid >= n-1 {
		return nil
	}
	half := func(from, to int) models.StreamHalf {
		h := models.StreamHalf{ElapsedSeconds: int64(times[to] - times[from])}
		if streams.Heartrate != nil {
			h.AvgHeartrateBpm = avgIntOver(streams.Heartrate, from, to, true)
		}
		if cumMeters != nil && to < len(cumMeters) {
			dist := cumMeters[to] - cumMeters[from]
			if dist > 0 {
				h.DistanceKm = round2(dist / metersPerKm)
				if secs := float64(times[to] - times[from]); secs > 0 {
					h.AvgPaceMinKm = round2(secs * metersPerKm / (60.0 * dist))
				}
			}
		}
		return h
	}
	first := half(0, mid)
	second := half(mid, n-1)
	if first.AvgHeartrateBpm == nil && first.DistanceKm == 0 {
		return nil
	}
	return &models.StreamSplitHalves{First: first, Second: second}
}

// computePaceStdDev returns the standard deviation of the full-length splits' pace, in seconds
// per unit distance. The trailing partial split is excluded. Returns nil with fewer than two
// full splits.
func computePaceStdDev(segments []models.StreamSegment) *float64 {
	paces := []float64{}
	for _, s := range segments {
		// Full-length splits only (the last split may be a partial unit).
		if s.AvgPaceMinKm > 0 && s.Distance >= 0.95 {
			paces = append(paces, s.AvgPaceMinKm*60)
		}
	}
	if len(paces) < 2 {
		return nil
	}
	mean := 0.0
	for _, p := range paces {
		mean += p
	}
	mean /= float64(len(paces))
	varSum := 0.0
	for _, p := range paces {
		d := p - mean
		varSum += d * d
	}
	std := round1(math.Sqrt(varSum / float64(len(paces))))
	return &std
}

// computeBreaks finds sustained stretches of near-zero speed (stops and walk breaks). Speed
// comes from velocity_smooth when present, else from the distance deltas. Returns nil without
// any speed signal or when no stretch is long enough to count.
func computeBreaks(streams *models.StravaActivityStreams, times []int, cumMeters []float64) *models.StreamBreaks {
	n := len(times)
	if n < 2 {
		return nil
	}
	if streams.VelocitySmooth == nil && cumMeters == nil {
		return nil
	}
	speedAt := func(i int) float64 {
		if streams.VelocitySmooth != nil && i < len(streams.VelocitySmooth.Data) {
			return streams.VelocitySmooth.Data[i]
		}
		if cumMeters != nil && i > 0 && i < len(cumMeters) {
			if dt := times[i] - times[i-1]; dt > 0 {
				return (cumMeters[i] - cumMeters[i-1]) / float64(dt)
			}
		}
		return 0
	}

	breaks := []models.StreamBreak{}
	inBreak := false
	startIdx := 0
	flush := func(endIdx int) {
		if !inBreak {
			return
		}
		if dur := times[endIdx] - times[startIdx]; dur >= minBreakSeconds {
			breaks = append(breaks, models.StreamBreak{
				FromSeconds:     times[startIdx],
				ToSeconds:       times[endIdx],
				DurationSeconds: dur,
			})
		}
		inBreak = false
	}
	for i := 0; i < n; i++ {
		if speedAt(i) <= breakSpeedMS {
			if !inBreak {
				inBreak, startIdx = true, i
			}
		} else {
			flush(i)
		}
	}
	flush(n - 1)

	if len(breaks) == 0 {
		return nil
	}
	total := 0
	for _, b := range breaks {
		total += b.DurationSeconds
	}
	return &models.StreamBreaks{Count: len(breaks), TotalDurationSeconds: total, Breaks: breaks}
}

// computeHRByGradient buckets time and average heart rate by terrain gradient. Gradient is
// smoothed over gradientWindowMeters so sensor noise doesn't scatter samples across bands.
// Needs heart rate, altitude and a distance signal; returns nil otherwise.
func computeHRByGradient(streams *models.StravaActivityStreams, times []int, cumMeters []float64) []models.StreamHRGradientBucket {
	if streams.Heartrate == nil || streams.Altitude == nil || cumMeters == nil {
		return nil
	}
	alt := streams.Altitude.Data
	hr := streams.Heartrate.Data
	n := len(times)
	if len(alt) < n {
		n = len(alt)
	}
	if len(cumMeters) < n {
		n = len(cumMeters)
	}
	if n < 2 {
		return nil
	}

	type acc struct {
		seconds int64
		hrSum   float64
		hrDt    int64
		meters  float64
	}
	accs := make([]acc, len(gradientBandLabels))
	j := 0
	for i := 1; i < n; i++ {
		// Slide j back to ~gradientWindowMeters behind i for a smoothed gradient.
		for j < i-1 && cumMeters[i]-cumMeters[j] > gradientWindowMeters {
			j++
		}
		dist := cumMeters[i] - cumMeters[j]
		if dist <= 0 {
			continue
		}
		dt := int64(times[i] - times[i-1])
		if dt <= 0 {
			continue
		}
		grad := (alt[i] - alt[j]) / dist * 100
		bi := gradientBandIndex(grad)
		accs[bi].seconds += dt
		accs[bi].meters += cumMeters[i] - cumMeters[i-1]
		if i < len(hr) && hr[i] > 0 {
			accs[bi].hrSum += float64(hr[i]) * float64(dt)
			accs[bi].hrDt += dt
		}
	}

	buckets := []models.StreamHRGradientBucket{}
	for bi, a := range accs {
		if a.seconds == 0 {
			continue
		}
		b := models.StreamHRGradientBucket{
			Label:   gradientBandLabels[bi],
			Seconds: a.seconds,
		}
		if bi > 0 {
			lo := gradientBandBounds[bi-1]
			b.MinGradePct = &lo
		}
		if bi < len(gradientBandBounds) {
			hi := gradientBandBounds[bi]
			b.MaxGradePct = &hi
		}
		if a.hrDt > 0 {
			avg := int(math.Round(a.hrSum / float64(a.hrDt)))
			b.AvgHeartrateBpm = &avg
		}
		if a.meters > 0 {
			b.AvgPaceMinKm = round2(float64(a.seconds) * metersPerKm / (60.0 * a.meters))
		}
		buckets = append(buckets, b)
	}
	if len(buckets) == 0 {
		return nil
	}
	return buckets
}

// gradientBandIndex maps a percent grade to a band index using the upper-edge bounds (mirrors
// hrZoneIndex): the first band whose upper edge the grade is below, else the open-ended top.
func gradientBandIndex(grad float64) int {
	for i, b := range gradientBandBounds {
		if grad < b {
			return i
		}
	}
	return len(gradientBandBounds)
}

// resolveUserHR turns a user's configured heart-rate settings into the inputs the zone
// model needs: their maximum HR, their resting HR (for reserve zones), and a basis label.
// Max precedence: an explicit setting wins, then the all-time max observed across their
// activities (real data beats a formula), then the age-based estimate. A zero max means
// "use this activity's observed peak" — resolved inside computeHRZones.
func resolveUserHR(user models.User, now time.Time) (hrMax int, hrRest int, basis string) {
	if user.MaxHeartrate != nil && *user.MaxHeartrate > 0 {
		hrMax, basis = *user.MaxHeartrate, "max"
	} else if user.ObservedMaxHeartrate != nil && *user.ObservedMaxHeartrate > 0 {
		hrMax, basis = *user.ObservedMaxHeartrate, "observed_max"
	} else if age := hrMaxFromBirthDate(user.BirthDate, now); age > 0 {
		hrMax, basis = age, "age"
	}
	if user.RestingHeartrate != nil && *user.RestingHeartrate > 0 {
		hrRest = *user.RestingHeartrate
	}
	return
}

// computeElevationProfile builds a down-sampled altitude-over-distance profile. Returns
// nil without both an altitude channel and a distance signal to plot it against.
func computeElevationProfile(streams *models.StravaActivityStreams, cumMeters []float64) []models.StreamElevationPoint {
	if streams.Altitude == nil || len(streams.Altitude.Data) == 0 || cumMeters == nil {
		return nil
	}
	alt := streams.Altitude.Data
	n := len(alt)
	if len(cumMeters) < n {
		n = len(cumMeters)
	}
	if n < 2 {
		return nil
	}
	stride := 1
	if n > routeOverviewMaxPoints {
		stride = int(math.Ceil(float64(n) / float64(routeOverviewMaxPoints)))
	}
	profile := []models.StreamElevationPoint{}
	for i := 0; i < n; i += stride {
		profile = append(profile, models.StreamElevationPoint{
			DistanceKm: round2(cumMeters[i] / metersPerKm),
			AltitudeM:  round1(alt[i]),
		})
	}
	// Keep the true final sample so the profile ends at the real finish.
	last := models.StreamElevationPoint{DistanceKm: round2(cumMeters[n-1] / metersPerKm), AltitudeM: round1(alt[n-1])}
	if len(profile) == 0 || profile[len(profile)-1] != last {
		profile = append(profile, last)
	}
	return profile
}

// computeBiggestClimb finds the single largest sustained ascent — the contiguous stretch
// with the greatest net altitude gain (a max-subarray over the altitude deltas) — and
// reports its gain, distance and average gradient. Trivial bumps (< minClimbGainM) are
// ignored. Returns nil without altitude + distance.
func computeBiggestClimb(streams *models.StravaActivityStreams, cumMeters []float64) *models.StreamClimb {
	if streams.Altitude == nil || len(streams.Altitude.Data) == 0 || cumMeters == nil {
		return nil
	}
	const minClimbGainM = 10.0
	alt := streams.Altitude.Data
	n := len(alt)
	if len(cumMeters) < n {
		n = len(cumMeters)
	}
	if n < 2 {
		return nil
	}

	bestGain, bestFrom, bestTo := 0.0, 0, 0
	curSum, curStart := 0.0, 0
	for i := 1; i < n; i++ {
		d := alt[i] - alt[i-1]
		if curSum <= 0 {
			curStart, curSum = i-1, d
		} else {
			curSum += d
		}
		if curSum > bestGain {
			bestGain, bestFrom, bestTo = curSum, curStart, i
		}
	}
	if bestGain < minClimbGainM || bestTo <= bestFrom {
		return nil
	}
	dist := cumMeters[bestTo] - cumMeters[bestFrom]
	grade := 0.0
	if dist > 0 {
		grade = bestGain / dist * 100
	}
	return &models.StreamClimb{
		GainM:      round1(bestGain),
		DistanceKm: round2(dist / metersPerKm),
		GradePct:   round1(grade),
		FromPoint:  bestFrom,
		ToPoint:    bestTo,
	}
}

// attachStreamSummaries walks an exercise day and populates OperationObject.StreamSummary
// for every moving activity that carries Strava streams, so the /exercises detail page
// renders the same processed summary the MCP tool exposes instead of re-deriving stats in
// JS. HR zones anchor from the day owner's settings; the age-based estimate uses the
// activity's own date (not today), so an old activity's zones stay historically accurate
// and don't drift as the user ages.
func attachStreamSummaries(day *models.ExerciseDayObject) {
	if day == nil {
		return
	}
	hrMax, hrRest, hrBasis := resolveUserHR(day.User, day.Date)
	for ei := range day.Exercises {
		ops := day.Exercises[ei].Operations
		for oi := range ops {
			var streams *models.StravaActivityStreams
			for si := range ops[oi].OperationSets {
				if ops[oi].OperationSets[si].StravaStreams != nil {
					streams = &ops[oi].OperationSets[si].StravaStreams.StravaActivityStreams
					break
				}
			}
			if streams == nil {
				continue
			}
			ops[oi].StreamSummary = SummarizeStreams(streams, ops[oi].DistanceUnit, hrMax, hrRest, hrBasis)
		}
	}
}

// streamTimes returns seconds-from-start per sample, synthesizing a 1 Hz clock when the
// activity has no time channel (mirrors assembleWorkoutStreams).
func streamTimes(streams *models.StravaActivityStreams, n int) []int {
	times := make([]int, n)
	if streams.Time != nil && len(streams.Time.Data) == n {
		copy(times, streams.Time.Data)
		return times
	}
	for i := range times {
		times[i] = i
	}
	return times
}

// cumulativeDistanceMeters builds a per-sample cumulative distance. GPS is authoritative
// (haversine over latlng); without it, velocity_smooth is integrated over the time deltas.
// Returns nil when neither signal exists (e.g. a treadmill run with only HR).
func cumulativeDistanceMeters(streams *models.StravaActivityStreams, times []int, n int) []float64 {
	if streams.LatLng != nil && len(streams.LatLng.Data) > 0 {
		pts := streams.LatLng.Data
		cum := make([]float64, n)
		for i := 1; i < n; i++ {
			cum[i] = cum[i-1]
			if i < len(pts) && len(pts[i]) == 2 && len(pts[i-1]) == 2 {
				cum[i] += haversineMeters(pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1])
			}
		}
		return cum
	}
	if streams.VelocitySmooth != nil && len(streams.VelocitySmooth.Data) > 0 {
		vel := streams.VelocitySmooth.Data
		cum := make([]float64, n)
		for i := 1; i < n; i++ {
			cum[i] = cum[i-1]
			dt := times[i] - times[i-1]
			if dt < 0 {
				dt = 0
			}
			if i < len(vel) && vel[i] > 0 {
				cum[i] += vel[i] * float64(dt)
			}
		}
		return cum
	}
	return nil
}

// computeSegments splits a moving activity into per-distance-unit pieces (per km, or per
// mile when distanceUnit reads as miles). The final split is whatever distance remains.
// Returns nil when there is no distance signal to split on.
func computeSegments(streams *models.StravaActivityStreams, times []int, cumMeters []float64, distanceUnit string) []models.StreamSegment {
	if cumMeters == nil {
		return nil
	}
	n := len(cumMeters)
	if n < 2 || cumMeters[n-1] <= 0 {
		return nil
	}

	unitMeters := metersPerKm
	if isMileUnit(distanceUnit) {
		unitMeters = metersPerMile
	}

	segments := []models.StreamSegment{}
	from := 0
	boundary := unitMeters
	appendSegment := func(fromIdx, toIdx int) {
		if toIdx <= fromIdx {
			return
		}
		segMeters := cumMeters[toIdx] - cumMeters[fromIdx]
		if segMeters <= 0 {
			return
		}
		elapsed := int64(times[toIdx] - times[fromIdx])
		seg := models.StreamSegment{
			Index:          len(segments) + 1,
			DistanceUnit:   distanceUnit,
			Distance:       round2(segMeters / unitMeters),
			ElapsedSeconds: elapsed,
			ElevationGainM: round1(elevationGainOver(streams.Altitude, fromIdx, toIdx)),
			FromPoint:      fromIdx,
			ToPoint:        toIdx,
		}
		if elapsed > 0 {
			seg.AvgSpeedKmh = round1(segMeters / float64(elapsed) * 3.6)
			seg.AvgPaceMinKm = round2(float64(elapsed) * metersPerKm / (60.0 * segMeters))
		}
		seg.AvgHeartrateBpm = avgIntOver(streams.Heartrate, fromIdx, toIdx, true)
		seg.AvgCadenceRpm = avgIntOver(streams.Cadence, fromIdx, toIdx, true)
		seg.AvgWatts = avgIntOver(streams.Watts, fromIdx, toIdx, false)
		segments = append(segments, seg)
	}

	for i := 0; i < n; i++ {
		if cumMeters[i] >= boundary {
			appendSegment(from, i)
			from = i
			// Skip whole units at once for the rare long gap between samples.
			for cumMeters[i] >= boundary {
				boundary += unitMeters
			}
		}
	}
	// Trailing partial split.
	appendSegment(from, n-1)

	if len(segments) == 0 {
		return nil
	}
	return segments
}

// computeRoute summarizes the GPS path: extent, endpoints, total length and a
// down-sampled overview polyline. Returns nil when the activity has no GPS.
func computeRoute(streams *models.StravaActivityStreams, cumMeters []float64) *models.StreamRoute {
	if streams.LatLng == nil || len(streams.LatLng.Data) == 0 {
		return nil
	}
	pts := streams.LatLng.Data

	route := &models.StreamRoute{PointCount: len(pts)}

	minLat, minLng := math.Inf(1), math.Inf(1)
	maxLat, maxLng := math.Inf(-1), math.Inf(-1)
	var first, last []float64
	for _, p := range pts {
		if len(p) != 2 {
			continue
		}
		if first == nil {
			first = []float64{round5(p[0]), round5(p[1])}
		}
		last = []float64{round5(p[0]), round5(p[1])}
		minLat, maxLat = math.Min(minLat, p[0]), math.Max(maxLat, p[0])
		minLng, maxLng = math.Min(minLng, p[1]), math.Max(maxLng, p[1])
	}
	route.Start = first
	route.End = last
	if first != nil {
		route.BoundingBox = &models.StreamBBox{
			MinLat: round5(minLat), MinLng: round5(minLng),
			MaxLat: round5(maxLat), MaxLng: round5(maxLng),
		}
	}

	if cumMeters != nil && len(cumMeters) > 0 {
		route.DistanceKm = round2(cumMeters[len(cumMeters)-1] / metersPerKm)
	}

	// Uniformly stride the trace down to an overview polyline.
	stride := 1
	if len(pts) > routeOverviewMaxPoints {
		stride = int(math.Ceil(float64(len(pts)) / float64(routeOverviewMaxPoints)))
	}
	poly := [][]float64{}
	for i := 0; i < len(pts); i += stride {
		if len(pts[i]) == 2 {
			poly = append(poly, []float64{round5(pts[i][0]), round5(pts[i][1])})
		}
	}
	// Always keep the true final point so the overview closes on the real finish.
	if last != nil && (len(poly) == 0 || !sameLatLng(poly[len(poly)-1], last)) {
		poly = append(poly, last)
	}
	route.Polyline = poly

	return route
}

// computeHRZones buckets heart-rate time into five zones. Boundaries are percentages of
// hrMax, unless a resting HR below hrMax is given — then they use heart-rate reserve
// (Karvonen: rest + pct·(max−rest)) and the basis becomes "reserve". When hrMax<=0
// nothing is configured, so the zones anchor to the activity's own peak and the basis is
// "observed". Returns nil when there is no heart-rate data.
func computeHRZones(streams *models.StravaActivityStreams, times []int, hrMax int, hrRest int, basis string) ([]models.StreamHRZone, string, int) {
	if streams.Heartrate == nil || len(streams.Heartrate.Data) == 0 {
		return nil, "", 0
	}
	hr := streams.Heartrate.Data

	if hrMax <= 0 {
		basis = "observed"
		for _, v := range hr {
			if v > hrMax {
				hrMax = v
			}
		}
	}
	if hrMax <= 0 {
		return nil, "", 0
	}

	useReserve := hrRest > 0 && hrRest < hrMax
	if useReserve {
		basis = "reserve"
	}

	// Zone bpm boundaries, rounded once so display bounds and bucketing agree.
	bounds := make([]int, len(hrZoneBounds))
	for i, f := range hrZoneBounds {
		if useReserve {
			bounds[i] = int(math.Round(float64(hrRest) + f*float64(hrMax-hrRest)))
		} else {
			bounds[i] = int(math.Round(f * float64(hrMax)))
		}
	}

	floorBpm := 0
	if useReserve {
		floorBpm = hrRest
	}

	seconds := make([]int64, len(hrZoneNames))
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
		return nil, "", 0
	}

	zones := make([]models.StreamHRZone, len(hrZoneNames))
	for i := range hrZoneNames {
		minBpm := floorBpm
		if i > 0 {
			minBpm = bounds[i-1]
		}
		maxBpm := 0 // open-ended top zone
		if i < len(bounds) {
			maxBpm = bounds[i]
		}
		zones[i] = models.StreamHRZone{
			Zone:    i + 1,
			Name:    hrZoneNames[i],
			MinBpm:  minBpm,
			MaxBpm:  maxBpm,
			Seconds: seconds[i],
			Percent: round1(float64(seconds[i]) / float64(total) * 100),
		}
	}
	return zones, basis, hrMax
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

// elevationGainOver sums positive altitude changes across [fromIdx, toIdx].
func elevationGainOver(alt *models.StravaStream[float64], fromIdx, toIdx int) float64 {
	if alt == nil {
		return 0
	}
	data := alt.Data
	gain := 0.0
	for i := fromIdx + 1; i <= toIdx && i < len(data); i++ {
		if d := data[i] - data[i-1]; d > 0 {
			gain += d
		}
	}
	return gain
}

// avgIntOver averages an int channel across [fromIdx, toIdx], optionally skipping
// zero/negative samples (HR/cadence read 0 while paused). Returns nil when empty.
func avgIntOver(ch *models.StravaStream[int], fromIdx, toIdx int, filterZero bool) *int {
	if ch == nil {
		return nil
	}
	data := ch.Data
	sum, count := 0, 0
	for i := fromIdx; i <= toIdx && i < len(data); i++ {
		if filterZero && data[i] <= 0 {
			continue
		}
		sum += data[i]
		count++
	}
	if count == 0 {
		return nil
	}
	v := int(math.Round(float64(sum) / float64(count)))
	return &v
}

// haversineMeters is the great-circle distance between two lat/lng points, in metres.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func isMileUnit(unit string) bool {
	switch unit {
	case "mi", "mile", "miles", "Mile", "Miles", "MI", "Mi":
		return true
	}
	return false
}

func sameLatLng(a, b []float64) bool {
	return len(a) == 2 && len(b) == 2 && a[0] == b[0] && a[1] == b[1]
}

// --- per-channel header stats (shared by SummarizeStreams and the MCP series header) ---

// intStat summarizes an int channel. When filterZero is set, zero/negative samples are
// ignored (heart rate and cadence record 0 while paused/stopped).
func intStat(data []int, filterZero bool) *models.StreamStat {
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
	return &models.StreamStat{
		Avg: round1(float64(sum) / float64(count)),
		Min: float64(min),
		Max: float64(max),
	}
}

func elevationStat(data []float64) *models.StreamElevationStat {
	if len(data) == 0 {
		return nil
	}
	min, max := math.Inf(1), math.Inf(-1)
	gain, loss := 0.0, 0.0
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
		} else if v < prev {
			loss += prev - v
		}
		prev = v
	}
	return &models.StreamElevationStat{
		GainM: round1(gain),
		LossM: round1(loss),
		MinM:  round1(min),
		MaxM:  round1(max),
	}
}

func speedStat(data []float64) *models.StreamSpeedStat {
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
	return &models.StreamSpeedStat{
		AvgKmh:       round1(avgKmh),
		MaxKmh:       round1(max * 3.6),
		AvgPaceMinKm: round2(pace),
	}
}

// powerStat summarizes watts and integrates work (kJ) using the time deltas so it is
// correct even when the series is not 1 Hz.
func powerStat(data []int, times []int) *models.StreamPowerStat {
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
	return &models.StreamPowerStat{
		AvgW:   round1(float64(sum) / float64(count)),
		MaxW:   float64(max),
		WorkKj: round1(work / 1000.0),
	}
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

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round5(v float64) float64 { return math.Round(v*100000) / 100000 }
