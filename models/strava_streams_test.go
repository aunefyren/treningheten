package models

import "testing"

// TestStravaStreamsJSONRoundTrip covers the JSON column that stores an activity's sensor
// streams: what Value writes must come back out of Scan intact, for both the []byte and
// string forms a driver can hand back. The streams are the input to every derived metric
// (splits, HR zones, per-track effort), so a lossy round-trip is silently wrong data.
func TestStravaStreamsJSONRoundTrip(t *testing.T) {
	original := StravaStreamsJSON{StravaActivityStreams{
		Time:      &StravaStream[int]{Data: []int{0, 1, 2}, SeriesType: "distance", OriginalSize: 3, Resolution: "high"},
		Heartrate: &StravaStream[int]{Data: []int{120, 130, 141}},
		Altitude:  &StravaStream[float64]{Data: []float64{0, 2.5, 5.25}},
	}}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	encoded, ok := value.([]byte)
	if !ok {
		t.Fatalf("Value returned %T, want []byte", value)
	}

	for _, form := range []interface{}{encoded, string(encoded)} {
		var scanned StravaStreamsJSON
		if err := scanned.Scan(form); err != nil {
			t.Fatalf("Scan(%T) returned error: %v", form, err)
		}

		if scanned.Time == nil || len(scanned.Time.Data) != 3 || scanned.Time.Data[2] != 2 {
			t.Errorf("time stream did not survive: %+v", scanned.Time)
		}
		if scanned.Time.SeriesType != "distance" || scanned.Time.OriginalSize != 3 || scanned.Time.Resolution != "high" {
			t.Errorf("stream metadata did not survive: %+v", scanned.Time)
		}
		if scanned.Heartrate == nil || scanned.Heartrate.Data[1] != 130 {
			t.Errorf("heartrate stream did not survive: %+v", scanned.Heartrate)
		}
		// Floats must not be truncated to ints on the way through.
		if scanned.Altitude == nil || scanned.Altitude.Data[2] != 5.25 {
			t.Errorf("altitude stream lost precision: %+v", scanned.Altitude)
		}
		// A channel the activity never had stays absent rather than becoming an empty stream.
		if scanned.Cadence != nil {
			t.Errorf("cadence = %+v, want nil for a stream that was never recorded", scanned.Cadence)
		}
	}
}

// TestStravaStreamsJSONScanEdgeCases covers the column values Scan must survive: NULL, an
// empty column and the literal "null" all leave the value untouched rather than erroring,
// while a wrong type or malformed JSON is reported.
func TestStravaStreamsJSONScanEdgeCases(t *testing.T) {
	for _, value := range []interface{}{nil, []byte(""), []byte("null"), "null", ""} {
		var streams StravaStreamsJSON
		if err := streams.Scan(value); err != nil {
			t.Errorf("Scan(%#v) returned error: %v", value, err)
		}
		if streams.Time != nil || streams.Heartrate != nil {
			t.Errorf("Scan(%#v) populated streams from an empty column", value)
		}
	}

	var streams StravaStreamsJSON
	if err := streams.Scan(12345); err == nil {
		t.Errorf("Scan accepted an int column value")
	}
	if err := streams.Scan([]byte("{\"time\": ")); err == nil {
		t.Errorf("Scan accepted malformed JSON")
	}
}

// TestStravaStreamsJSONValueOfEmpty covers that an activity with no streams still stores as
// valid JSON — the column is written on every synced activity, streams or not.
func TestStravaStreamsJSONValueOfEmpty(t *testing.T) {
	var empty StravaStreamsJSON

	value, err := empty.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}

	var scanned StravaStreamsJSON
	if err := scanned.Scan(value); err != nil {
		t.Fatalf("Scan of an empty streams value returned error: %v", err)
	}
	if scanned.Time != nil {
		t.Errorf("empty streams scanned back with data: %+v", scanned.Time)
	}
}
