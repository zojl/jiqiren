package handlers

import (
	"net/http"
	"strconv"

	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
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

	http.Redirect(w, r, "/panel", http.StatusFound)
}

func SquadTokenForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squadId, parseErr := squadIdFromPath(r)
	if (parseErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}

	if (!permissions.CanSetSquadToken(squadId)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	squad, squadErr := repository.FindOneTeamById(squadId)
	if (squadErr != nil || squad.FractionId != permissions.FractionId()) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	templates.Render(w, "token_form", tokenFormData{
		Action: "/panel/squads/" + strconv.FormatUint(uint64(squadId), 10) + "/token",
	})
}

func SetSquadToken(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squadId, parseErr := squadIdFromPath(r)
	if (parseErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}

	if (!permissions.CanSetSquadToken(squadId)) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	squad, squadErr := repository.FindOneTeamById(squadId)
	if (squadErr != nil || squad.FractionId != permissions.FractionId()) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if setErr := repository.SetTeamToken(squad, r.FormValue("token")); setErr != nil {
		http.Error(w, "Не удалось сохранить токен", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel", http.StatusFound)
}
