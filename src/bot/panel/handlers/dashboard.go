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
	GoalPlan goalPlanSectionData
}

type dashboardData struct {
	FractionName string
	FractionIcon string
	FractionCurrentTargetIcon string
	FractionTargetOptions []targetIconOption
	CanSetFractionTarget bool
	CanManageSquads bool
	Squads []squadView
	GoalPlan goalPlanSectionData
}

// Dashboard is the fraction-wide "Отряды" page — a plain squad lead has no
// use for it, so a failed permission check falls back to redirecting them
// to their own squad page instead of a flat 403.
func Dashboard(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewDashboard()) {
		if squadId, isHome := permissions.HomeSquadId(); (isHome) {
			if squad, squadErr := repository.FindOneTeamById(squadId); (squadErr == nil) {
				http.Redirect(w, r, "/panel/squads/"+squad.Code, http.StatusFound)
				return
			}
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

	fractionGoalTargets, fractionGoalErr := repository.FindPlannedTargetsByFractionId(fraction.Id)
	if (fractionGoalErr != nil) {
		fractionGoalTargets = nil
	}

	data := dashboardData{
		FractionName: fraction.Name,
		FractionIcon: fraction.Icon,
		FractionCurrentTargetIcon: targetIcon(allFractions, fraction.Target),
		FractionTargetOptions: buildTargetIconOptions(allFractions, fraction.Id, fractionTarget),
		CanSetFractionTarget: permissions.CanSetFractionTarget(),
		CanManageSquads: permissions.CanManageSquads(),
		GoalPlan: buildGoalPlanSection(fractionGoalTargets, now, allFractions, "/panel/fraction/goals", permissions.CanSetFractionTarget()),
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

			squadGoalTargets, squadGoalErr := repository.FindPlannedTargetsBySquadId(squad.Id)
			if (squadGoalErr != nil) {
				squadGoalTargets = nil
			}

			data.Squads = append(data.Squads, squadView{
				Id: squad.Id,
				Name: squad.Name,
				Code: squad.Code,
				CurrentTargetIcon: targetIcon(allFractions, squad.Target),
				TargetOptions: buildTargetIconOptions(allFractions, fraction.Id, target),
				CanSetTarget: permissions.CanSetSquadTarget(squad.Id),
				GoalPlan: buildGoalPlanSection(squadGoalTargets, now, allFractions, "/panel/squads/"+squad.Code+"/goals", permissions.CanSetSquadTarget(squad.Id)),
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
