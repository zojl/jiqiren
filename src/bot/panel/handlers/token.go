package handlers

import (
	"fmt"
	"net/http"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/go-chi/chi/v5"
)

type tokenFormData struct {
	Action string
}

func FractionTokenForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionToken()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	templates.Render(w, "token_form", tokenFormData{Action: "/panel/fraction/token"})
}

func SetFractionToken(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionToken()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, fractionErr := repository.FindOneFractionById(permissions.FractionId())
	if (fractionErr != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	if setErr := repository.SetFractionToken(fraction, r.FormValue("token")); setErr != nil {
		http.Error(w, "Не удалось сохранить токен", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionSetFractionToken, "обновлён токен команды")
	http.Redirect(w, r, "/panel", http.StatusFound)
}

func SquadTokenForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadToken(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	templates.Render(w, "token_form", tokenFormData{
		Action: "/panel/squads/" + squad.Code + "/token",
	})
}

func SetSquadToken(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squad, squadErr := repository.FindOneTeamByCodeAndFraction(chi.URLParam(r, "code"), permissions.FractionId())
	if (squadErr != nil) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (!permissions.CanSetSquadToken(squad.Id)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	if setErr := repository.SetTeamToken(squad, r.FormValue("token")); setErr != nil {
		http.Error(w, "Не удалось сохранить токен", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionSetSquadToken,
		fmt.Sprintf("обновлён токен отряда %s", squad.Code))
	http.Redirect(w, r, "/panel", http.StatusFound)
}
