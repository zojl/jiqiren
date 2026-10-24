package scheduler

import (
	"fmt"
	"time"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

// silentAfter: a profile this old says nothing useful about the player's
// current state, and pinging someone who's been gone a week is just
// noise — exclude them entirely rather than flag them for anything.
const silentAfter = 7 * 24 * time.Hour

// activeSquadProfiles returns each current member's latest profile —
// effective squad (model.Profile.EffectiveSquadId: a manual override wins
// over the profile's own TeamId) matching this one, skipping anyone
// who's left or gone silent for a week — plus a UserId → VkId lookup for
// building mentions.
func activeSquadProfiles(squad *model.Team, now time.Time) ([]model.Profile, map[uint]uint, error) {
	profiles, err := repository.FindLatestProfilesForTeam(squad.Id)
	if (err != nil) {
		return nil, nil, err
	}

	candidateIds := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		candidateIds = append(candidateIds, profile.UserId)
	}

	users, _ := repository.FindUsersByIds(candidateIds)
	vkIdByUserId := make(map[uint]uint, len(users))
	overrideByUserId := make(map[uint]*uint, len(users))
	for _, user := range users {
		vkIdByUserId[user.Id] = user.VkId
		overrideByUserId[user.Id] = user.SquadOverrideId
	}

	active := make([]model.Profile, 0, len(profiles))
	for _, profile := range profiles {
		effectiveSquadId := profile.EffectiveSquadId(overrideByUserId[profile.UserId])
		if (effectiveSquadId == nil || *effectiveSquadId != squad.Id) {
			continue
		}

		if (now.Sub(profile.MessageDate) > silentAfter) {
			continue
		}

		active = append(active, profile)
	}

	return active, vkIdByUserId, nil
}

func mention(vkId uint, name string) string {
	return fmt.Sprintf("[id%d|%s]", vkId, name)
}
