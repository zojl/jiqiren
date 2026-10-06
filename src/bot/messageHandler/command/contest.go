package command

import (
    "fmt"
    "time"

    "jiqiren/bot/messageHandler/reply"
    "jiqiren/bot/database/model"
    "jiqiren/bot/database/repository"
    
    "github.com/SevereCloud/vksdk/v2/api"
    "github.com/SevereCloud/vksdk/v2/events"
)

type Contest struct {
    Message *events.MessageNewObject
    Vk *api.VK
}

func (params Contest) Handle() {
    vkId := params.Message.Message.FromID
    replyParams := reply.MakeParams(&params.Message.Message, params.Vk)
    
    profile, profileErr := repository.FindLastProfileByVkId(uint(vkId))
    if (profileErr != nil) {
        replyParams.Reply("Не найден профиль.")
        return
    }
    
    currentTime := time.Now()
    contest, contestErr := repository.FindOneContestByFractionIdAndDate(profile.FractionId, &currentTime)
    if (contestErr != nil) {
        replyParams.Reply("В твоей фракции сейчас нет конкурсов.")
        return
    }
    
    userMock := model.User{
        Id: profile.UserId,
    }

    var pointsSum int64 = 0
    
    contestType, _ := (repository.FindOneContestTypeById(contest.TypeId));
    if (contestType.Code == "project") {
        pointsSum = repository.CountContestProjectMessagesByVkIdAndContestId(contest, &userMock)
    } else {
        pointsSum = repository.SumPointsByContestAndUser(contest, &userMock)
    }

    replyParams.Reply(fmt.Sprintf("Твой текущий счёт в конкурсе «%s»: %d", contest.Name, pointsSum))
}