package controllers

import "github.com/aunefyren/treningheten/models"

// Cadence units. Strava records foot-sport cadence per leg (a runner at 170 steps/min reads
// ~85), so for those sports the stream is doubled into steps per minute — the number runners
// and their watches use. Everything else (cycling, rowing, …) stays revolutions/strokes per
// minute as recorded.
const (
	cadenceUnitSteps       = "spm"
	cadenceUnitRevolutions = "rpm"
)

// stepCadenceSports are the Action.StravaName values whose cadence counts single-leg steps.
var stepCadenceSports = map[string]bool{
	"Run":        true,
	"TrailRun":   true,
	"VirtualRun": true,
	"Walk":       true,
	"Hike":       true,
}

// cadenceUnitForSport returns the unit an activity's cadence is presented in, keyed on the
// action's Strava sport name. Unknown or empty sports keep the recorded rpm.
func cadenceUnitForSport(stravaName string) string {
	if stepCadenceSports[stravaName] {
		return cadenceUnitSteps
	}
	return cadenceUnitRevolutions
}

// cadenceUnitForAction is cadenceUnitForSport for an optional action.
func cadenceUnitForAction(action *models.Action) string {
	if action == nil {
		return cadenceUnitRevolutions
	}
	return cadenceUnitForSport(action.StravaName)
}

// cadenceFactor is the multiplier from the recorded stream value to the presented unit.
func cadenceFactor(unit string) int {
	if unit == cadenceUnitSteps {
		return 2
	}
	return 1
}

// scaleCadence returns a copy of a recorded cadence value in the presented unit.
func scaleCadence(value *int, unit string) *int {
	if value == nil {
		return nil
	}
	scaled := *value * cadenceFactor(unit)
	return &scaled
}

// applyCadenceUnit converts a summary's cadence (header stat and per-split averages) from the
// recorded per-leg value into the sport's presented unit, and records that unit. A summary
// without cadence is left untouched. Applied after SummarizeStreams so the summary math stays
// sport-agnostic.
func applyCadenceUnit(summary *models.StreamSummary, unit string) {
	if summary == nil || summary.Cadence == nil {
		return
	}
	factor := float64(cadenceFactor(unit))
	summary.CadenceUnit = unit
	summary.Cadence = &models.StreamStat{
		Avg: round1(summary.Cadence.Avg * factor),
		Min: summary.Cadence.Min * factor,
		Max: summary.Cadence.Max * factor,
	}
	for i := range summary.Segments {
		summary.Segments[i].AvgCadenceRpm = scaleCadence(summary.Segments[i].AvgCadenceRpm, unit)
	}
}
