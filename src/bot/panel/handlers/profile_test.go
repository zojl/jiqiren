package handlers

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/inventory"
	"jiqiren/bot/panel/templates"
)

func TestParseProfileHistoryQuery(t *testing.T) {
	values, _ := url.ParseQuery("from=2026-01-02&to=not-a-date&hidden=1&page=3")
	query := parseProfileHistoryQuery(values)
	if (query.From != "2026-01-02" || query.To != "" || !query.ShowHidden || query.Page != 3) {
		t.Fatalf("parsed: %+v", query)
	}

	for _, raw := range []string{"page=0", "page=-5", "page=abc", ""} {
		values, _ := url.ParseQuery(raw)
		if got := parseProfileHistoryQuery(values).Page; (got != 1) {
			t.Errorf("%q → page %d, want 1", raw, got)
		}
	}

	values, _ = url.ParseQuery("from=yesterday&to=2026-01-05")
	if query := parseProfileHistoryQuery(values); (query.From != "" || query.To != "2026-01-05") {
		t.Fatalf("an invalid «from» must be dropped too: %+v", query)
	}

	values, _ = url.ParseQuery("hidden=true")
	if (parseProfileHistoryQuery(values).ShowHidden) {
		t.Fatal(`only hidden=1 shows hidden profiles`)
	}
}

func TestProfileHistoryHref(t *testing.T) {
	query := profileHistoryQuery{From: "2026-01-02", ShowHidden: true, Page: 1}

	if got := query.href(1); (got != "/panel/profile/history?from=2026-01-02&hidden=1") {
		t.Fatalf("page 1 has no page param: %s", got)
	}

	if got := query.href(2); (got != "/panel/profile/history?from=2026-01-02&hidden=1&page=2") {
		t.Fatal(got)
	}

	if got := (profileHistoryQuery{Page: 1}).href(1); (got != "/panel/profile/history") {
		t.Fatal(got)
	}
}

// The hide/restore forms post back the current query, and the redirect is
// rebuilt from it — so whatever is posted, it can only ever lead back into
// the history page, never to another URL.
func TestProfileHistoryRedirectCantBeHijacked(t *testing.T) {
	for _, posted := range []string{
		"https://evil.example/",
		"//evil.example",
		"from=2026-01-02&next=https://evil.example",
		"page=2%0d%0aLocation:%20https://evil.example",
		"/panel/managers",
	} {
		values, _ := url.ParseQuery(posted)
		query := parseProfileHistoryQuery(values)
		href := query.href(query.Page)
		if (!strings.HasPrefix(href, "/panel/profile/history")) {
			t.Errorf("%q → %q", posted, href)
		}
	}
}

func TestProfileHistoryFilter(t *testing.T) {
	filter := profileHistoryQuery{From: "2026-01-02", To: "2026-01-05", ShowHidden: true}.filter()

	if (!filter.IncludeHidden || filter.From == nil || filter.To == nil) {
		t.Fatalf("filter: %+v", filter)
	}

	if (!filter.From.Equal(time.Date(2026, 1, 2, 0, 0, 0, 0, moscow))) {
		t.Fatalf("from is the start of the MSK day: %v", filter.From)
	}

	// "to" is inclusive in the form, so the bound is the next MSK midnight.
	if (!filter.To.Equal(time.Date(2026, 1, 6, 0, 0, 0, 0, moscow))) {
		t.Fatalf("to: %v", filter.To)
	}

	empty := profileHistoryQuery{}.filter()
	if (empty.From != nil || empty.To != nil || empty.IncludeHidden) {
		t.Fatalf("empty filter: %+v", empty)
	}
}

func TestProfileHeadline(t *testing.T) {
	profile := model.Profile{Lead: 2, Way: 3, Status: "💡", Username: "Testoviy", Level: 39}
	if got := profileHeadline(profile, "DD", "🈵"); (got != "🔸DD|🔲🈵💡Testoviy (39)") {
		t.Fatal(got)
	}

	profile.Level, profile.Prestige, profile.Lead = 40, 3, 0
	if got := profileHeadline(profile, "", "🈵"); (got != "🔲🈵💡Testoviy (40💎3)") {
		t.Fatal(got)
	}
}

func TestNumberFormatting(t *testing.T) {
	cases := map[uint64]string{0: "0", 999: "999", 1000: "1 000", 1234567: "1 234 567"}
	for value, want := range cases {
		if got := groupThousands(value); (got != want) {
			t.Errorf("groupThousands(%d) = %q, want %q", value, got, want)
		}
	}

	if (signedOrBlank(0) != "" || signedOrBlank(5) != "+5" || signedOrBlank(-1) != "-1") {
		t.Fatal("signedOrBlank")
	}
}

func TestFormatMskMoment(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, moscow)
	if got := formatMskMoment(time.Date(2026, 11, 13, 9, 5, 0, 0, moscow), now); (got != "09:05") {
		t.Fatalf("today: %s", got)
	}

	if got := formatMskMoment(time.Date(2026, 11, 12, 23, 59, 0, 0, moscow), now); (got != "12.11 23:59") {
		t.Fatalf("another day: %s", got)
	}
}

func TestBuildInventoryBuildView(t *testing.T) {
	worn := inventory.Item{Location: inventory.LocationBackpack, Class: "🖥", Name: "Old", Practice: 10}
	better := inventory.Item{Location: inventory.LocationStorage, Class: "🖥", Name: "New", Practice: 20}
	items := []inventory.Item{worn, better}

	view := buildInventoryBuildView("⚔️", "⚔️ атака", inventory.BestAttack(items, inventory.Player{}), inventory.Equipped(items), true, inventory.Build.AttackScore)
	if (view.Score != 40 || view.EquippedScore != 20 || view.Gain != "+20") {
		t.Fatalf("scores: %+v", view)
	}

	if (!view.Slots[0].Swap || view.Slots[0].Location != inventory.LocationStorage.Label() || view.Slots[1].Item != "—") {
		t.Fatalf("slots: %+v", view.Slots)
	}

	if noBackpack := buildInventoryBuildView("⚔️", "⚔️", inventory.BestAttack(items, inventory.Player{}), inventory.Equipped(items), false, inventory.Build.AttackScore); (noBackpack.Gain != "") {
		t.Fatal("no gain without a known backpack")
	}
}

// Rendering needs no database: nav and theme fall back to a logged-out
// view when there's no viewer in the request.
func render(t *testing.T, name string, data interface{}) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	templates.Render(recorder, httptest.NewRequest("GET", "/profile", nil), name, data)

	body := recorder.Body.String()
	if (recorder.Code != 200 || strings.Contains(body, "Ошибка шаблона")) {
		t.Fatalf("%s: %d %s", name, recorder.Code, body)
	}

	return body
}

func TestProfileTemplates(t *testing.T) {
	history := render(t, "profile_history", profileHistoryData{
		Tabs: profileTabsData{Active: "history"},
		Rows: []profileHistoryRow{{Id: 5, Headline: "🔸DD|🔲🈵💡Testoviy (39)", Hidden: true}, {Id: 6}},
		Page: 2, TotalPages: 3, Total: 25, Query: "from=2026-01-02&page=2",
		NewerHref: "/panel/profile/history", OlderHref: "/panel/profile/history?page=3",
	})
	for _, want := range []string{
		`/panel/profile/history/5/restore`, `/panel/profile/history/6/hide`,
		`class="profile-hidden"`, `name="query" value="from=2026-01-02&amp;page=2"`, "стр. 2 из 3",
		`<a href="/panel/profile/history" class="active">история</a>`,
	} {
		if (!strings.Contains(history, want)) {
			t.Errorf("history page lacks %q", want)
		}
	}

	render(t, "profile_empty", profileEmptyData{Tabs: profileTabsData{Active: "summary"}})
	render(t, "profile_inventory", profileInventoryData{Tabs: profileTabsData{Active: "inventory"}})

	own := render(t, "player_page", playerPageData{Tabs: &profileTabsData{Active: "summary"}, ExperienceChartUrl: "/panel/profile/experience"})
	if (!strings.Contains(own, "profile-tabs") || !strings.Contains(own, "exp-chart-block")) {
		t.Fatal("«Мой профиль» shows tabs and the chart")
	}

	managerView := render(t, "player_page", playerPageData{BackHref: "/panel", BackLabel: "←"})
	if (strings.Contains(managerView, "profile-tabs") || strings.Contains(managerView, "exp-chart-block")) {
		t.Fatal("a manager's player page must not get the player's own tabs or chart")
	}
}
