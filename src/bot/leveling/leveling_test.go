package leveling

import (
	"math"
	"testing"
)

func TestCumulativeExperience(t *testing.T) {
	// Known points of the game's curve; 2080 at level 10 is what a level-9
	// profile shows as its next threshold ("💡: …/2080").
	cases := map[uint]uint64{1: 0, 2: 100, 3: 220, 10: 2080}
	for level, want := range cases {
		if got := CumulativeExperience(level); (got != want) {
			t.Errorf("CumulativeExperience(%d) = %d, want %d", level, got, want)
		}
	}

	for level := uint(1); level < LevelMax; level++ {
		if (CumulativeExperience(level+1) <= CumulativeExperience(level)) {
			t.Fatalf("curve must grow: level %d → %d", level, level+1)
		}
	}
}

func TestProgressFor(t *testing.T) {
	progress := ProgressFor(9, 1984)
	baseline, target := CumulativeExperience(9), CumulativeExperience(10)
	if (!progress.Ok || progress.Remaining != target-1984) {
		t.Fatalf("level 9: %+v", progress)
	}

	wantPercent := float64(1984-baseline) / float64(target-baseline) * 100
	if (math.Abs(progress.Percent-wantPercent) > 1e-9) {
		t.Fatalf("percent: got %v, want %v", progress.Percent, wantPercent)
	}

	if (ProgressFor(0, 10).Ok) {
		t.Fatal("level 0 has no progress")
	}

	if (ProgressFor(9, target+1).Ok) {
		t.Fatal("experience past the next level is not a valid progress")
	}

	// At the cap, progress re-counts from zero toward the next prestige.
	capped := ProgressFor(LevelMax, 1000)
	if (!capped.Ok || capped.Remaining != CumulativeExperience(LevelMax)-1000) {
		t.Fatalf("level 40: %+v", capped)
	}
}

func TestTotalExperience(t *testing.T) {
	if got := TotalExperience(39, 0, 12345); (got != 12345) {
		t.Fatalf("below the cap experience is already cumulative: %d", got)
	}

	full := CumulativeExperience(LevelMax)
	if got := TotalExperience(LevelMax, 0, 10); (got != full+10) {
		t.Fatalf("reaching 40 banks one full curve: %d", got)
	}

	if got := TotalExperience(LevelMax, 2, 10); (got != full*3+10) {
		t.Fatalf("each prestige banks another: %d", got)
	}

	// Never drops across a prestige.
	before := TotalExperience(LevelMax, 2, full)
	after := TotalExperience(LevelMax, 3, 0)
	if (after < before) {
		t.Fatalf("total dropped across a prestige: %d → %d", before, after)
	}
}
