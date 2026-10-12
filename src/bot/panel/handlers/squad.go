package handlers

import (
	"net/http"
	"strconv"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"

	"github.com/go-chi/chi/v5"
)

type squadFormData struct {
	Action string
	Name string
	ShowCodeFields bool
}

func NewSquadForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageSquads()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	templates.Render(w, "squad_form", squadFormData{Action: "/panel/squads", ShowCodeFields: true})
}

func CreateSquad(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageSquads()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	squad := model.Team{
		Name: r.FormValue("name"),
		Code: r.FormValue("code"),
		FractionId: permissions.FractionId(),
	}

	if chatIdRaw := r.FormValue("chat_id"); chatIdRaw != "" {
		if chatId, parseErr := strconv.ParseInt(chatIdRaw, 10, 64); parseErr == nil {
			squad.ChatId = &chatId
		}
	}

	if createErr := repository.CreateTeam(&squad); createErr != nil {
		http.Error(w, "Не удалось создать отряд", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel", http.StatusFound)
}

func RenameSquadForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squadId, parseErr := squadIdFromPath(r)
	if (parseErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}

	if (!permissions.CanRenameSquad(squadId)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	squad, squadErr := repository.FindOneTeamById(squadId)
	if (squadErr != nil || squad.FractionId != permissions.FractionId()) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	templates.Render(w, "squad_form", squadFormData{
		Action: "/panel/squads/" + strconv.FormatUint(uint64(squadId), 10) + "/rename",
		Name: squad.Name,
	})
}

func RenameSquad(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)

	squadId, parseErr := squadIdFromPath(r)
	if (parseErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}

	if (!permissions.CanRenameSquad(squadId)) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	squad, squadErr := repository.FindOneTeamById(squadId)
	if (squadErr != nil || squad.FractionId != permissions.FractionId()) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if renameErr := repository.RenameTeam(squad, r.FormValue("name")); renameErr != nil {
		http.Error(w, "Не удалось переименовать отряд", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel", http.StatusFound)
}

func DeleteSquad(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageSquads()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	squadId, parseErr := squadIdFromPath(r)
	if (parseErr != nil) {
		http.Error(w, "Некорректный отряд", http.StatusBadRequest)
		return
	}

	squad, squadErr := repository.FindOneTeamById(squadId)
	if (squadErr != nil || squad.FractionId != permissions.FractionId()) {
		http.Error(w, "Отряд не найден", http.StatusNotFound)
		return
	}

	if deleteErr := repository.DeleteTeam(squadId); deleteErr != nil {
		http.Error(w, "Не удалось удалить отряд", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel", http.StatusFound)
}

func squadIdFromPath(r *http.Request) (uint, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (err != nil) {
		return 0, err
	}

	return uint(id), nil
}
