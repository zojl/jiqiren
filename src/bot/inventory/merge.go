package inventory

import (
	"sort"
	"time"
)

// Forward is one collected message together with when it was sent in the
// game chat (the forward's own date, not when it reached the bot).
type Forward struct {
	Message Message
	SentAt time.Time
}

// Merge combines every collected message into one item set, oldest
// message first:
//   - fully identical items are one item, even on the same page or in
//     different locations — the player can't tell copies apart when
//     putting a build together anyway, and items get moved between
//     home/storage while they're being collected;
//   - an item without firmware merges into an otherwise identical one with
//     firmware (the summary's terse lines into the detailed backpack's);
//     two items with different known firmware stay separate;
//   - on a merge, the newer message decides the location (the item was
//     moved since) and fills in any detail the older one lacked.
func Merge(forwards []Forward) []Item {
	sorted := make([]Forward, len(forwards))
	copy(sorted, forwards)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].SentAt.Before(sorted[j].SentAt) })

	items := []Item{}
	for _, forward := range sorted {
		for _, item := range forward.Message.Items {
			if index := findSame(items, item); (index >= 0) {
				items[index] = mergeItems(items[index], item)
				continue
			}

			items = append(items, item)
		}
	}

	return items
}

func findSame(items []Item, item Item) int {
	key := item.key()
	for index, existing := range items {
		if (existing.key() != key) {
			continue
		}

		if (existing.FirmwareName != "" && item.FirmwareName != "" && existing.Firmware() != item.Firmware()) {
			continue
		}

		return index
	}

	return -1
}

// mergeItems folds newer into older: newer's location wins, and each
// detail is taken from whichever side actually has it (newer first).
func mergeItems(older Item, newer Item) Item {
	merged := newer

	if (merged.FirmwareName == "") {
		merged.FirmwareRarity = older.FirmwareRarity
		merged.FirmwareName = older.FirmwareName
	}

	if (merged.RequiredLevel == 0) {
		merged.RequiredLevel = older.RequiredLevel
	}

	if (merged.Price == 0) {
		merged.Price = older.Price
	}

	merged.Unsellable = older.Unsellable || newer.Unsellable

	if (len(merged.Counters) == 0) {
		merged.Counters = older.Counters
	}

	if (merged.Description == "") {
		merged.Description = older.Description
	}

	if (len(older.Raw) > len(merged.Raw)) {
		merged.Raw = older.Raw
	}

	return merged
}

// Progress summarizes what's been collected so far, for the bot's replies.
type Progress struct {
	HasBackpack bool
	// Count is the number of items per location as listed by the game
	// (before merging), for comparison with Expected.
	Count map[Location]int
	// Expected per location from the latest summary message, -1 if none.
	Expected map[Location]int
	// MissingPages per location, for paged screens with pages not yet sent.
	MissingPages map[Location][]int
}

// Summarize computes Progress. Pages are keyed by (kind, page) upstream,
// so each page is counted once.
func Summarize(forwards []Forward) Progress {
	progress := Progress{
		Count: map[Location]int{},
		Expected: map[Location]int{LocationHome: -1, LocationStorage: -1, LocationServer: -1},
		MissingPages: map[Location][]int{},
	}

	var latestSummary *Forward
	backpackCount, summaryCount := 0, 0
	hasBackpack, hasSummary := false, false
	pages := map[Kind]map[int]bool{}
	totalPages := map[Kind]int{}

	for i, forward := range forwards {
		message := forward.Message
		switch (message.Kind) {
		case KindSummary:
			hasSummary = true
			summaryCount = len(message.Items)
			if (latestSummary == nil || forward.SentAt.After(latestSummary.SentAt)) {
				latestSummary = &forwards[i]
			}
		case KindBackpack:
			hasBackpack = true
			backpackCount = len(message.Items)
		default:
			progress.Count[message.Kind.Location()] += len(message.Items)
			if (pages[message.Kind] == nil) {
				pages[message.Kind] = map[int]bool{}
			}
			pages[message.Kind][message.Page] = true
			if (message.TotalPages > totalPages[message.Kind]) {
				totalPages[message.Kind] = message.TotalPages
			}
		}
	}

	progress.HasBackpack = hasBackpack || hasSummary
	if (hasBackpack) {
		progress.Count[LocationBackpack] = backpackCount
	} else {
		progress.Count[LocationBackpack] = summaryCount
	}

	if (latestSummary != nil) {
		progress.Expected[LocationHome] = latestSummary.Message.ExpectedHome
		progress.Expected[LocationStorage] = latestSummary.Message.ExpectedStorage
		progress.Expected[LocationServer] = latestSummary.Message.ExpectedServer
	}

	for kind, total := range totalPages {
		for page := 1; page <= total; page++ {
			if (!pages[kind][page]) {
				progress.MissingPages[kind.Location()] = append(progress.MissingPages[kind.Location()], page)
			}
		}
	}

	return progress
}
