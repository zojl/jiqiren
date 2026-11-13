package battletime

import (
	"testing"
	"time"
)

func msk(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, Moscow)
}

func utcDay(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestNextAndLast(t *testing.T) {
	cases := []struct {
		now  time.Time
		next time.Time
	}{
		{msk(2026, 11, 5, 17, 59), msk(2026, 11, 5, 18, 0)},
		{msk(2026, 11, 5, 18, 0), msk(2026, 11, 6, 18, 0)},
		{msk(2026, 11, 5, 0, 30), msk(2026, 11, 5, 18, 0)},
		{msk(2026, 12, 31, 23, 0), msk(2027, 1, 1, 18, 0)},
	}

	for _, c := range cases {
		if got := Next(c.now); (!got.Equal(c.next)) {
			t.Errorf("Next(%v) = %v, want %v", c.now, got, c.next)
		}

		if got := Last(c.now); (!got.Equal(c.next.Add(-24 * time.Hour))) {
			t.Errorf("Last(%v) = %v", c.now, got)
		}
	}
}

func TestToday(t *testing.T) {
	// 02:30 MSK on the 5th is still the 4th in UTC — Today must follow MSK.
	if got := Today(msk(2026, 11, 5, 2, 30)); (!got.Equal(utcDay(2026, 11, 5))) {
		t.Fatalf("Today = %v", got)
	}
}

func TestBattleToday(t *testing.T) {
	// Before the 5th's battle, the running cycle started on the 4th.
	if got := BattleToday(msk(2026, 11, 5, 0, 30)); (!got.Equal(utcDay(2026, 11, 4))) {
		t.Fatalf("00:30 MSK: %v", got)
	}

	if got := BattleToday(msk(2026, 11, 5, 18, 30)); (!got.Equal(utcDay(2026, 11, 5))) {
		t.Fatalf("18:30 MSK: %v", got)
	}
}

func TestIsPastToday(t *testing.T) {
	now := msk(2026, 11, 5, 12, 0)
	if (!IsPastToday(11, 59, now) || IsPastToday(12, 1, now)) {
		t.Fatal("IsPastToday around noon")
	}
}

func TestBattleAnchors(t *testing.T) {
	day := utcDay(2026, 11, 5)
	if (!BattleAt(day).Equal(msk(2026, 11, 5, 18, 0))) {
		t.Fatalf("BattleAt: %v", BattleAt(day))
	}

	if (!PreBattleAnchor(day).Equal(msk(2026, 11, 5, 17, 58)) || !PostBattleAnchor(day).Equal(msk(2026, 11, 5, 18, 3))) {
		t.Fatalf("anchors: %v / %v", PreBattleAnchor(day), PostBattleAnchor(day))
	}
}
