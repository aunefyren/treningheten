package models

import "math"

// Distance conversions shared by the stream maths (splits, routes, best efforts).
const (
	MetersPerKm   = 1000.0
	MetersPerMile = 1609.344
)

// StreamLength is the sample count of a stream: the longest channel it carries.
func StreamLength(s *StravaActivityStreams) int {
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

// StreamTimes returns seconds-from-start per sample, synthesizing a 1 Hz clock when the
// activity has no time channel (mirrors assembleWorkoutStreams).
func StreamTimes(streams *StravaActivityStreams, n int) []int {
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

// CumulativeDistanceMeters builds a per-sample cumulative distance. GPS is authoritative
// (haversine over latlng); without it, velocity_smooth is integrated over the time deltas.
// Returns nil when neither signal exists (e.g. a treadmill run with only HR).
func CumulativeDistanceMeters(streams *StravaActivityStreams, times []int, n int) []float64 {
	if streams.LatLng != nil && len(streams.LatLng.Data) > 0 {
		pts := streams.LatLng.Data
		cum := make([]float64, n)
		for i := 1; i < n; i++ {
			cum[i] = cum[i-1]
			if i < len(pts) && len(pts[i]) == 2 && len(pts[i-1]) == 2 {
				cum[i] += HaversineMeters(pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1])
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

// HaversineMeters is the great-circle distance between two lat/lng points, in metres.
func HaversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0
	rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
