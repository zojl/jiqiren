package messageHandler

import (
	"testing"

	"jiqiren/bot/inventory"
)

// A level-9 profile: no 💿/📄/💽/📑 resource lines yet, ⚡ status, a
// fraction ad above it.
const lowLevelProfile = `Huǒqiáng> Нет отряда? Подайте заявку в AH! Там помогут стать сильнее! Офис — отряд — подать заявку — AH

Не забывайте сливать деньги перед битвой!

⚡Ave Huoqiang (AH | 🈵Huǒqiáng)

💻: 9; 💡: 2007/2080
💵: 14; 📈: 11
🔘: 0; 💸: 4929318.138
🔥: 5/5; 🔋: 199%

📡: 14; 💾: 6
📱: 13; 🔎: 5

До 🛌: 41 ч. 33 мин.`

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
