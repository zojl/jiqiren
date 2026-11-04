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

	LiveAvailable      bool
	Live               budgetFieldSet
	LiveHasDiff        bool
	LiveIncomeDiffLabel string
	LiveIncomeDiffBase uint64

	BattleResultsAvailable bool
	BattleResults          budgetFieldSet
	BattleDayLabel         string

	YesterdayAvailable           bool
	YesterdayIncomeLabel         string
	YesterdayContributionsLabel string

	ChartUrl string
}

// parseSelectedDate reads ?date=YYYY-MM-DD (MSK calendar day) from the
// request, falling back to today. The result uses the same UTC-midnight
// representation as battletime.Today, so it can be fed straight into
// resolveBattleDay and the repository's day-bounded lookups.
func parseSelectedDate(r *http.Request, now time.Time) time.Time {
	raw := r.URL.Query().Get("date")
	if (raw != "") {
		if parsed, err := time.ParseInLocation("2006-01-02", raw, time.UTC); (err == nil) {
			return parsed
		}
	}

	return battletime.Today(now)
}

// resolveBattleDay maps a selected calendar day to the battle cycle whose
// results should be shown. For any day already fully elapsed this is just
// that day itself; for "today" specifically, today's battle may not have
// produced results yet, in which case the previous cycle's results are
// shown instead.
func resolveBattleDay(selected time.Time, now time.Time) time.Time {
	if (!selected.Equal(battletime.Today(now))) {
		return selected
	}

	if (now.Before(battletime.PostBattleAnchor(selected))) {
		return selected.AddDate(0, 0, -1)
	}

	return selected
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
	battleDay := resolveBattleDay(selected, now)

	data := budgetPageData{
		SelectedDate:  selected.Format("2006-01-02"),
		TodayDate:     battletime.Today(now).Format("2006-01-02"),
		YesterdayDate: battletime.Today(now).AddDate(0, 0, -1).Format("2006-01-02"),
		IsToday:       selected.Equal(battletime.Today(now)),
		ChartUrl:      "/panel/fraction/budget/chart?date=" + selected.Format("2006-01-02"),
	}

	if (fraction.TargetToken == "") {
		data.Error = "токен команды не задан"
		templates.Render(w, "budget_page", data)
		return
	}

	var battleResults *model.BudgetSnapshot
	if br, brErr := repository.FindBudgetSnapshotByKindAndDate(fraction.Id, model.BudgetSnapshotKindPostBattle, battleDay); (brErr == nil) {
		battleResults = br
		data.BattleResultsAvailable = true
		data.BattleResults = budgetFieldSetFromSnapshot(br)
		data.BattleDayLabel = battleDay.In(battletime.Moscow).Format("02.01.2006")
	}

	if seed, seedErr := repository.FindLatestBudgetSnapshot(fraction.Id, selected); (seedErr == nil) {
		data.LiveAvailable = true
		data.Live = budgetFieldSetFromSnapshot(seed)

		if (battleResults != nil) {
			data.LiveHasDiff = true
			data.LiveIncomeDiffLabel = fmt.Sprintf("%+d", int64(seed.IncomeToday)-int64(battleResults.IncomeToday))
			data.LiveIncomeDiffBase = battleResults.IncomeToday
		}
	}

	if preBattle, preErr := repository.FindBudgetSnapshotByKindAndDate(fraction.Id, model.BudgetSnapshotKindPreBattle, battleDay); (preErr == nil && battleResults != nil) {
		data.YesterdayAvailable = true
		data.YesterdayIncomeLabel = fmt.Sprintf("%+d", int64(preBattle.IncomeToday)-int64(battleResults.IncomeToday))
		data.YesterdayContributionsLabel = fmt.Sprintf("%+d", int64(preBattle.ContributionsToday)-int64(battleResults.ContributionsToday))
	}

	templates.Render(w, "budget_page", data)
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

// BudgetChart returns the fixed 24h window (previous cycle's post-battle
// anchor through the selected cycle's pre-battle anchor) of every snapshot
// recorded for the fraction, for the page's graph block.
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
	battleDay := resolveBattleDay(selected, now)

	from := battletime.PostBattleAnchor(battleDay.AddDate(0, 0, -1))
	to := battletime.PreBattleAnchor(battleDay)

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
