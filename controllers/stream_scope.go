package controllers

import "github.com/aunefyren/treningheten/models"

// isDistanceActivityType reports whether an operation type is measured over distance, so the
// speed, pace and split views of its stream summary carry meaning. Court and gym sports
// (timing, lifting) still record heart rate and cadence, but their speed channel is just
// shuffling around a court — splits of 186 min/km and "breaks" covering 95% of the session.
// Unknown or empty types are treated as distance activities so legacy rows keep their views.
func isDistanceActivityType(operationType string) bool {
	return operationType != "timing" && operationType != "lifting"
}

// scopeSummaryToActivityType drops the distance-derived blocks (speed, splits and the
// pace-based analysis) from a summary of a non-distance activity, keeping the sensor header
// (HR, cadence, power, temperature), HR zones, any GPS route/elevation and the terrain-based
// HR-by-gradient view. Applied after SummarizeStreams so the summary math stays
// sport-agnostic, like applyCadenceUnit.
func scopeSummaryToActivityType(summary *models.StreamSummary, operationType string) {
	if summary == nil || isDistanceActivityType(operationType) {
		return
	}
	summary.Speed = nil
	summary.Segments = nil
	if summary.Analysis == nil {
		return
	}
	summary.Analysis.DecouplingPct = nil
	summary.Analysis.SplitHalves = nil
	summary.Analysis.PaceStdDevSeconds = nil
	summary.Analysis.PaceVariationPct = nil
	summary.Analysis.Breaks = nil
	if len(summary.Analysis.HRByGradient) == 0 {
		summary.Analysis = nil
	}
}
