package models

import "testing"

func TestObservedMaxHeartrate(t *testing.T) {
	tests := []struct {
		name    string
		streams *StravaActivityStreams
		want    int
	}{
		{"nil streams", nil, 0},
		{"no heartrate channel", &StravaActivityStreams{}, 0},
		{"empty heartrate", &StravaActivityStreams{Heartrate: &StravaStream[int]{Data: []int{}}}, 0},
		{"normal peak", &StravaActivityStreams{Heartrate: &StravaStream[int]{Data: []int{120, 165, 158}}}, 165},
		{"dropout spike ignored", &StravaActivityStreams{Heartrate: &StravaStream[int]{Data: []int{150, 255, 300, 172}}}, 172},
		{"at plausible ceiling", &StravaActivityStreams{Heartrate: &StravaStream[int]{Data: []int{200, 226}}}, 226},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ObservedMaxHeartrate(tc.streams); got != tc.want {
				t.Fatalf("ObservedMaxHeartrate = %d, want %d", got, tc.want)
			}
		})
	}
}
