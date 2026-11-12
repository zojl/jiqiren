package handlers

import (
	"net/http"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

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
	TargetPlan targetPlanSectionData
}

type dashboardData struct {
	FractionName string
	FractionIcon string
	FractionCurrentTargetIcon string
	FractionTargetOptions []targetIconOption
	CanSetFractionTarget bool
	CanManageSquads bool
	Squads []squadView
	TargetPlan targetPlanSectionData
}

// Dashboard is the fraction-wide "Отряды" page and the panel's landing
// route. Anyone without access to it is redirected to their own default
// section instead of a flat 403: a plain squad lead to their own squad
// page, and anyone else with a profile (a plain player) to "Мой профиль".
func Dashboard(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewDashboard()) {
		if squadId, isHome := permissions.HomeSquadId(); (isHome) {
			if squad, squadErr := repository.FindOneTeamById(squadId); (squadErr == nil) {
				http.Redirect(w, r, "/panel/squads/"+squad.Code, http.StatusFound)
				return
			}
		}

		if (permissions.CanViewOwnProfile()) {
			http.Redirect(w, r, "/panel/profile", http.StatusFound)
			return
		}

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

	fractionTarget := model.TargetNone
	if (fraction.Target != nil) {
		fractionTarget = *fraction.Target
	}

	now := time.Now()

	fractionPlannedTargets, fractionTargetsErr := repository.FindPlannedTargetsByFractionId(fraction.Id)
	if (fractionTargetsErr != nil) {
		fractionPlannedTargets = nil
	}

	data := dashboardData{
		FractionName: fraction.Name,
		FractionIcon: fraction.Icon,
		FractionCurrentTargetIcon: targetIcon(allFractions, fraction.Target),
		FractionTargetOptions: buildTargetIconOptions(allFractions, fraction.Id, fractionTarget),
		CanSetFractionTarget: permissions.CanSetFractionTarget(),
		CanManageSquads: permissions.CanManageSquads(),
		TargetPlan: buildTargetPlanSection(fractionPlannedTargets, now, allFractions, "/panel/fraction/targets", permissions.CanSetFractionTarget()),
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

			squadPlannedTargets, squadTargetsErr := repository.FindPlannedTargetsBySquadId(squad.Id)
			if (squadTargetsErr != nil) {
				squadPlannedTargets = nil
			}

			data.Squads = append(data.Squads, squadView{
				Id: squad.Id,
				Name: squad.Name,
				Code: squad.Code,
				CurrentTargetIcon: targetIcon(allFractions, squad.Target),
				TargetOptions: buildTargetIconOptions(allFractions, fraction.Id, target),
				CanSetTarget: permissions.CanSetSquadTarget(squad.Id),
				TargetPlan: buildTargetPlanSection(squadPlannedTargets, now, allFractions, "/panel/squads/"+squad.Code+"/targets", permissions.CanSetSquadTarget(squad.Id)),
			})
		}
	}

	templates.Render(w, r, "dashboard", data)
}

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
