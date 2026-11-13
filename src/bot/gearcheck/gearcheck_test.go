package gearcheck

import (
	"testing"

	"jiqiren/bot/database/model"
	"jiqiren/bot/inventory"
)

func u(value uint) *uint {
	return &value
}

func TestRoleFor(t *testing.T) {
	const own = 4
	squadDefends := &model.Team{Id: 1, Target: u(model.TargetDefense)}
	squadAttacks := &model.Team{Id: 1, Target: u(2)}
	squadIdle := &model.Team{Id: 1}

	cases := []struct {
		name   string
		target *uint
		squad  *model.Team
		want   Role
	}{
		{"own fraction is defense", u(own), nil, RoleDefense},
		{"another fraction is attack", u(2), squadDefends, RoleAttack},
		{"no target — squad defends", nil, squadDefends, RoleDefense},
		{"no target — squad attacks", nil, squadAttacks, RoleAttack},
		{"no target, squad without one", nil, squadIdle, RoleNone},
		{"no target, no squad", nil, nil, RoleNone},
		{"explicit none falls back to the squad", u(model.TargetNone), squadAttacks, RoleAttack},
	}

	for _, c := range cases {
		profile := model.Profile{FractionId: own, TargetId: c.target}
		if got := RoleFor(profile, c.squad); (got != c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestLabel(t *testing.T) {
	if got := (Result{Role: RoleAttack, Gain: 12}).Label(); (got != "можно +12⚔️") {
		t.Fatal(got)
	}

	if got := (Result{Role: RoleDefense, Gain: 3}).Label(); (got != "можно +3🛡️") {
		t.Fatal(got)
	}
}

func TestResultFor(t *testing.T) {
	// Attack 2·100 + 3·50 = 350.
	profile := model.Profile{Practice: 100, Cunning: 50}

	if got := resultFor(profile, RoleAttack, []int{300, 362, 340}); (got.Gain != 12) {
		t.Fatalf("best candidate wins: %+v", got)
	}

	if got := resultFor(profile, RoleAttack, []int{300}); (got.Gain != 0) {
		t.Fatalf("a profile above every optimum is the optimum: %+v", got)
	}

	if got := resultFor(profile, RoleAttack, nil); (got.Gain != 0) {
		t.Fatalf("no candidates, nothing to compare against: %+v", got)
	}
}

func TestInventoryOptimumFrom(t *testing.T) {
	// Before the analysis: defense 2·100 + 3·50 = 350, wearing "Old".
	base := model.Profile{Theory: 100, Wisdom: 50}
	items := []inventory.Item{
		{Location: inventory.LocationBackpack, Class: "📳", Name: "Old", Theory: 10, Wisdom: 5},
		{Location: inventory.LocationHome, Class: "📳", Name: "New", Theory: 30, Wisdom: 20},
		{Location: inventory.LocationHome, Class: "🖥", Name: "TooHigh", Theory: 99, RequiredLevel: 35},
	}

	// Base without items: 💾90 🔎45; plus "New": 💾120 🔎65 → 2·120 + 3·65 = 435.
	got := inventoryOptimumFrom(base, items, inventory.Player{Known: true, Level: 30}, RoleDefense)
	if (got != 435) {
		t.Fatalf("defense optimum: %d", got)
	}

	// At level 35 the 🖥 also fits: +99💾 → 435 + 198.
	if got := inventoryOptimumFrom(base, items, inventory.Player{Known: true, Level: 35}, RoleDefense); (got != 633) {
		t.Fatalf("level-gated item at the right level: %d", got)
	}
}
