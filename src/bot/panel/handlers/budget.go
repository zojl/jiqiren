package handlers

import (
	"net/http"

	"jiqiren/bot/database/repository"
	"jiqiren/bot/external/hackerwars"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

type budgetFragmentData struct {
	Budget *hackerwars.Budget
	Error string
}

// FractionBudget is fetched lazily by JS each time the balance spoiler on
// the dashboard is opened — the response is never cached server- or
// client-side, so every open re-queries hackerwars.ru for fresh numbers.
func FractionBudget(w http.ResponseWriter, r *http.Request) {
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

	if (fraction.TargetToken == "") {
		templates.RenderFragment(w, "budget_fragment", budgetFragmentData{Error: "токен команды не задан"})
		return
	}

	budget, budgetErr := hackerwars.GetTeamBudget(fraction.TargetToken)
	if (budgetErr != nil) {
		templates.RenderFragment(w, "budget_fragment", budgetFragmentData{Error: budgetErr.Error()})
		return
	}

	templates.RenderFragment(w, "budget_fragment", budgetFragmentData{Budget: budget})
}
