package messageHandler

import (
	"testing"

	"jiqiren/bot/inventory"
)

// An anonymized level-9 profile (see CLAUDE.md): no 💿/📄/💽/📑 resource
// lines yet, ⚡ status, a fraction ad above it.
const lowLevelProfile = `Huǒqiáng> Нет отряда? Подайте заявку в AH! Там помогут стать сильнее! Офис — отряд — подать заявку — AH

Не забывайте сливать деньги перед битвой!

⚡Testoviy Igrok (AH | 🈵Huǒqiáng)

💻: 9; 💡: 1984/2080
💵: 17; 📈: 9
🔘: 0; 💸: 4810233.517
🔥: 5/5; 🔋: 187%

📡: 12; 💾: 7
📱: 11; 🔎: 4

До 🛌: 39 ч. 12 мин.`

func TestIsProfileMessage(t *testing.T) {
	if (!isProfileMessage(lowLevelProfile)) {
		t.Fatal("a low-level profile without resource lines must count as a profile")
	}

	summary := "Инвентарь:\n\n🎒Рюкзак:\n👨🏻‍💻💻Домашний сервак (+68📡, +43💾, +30📱, +4🔎)\n\n🏡Дом: 2 предмета"
	if (isProfileMessage(summary)) {
		t.Fatal("an inventory screen must not count as a profile")
	}

	if _, isInventory := inventory.DetectKind(lowLevelProfile); (isInventory) {
		t.Fatal("a profile must not be detected as an inventory screen")
	}
}

func TestMessageKindDetectors(t *testing.T) {
	report := battleReportStart + "12.11 …\n" + battleReportEnd
	if (!isBattleReport(report) || isBattleReport(battleReportStart+"…") || isBattleReport("…"+battleReportEnd)) {
		t.Fatal("a battle report is framed by both its start and its end")
	}

	if (!isTeamProjectResult(teamProjectResultPrefix1+" …") || !isTeamProjectResult(teamProjectResultPrefix2+" …") || isTeamProjectResult("…")) {
		t.Fatal("team project result prefixes")
	}

	for _, cmd := range []string{cmdInventoryStart, cmdInventoryFinish, cmdInventoryContinue, cmdInventoryApply, cmdInventoryCancel} {
		if (!isInventoryCommand(cmd)) {
			t.Errorf("%s is an inventory command", cmd)
		}
	}

	if (isInventoryCommand("profile") || isInventoryCommand("")) {
		t.Fatal("not inventory commands")
	}
}
