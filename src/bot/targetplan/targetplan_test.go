package targetplan

import (
	"testing"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
)

func day(year int, month time.Month, d int) time.Time {
	return time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
}

func at(year int, month time.Month, d, hour, minute int) time.Time {
	return time.Date(year, month, d, hour, minute, 0, 0, battletime.Moscow)
}

func ptr(t time.Time) *time.Time {
	return &t
}

func TestNextOccurrenceOnce(t *testing.T) {
	now := at(2026, 11, 13, 12, 0)
	target := model.PlannedTarget{Type: model.PlannedTargetTypeOnce, StartDate: day(2026, 11, 20), Hour: 17, Minute: 30}

	moment, ok := NextOccurrence(target, now)
	if (!ok || !moment.Equal(at(2026, 11, 20, 17, 30))) {
		t.Fatalf("once, pending: %v %v", moment, ok)
	}

	// Past-due still reports its moment — it's simply due now.
	target.StartDate = day(2026, 11, 1)
	if moment, ok := NextOccurrence(target, now); (!ok || !moment.Equal(at(2026, 11, 1, 17, 30))) {
		t.Fatalf("once, overdue: %v %v", moment, ok)
	}

	target.LastAppliedDate = ptr(day(2026, 11, 1))
	if _, ok := NextOccurrence(target, now); (ok) {
		t.Fatal("an applied once entry has no next occurrence")
	}
}

func TestNextOccurrenceRecurring(t *testing.T) {
	now := at(2026, 11, 13, 12, 0)
	target := model.PlannedTarget{Type: model.PlannedTargetTypeRecurring, StartDate: day(2026, 11, 10), Hour: 17, Minute: 55}

	if moment, ok := NextOccurrence(target, now); (!ok || !moment.Equal(at(2026, 11, 13, 17, 55))) {
		t.Fatalf("recurring, today: %v %v", moment, ok)
	}

	target.LastAppliedDate = ptr(battletime.Today(now))
	if moment, ok := NextOccurrence(target, now); (!ok || !moment.Equal(at(2026, 11, 14, 17, 55))) {
		t.Fatalf("recurring, applied today → tomorrow: %v %v", moment, ok)
	}

	target.LastAppliedDate = nil
	target.StartDate = day(2026, 11, 20)
	if moment, ok := NextOccurrence(target, now); (!ok || !moment.Equal(at(2026, 11, 20, 17, 55))) {
		t.Fatalf("recurring, not started yet: %v %v", moment, ok)
	}

	target.StartDate = day(2026, 11, 1)
	target.EndDate = ptr(day(2026, 11, 12))
	if _, ok := NextOccurrence(target, now); (ok) {
		t.Fatal("recurring past its end date has no next occurrence")
	}

	// The end date itself is still included.
	target.EndDate = ptr(day(2026, 11, 13))
	if _, ok := NextOccurrence(target, now); (!ok) {
		t.Fatal("the end date is inclusive")
	}
}

func TestBuildUpcoming(t *testing.T) {
	now := at(2026, 11, 13, 12, 0)
	targets := []model.PlannedTarget{
		{Id: 1, Type: model.PlannedTargetTypeOnce, StartDate: day(2026, 11, 20), Hour: 10},
		{Id: 2, Type: model.PlannedTargetTypeRecurring, StartDate: day(2026, 11, 1), Hour: 17},
		{Id: 3, Type: model.PlannedTargetTypeOnce, StartDate: day(2026, 11, 1), LastAppliedDate: ptr(day(2026, 11, 1))},
		{Id: 4, Type: model.PlannedTargetTypeOnce, StartDate: day(2026, 11, 15), Hour: 9},
	}

	upcoming := BuildUpcoming(targets, now, 0)
	if (len(upcoming) != 3 || upcoming[0].Target.Id != 2 || upcoming[1].Target.Id != 4 || upcoming[2].Target.Id != 1) {
		t.Fatalf("order / spent entry dropped: %+v", upcoming)
	}

	if limited := BuildUpcoming(targets, now, 2); (len(limited) != 2 || limited[1].Target.Id != 4) {
		t.Fatalf("limit: %+v", limited)
	}
}
