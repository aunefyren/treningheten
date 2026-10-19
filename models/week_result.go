package models

// The weekly outcome rules for one participant, in one place so debt generation, the wheel,
// achievements and the AI greeting all judge a week the same way. See docs/streaks.md and
// docs/seasons-and-goals.md.

// GoalMet reports whether the week's workouts reached the weekly goal. On its own it says
// nothing about sick leave or competing; WonWeek and OwesSpin add those.
func (r UserWeekResults) GoalMet() bool {
	return r.WeekCompletion >= 1
}

// WonWeek reports whether the participant earns a place on the wheel this week: competing,
// goal met, and not on sick leave (a sick-leave week can't count for or against anyone).
func (r UserWeekResults) WonWeek() bool {
	return r.Competing && r.GoalMet() && !r.SickLeave
}

// OwesSpin reports whether the participant has to spin the wheel for this week: competing,
// goal missed, not on sick leave, and in the season for the whole week (joining mid-week
// never costs a spin).
func (r UserWeekResults) OwesSpin() bool {
	return r.Competing && !r.GoalMet() && !r.SickLeave && r.FullWeekParticipation
}

// WheelTickets is how many entries a week winner gets on the wheel: one, plus one per week of
// the season streak they carried into the week. Only meaningful when WonWeek is true.
func (r UserWeekResults) WheelTickets() int {
	return r.CurrentStreak + 1
}
