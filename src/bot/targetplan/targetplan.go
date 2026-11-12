// Package targetplan computes when a model.PlannedTarget should next fire —
// shared by the scheduler (deciding whether it's due right now) and the
// panel (sorting the upcoming-targets preview), so the two never drift apart
// on what "next" means.
package targetplan

import (
	"sort"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
)

// NextOccurrence returns the next MSK moment a planned target should
// (attempt to) fire, and false if there is no such moment left — a "once"
// entry that has already been applied. The returned moment can be in the
// past, which just means it's due now; callers looking to schedule don't
// need any separate catch-up logic for that.
//
// A reserve entry skipped because a target was already set does not
// advance LastAppliedDate, so it keeps reporting the same due moment until
// the condition clears (same day for recurring, indefinitely for once) —
// that's the retry behavior the reserve flag is meant to have.
func NextOccurrence(pt model.PlannedTarget, now time.Time) (time.Time, bool) {
	moscow := battletime.Moscow

	if (pt.Type == model.PlannedTargetTypeOnce) {
		if (pt.LastAppliedDate != nil) {
			return time.Time{}, false
		}

		startMsk := pt.StartDate.In(moscow)
		return time.Date(startMsk.Year(), startMsk.Month(), startMsk.Day(), int(pt.Hour), int(pt.Minute), 0, 0, moscow), true
	}

	today := battletime.Today(now)
	candidateDate := today
	if (pt.StartDate.After(candidateDate)) {
		candidateDate = pt.StartDate
	}

	if (pt.LastAppliedDate != nil && pt.LastAppliedDate.Equal(candidateDate)) {
		candidateDate = candidateDate.AddDate(0, 0, 1)
	}

	if (pt.EndDate != nil && candidateDate.After(*pt.EndDate)) {
		return time.Time{}, false
	}

	candidateMsk := candidateDate.In(moscow)
	return time.Date(candidateMsk.Year(), candidateMsk.Month(), candidateMsk.Day(), int(pt.Hour), int(pt.Minute), 0, 0, moscow), true
}

type Upcoming struct {
	Target model.PlannedTarget
	Moment time.Time
}

// BuildUpcoming sorts targets by their next occurrence and returns the
// first limit of them — entries with no next occurrence (a spent "once"
// entry) are dropped. limit <= 0 means no limit, used by the scheduler
// which needs every due entry, not just a handful.
func BuildUpcoming(targets []model.PlannedTarget, now time.Time, limit int) []Upcoming {
	upcoming := make([]Upcoming, 0, len(targets))
	for _, target := range targets {
		if moment, ok := NextOccurrence(target, now); (ok) {
			upcoming = append(upcoming, Upcoming{Target: target, Moment: moment})
		}
	}

	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].Moment.Before(upcoming[j].Moment) })

	if (limit > 0 && len(upcoming) > limit) {
		upcoming = upcoming[:limit]
	}

	return upcoming
}
