package handlers

import (
	"net/http"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

type targetOption struct {
	Value uint
	Label string
}

type squadView struct {
	Id uint
	Name string
	Code string
	TargetLabel string
	CanSetTarget bool
	CanSetToken bool
	CanRename bool
}

type dashboardData struct {
	FractionName string
	FractionIcon string
	TargetOptions []targetOption
	CanViewBalance bool
	CanSetFractionTarget bool
	CanSetFractionToken bool
	FractionTargetLabel string
	CanManageSquads bool
	Squads []squadView
}

func Dashboard(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewTargets()) {
		http.Error(w, "Доступ запрещён", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	allFractions, err := repository.FindAllFractions()
	if (err != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	data := dashboardData{
		FractionName: fraction.Name,
		FractionIcon: fraction.Icon,
		TargetOptions: buildTargetOptions(allFractions, fraction.Id),
		CanViewBalance: permissions.CanViewBalance(),
		CanSetFractionTarget: permissions.CanSetFractionTarget(),
		CanSetFractionToken: permissions.CanSetFractionToken(),
		FractionTargetLabel: targetLabel(allFractions, fraction.Target),
		CanManageSquads: permissions.CanManageSquads(),
	}

	squads, squadsErr := repository.FindTeamsByFractionId(fraction.Id)
	if (squadsErr == nil) {
		for _, squad := range squads {
			data.Squads = append(data.Squads, squadView{
				Id: squad.Id,
				Name: squad.Name,
				Code: squad.Code,
				TargetLabel: targetLabel(allFractions, squad.Target),
				CanSetTarget: permissions.CanSetSquadTarget(squad.Id),
				CanSetToken: permissions.CanSetSquadToken(squad.Id),
				CanRename: permissions.CanRenameSquad(squad.Id),
			})
		}
	}

	templates.Render(w, "dashboard", data)
}

func buildTargetOptions(fractions []model.Fraction, ownFractionId uint) []targetOption {
	options := []targetOption{{Value: model.TargetNone, Label: "✕ снять цель"}}
	for _, fraction := range fractions {
		if (fraction.Id == ownFractionId) {
			continue
		}
		options = append(options, targetOption{Value: fraction.Id, Label: fraction.Icon + " " + fraction.Name})
	}
	options = append(options, targetOption{Value: model.TargetDefense, Label: "🛡 защита"})

	return options
}

func targetLabel(fractions []model.Fraction, target *uint) string {
	if (target == nil || *target == model.TargetNone) {
		return "—"
	}

	if (*target == model.TargetDefense) {
		return "🛡 защита"
	}

	for _, fraction := range fractions {
		if (fraction.Id == *target) {
			return fraction.Icon + " " + fraction.Name
		}
	}

	return "?"
}
