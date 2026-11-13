package repository

import "testing"

func TestCanHideAnotherProfile(t *testing.T) {
	cases := map[int64]bool{0: false, 1: false, 2: true, 10: true}
	for visible, want := range cases {
		if got := CanHideAnotherProfile(visible); (got != want) {
			t.Errorf("CanHideAnotherProfile(%d) = %v, want %v", visible, got, want)
		}
	}
}
