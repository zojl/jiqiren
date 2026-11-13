package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/leveling"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/go-chi/chi/v5"
)

// "Мой профиль" — the logged-in user's own section, split into tabs that
// are each a plain server-rendered page. Which user's data to show comes
// solely from the session — no route here takes a user/vk id — so none of
// them can be used to look at anyone else. Nothing here is audited: the
// audit log records managers acting on fraction data, not players looking
// at (or tidying up) their own.

const profileHistoryPageSize = 10
const experienceChartWindow = 1 // years

type profileTabsData struct {
	Active string
}

type profileEmptyData struct {
	Tabs profileTabsData
}

type profileInventoryData struct {
	Tabs profileTabsData
}

// requireOwnProfile returns the logged-in user if they may use "Мой
// профиль" at all, writing the error response itself otherwise.
func requireOwnProfile(w http.ResponseWriter, r *http.Request) *model.User {
	user := auth.UserFromContext(r)
	if (user == nil || !auth.PermissionsFromRequest(r).CanViewOwnProfile()) {
		http.Error(w, "Профиль не найден — отправьте его боту", http.StatusNotFound)
		return nil
	}

	return user
}

// MyProfile is the summary tab: the latest visible profile plus best
// attack/defense snapshots, same as PlayerPage shows managers, and the
// experience chart (loaded separately, see ProfileExperienceChart).
func MyProfile(w http.ResponseWriter, r *http.Request) {
	user := requireOwnProfile(w, r)
	if (user == nil) {
		return
	}

	tabs := profileTabsData{Active: "summary"}

	// Every profile hidden — the user still has the section (see
	// repository.UserHasProfiles), just nothing current to show.
	profile, profileErr := repository.FindLastProfileByVkId(user.VkId)
	if (profileErr != nil) {
		templates.Render(w, r, "profile_empty", profileEmptyData{Tabs: tabs})
		return
	}

	allFractions, fractionsErr := repository.FindAllFractions()
	if (fractionsErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	naturalSquadId := profile.TeamId
	effectiveSquadId := profile.EffectiveSquadId(user.SquadOverrideId)

	data := playerPageData{
		VkId: user.VkId,
		Profile: buildPlayerProfileView(*profile, allFractions, naturalSquadId, effectiveSquadId, now),
		Tabs: &tabs,
		ExperienceChartUrl: "/panel/profile/experience",
	}

	if attackProfile, err := repository.FindBestAttackProfile(profile.UserId); (err == nil) {
		data.BestAttack = buildPlayerStatsSnapshotView(*attackProfile, now)
	}

	if defenseProfile, err := repository.FindBestDefenseProfile(profile.UserId); (err == nil) {
		data.BestDefense = buildPlayerStatsSnapshotView(*defenseProfile, now)
	}

	templates.Render(w, r, "player_page", data)
}

type experienceChartPoint struct {
	At string `json:"at"`
	Level uint `json:"level"`
	Prestige uint `json:"prestige"`
	Total uint64 `json:"total"`
}

type experienceChartResponse struct {
	Error string `json:"error,omitempty"`
	From string `json:"from"`
	To string `json:"to"`
	Points []experienceChartPoint `json:"points"`
}

// ProfileExperienceChart returns the user's experience growth over the
// last year as JSON, flattened across prestiges (leveling.TotalExperience)
// so it never drops. The x-axis starts at the first point rather than a
// full year back, so a newer player's chart isn't mostly empty.
func ProfileExperienceChart(w http.ResponseWriter, r *http.Request) {
	user := requireOwnProfile(w, r)
	if (user == nil) {
		return
	}

	now := time.Now()
	since := now.AddDate(-experienceChartWindow, 0, 0)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	profiles, err := repository.FindExperienceSeries(user.Id, since)
	if (err != nil) {
		json.NewEncoder(w).Encode(experienceChartResponse{Error: "не удалось загрузить график"})
		return
	}

	from := since
	points := make([]experienceChartPoint, 0, len(profiles))
	for _, profile := range profiles {
		// A level-0 profile is a parse failure, not real data — it would
		// show up as a spike down to zero.
		if (profile.Level == 0) {
			continue
		}

		if (len(points) == 0) {
			from = profile.MessageDate
		}

		points = append(points, experienceChartPoint{
			At: profile.MessageDate.Format(time.RFC3339),
			Level: profile.Level,
			Prestige: profile.Prestige,
			Total: leveling.TotalExperience(profile.Level, profile.Prestige, uint64(profile.Experience)),
		})
	}

	json.NewEncoder(w).Encode(experienceChartResponse{
		From: from.Format(time.RFC3339),
		To: now.Format(time.RFC3339),
		Points: points,
	})
}

// profileHistoryQuery is the history tab's filter + page, round-tripped
// through the URL (and through the hide/restore forms, to land back on
// the same view). Dates are MSK calendar days as typed into the date
// inputs, both ends inclusive.
type profileHistoryQuery struct {
	From string
	To string
	ShowHidden bool
	Page int
}

func parseProfileHistoryQuery(values url.Values) profileHistoryQuery {
	query := profileHistoryQuery{Page: 1}

	if _, err := time.Parse("2006-01-02", values.Get("from")); (err == nil) {
		query.From = values.Get("from")
	}

	if _, err := time.Parse("2006-01-02", values.Get("to")); (err == nil) {
		query.To = values.Get("to")
	}

	query.ShowHidden = values.Get("hidden") == "1"

	if page, err := strconv.Atoi(values.Get("page")); (err == nil && page > 1) {
		query.Page = page
	}

	return query
}

func (query profileHistoryQuery) encode(page int) string {
	values := url.Values{}
	if (query.From != "") {
		values.Set("from", query.From)
	}

	if (query.To != "") {
		values.Set("to", query.To)
	}

	if (query.ShowHidden) {
		values.Set("hidden", "1")
	}

	if (page > 1) {
		values.Set("page", strconv.Itoa(page))
	}

	return values.Encode()
}

func (query profileHistoryQuery) href(page int) string {
	encoded := query.encode(page)
	if (encoded == "") {
		return "/panel/profile/history"
	}

	return "/panel/profile/history?" + encoded
}

func (query profileHistoryQuery) filter() repository.ProfileHistoryFilter {
	filter := repository.ProfileHistoryFilter{IncludeHidden: query.ShowHidden}

	if from, err := time.ParseInLocation("2006-01-02", query.From, moscow); (err == nil) {
		filter.From = &from
	}

	if to, err := time.ParseInLocation("2006-01-02", query.To, moscow); (err == nil) {
		end := to.AddDate(0, 0, 1)
		filter.To = &end
	}

	return filter
}

type profileHistoryRow struct {
	Id uint
	Date string
	Headline string
	Experience string
	Hidden bool
}

type profileHistoryData struct {
	Tabs profileTabsData
	From string
	To string
	ShowHidden bool
	Rows []profileHistoryRow
	Total int64
	Page int
	TotalPages int
	NewerHref string
	OlderHref string
	// Query is the current view's query string, posted back by the
	// hide/restore forms so the redirect lands on the same page.
	Query string
}

// ProfileHistory lists the user's own profiles, newest first, a page at a
// time, filterable by date. Each row shows the profile's headline exactly
// as it was when sent (squad, way, fraction — not the user's current
// ones), so a profile that isn't theirs stands out by nickname.
func ProfileHistory(w http.ResponseWriter, r *http.Request) {
	user := requireOwnProfile(w, r)
	if (user == nil) {
		return
	}

	query := parseProfileHistoryQuery(r.URL.Query())
	filter := query.filter()

	profiles, total, err := repository.FindProfileHistory(user.Id, filter, (query.Page-1)*profileHistoryPageSize, profileHistoryPageSize)
	if (err != nil) {
		http.Error(w, "Не удалось загрузить историю", http.StatusInternalServerError)
		return
	}

	totalPages := int((total + profileHistoryPageSize - 1) / profileHistoryPageSize)

	// A page past the end — e.g. just hid the last profile on the last
	// page — falls back to the actual last page instead of an empty list.
	if (totalPages > 0 && query.Page > totalPages) {
		query.Page = totalPages
		profiles, total, err = repository.FindProfileHistory(user.Id, filter, (query.Page-1)*profileHistoryPageSize, profileHistoryPageSize)
		if (err != nil) {
			http.Error(w, "Не удалось загрузить историю", http.StatusInternalServerError)
			return
		}
	}

	allFractions, fractionsErr := repository.FindAllFractions()
	if (fractionsErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	fractionIcons := make(map[uint]string, len(allFractions))
	for _, fraction := range allFractions {
		fractionIcons[fraction.Id] = fraction.Icon
	}

	squadCodes := map[uint]string{}
	rows := make([]profileHistoryRow, 0, len(profiles))
	for _, profile := range profiles {
		squadCode := ""
		if (profile.TeamId != nil) {
			code, known := squadCodes[*profile.TeamId]
			if (!known) {
				if squad, squadErr := repository.FindOneTeamById(*profile.TeamId); (squadErr == nil) {
					code = squad.Code
				}
				squadCodes[*profile.TeamId] = code
			}
			squadCode = code
		}

		rows = append(rows, profileHistoryRow{
			Id: profile.Id,
			Date: profile.MessageDate.In(moscow).Format("02.01.2006 15:04"),
			Headline: profileHeadline(profile, squadCode, fractionIcons[profile.FractionId]),
			Experience: groupThousands(uint64(profile.Experience)),
			Hidden: profile.Hidden,
		})
	}

	data := profileHistoryData{
		Tabs: profileTabsData{Active: "history"},
		From: query.From,
		To: query.To,
		ShowHidden: query.ShowHidden,
		Rows: rows,
		Total: total,
		Page: query.Page,
		TotalPages: totalPages,
		Query: query.encode(query.Page),
	}

	if (query.Page > 1) {
		data.NewerHref = query.href(query.Page - 1)
	}

	if (query.Page < totalPages) {
		data.OlderHref = query.href(query.Page + 1)
	}

	templates.Render(w, r, "profile_history", data)
}

func HideProfile(w http.ResponseWriter, r *http.Request) {
	setOwnProfileHidden(w, r, true)
}

func RestoreProfile(w http.ResponseWriter, r *http.Request) {
	setOwnProfileHidden(w, r, false)
}

// setOwnProfileHidden answers someone else's (or a nonexistent) profile
// id with the same 404, so the endpoint can't be used to probe ids.
func setOwnProfileHidden(w http.ResponseWriter, r *http.Request, hidden bool) {
	permissions := auth.PermissionsFromRequest(r)

	id, parseErr := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (parseErr != nil) {
		http.Error(w, "Профиль не найден", http.StatusNotFound)
		return
	}

	profile, profileErr := repository.FindOneProfileById(uint(id))
	if (profileErr != nil || !permissions.CanHideProfile(profile.UserId)) {
		http.Error(w, "Профиль не найден", http.StatusNotFound)
		return
	}

	if err := repository.SetProfileHidden(profile, hidden); (err != nil) {
		http.Error(w, "Не удалось сохранить", http.StatusInternalServerError)
		return
	}

	// Re-parsed rather than redirected to verbatim, so the posted value
	// can only ever produce a history URL.
	values, _ := url.ParseQuery(r.FormValue("query"))
	query := parseProfileHistoryQuery(values)
	http.Redirect(w, r, query.href(query.Page), http.StatusFound)
}

// ProfileInventory is a placeholder tab — inventory storage is still to be
// designed.
func ProfileInventory(w http.ResponseWriter, r *http.Request) {
	if (requireOwnProfile(w, r) == nil) {
		return
	}

	templates.Render(w, r, "profile_inventory", profileInventoryData{Tabs: profileTabsData{Active: "inventory"}})
}

// profileHeadline renders a profile's identity line the way the game
// itself formats it, e.g. "🔸DD|🔲🈵💡c0nd0r (39)" or "...(40💎3)".
func profileHeadline(profile model.Profile, squadCode string, fractionIcon string) string {
	var headline strings.Builder

	headline.WriteString(leadIcon(profile.Lead))
	if (squadCode != "") {
		headline.WriteString(squadCode)
		headline.WriteString("|")
	}

	headline.WriteString(wayIcon(profile.Way))
	headline.WriteString(fractionIcon)
	headline.WriteString(profile.Status)
	headline.WriteString(profile.Username)
	headline.WriteString(" (")
	headline.WriteString(strconv.FormatUint(uint64(profile.Level), 10))
	if (profile.Prestige > 0) {
		headline.WriteString("💎")
		headline.WriteString(strconv.FormatUint(uint64(profile.Prestige), 10))
	}
	headline.WriteString(")")

	return headline.String()
}

// groupThousands formats n with no-break spaces between digit groups,
// e.g. 1234567 → "1 234 567".
func groupThousands(n uint64) string {
	digits := strconv.FormatUint(n, 10)

	var grouped strings.Builder
	for i, digit := range digits {
		if (i > 0 && (len(digits)-i)%3 == 0) {
			grouped.WriteString(" ")
		}
		grouped.WriteRune(digit)
	}

	return grouped.String()
}
