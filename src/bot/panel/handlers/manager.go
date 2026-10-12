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

type managerView struct {
	Id uint
	VkId uint
	AccountType string
	SquadId *uint
}

func ManagerList(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	managers, managersErr := repository.FindManagersByFractionId(permissions.FractionId())
	if (managersErr != nil) {
		http.Error(w, "Не удалось загрузить менеджеров", http.StatusInternalServerError)
		return
	}

	views := make([]managerView, 0, len(managers))
	for _, manager := range managers {
		accountType := "—"
		if (manager.AccountType != nil) {
			accountType = string(*manager.AccountType)
		}

		views = append(views, managerView{
			Id: manager.Id,
			VkId: manager.VkId,
			AccountType: accountType,
			SquadId: manager.SquadId,
		})
	}

	templates.Render(w, "manager_list", views)
}

func CreateManager(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	vkId, parseErr := strconv.ParseUint(r.FormValue("vk_id"), 10, 64)
	if (parseErr != nil) {
		http.Error(w, "Некорректный VK id", http.StatusBadRequest)
		return
	}

	manager := model.Manager{
		VkId: uint(vkId),
		FractionId: permissions.FractionId(),
	}

	if accountTypeRaw := r.FormValue("account_type"); accountTypeRaw != "" {
		accountType := model.AccountType(accountTypeRaw)
		manager.AccountType = &accountType
	}

	if squadIdRaw := r.FormValue("squad_id"); squadIdRaw != "" {
		if squadId, squadParseErr := strconv.ParseUint(squadIdRaw, 10, 64); squadParseErr == nil {
			squadIdValue := uint(squadId)
			manager.SquadId = &squadIdValue
		}
	}

	if createErr := repository.CreateManager(&manager); createErr != nil {
		http.Error(w, "Не удалось создать менеджера", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel/managers", http.StatusFound)
}

func DeleteManager(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanManageManagers()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	id, parseErr := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if (parseErr != nil) {
		http.Error(w, "Некорректный id", http.StatusBadRequest)
		return
	}

	manager, managerErr := repository.FindOneManagerById(uint(id))
	if (managerErr != nil || manager.FractionId != permissions.FractionId()) {
		http.Error(w, "Менеджер не найден", http.StatusNotFound)
		return
	}

	if deleteErr := repository.DeleteManager(uint(id)); deleteErr != nil {
		http.Error(w, "Не удалось удалить менеджера", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/panel/managers", http.StatusFound)
}
