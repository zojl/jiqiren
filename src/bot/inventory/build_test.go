package inventory

import "testing"

func TestCanWear(t *testing.T) {
	black := Item{Way: WayBlack}
	white := Item{Way: WayWhite}
	any := Item{}
	level35 := Item{RequiredLevel: 35}

	cases := []struct {
		name   string
		player Player
		item   Item
		want   bool
	}{
		{"unknown player wears anything", Player{}, black, true},
		{"blackhat, ⬛ item", PlayerFrom(30, 1), black, true},
		{"blackhat, ⬜ item", PlayerFrom(30, 1), white, false},
		{"whitehat, ⬜ item", PlayerFrom(30, 2), white, true},
		{"whitehat, ⬛ item", PlayerFrom(30, 2), black, false},
		{"🔲 wears both", PlayerFrom(30, 3), black, true},
		{"🔳 wears both", PlayerFrom(30, 4), white, true},
		{"no way yet, restricted item", PlayerFrom(9, 0), black, false},
		{"no way yet, unrestricted item", PlayerFrom(9, 0), any, true},
		{"level 34, needs 35", PlayerFrom(34, 2), level35, false},
		{"level 35, needs 35", PlayerFrom(35, 2), level35, true},
	}

	for _, c := range cases {
		if got, reason := c.player.CanWear(c.item); (got != c.want || (!got && reason == "")) {
			t.Errorf("%s: got %v (%q), want %v", c.name, got, reason, c.want)
		}
	}
}

func TestPlayerFrom(t *testing.T) {
	if player := PlayerFrom(30, 3); (!player.Known || player.Level != 30 || player.Way != "🔲") {
		t.Fatalf("%+v", player)
	}

	if player := PlayerFrom(5, 0); (player.Way != "") {
		t.Fatalf("no way: %+v", player)
	}
}
