package inventory

import (
	"testing"
	"time"
)

const sampleSummary = `🎒Рюкзак:
👨🏻‍💻🖥PDP-1 (+56📡, +39💾, -1📱)
👨🏻‍💻💻Домашний сервак (+68📡, +43💾, +30📱)
🔹👨🏻‍💻🔀WireGuard сервер (+27📡, +1💾, +35📱, +4🔎)
👨🏻‍💻⌚NTP server (+40📡, +54💾, +3📱)
🔹👨🏻‍💻📳P2P нода (+43📡, +66💾, +5📱, +32🔎)
📵NetShark (⬛, +8📡, +6💾, +6📱, +13🔎)

🏡Дом: 2 предмета
🏭Склад: 8/10
🗄Серверная: 1/1

*⃣Комплектуха: 14 видов; 667 шт.
📦Коробки: 2 вида; 309 шт.
🎁Подарки: 2 вида; 9 шт.
🎁Подарок с 10kkk VKCoins (Код: 38BB) - 8 шт.
🎁Подарок с 50 биткоинами (Код: 6205) - 1 шт.`

// The same summary screen in its other form: ❌/price inside the stats,
// stats wrapped onto their own line.
const sampleSummaryInline = `🎒Рюкзак:
👨🏻‍💻🖥PDP-1
(+56📡, +39💾, -1📱, ❌)
🔰: 🔺Редкая
📵NetShark
(⬛, +8📡, +6💾, +6📱, +13🔎, 10260💵)
🔰: 🔺Особая

🏡Дом: 2 предмета`

const sampleBackpack = `🎒Рюкзак:

👨🏻‍💻🖥PDP-1 (+56📡, +39💾, -1📱)
🔰: 🔺Редкая
Разработанный в 1960 году PDP-1 оказал огромное влияние на хакерскую культуру.
❌Невозможно продать

👨🏻‍💻💻Домашний сервак (+68📡, +43💾, +30📱)
🔰: 🔻Простая
Ноутбуки часто используются в качестве домашних серверов.
❌Невозможно продать

🔹👨🏻‍💻🔀WireGuard сервер (+27📡, +1💾, +35📱, +4🔎)
🔰: 🔸Отличная
WireGuard - протокол VPN.
❌Невозможно продать

👨🏻‍💻⌚NTP server (+40📡, +54💾, +3📱)
🔰: 🔹Оптимизированная
Протокол NTP был разработан Дэвидом Л. Миллсом в 1985 году.
❌Невозможно продать

🔹👨🏻‍💻📳P2P нода (+43📡, +66💾, +5📱, +32🔎)
🔰: 🔸Красная
В P2P сетях устройства одновременно выполняют функции серверов и клиентов.
❌Невозможно продать

📵NetShark (⬛, +8📡, +6💾, +6📱, +13🔎)
🔰: 🔺Особая
 - Транзакций перехвачено: 360
Цена: 10260💵

Ты можешь оставить предметы 🏡дома`

const sampleHome = `🏡Дом:

1) ⌚NeverFit (⬛, +25📡, +30💾, -1📱)
🔰: 🔺Редкая
Цена: 4783💵

2) 🔹👨🏻‍💻💻Портативный сервак (+68📡, +43💾, +35📱)
🔰: 🔸Недавняя
Ноутбуки часто используются в качестве портативных серваков.
Требуется 35 уровень
❌Невозможно продать

Страница 1 из 1`

const sampleStorage = `🏭Склад:

1) ▪⌚🏷Cамодельные часы (+3💾)
 - Побед в битвах: 0
Цена: 158💵

2) 📳FlyParrot (⬛, +16📡, +37💾, +1📱, +15🔎)
Цена: 3925💵

3) ▫💻Ноут CTFника (+41📡, +13💾, +15📱)
Этот девайс получают опытные игроки арены
❌Невозможно продать

4) ⌚WatchPod (⬛, +22📡, +29💾)
Цена: 4333💵

5) 🔸📳iSmart (+12📡, +42💾, +16🔎)
 - Побед в битвах: 0
Выдано 25 выдающимся участникам применения 0day 🌅WaterWalk.
❌Невозможно продать

6) ▫📳iSmart (+12📡, +42💾, +16🔎)
 - Побед в битвах: 0
Выдано 25 выдающимся участникам применения 0day 🌅WaterWalk.
❌Невозможно продать

7) ▫📵Тындекс Стоп (+2📡, +10🔎)
 - Отбитых грабежей: 0
Выдано 10 выдающимся участникам применения 0day 🌆ShadowGrid
❌Невозможно продать

8) ▫🔀Doodle Pass (+2💾, +10📱)
 - Удачных грабежей: 0
Выдано 10 выдающимся участникам применения 0day 🏙MorningHaze
❌Невозможно продать

Страница 1 из 2`

const sampleServer = `🗄Серверная:

1) 🔹👨🏻‍💻📵NGFW (+27💾, +2📱, +38🔎)
🔰: 🔸Полезная
Файрволл нового поколения обеспечивает глубокую проверку трафика (DPI).
❌Невозможно продать

Страница 1 из 1`

func mustParse(t *testing.T, text string, kind Kind) Message {
	t.Helper()
	message, ok := Parse(text)
	if (!ok || message.Kind != kind) {
		t.Fatalf("expected %s, got %s (ok=%v)", kind, message.Kind, ok)
	}

	return message
}

func TestDetectKind(t *testing.T) {
	mustParse(t, sampleSummary, KindSummary)
	mustParse(t, sampleSummaryInline, KindSummary)
	mustParse(t, sampleBackpack, KindBackpack)
	mustParse(t, sampleHome, KindHome)
	mustParse(t, sampleStorage, KindStorage)
	mustParse(t, sampleServer, KindServer)

	if _, ok := DetectKind("🥷 c0nd0r (DD | ⬛Huǒqiáng)"); (ok) {
		t.Fatal("profile detected as inventory")
	}
}

func TestParseSummary(t *testing.T) {
	message := mustParse(t, sampleSummary, KindSummary)
	if (len(message.Items) != 6) {
		t.Fatalf("expected 6 items, got %d: %+v", len(message.Items), message.Items)
	}

	if (message.ExpectedHome != 2 || message.ExpectedStorage != 8 || message.ExpectedServer != 1) {
		t.Fatalf("expected counts: %+v", message)
	}

	pdp := message.Items[0]
	if (!pdp.IsServer || pdp.Class != "🖥" || pdp.Name != "PDP-1" || pdp.Practice != 56 || pdp.Theory != 39 || pdp.Cunning != -1 || pdp.Location != LocationBackpack) {
		t.Fatalf("PDP-1: %+v", pdp)
	}

	wireguard := message.Items[2]
	if (wireguard.Rarity != "🔹" || !wireguard.IsServer || wireguard.Class != "🔀" || wireguard.Name != "WireGuard сервер" || wireguard.Wisdom != 4) {
		t.Fatalf("WireGuard: %+v", wireguard)
	}

	netshark := message.Items[5]
	if (netshark.Way != WayBlack || netshark.IsServer || netshark.Class != "📵" || netshark.Wisdom != 13) {
		t.Fatalf("NetShark: %+v", netshark)
	}
}

func TestParseSummaryInline(t *testing.T) {
	message := mustParse(t, sampleSummaryInline, KindSummary)
	if (len(message.Items) != 2) {
		t.Fatalf("expected 2 items, got %+v", message.Items)
	}

	pdp, netshark := message.Items[0], message.Items[1]
	if (pdp.Name != "PDP-1" || !pdp.Unsellable || pdp.FirmwareName != "Редкая" || pdp.FirmwareRarity != "🔺") {
		t.Fatalf("PDP-1: %+v", pdp)
	}

	if (netshark.Price != 10260 || netshark.Way != WayBlack || netshark.FirmwareName != "Особая") {
		t.Fatalf("NetShark: %+v", netshark)
	}
}

func TestParseDetails(t *testing.T) {
	backpack := mustParse(t, sampleBackpack, KindBackpack)
	if (len(backpack.Items) != 6) {
		t.Fatalf("expected 6 items, got %d", len(backpack.Items))
	}

	netshark := backpack.Items[5]
	if (netshark.FirmwareRarity != "🔺" || netshark.FirmwareName != "Особая" || netshark.Price != 10260 || netshark.Unsellable ||
		len(netshark.Counters) != 1 || netshark.Counters[0] != "Транзакций перехвачено: 360") {
		t.Fatalf("NetShark: %+v", netshark)
	}

	if (backpack.Items[0].Description == "" || !backpack.Items[0].Unsellable) {
		t.Fatalf("PDP-1: %+v", backpack.Items[0])
	}

	home := mustParse(t, sampleHome, KindHome)
	if (len(home.Items) != 2 || home.Page != 1 || home.TotalPages != 1) {
		t.Fatalf("home: %+v", home)
	}

	if (home.Items[1].RequiredLevel != 35 || home.Items[1].Location != LocationHome || home.Items[1].Name != "Портативный сервак") {
		t.Fatalf("Портативный сервак: %+v", home.Items[1])
	}

	storage := mustParse(t, sampleStorage, KindStorage)
	if (len(storage.Items) != 8 || storage.TotalPages != 2) {
		t.Fatalf("storage: %d items, %d pages", len(storage.Items), storage.TotalPages)
	}

	watch := storage.Items[0]
	if (watch.Rarity != "▪" || watch.Class != "⌚" || watch.Name != "🏷Cамодельные часы" || watch.Theory != 3) {
		t.Fatalf("watch: %+v", watch)
	}

	server := mustParse(t, sampleServer, KindServer)
	if (len(server.Items) != 1 || server.Items[0].Description == "" || server.Items[0].Location != LocationServer) {
		t.Fatalf("server: %+v", server)
	}
}

func TestMerge(t *testing.T) {
	start := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	summary := mustParse(t, sampleSummary, KindSummary)
	backpack := mustParse(t, sampleBackpack, KindBackpack)
	storage := mustParse(t, sampleStorage, KindStorage)

	// A NetShark copy lying at home, identical to the equipped one — one item.
	home := mustParse(t, "🏡Дом:\n\n1) 📵NetShark (⬛, +8📡, +6💾, +6📱, +13🔎)\nЦена: 10260💵\n\nСтраница 1 из 1", KindHome)

	items := Merge([]Forward{
		{Message: backpack, SentAt: start.Add(2 * time.Minute)},
		{Message: summary, SentAt: start.Add(1 * time.Minute)},
		{Message: storage, SentAt: start.Add(3 * time.Minute)},
		{Message: storage, SentAt: start.Add(4 * time.Minute)},
		{Message: home, SentAt: start.Add(5 * time.Minute)},
	})

	// 6 equipped + 8 in storage (both iSmarts differ by rarity) — the
	// summary merges into the backpack, the repeated storage page into
	// itself, and the home NetShark into the equipped one.
	if (len(items) != 14) {
		for _, item := range items {
			t.Log(item.Location, item.Line())
		}
		t.Fatalf("expected 14 items, got %d", len(items))
	}

	for _, item := range items {
		if (item.Name == "PDP-1" && item.FirmwareName != "Редкая") {
			t.Fatalf("PDP-1 lost its firmware: %+v", item)
		}

		if (item.Name == "NetShark" && (item.Location != LocationHome || item.FirmwareName != "Особая")) {
			t.Fatalf("NetShark should have moved home, keeping firmware: %+v", item)
		}
	}

	twoFirmwares := Merge([]Forward{{Message: mustParse(t, "🏡Дом:\n\n1) 📵A (+1📡)\n🔰: 🔸X\n\n2) 📵A (+1📡)\n🔰: 🔸Y\n\n3) 📵A (+1📡)", KindHome)}})
	if (len(twoFirmwares) != 2) {
		t.Fatalf("different firmware must stay apart: %+v", twoFirmwares)
	}
}

func TestSummarize(t *testing.T) {
	progress := Summarize([]Forward{
		{Message: mustParse(t, sampleSummary, KindSummary)},
		{Message: mustParse(t, sampleStorage, KindStorage)},
	})

	if (!progress.HasBackpack || progress.Count[LocationBackpack] != 6 || progress.Count[LocationStorage] != 8) {
		t.Fatalf("progress: %+v", progress)
	}

	if (progress.Expected[LocationHome] != 2 || len(progress.MissingPages[LocationStorage]) != 1 || progress.MissingPages[LocationStorage][0] != 2) {
		t.Fatalf("progress: %+v", progress)
	}
}

func TestBuilds(t *testing.T) {
	items := Merge([]Forward{
		{Message: mustParse(t, sampleBackpack, KindBackpack)},
		{Message: mustParse(t, sampleHome, KindHome)},
		{Message: mustParse(t, sampleStorage, KindStorage)},
		{Message: mustParse(t, sampleServer, KindServer)},
	})

	pick := func(build Build, class string) string {
		for _, slot := range build.Slots {
			if (slot.Class.Icon == class && slot.Item != nil) {
				return slot.Item.Name
			}
		}
		return ""
	}

	whitehat := Player{Known: true, Level: 30, Way: "⬜"}
	attack := BestAttack(items, whitehat)
	// Портативный сервак (68📡 35📱) beats Домашний сервак (68📡 30📱) — but
	// needs level 35, so at level 30 the equipped one stays.
	if (pick(attack, "💻") != "Домашний сервак") {
		t.Fatalf("level gate: got %q", pick(attack, "💻"))
	}

	// ⬛-only NetShark isn't wearable for a whitehat: of the firewalls
	// left, NGFW (+2📱 → attack 6) beats Тындекс Стоп (+2📡 → attack 4).
	if (pick(attack, "📵") != "NGFW") {
		t.Fatalf("way gate: got %q", pick(attack, "📵"))
	}

	grey := Player{Known: true, Level: 40, Way: "🔲"}
	attack = BestAttack(items, grey)
	if (pick(attack, "💻") != "Портативный сервак" || pick(attack, "📵") != "NetShark") {
		t.Fatalf("grey player: 💻 %q, 📵 %q", pick(attack, "💻"), pick(attack, "📵"))
	}

	defense := BestDefense(items, grey)
	// P2P нода: 66💾 32🔎 → 228 vs iSmart 42💾 16🔎 → 132 vs FlyParrot 37💾 15🔎 → 119.
	if (pick(defense, "📳") != "P2P нода" || pick(defense, "📵") != "NGFW") {
		t.Fatalf("defense: 📳 %q, 📵 %q", pick(defense, "📳"), pick(defense, "📵"))
	}

	equipped := Equipped(items)
	if (equipped.Practice != 56+68+27+40+43+8) {
		t.Fatalf("equipped practice: %d", equipped.Practice)
	}
}
