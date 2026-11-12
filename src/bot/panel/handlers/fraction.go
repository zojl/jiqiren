package handlers

import (
	"net/http"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
	"jiqiren/bot/readiness"
)

type fractionPageData struct {
	Name               string
	Icon               string
	CurrentTargetIcon  string
	TargetOptions      []targetIconOption
	CanSetTarget       bool
	HasToken           bool
	CanSetToken        bool
	Message            string
	CanSetMessage       bool

	StaminaVerySevere  uint
	StaminaSevere      uint
	StaminaRecommended uint
	MoneyRecommended   uint
	MoneySevere        uint
	MoneyVerySevere    uint
	CanSetSettings     bool

	TargetPlan targetPlanSectionData
}

// FractionPage is "Команда" — the fraction-wide counterpart of a squad
// page, deliberately minimal: target, token and the team message, with
// none of a squad's battle/member/schedule concerns.
func FractionPage(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanViewFraction()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	allFractions, allErr := repository.FindAllFractions()
	if (allErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	target := model.TargetNone
	if (fraction.Target != nil) {
		target = *fraction.Target
	}

	thresholds := readiness.ThresholdsFor(fraction)

	plannedTargets, plannedTargetsErr := repository.FindPlannedTargetsByFractionId(fraction.Id)
	if (plannedTargetsErr != nil) {
		plannedTargets = nil
	}
	targetPlan := buildTargetPlanSection(plannedTargets, time.Now(), allFractions, "/panel/fraction/targets", permissions.CanSetFractionTarget())

	templates.Render(w, r, "fraction_page", fractionPageData{
		Name:              fraction.Name,
		Icon:              fraction.Icon,
		CurrentTargetIcon: targetIcon(allFractions, fraction.Target),
		TargetOptions:     buildTargetIconOptions(allFractions, fraction.Id, target),
		CanSetTarget:      permissions.CanSetFractionTarget(),
		HasToken:          fraction.TargetToken != "",
		CanSetToken:       permissions.CanSetFractionToken(),
		Message:           fraction.Message,
		CanSetMessage:     permissions.CanSetFractionMessage(),

		StaminaVerySevere:  thresholds.StaminaVerySevere,
		StaminaSevere:      thresholds.StaminaSevere,
		StaminaRecommended: thresholds.StaminaRecommended,
		MoneyRecommended:   thresholds.MoneyRecommended,
		MoneySevere:        thresholds.MoneySevere,
		MoneyVerySevere:    thresholds.MoneyVerySevere,
		CanSetSettings:     permissions.CanSetFractionSettings(),

		TargetPlan: targetPlan,
	})
}
