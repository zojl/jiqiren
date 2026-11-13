package handlers

import (
	"net/http"
	"strconv"
	"time"

	"jiqiren/bot/database/repository"
	"jiqiren/bot/inventory"
	"jiqiren/bot/panel/templates"
)

type inventoryItemView struct {
	Title string
	Way string
	Practice string
	Theory string
	Cunning string
	Wisdom string
	Firmware string
	Price string
	// Unwearable is why the player can't equip it ("" if they can).
	Unwearable string
}

type inventoryLocationView struct {
	Label string
	Items []inventoryItemView
}

type buildSlotView struct {
	Class string
	Item string
	Location string
	// Swap marks a pick that isn't worn right now.
	Swap bool
}

type inventoryBuildView struct {
	Title string
	ScoreLabel string
	Slots []buildSlotView
	Stats string
	Score int
	HasEquipped bool
	EquippedScore int
	Gain string
}

type profileInventoryData struct {
	Tabs profileTabsData
	Analyzed bool
	AnalyzedAt string
	HasBackpack bool
	PlayerNote string
	Locations []inventoryLocationView
	Builds []inventoryBuildView
}

// ProfileInventory shows the user's last analyzed inventory (collected in
// the bot, see messageHandler/inventory.go) by location, plus the best
// attack and defense builds that can be put together from it.
func ProfileInventory(w http.ResponseWriter, r *http.Request) {
	user := requireOwnProfile(w, r)
	if (user == nil) {
		return
	}

	data := profileInventoryData{Tabs: profileTabsData{Active: "inventory"}}

	analysis, analysisErr := repository.FindInventoryAnalysis(user.Id)
	if (analysisErr != nil) {
		templates.Render(w, r, "profile_inventory", data)
		return
	}

	stored, itemsErr := repository.FindInventoryItems(user.Id)
	if (itemsErr != nil) {
		http.Error(w, "Не удалось загрузить инвентарь", http.StatusInternalServerError)
		return
	}

	items := make([]inventory.Item, 0, len(stored))
	for _, item := range stored {
		items = append(items, inventory.FromModel(item))
	}

	player := inventory.Player{}
	if profile, err := repository.FindLastProfileByVkId(user.VkId); (err == nil) {
		player = inventory.Player{Known: true, Level: profile.Level, Way: wayIcon(profile.Way)}
		data.PlayerNote = "с учётом уровня " + strconv.FormatUint(uint64(profile.Level), 10)
		if (player.Way != "") {
			data.PlayerNote += " и пути " + player.Way
		}
		data.PlayerNote += " из последнего профиля"
	} else {
		data.PlayerNote = "нет актуального профиля — уровень и путь не учитываются"
	}

	data.Analyzed = true
	data.AnalyzedAt = formatMskMoment(analysis.AnalyzedAt, time.Now())
	data.HasBackpack = analysis.HasBackpack

	for _, location := range inventory.Locations {
		view := inventoryLocationView{Label: location.Label()}
		for _, item := range items {
			if (item.Location == location) {
				view.Items = append(view.Items, buildInventoryItemView(item, player))
			}
		}
		data.Locations = append(data.Locations, view)
	}

	equipped := inventory.Equipped(items)
	data.Builds = []inventoryBuildView{
		buildInventoryBuildView("⚔️ лучший билд атаки", "⚔️ атака от вещей", inventory.BestAttack(items, player), equipped, analysis.HasBackpack, inventory.Build.AttackScore),
		buildInventoryBuildView("🛡️ лучший билд защиты", "🛡️ защита от вещей", inventory.BestDefense(items, player), equipped, analysis.HasBackpack, inventory.Build.DefenseScore),
	}

	templates.Render(w, r, "profile_inventory", data)
}

func buildInventoryItemView(item inventory.Item, player inventory.Player) inventoryItemView {
	view := inventoryItemView{
		Title: item.Title(),
		Way: item.Way,
		Practice: signedOrBlank(item.Practice),
		Theory: signedOrBlank(item.Theory),
		Cunning: signedOrBlank(item.Cunning),
		Wisdom: signedOrBlank(item.Wisdom),
		Firmware: item.Firmware(),
	}

	switch {
	case item.Price > 0:
		view.Price = groupThousands(uint64(item.Price)) + "💵"
	case item.Unsellable:
		view.Price = "❌"
	}

	if wearable, reason := player.CanWear(item); (!wearable) {
		view.Unwearable = reason
	}

	return view
}

func buildInventoryBuildView(title string, scoreLabel string, build inventory.Build, equipped inventory.Build, hasEquipped bool, score func(inventory.Build) int) inventoryBuildView {
	view := inventoryBuildView{
		Title: title,
		ScoreLabel: scoreLabel,
		Score: score(build),
		HasEquipped: hasEquipped,
		EquippedScore: score(equipped),
		Stats: "📡" + strconv.Itoa(build.Practice) + " 💾" + strconv.Itoa(build.Theory) + " 📱" + strconv.Itoa(build.Cunning) + " 🔎" + strconv.Itoa(build.Wisdom),
	}

	if (hasEquipped) {
		gain := view.Score - view.EquippedScore
		view.Gain = strconv.Itoa(gain)
		if (gain > 0) {
			view.Gain = "+" + view.Gain
		}
	}

	for _, slot := range build.Slots {
		slotView := buildSlotView{Class: slot.Class.Icon + " " + slot.Class.Name, Item: "—"}
		if (slot.Item != nil) {
			slotView.Item = slot.Item.Title() + " (" + slot.Item.StatsLabel() + ")"
			slotView.Location = slot.Item.Location.Label()
			slotView.Swap = slot.Item.Location != inventory.LocationBackpack
		}
		view.Slots = append(view.Slots, slotView)
	}

	return view
}

func signedOrBlank(value int) string {
	if (value == 0) {
		return ""
	}

	if (value > 0) {
		return "+" + strconv.Itoa(value)
	}

	return strconv.Itoa(value)
}
