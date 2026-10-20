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

type targetIconOption struct {
	Value uint
	Icon string
	Name string
	Selected bool
}

type squadView struct {
	Id uint
	Name string
	Code string
	CurrentTargetIcon string
	TargetOptions []targetIconOption
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
	CanViewAuditLog bool
	Squads []squadView
}

func Dashboard(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewTargets()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
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
		CanViewAuditLog: permissions.CanViewAuditLog(),
	}

	squads, squadsErr := repository.FindTeamsByFractionId(fraction.Id)
	if (squadsErr == nil) {
		for _, squad := range squads {
			if (!permissions.CanViewSquad(squad.Id)) {
				continue
			}

			target := model.TargetNone
			if (squad.Target != nil) {
				target = *squad.Target
			}

			data.Squads = append(data.Squads, squadView{
				Id: squad.Id,
				Name: squad.Name,
				Code: squad.Code,
				CurrentTargetIcon: targetIcon(allFractions, squad.Target),
				TargetOptions: buildTargetIconOptions(allFractions, fraction.Id, target),
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
	options = append(options, targetOption{Value: model.TargetDefense, Label: "🔐 защита"})

	return options
}

// buildTargetIconOptions mirrors buildTargetOptions but renders icon-only
// options for the compact per-squad target picker.
func buildTargetIconOptions(fractions []model.Fraction, ownFractionId uint, current uint) []targetIconOption {
	options := []targetIconOption{{Value: model.TargetNone, Icon: "✕", Name: "снять цель", Selected: current == model.TargetNone}}
	for _, fraction := range fractions {
		if (fraction.Id == ownFractionId) {
			continue
		}
		options = append(options, targetIconOption{Value: fraction.Id, Icon: fraction.Icon, Name: fraction.Name, Selected: current == fraction.Id})
	}
	options = append(options, targetIconOption{Value: model.TargetDefense, Icon: "🔐", Name: "защита", Selected: current == model.TargetDefense})

	return options
}

func targetIcon(fractions []model.Fraction, target *uint) string {
	if (target == nil || *target == model.TargetNone) {
		return "✕"
	}

	if (*target == model.TargetDefense) {
		return "🔐"
	}

	for _, fraction := range fractions {
		if (fraction.Id == *target) {
			return fraction.Icon
		}
	}

	return "?"
}

func targetLabel(fractions []model.Fraction, target *uint) string {
	if (target == nil || *target == model.TargetNone) {
		return "—"
	}

	if (*target == model.TargetDefense) {
		return "🔐 защита"
	}

	for _, fraction := range fractions {
		if (fraction.Id == *target) {
			return fraction.Icon + " " + fraction.Name
		}
	}

	return "?"
}
