package inventory

import (
	"strings"

	"jiqiren/bot/database/model"
)

// ToModel and FromModel convert between the parser's Item and the stored
// model.InventoryItem.
func ToModel(userId uint, item Item) model.InventoryItem {
	return model.InventoryItem{
		UserId: userId,
		Location: string(item.Location),
		Rarity: item.Rarity,
		IsServer: item.IsServer,
		Class: item.Class,
		Name: item.Name,
		Way: item.Way,
		Practice: item.Practice,
		Theory: item.Theory,
		Cunning: item.Cunning,
		Wisdom: item.Wisdom,
		FirmwareRarity: item.FirmwareRarity,
		FirmwareName: item.FirmwareName,
		RequiredLevel: item.RequiredLevel,
		Price: item.Price,
		Unsellable: item.Unsellable,
		Counters: strings.Join(item.Counters, "\n"),
		Description: item.Description,
		Raw: item.Raw,
	}
}

func FromModel(stored model.InventoryItem) Item {
	item := Item{
		Location: Location(stored.Location),
		Rarity: stored.Rarity,
		IsServer: stored.IsServer,
		Class: stored.Class,
		Name: stored.Name,
		Way: stored.Way,
		Practice: stored.Practice,
		Theory: stored.Theory,
		Cunning: stored.Cunning,
		Wisdom: stored.Wisdom,
		FirmwareRarity: stored.FirmwareRarity,
		FirmwareName: stored.FirmwareName,
		RequiredLevel: stored.RequiredLevel,
		Price: stored.Price,
		Unsellable: stored.Unsellable,
		Description: stored.Description,
		Raw: stored.Raw,
	}

	if (stored.Counters != "") {
		item.Counters = strings.Split(stored.Counters, "\n")
	}

	return item
}

// ForwardsFromDraft parses a draft's raw messages back into Forwards;
// messages that no longer parse (format drift) are skipped.
func ForwardsFromDraft(messages []model.InventoryDraftMessage) []Forward {
	forwards := make([]Forward, 0, len(messages))
	for _, stored := range messages {
		if message, ok := Parse(stored.Text); (ok) {
			forwards = append(forwards, Forward{Message: message, SentAt: stored.SentAt})
		}
	}

	return forwards
}
