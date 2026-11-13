// Package gearcheck tells whether a player's profile shows them wearing a
// worse item set than they could for their next battle — the "неоптимальный
// набор вещей" readiness check, shared by the profile-accepted reply and
// the scheduled profile check.
package gearcheck

import (
	"strconv"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/inventory"
)

// historyWindow is how far back the best past profile counts as reachable
// — older records may have been set with items since sold.
const historyWindow = 30 * 24 * time.Hour

// baseProfileMaxAge bounds how long before an inventory analysis the
// profile its base stats come from may be — the older it is, the likelier
// the player wore something else back then than the analyzed backpack.
const baseProfileMaxAge = 24 * time.Hour

type Role int

const (
	RoleNone Role = iota
	RoleAttack
	RoleDefense
)

// RoleFor decides whether the player's next battle is attack or defense:
// by their own chosen target (their own fraction means defending, see
// readiness.TargetMatches), or, with none chosen, by their squad's target.
// No target anywhere — RoleNone, nothing to check.
func RoleFor(profile model.Profile, squad *model.Team) Role {
	if (profile.TargetId != nil && *profile.TargetId != model.TargetNone) {
		if (*profile.TargetId == profile.FractionId || *profile.TargetId == model.TargetDefense) {
			return RoleDefense
		}
		return RoleAttack
	}

	if (squad == nil || squad.Id == 0 || squad.Target == nil || *squad.Target == model.TargetNone) {
		return RoleNone
	}

	if (*squad.Target == model.TargetDefense) {
		return RoleDefense
	}

	return RoleAttack
}

type Result struct {
	Role Role
	// Gain is how many ⚔️/🛡️ points the optimum is above the profile; 0
	// when the profile is already at (or above) it.
	Gain int
}

// Label: "можно +12⚔️".
func (result Result) Label() string {
	icon := "⚔️"
	if (result.Role == RoleDefense) {
		icon = "🛡️"
	}

	return "можно +" + strconv.Itoa(result.Gain) + icon
}

type stats struct {
	practice, theory, cunning, wisdom int
}

func (s stats) score(role Role) int {
	if (role == RoleDefense) {
		return s.theory*2 + s.wisdom*3
	}

	return s.practice*2 + s.cunning*3
}

func profileStats(profile model.Profile) stats {
	return stats{int(profile.Practice), int(profile.Theory), int(profile.Cunning), int(profile.Wisdom)}
}

// Check compares profile against the best of:
//   - the inventory optimum: base stats (the last profile before the
//     inventory analysis, minus what was worn then) plus the best build
//     from the analyzed inventory, wearable at the profile's level/way;
//   - the best profile of the last 30 days.
//
// Whichever is higher is the optimum; a profile above both simply becomes
// the new 30-day best, so it's never flagged.
func Check(profile model.Profile, squad *model.Team, now time.Time) Result {
	role := RoleFor(profile, squad)
	if (role == RoleNone) {
		return Result{}
	}

	candidates := []int{}

	scoreSql := repository.AttackScoreSql
	if (role == RoleDefense) {
		scoreSql = repository.DefenseScoreSql
	}
	if best, err := repository.FindBestProfileSince(profile.UserId, now.Add(-historyWindow), scoreSql); (err == nil) {
		candidates = append(candidates, profileStats(*best).score(role))
	}

	if inventoryOptimum, ok := inventoryOptimum(profile, role); (ok) {
		candidates = append(candidates, inventoryOptimum)
	}

	return resultFor(profile, role, candidates)
}

// resultFor compares the profile against the best of the candidate
// optimums; a profile above all of them is its own optimum (gain 0).
func resultFor(profile model.Profile, role Role, candidates []int) Result {
	current := profileStats(profile).score(role)
	optimum := current
	for _, candidate := range candidates {
		optimum = max(optimum, candidate)
	}

	return Result{Role: role, Gain: optimum - current}
}

func inventoryOptimum(profile model.Profile, role Role) (int, bool) {
	analysis, err := repository.FindInventoryAnalysis(profile.UserId)
	if (err != nil || !analysis.HasBackpack) {
		return 0, false
	}

	baseProfile, err := repository.FindLastProfileBetween(profile.UserId, analysis.AnalyzedAt.Add(-baseProfileMaxAge), analysis.AnalyzedAt)
	if (err != nil) {
		return 0, false
	}

	stored, err := repository.FindInventoryItems(profile.UserId)
	if (err != nil) {
		return 0, false
	}

	items := make([]inventory.Item, 0, len(stored))
	for _, item := range stored {
		items = append(items, inventory.FromModel(item))
	}

	return inventoryOptimumFrom(*baseProfile, items, inventory.PlayerFrom(profile.Level, profile.Way), role), true
}

// inventoryOptimumFrom is the pure part of inventoryOptimum: base stats
// are baseProfile minus what the analyzed backpack wears; on top of them
// goes the best build player can wear from items.
func inventoryOptimumFrom(baseProfile model.Profile, items []inventory.Item, player inventory.Player, role Role) int {
	equipped := inventory.Equipped(items)
	base := profileStats(baseProfile)
	base.practice -= equipped.Practice
	base.theory -= equipped.Theory
	base.cunning -= equipped.Cunning
	base.wisdom -= equipped.Wisdom

	build := inventory.BestAttack(items, player)
	if (role == RoleDefense) {
		build = inventory.BestDefense(items, player)
	}

	optimum := base
	optimum.practice += build.Practice
	optimum.theory += build.Theory
	optimum.cunning += build.Cunning
	optimum.wisdom += build.Wisdom

	return optimum.score(role)
}
