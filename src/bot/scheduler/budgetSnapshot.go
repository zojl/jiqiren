package scheduler

import (
	"log"
	"time"

	"jiqiren/bot/battletime"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/external/hackerwars"
)

const budgetSnapshotGridMinutes = 5

func tickBudgetSnapshots(nowMsk time.Time) {
	kind, due := budgetSnapshotKindDue(nowMsk)
	if (!due) {
		return
	}

	fractions, err := repository.FindAllFractions()
	if (err != nil) {
		log.Println("scheduler: " + err.Error())
		return
	}

	captured := nowMsk.Truncate(time.Minute)

	for _, fraction := range fractions {
		if (fraction.TargetToken == "") {
			continue
		}

		budget, budgetErr := hackerwars.GetTeamBudget(fraction.TargetToken)
		if (budgetErr != nil) {
			log.Println("scheduler: budget snapshot: " + budgetErr.Error())
			continue
		}

		snapshot := model.BudgetSnapshot{
			FractionId:         fraction.Id,
			Kind:               kind,
			CapturedAt:         captured,
			Budget:             budget.Info.Budget,
			ContributionsToday: budget.Info.Contributions.Today,
			ContributionsTotal: budget.Info.Contributions.Total,
			IncomeToday:        budget.Info.Income.Today,
			IncomeTotal:        budget.Info.Income.Total,
		}

		if saveErr := repository.CreateBudgetSnapshot(&snapshot); (saveErr != nil) {
			log.Println("scheduler: " + saveErr.Error())
		}
	}
}

// budgetSnapshotKindDue reports which kind of snapshot (if any) this minute
// tick corresponds to. The two battle anchors take priority since they
// never land on a 5-minute grid mark (:58 and :03). A missed anchor isn't
// backfilled later — an off-schedule value wouldn't represent the specific
// pre/post-battle moment it's meant to capture, unlike scheduled messages
// which tolerate catching up after downtime.
func budgetSnapshotKindDue(nowMsk time.Time) (model.BudgetSnapshotKind, bool) {
	preAnchor := battletime.PreBattleAnchor(nowMsk)
	if (nowMsk.Hour() == preAnchor.Hour() && nowMsk.Minute() == preAnchor.Minute()) {
		return model.BudgetSnapshotKindPreBattle, true
	}

	postAnchor := battletime.PostBattleAnchor(nowMsk)
	if (nowMsk.Hour() == postAnchor.Hour() && nowMsk.Minute() == postAnchor.Minute()) {
		return model.BudgetSnapshotKindPostBattle, true
	}

	if (nowMsk.Minute()%budgetSnapshotGridMinutes == 0) {
		return model.BudgetSnapshotKindInterval, true
	}

	return "", false
}
