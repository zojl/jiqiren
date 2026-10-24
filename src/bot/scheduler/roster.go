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

// activeSquadProfiles returns each current member's latest profile
// (skipping anyone who's left or gone silent for a week) plus a
// UserId → VkId lookup for building mentions.
func activeSquadProfiles(squad *model.Team, now time.Time) ([]model.Profile, map[uint]uint, error) {
	profiles, err := repository.FindLatestProfilesForTeam(squad.Id)
	if (err != nil) {
		return nil, nil, err
	}

	active := make([]model.Profile, 0, len(profiles))
	userIds := make([]uint, 0, len(profiles))
	for _, profile := range profiles {
		if (profile.TeamId == nil || *profile.TeamId != squad.Id) {
			continue
		}

		if (now.Sub(profile.MessageDate) > silentAfter) {
			continue
		}

		active = append(active, profile)
		userIds = append(userIds, profile.UserId)
	}

	users, _ := repository.FindUsersByIds(userIds)
	vkIdByUserId := make(map[uint]uint, len(users))
	for _, user := range users {
		vkIdByUserId[user.Id] = user.VkId
	}

	return active, vkIdByUserId, nil
}

func mention(vkId uint, name string) string {
	return fmt.Sprintf("[id%d|%s]", vkId, name)
}
