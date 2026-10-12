package handlers

import (
	"net/http"
	"strconv"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/external/hackerwars"
	"jiqiren/bot/panel/auth"
)

func SetTarget(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	target, parseErr := strconv.ParseUint(r.FormValue("target"), 10, 64)
	if (parseErr != nil || !model.IsValidTarget(uint(target))) {
		http.Error(w, "Некорректная цель", http.StatusBadRequest)
		return
	}

	if (r.FormValue("scope") == "squad") {
		setSquadTarget(w, r, permissions, uint(target))
	} else {
		setFractionTarget(w, r, permissions, uint(target))
	}
}

func setSquadTarget(w http.ResponseWriter, r *http.Request, permissions *auth.Permissions, target uint) {
	squadId, parseErr := strconv.ParseUint(r.FormValue("squad_id"), 10, 64)
	if (parseErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}

	if (!permissions.CanSetSquadTarget(uint(squadId))) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	squad, squadErr := repository.FindOneTeamById(uint(squadId))
	if (squadErr != nil || squad.FractionId != permissions.FractionId()) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if (squad.TargetToken == "") {
		http.Error(w, "Для отряда не задан токен", http.StatusBadRequest)
		return
	}

	if setErr := hackerwars.SetTarget(squad.TargetToken, squad.Code, target); setErr != nil {
		http.Error(w, "Внешний API отклонил запрос: "+setErr.Error(), http.StatusBadGateway)
		return
	}

	repository.SetTeamTarget(squad, target)
	http.Redirect(w, r, "/panel", http.StatusFound)
}

func setFractionTarget(w http.ResponseWriter, r *http.Request, permissions *auth.Permissions, target uint) {
	if (!permissions.CanSetFractionTarget()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	fraction, fractionErr := repository.FindOneFractionById(permissions.FractionId())
	if (fractionErr != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	if (fraction.TargetToken == "") {
		http.Error(w, "Для команды не задан токен", http.StatusBadRequest)
		return
	}

	source := strconv.FormatUint(uint64(fraction.Id), 10)
	if setErr := hackerwars.SetTarget(fraction.TargetToken, source, target); setErr != nil {
		http.Error(w, "Внешний API отклонил запрос: "+setErr.Error(), http.StatusBadGateway)
		return
	}

	repository.SetFractionTarget(fraction, target)
	http.Redirect(w, r, "/panel", http.StatusFound)
}
