package handlers

import (
	"net/http"
	"time"

	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
)

// MyProfile is "Мой профиль": the logged-in user's own last profile plus
// best attack/defense snapshots, same as PlayerPage shows managers. Which
// profile to show comes solely from the session — there's no id in the
// URL — so it can never be used to look at anyone else. Not audited: the
// audit log records managers acting on fraction data, not players looking
// at themselves.
func MyProfile(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	user := auth.UserFromContext(r)
	if (user == nil || !permissions.CanViewOwnProfile()) {
		http.Error(w, "Профиль не найден — отправьте его боту", http.StatusNotFound)
		return
	}

	profile, profileErr := repository.FindLastProfileByVkId(user.VkId)
	if (profileErr != nil) {
		http.Error(w, "Профиль не найден — отправьте его боту", http.StatusNotFound)
		return
	}

	allFractions, fractionsErr := repository.FindAllFractions()
	if (fractionsErr != nil) {
		http.Error(w, "Ошибка загрузки фракций", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	naturalSquadId := profile.TeamId
	effectiveSquadId := profile.EffectiveSquadId(user.SquadOverrideId)

	data := playerPageData{
		VkId: user.VkId,
		Profile: buildPlayerProfileView(*profile, allFractions, naturalSquadId, effectiveSquadId, now),
	}

	if attackProfile, err := repository.FindBestAttackProfile(profile.UserId); (err == nil) {
		data.BestAttack = buildPlayerStatsSnapshotView(*attackProfile, now)
	}

	if defenseProfile, err := repository.FindBestDefenseProfile(profile.UserId); (err == nil) {
		data.BestDefense = buildPlayerStatsSnapshotView(*defenseProfile, now)
	}

	templates.Render(w, r, "player_page", data)
}
