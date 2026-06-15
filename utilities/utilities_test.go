package utilities

import (
	"reflect"
	"testing"
	"time"
)

// Reference week, anchored on a known Monday so weekday math is explicit:
//
//	Mon 2026-06-15 ... Sun 2026-06-21
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 12, 30, 45, 0, time.UTC)
}

func TestFindNextSunday(t *testing.T) {
	tests := []struct {
		name  string
		in    time.Time
		wantY int
		wantM time.Month
		wantD int
	}{
		{"monday", day(2026, 6, 15), 2026, 6, 21},
		{"tuesday", day(2026, 6, 16), 2026, 6, 21},
		{"wednesday", day(2026, 6, 17), 2026, 6, 21},
		{"thursday", day(2026, 6, 18), 2026, 6, 21},
		{"friday", day(2026, 6, 19), 2026, 6, 21},
		{"saturday", day(2026, 6, 20), 2026, 6, 21},
		{"sunday returns itself", day(2026, 6, 21), 2026, 6, 21},
		{"crosses month boundary", day(2026, 6, 30), 2026, 7, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindNextSunday(tt.in)
			if err != nil {
				t.Fatalf("FindNextSunday() unexpected error: %v", err)
			}
			if got.Weekday() != time.Sunday {
				t.Errorf("weekday = %v, want Sunday", got.Weekday())
			}
			if got.Year() != tt.wantY || got.Month() != tt.wantM || got.Day() != tt.wantD {
				t.Errorf("date = %04d-%02d-%02d, want %04d-%02d-%02d",
					got.Year(), got.Month(), got.Day(), tt.wantY, tt.wantM, tt.wantD)
			}
			// FindNextSunday stamps the end of day (SetClockToMaximum).
			if got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 {
				t.Errorf("clock = %02d:%02d:%02d, want 23:59:59", got.Hour(), got.Minute(), got.Second())
			}
			// The result must never be before the input's day.
			if got.Before(SetClockToMinimum(tt.in)) {
				t.Errorf("next sunday %v is before input day %v", got, tt.in)
			}
		})
	}
}

func TestFindEarlierMonday(t *testing.T) {
	tests := []struct {
		name  string
		in    time.Time
		wantY int
		wantM time.Month
		wantD int
	}{
		{"monday returns itself", day(2026, 6, 15), 2026, 6, 15},
		{"tuesday", day(2026, 6, 16), 2026, 6, 15},
		{"wednesday", day(2026, 6, 17), 2026, 6, 15},
		{"thursday", day(2026, 6, 18), 2026, 6, 15},
		{"friday", day(2026, 6, 19), 2026, 6, 15},
		{"saturday", day(2026, 6, 20), 2026, 6, 15},
		{"sunday", day(2026, 6, 21), 2026, 6, 15},
		{"crosses month boundary", day(2026, 7, 1), 2026, 6, 29},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindEarlierMonday(tt.in)
			if err != nil {
				t.Fatalf("FindEarlierMonday() unexpected error: %v", err)
			}
			if got.Weekday() != time.Monday {
				t.Errorf("weekday = %v, want Monday", got.Weekday())
			}
			if got.Year() != tt.wantY || got.Month() != tt.wantM || got.Day() != tt.wantD {
				t.Errorf("date = %04d-%02d-%02d, want %04d-%02d-%02d",
					got.Year(), got.Month(), got.Day(), tt.wantY, tt.wantM, tt.wantD)
			}
			// FindEarlierMonday stamps the start of day (SetClockToMinimum).
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
				t.Errorf("clock = %02d:%02d:%02d.%09d, want 00:00:00.000000000",
					got.Hour(), got.Minute(), got.Second(), got.Nanosecond())
			}
			// The result must never be after the input's day.
			if got.After(SetClockToMaximum(tt.in)) {
				t.Errorf("earlier monday %v is after input day %v", got, tt.in)
			}
		})
	}
}

func TestFindEarlierSunday(t *testing.T) {
	tests := []struct {
		name  string
		in    time.Time
		wantY int
		wantM time.Month
		wantD int
	}{
		{"sunday returns itself", day(2026, 6, 21), 2026, 6, 21},
		{"monday", day(2026, 6, 15), 2026, 6, 14},
		{"saturday", day(2026, 6, 20), 2026, 6, 14},
		{"crosses month boundary", day(2026, 7, 2), 2026, 6, 28},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindEarlierSunday(tt.in)
			if err != nil {
				t.Fatalf("FindEarlierSunday() unexpected error: %v", err)
			}
			if got.Weekday() != time.Sunday {
				t.Errorf("weekday = %v, want Sunday", got.Weekday())
			}
			if got.Year() != tt.wantY || got.Month() != tt.wantM || got.Day() != tt.wantD {
				t.Errorf("date = %04d-%02d-%02d, want %04d-%02d-%02d",
					got.Year(), got.Month(), got.Day(), tt.wantY, tt.wantM, tt.wantD)
			}
		})
	}
}

// FindEarlierMonday and FindNextSunday must bracket the same calendar week.
func TestWeekBracketIsSevenDays(t *testing.T) {
	for _, in := range []time.Time{
		day(2026, 6, 15), day(2026, 6, 18), day(2026, 6, 21), day(2026, 1, 1), day(2026, 12, 31),
	} {
		monday, err := FindEarlierMonday(in)
		if err != nil {
			t.Fatalf("FindEarlierMonday(%v): %v", in, err)
		}
		sunday, err := FindNextSunday(in)
		if err != nil {
			t.Fatalf("FindNextSunday(%v): %v", in, err)
		}
		// Monday 00:00:00 to the following Sunday 23:59:59 spans 6 full days.
		gap := sunday.Sub(monday)
		if gap < 6*24*time.Hour || gap >= 7*24*time.Hour {
			t.Errorf("for %v: week span = %v, want within [6d, 7d)", in, gap)
		}
	}
}

func TestValidatePasswordFormat(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"valid", "Password1", true},
		{"valid with norwegian upper", "Ærlig123", true},
		{"too short", "Pass1", false},
		{"no uppercase", "password1", false},
		{"no lowercase", "PASSWORD1", false},
		{"no number", "Password", false},
		{"empty", "", false},
		{"exactly eight valid", "Abcdefg1", true},
		{"too long", string(make([]byte, 129)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := ValidatePasswordFormat(tt.password)
			if err != nil {
				t.Fatalf("ValidatePasswordFormat() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ValidatePasswordFormat(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}

func TestRemoveIntFromArray(t *testing.T) {
	tests := []struct {
		name   string
		in     []int
		remove int
		want   []int
	}{
		{"removes single", []int{1, 2, 3}, 2, []int{1, 3}},
		{"removes all duplicates", []int{1, 2, 2, 3, 2}, 2, []int{1, 3}},
		{"absent value", []int{1, 2, 3}, 9, []int{1, 2, 3}},
		{"empty input", []int{}, 1, []int{}},
		{"removes only element", []int{5}, 5, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveIntFromArray(tt.in, tt.remove)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveIntFromArray(%v, %d) = %v, want %v", tt.in, tt.remove, got, tt.want)
			}
		})
	}
}

func TestIntToPaddedString(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "00"},
		{9, "09"},
		{10, "10"},
		{99, "99"},
		{2026, "2026"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := IntToPaddedString(tt.in); got != tt.want {
				t.Errorf("IntToPaddedString(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTimeToMySQLTimestamp(t *testing.T) {
	in := time.Date(2026, 6, 5, 8, 7, 6, 0, time.UTC)
	want := "2026-06-05 08:07:06.000"
	if got := TimeToMySQLTimestamp(in); got != want {
		t.Errorf("TimeToMySQLTimestamp() = %q, want %q", got, want)
	}
}
