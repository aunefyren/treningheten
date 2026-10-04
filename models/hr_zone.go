package models

// Keys of the selectable heart-rate zone systems. Stored on User.HRZoneSystem and reported
// back on StreamSummary.HRZoneSystem; the definitions live in controllers (hrZoneSystems).
const (
	HRZoneSystemPercentMax    = "percent_max"
	HRZoneSystemReserve       = "reserve"
	HRZoneSystemOlympiatoppen = "olympiatoppen"
)

// HRZoneSystem defines one heart-rate zone model: where the zone edges sit and what the
// zones are called. Bounds are the upper edges of every zone but the last (which is
// open-ended), as fractions of maximum heart rate — or of heart-rate reserve when
// UsesResting is set. Codes and Names hold one entry per zone (len(Bounds)+1). The bottom
// zone has no lower edge, so time under the model's nominal floor counts toward it.
type HRZoneSystem struct {
	Key         string    `json:"key"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	UsesResting bool      `json:"uses_resting"`
	Bounds      []float64 `json:"bounds"`
	Codes       []string  `json:"codes"`
	Names       []string  `json:"names"`
}
