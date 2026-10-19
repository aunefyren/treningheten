package models

import "testing"

func TestUserWeekResultsOutcome(t *testing.T) {
	tests := []struct {
		name        string
		result      UserWeekResults
		wantMet     bool
		wantWon     bool
		wantOwes    bool
		wantTickets int
	}{
		{"competing, goal met", UserWeekResults{WeekCompletion: 1, CurrentStreak: 3, Competing: true, FullWeekParticipation: true}, true, true, false, 4},
		{"competing, overachieved", UserWeekResults{WeekCompletion: 2.5, Competing: true, FullWeekParticipation: true}, true, true, false, 1},
		{"competing, goal missed", UserWeekResults{WeekCompletion: 0.67, CurrentStreak: 2, Competing: true, FullWeekParticipation: true}, false, false, true, 3},
		{"sick leave with goal met neither wins", UserWeekResults{WeekCompletion: 1, Competing: true, SickLeave: true, FullWeekParticipation: true}, true, false, false, 1},
		{"sick leave with goal missed owes nothing", UserWeekResults{WeekCompletion: 0, Competing: true, SickLeave: true, FullWeekParticipation: true}, false, false, false, 1},
		{"joined mid-week and missed owes nothing", UserWeekResults{WeekCompletion: 0.33, Competing: true}, false, false, false, 1},
		{"joined mid-week and met still wins", UserWeekResults{WeekCompletion: 1, Competing: true}, true, true, false, 1},
		{"not competing never wins or owes", UserWeekResults{WeekCompletion: 1, FullWeekParticipation: true}, true, false, false, 1},
		{"not competing and missed owes nothing", UserWeekResults{WeekCompletion: 0, FullWeekParticipation: true}, false, false, false, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.GoalMet(); got != tc.wantMet {
				t.Errorf("GoalMet = %v, want %v", got, tc.wantMet)
			}
			if got := tc.result.WonWeek(); got != tc.wantWon {
				t.Errorf("WonWeek = %v, want %v", got, tc.wantWon)
			}
			if got := tc.result.OwesSpin(); got != tc.wantOwes {
				t.Errorf("OwesSpin = %v, want %v", got, tc.wantOwes)
			}
			if got := tc.result.WheelTickets(); got != tc.wantTickets {
				t.Errorf("WheelTickets = %d, want %d", got, tc.wantTickets)
			}
		})
	}
}
