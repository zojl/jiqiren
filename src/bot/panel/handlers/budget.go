package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/external/hackerwars"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

type budgetFieldSet struct {
	Budget             uint64
	ContributionsToday uint64
	ContributionsTotal uint64
	IncomeToday        uint64
	IncomeTotal        uint64
}

func budgetFieldSetFromSnapshot(snapshot *model.BudgetSnapshot) budgetFieldSet {
	return budgetFieldSet{
		Budget:             snapshot.Budget,
		ContributionsToday: snapshot.ContributionsToday,
		ContributionsTotal: snapshot.ContributionsTotal,
		IncomeToday:        snapshot.IncomeToday,
		IncomeTotal:        snapshot.IncomeTotal,
	}
}

type budgetPageData struct {
	Error string

	SelectedDate  string
	TodayDate     string
	YesterdayDate string
	IsToday       bool

	LiveAvailable               bool
	Live                        budgetFieldSet
	LiveHasDiff                 bool
	LiveIncomeDiffLabel         string
	LiveIncomeDiffBase          uint64
	LiveContributionsDiffLabel  string
	LiveContributionsDiffBase   uint64
	LiveUpdatedAtLabel          string

	BattleResultsAvailable bool
	BattleResults          budgetFieldSet
	BattleDayLabel         string

	// YesterdayResults is the pre_battle snapshot for battleDay as-is —
	// the same field set as BattleResults, just captured ~5 minutes
	// earlier (17:58 vs. 18:03), not a delta against anything.
	YesterdayAvailable bool
	YesterdayResults   budgetFieldSet

	ChartUrl string
}

// parseSelectedDate reads ?date=YYYY-MM-DD (MSK calendar day) from the
// request, falling back to battletime.BattleToday. The result uses the
// same UTC-midnight representation as battletime.Today, so it can be fed
// straight into the repository's day-bounded lookups and budgetCycleWindow
// — it already *is* the battle day (see BattleToday), no further
// resolution needed.
func parseSelectedDate(r *http.Request, now time.Time) time.Time {
	raw := r.URL.Query().Get("date")
	if (raw != "") {
		if parsed, err := time.ParseInLocation("2006-01-02", raw, time.UTC); (err == nil) {
			return parsed
		}
	}

	return battletime.BattleToday(now)
}

// budgetCycleWindow is the fixed 24h window of the cycle that starts right
// after battleDay's battle and runs until the next battle's pre-battle
// anchor — the "today" cycle relative to battleDay, which is what both the
// live-data seed and the chart visualize (as opposed to the already-
// concluded cycle shown by "доход за вчера", which ends at battleDay
// instead of starting there).
func budgetCycleWindow(battleDay time.Time) (time.Time, time.Time) {
	from := battletime.PostBattleAnchor(battleDay)
	to := battletime.PreBattleAnchor(battleDay.AddDate(0, 0, 1))
	return from, to
}

func BudgetPage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewBalance()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionViewBalance, "просмотр баланса фракции")

	now := time.Now()
	selected := parseSelectedDate(r, now)
	battleDay := selected
	today := battletime.BattleToday(now)

	data := budgetPageData{
		SelectedDate:  selected.Format("2006-01-02"),
		TodayDate:     today.Format("2006-01-02"),
		YesterdayDate: today.AddDate(0, 0, -1).Format("2006-01-02"),
		IsToday:       selected.Equal(today),
		ChartUrl:      "/panel/fraction/budget/chart?date=" + selected.Format("2006-01-02"),
	}

	if (fraction.TargetToken == "") {
		data.Error = "токен команды не задан"
		templates.Render(w, r, "budget_page", data)
		return
	}

	var battleResults *model.BudgetSnapshot
	if br, brErr := repository.FindBudgetSnapshotByKindAndDate(fraction.Id, model.BudgetSnapshotKindPostBattle, battleDay); (brErr == nil) {
		battleResults = br
		data.BattleResultsAvailable = true
		data.BattleResults = budgetFieldSetFromSnapshot(br)
		data.BattleDayLabel = battleDay.In(battletime.Moscow).Format("02.01.2006")
	}

	seedFrom, seedTo := budgetCycleWindow(battleDay)
	if seed, seedErr := repository.FindLatestBudgetSnapshotInRange(fraction.Id, seedFrom, seedTo); (seedErr == nil) {
		data.LiveAvailable = true
		data.Live = budgetFieldSetFromSnapshot(seed)
		data.LiveUpdatedAtLabel = seed.CapturedAt.In(battletime.Moscow).Format("15:04")

		if (battleResults != nil) {
			data.LiveHasDiff = true
			data.LiveIncomeDiffLabel = fmt.Sprintf("%+d", int64(seed.IncomeToday)-int64(battleResults.IncomeToday))
			data.LiveIncomeDiffBase = battleResults.IncomeToday
			data.LiveContributionsDiffLabel = fmt.Sprintf("%+d", int64(seed.ContributionsToday)-int64(battleResults.ContributionsToday))
			data.LiveContributionsDiffBase = battleResults.ContributionsToday
		}
	}

	if preBattle, preErr := repository.FindBudgetSnapshotByKindAndDate(fraction.Id, model.BudgetSnapshotKindPreBattle, battleDay); (preErr == nil) {
		data.YesterdayAvailable = true
		data.YesterdayResults = budgetFieldSetFromSnapshot(preBattle)
	}

	templates.Render(w, r, "budget_page", data)
}

type budgetActualResponse struct {
	Error              string `json:"error,omitempty"`
	Budget             uint64 `json:"budget"`
	ContributionsToday uint64 `json:"contributionsToday"`
	ContributionsTotal uint64 `json:"contributionsTotal"`
	IncomeToday        uint64 `json:"incomeToday"`
	IncomeTotal        uint64 `json:"incomeTotal"`
}

// BudgetActual is fetched by the budget page's JS, only while viewing
// today — every call re-queries hackerwars.ru for fresh numbers, nothing is
// cached server- or client-side.
func BudgetActual(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewBalance()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionViewActualBudget, "просмотр текущего бюджета")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if (fraction.TargetToken == "") {
		json.NewEncoder(w).Encode(budgetActualResponse{Error: "токен команды не задан"})
		return
	}

	budget, budgetErr := hackerwars.GetTeamBudget(fraction.TargetToken)
	if (budgetErr != nil) {
		json.NewEncoder(w).Encode(budgetActualResponse{Error: budgetErr.Error()})
		return
	}

	json.NewEncoder(w).Encode(budgetActualResponse{
		Budget:             budget.Info.Budget,
		ContributionsToday: budget.Info.Contributions.Today,
		ContributionsTotal: budget.Info.Contributions.Total,
		IncomeToday:        budget.Info.Income.Today,
		IncomeTotal:        budget.Info.Income.Total,
	})
}

type budgetChartPoint struct {
	CapturedAt         string `json:"capturedAt"`
	Kind               string `json:"kind"`
	Budget             uint64 `json:"budget"`
	ContributionsToday uint64 `json:"contributionsToday"`
	ContributionsTotal uint64 `json:"contributionsTotal"`
	IncomeToday        uint64 `json:"incomeToday"`
	IncomeTotal        uint64 `json:"incomeTotal"`
}

type budgetChartResponse struct {
	Error  string             `json:"error,omitempty"`
	From   string             `json:"from"`
	To     string             `json:"to"`
	Points []budgetChartPoint `json:"points"`
}

// BudgetChart returns the fixed 24h window (see budgetCycleWindow) of
// every snapshot recorded for the fraction, for the page's graph block.
func BudgetChart(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewBalance()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionViewBudgetChart, "просмотр графика бюджета")

	now := time.Now()
	selected := parseSelectedDate(r, now)
	battleDay := selected

	from, to := budgetCycleWindow(battleDay)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	snapshots, snapErr := repository.FindBudgetSnapshotsInRange(fraction.Id, from, to)
	if (snapErr != nil) {
		json.NewEncoder(w).Encode(budgetChartResponse{Error: snapErr.Error()})
		return
	}

	points := make([]budgetChartPoint, 0, len(snapshots))
	for _, snapshot := range snapshots {
		points = append(points, budgetChartPoint{
			CapturedAt:         snapshot.CapturedAt.Format(time.RFC3339),
			Kind:               string(snapshot.Kind),
			Budget:             snapshot.Budget,
			ContributionsToday: snapshot.ContributionsToday,
			ContributionsTotal: snapshot.ContributionsTotal,
			IncomeToday:        snapshot.IncomeToday,
			IncomeTotal:        snapshot.IncomeTotal,
		})
	}

	json.NewEncoder(w).Encode(budgetChartResponse{
		From:   from.Format(time.RFC3339),
		To:     to.Format(time.RFC3339),
		Points: points,
	})
}
