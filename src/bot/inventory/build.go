package inventory

import "strconv"

// Player is what decides which items someone can wear. Known is false when
// there's no profile to go by — then every item counts as wearable.
type Player struct {
	Known bool
	Level uint
	// Way is the profile's way icon: ⬛, ⬜, 🔲, 🔳 or "".
	Way string
}

// CanWear reports whether the player may equip item, and if not, why.
// ⬛/⬜ items are for blackhats/whitehats only; players on 🔲/🔳 may wear
// both, and a player with no way at all neither.
func (player Player) CanWear(item Item) (bool, string) {
	if (!player.Known) {
		return true, ""
	}

	if (item.RequiredLevel > 0 && player.Level < item.RequiredLevel) {
		return false, "нужен уровень " + strconv.Itoa(int(item.RequiredLevel))
	}

	if (item.Way != "" && player.Way != "🔲" && player.Way != "🔳" && player.Way != item.Way) {
		if (item.Way == WayBlack) {
			return false, "только для ⬛"
		}
		return false, "только для ⬜"
	}

	return true, ""
}

// Slot is one class's pick in a build; Item is nil when wearing nothing of
// that class is best.
type Slot struct {
	Class Class
	Item *Item
}

type Build struct {
	Slots []Slot
	Practice int
	Theory int
	Cunning int
	Wisdom int
}

func (build Build) AttackScore() int {
	return build.Practice*2 + build.Cunning*3
}

func (build Build) DefenseScore() int {
	return build.Theory*2 + build.Wisdom*3
}

// BestAttack and BestDefense pick at most one wearable item per class to
// maximize ⚔️ attack (📡×2 + 📱×3) or 🛡️ defense (💾×2 + 🔎×3). Classes are
// independent, so the best item per class is the best build overall; a
// class is left empty if no item adds to the score.
func BestAttack(items []Item, player Player) Build {
	return best(items, player, Item.AttackScore)
}

func BestDefense(items []Item, player Player) Build {
	return best(items, player, Item.DefenseScore)
}

// Equipped is the build currently worn — the backpack's items.
func Equipped(items []Item) Build {
	build := Build{}
	for _, class := range Classes {
		slot := Slot{Class: class}
		for i := range items {
			if (items[i].Class == class.Icon && items[i].Location == LocationBackpack) {
				slot.Item = &items[i]
				build.add(items[i])
				break
			}
		}
		build.Slots = append(build.Slots, slot)
	}

	return build
}

func best(items []Item, player Player, score func(Item) int) Build {
	build := Build{}
	for _, class := range Classes {
		slot := Slot{Class: class}
		for i := range items {
			item := items[i]
			if (item.Class != class.Icon || score(item) <= 0) {
				continue
			}

			if wearable, _ := player.CanWear(item); (!wearable) {
				continue
			}

			if (slot.Item == nil || better(item, *slot.Item, score)) {
				slot.Item = &items[i]
			}
		}

		if (slot.Item != nil) {
			build.add(*slot.Item)
		}
		build.Slots = append(build.Slots, slot)
	}

	return build
}

// better breaks score ties in favor of an already equipped item — one
// less thing to swap.
func better(candidate Item, current Item, score func(Item) int) bool {
	if (score(candidate) != score(current)) {
		return score(candidate) > score(current)
	}

	return candidate.Location == LocationBackpack && current.Location != LocationBackpack
}

func (build *Build) add(item Item) {
	build.Practice += item.Practice
	build.Theory += item.Theory
	build.Cunning += item.Cunning
	build.Wisdom += item.Wisdom
}
