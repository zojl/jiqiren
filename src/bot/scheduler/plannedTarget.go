package scheduler

import (
	"log"
	"strconv"
	"time"

	"jiqiren/bot/audit"
	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/external/hackerwars"
	"jiqiren/bot/targetplan"
)

// tickPlannedTargets applies every due planned target — squad- and
// fraction-scope alike. Author of the resulting audit entries is manager
// id 0 (no human manager triggered this), matching the requirement that
// scheduler-driven target changes are attributed to "0".
func tickPlannedTargets(now time.Time) {
	targets, err := repository.FindAllPlannedTargets()
	if (err != nil) {
		log.Println("scheduler: " + err.Error())
		return
	}

	for i := range targets {
		target := targets[i]

		moment, ok := targetplan.NextOccurrence(target, now)
		if (!ok || now.Before(moment)) {
			continue
		}

		if (target.SquadId != nil) {
			applySquadPlannedTarget(&target, now)
		} else {
			applyFractionPlannedTarget(&target, now)
		}
	}
}

func applySquadPlannedTarget(target *model.PlannedTarget, now time.Time) {
	squad, squadErr := repository.FindOneTeamById(*target.SquadId)
	if (squadErr != nil) {
		return // squad gone; nothing sensible to apply to
	}

	if (target.Reserve && squad.Target != nil && *squad.Target != model.TargetNone) {
		return // reserve condition not met yet; retry next tick
	}

	if (squad.TargetToken == "") {
		recordPlannedTargetFailure(target, "для отряда не задан токен")
		return
	}

	if setErr := hackerwars.SetTarget(squad.TargetToken, squad.Code, target.Target); (setErr != nil) {
		recordPlannedTargetFailure(target, setErr.Error())
		return
	}

	if applyErr := repository.SetTeamTarget(squad, target.Target); (applyErr != nil) {
		log.Println("scheduler: " + applyErr.Error())
		return
	}

	markPlannedTargetSucceeded(target, now,
		"отряд "+squad.Code+" → "+formatPlannedTargetDetail(target.Target))

	if (target.Message != "" && squad.ChatId != nil) {
		if sendErr := sendText(*squad.ChatId, target.Message); (sendErr != nil) {
			appendPlannedTargetError(target, "сообщение не отправлено: "+sendErr.Error())
		}
	}
}

func applyFractionPlannedTarget(target *model.PlannedTarget, now time.Time) {
	fraction, fractionErr := repository.FindOneFractionById(target.FractionId)
	if (fractionErr != nil) {
		return
	}

	if (target.Reserve && fraction.Target != nil && *fraction.Target != model.TargetNone) {
		return // reserve condition not met yet; retry next tick
	}

	if (fraction.TargetToken == "") {
		recordPlannedTargetFailure(target, "для команды не задан токен")
		return
	}

	source := strconv.FormatUint(uint64(fraction.Id), 10)
	if setErr := hackerwars.SetTarget(fraction.TargetToken, source, target.Target); (setErr != nil) {
		recordPlannedTargetFailure(target, setErr.Error())
		return
	}

	if applyErr := repository.SetFractionTarget(fraction, target.Target); (applyErr != nil) {
		log.Println("scheduler: " + applyErr.Error())
		return
	}

	markPlannedTargetSucceeded(target, now, "фракция → "+formatPlannedTargetDetail(target.Target))
}

func markPlannedTargetSucceeded(target *model.PlannedTarget, now time.Time, details string) {
	today := battletime.Today(now)
	if markErr := repository.MarkPlannedTargetApplied(target, today); (markErr != nil) {
		log.Println("scheduler: " + markErr.Error())
	}

	audit.Record(target.FractionId, 0, model.AuditActionApplyPlannedTarget, details)
}

func recordPlannedTargetFailure(target *model.PlannedTarget, errText string) {
	if markErr := repository.MarkPlannedTargetFailed(target, errText); (markErr != nil) {
		log.Println("scheduler: " + markErr.Error())
	}

	audit.Record(target.FractionId, 0, model.AuditActionApplyPlannedTargetFailed, errText)
}

// appendPlannedTargetError records a secondary failure (the chat message
// didn't go out) without undoing the target change that already
// succeeded — the target matters more than delivering the announcement.
func appendPlannedTargetError(target *model.PlannedTarget, errText string) {
	if markErr := repository.MarkPlannedTargetFailed(target, errText); (markErr != nil) {
		log.Println("scheduler: " + markErr.Error())
	}

	audit.Record(target.FractionId, 0, model.AuditActionApplyPlannedTargetFailed, errText)
}

func formatPlannedTargetDetail(target uint) string {
	switch (target) {
	case model.TargetNone:
		return "снята"
	case model.TargetDefense:
		return "🔐 защита"
	}

	fraction, err := repository.FindOneFractionById(target)
	if (err != nil) {
		return "?"
	}

	return fraction.Icon + " " + fraction.Name
}
