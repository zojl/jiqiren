package plain

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/readiness"
)

var regulars = [...]string {
	`(?P<Username>[a-zA-Z0-9а-яА-ЯёЁ ,_\-!?\$<>]{3,25})` +
		`\s\(` +
		`(?P<Lead>[▫🔸]{0,2})` +
		`(?P<Squad>[A-Z]{0,2})` +
		`\s?\|?\s?` +
		`(?P<Way>[⬛⬜🔲🔳]{0,2})` +
		`\S{0,2}` +
		`(?P<Fraction>Huǒqiáng|Aegis|V-hack|Phantoms|NetKings|NHS)` +
		`\)`,
	`💻: (?P<Level>\d+)`,
	`💻Уровень: (?P<Level>\d+)`,
	`💻: \d+💎(?P<Prestige>\d+)`,
	`💎Престиж: (?P<Prestige>\d+)`,
	`💡: (?P<Experience>\d+)/(?P<ExperienceLimit>\d+)`,
	`💡Опыт: (?P<Experience>\d+) из (?P<ExperienceLimit>\d+)`,
	`💵: (?P<Money>\d+)`,
	`💵Деньги: (?P<Money>\d+)`,
	`💸: (?P<Vkcoin>\d+\.?\d*)`,
	`💸VKCoin: (?P<Vkcoin>\d+\.?\d*)`,
	`🔘: (?P<Points>\d+)`,
	`🔘Поинты: (?P<Points>\d+)`,
	`💳: (?P<Bitcoins>\d+)`,
	`💳Биткоины: (?P<Bitcoins>\d+)`,
	`💿: (?P<Disks>\d+)`,
	`💿Радиодетали: (?P<Disks>\d+)`,
	`📄: (?P<Pages>\d+)`,
	`📄Страницы: (?P<Pages>\d+)`,
	`💽: (?P<Chips>\d+)`,
	`💽Микрочипы: (?P<Chips>\d+)`,
	`📑: (?P<Instructions>\d+)`,
	`📑Инструкции: (?P<Instructions>\d+)`,
	`📈: (?P<Stocks>\d+)`,
	`📈Акции: (?P<Stocks>\d+)`,
	`🔥: (?P<Motivation>\d+)/(?P<MotivationLimit>\d+)`,
	`🔥Мотивация: (?P<Motivation>\d+) из (?P<MotivationLimit>\d+)`,
	`📡: (?P<Practice>\d+)`,
	`📡Практика: (?P<Practice>\d+)`,
	`📡Практика: [\d]+\+[\d]+ \((?P<Practice>\d+)\)`,
	`💾: (?P<Theory>\d+)`,
	`💾Теория: (?P<Theory>\d+)`,
	`💾Теория: [\d]+\+[\d]+ \((?P<Theory>\d+)\)`,
	`📱: (?P<Cunning>\d+)`,
	`📱Хитрость: (?P<Cunning>\d+)`,
	`📱Хитрость: [\d]+\+[\d]+ \((?P<Cunning>\d+)\)`,
	`🔎: (?P<Wisdom>\d+)`,
	`🔎Мудрость: (?P<Wisdom>\d+)`,
	`🔎Мудрость: [\d]+\+[\d]+ \((?P<Wisdom>\d+)\)`,
	`🔋: (?P<Stamina>\d+)`,
	`🔋Выносливость: (?P<Stamina>\d+)`,
	`\n⚔: (?P<Target>[🔐💠🚧🎭🈵🔱🇺🇸]{1,2})`,
	`Занятие:\n[А-Яа-я ⚔]*(?P<Target>[🔐💠🚧🎭🈵🔱🇺🇸]{1,2})`,
	`(?:До 🛌:|🛌До сна осталось:) (?P<BeforeSleepHour>\d{1,2}) (?:ч\.|час.{0,2})(?:(?: и)? (?P<BeforeSleepMinute>\d{1,2}) (?:мин\.|минут.?))?`,
	`(?:До 🛌:|🛌До сна осталось:) (?P<BeforeSleepMinute>\d{1,2}) (?:мин\.|минут.?)`,
}

var compiledRegulars = make([]*regexp.Regexp, len(regulars), len(regulars))
var areRegularsCompiled = false

func HandleProfile(messageText string, senderId int, messageTime time.Time) *ProfileResponse {
	parsedProfile := parseProfile(messageText)
	return storeParsedProfile(parsedProfile, senderId, messageTime)
}

func storeParsedProfile(parsedProfile *ProfileParseResult, senderId int, messageTime time.Time) *ProfileResponse {
	user := repository.FindOrCreateUserByVkId(senderId)
	fraction := getFraction(parsedProfile)
	team := getTeam(parsedProfile, fraction)
	target := getTarget(parsedProfile, fraction)
	profile, isProfileInserted := insertProfile(parsedProfile, user, fraction, team, target, messageTime)
	return validateInsertedProfile(isProfileInserted, parsedProfile, profile, team, fraction)
}

func getFraction(parsedProfile *ProfileParseResult) *model.Fraction {
	db := database.GetDb()
	fraction := model.Fraction{}
	err := db.Unscoped().Where("Name = ?", parsedProfile.Fraction).First(&fraction).Error
	if (err != nil) {
		log.Fatal("Erroneous fraction name: " + parsedProfile.Fraction)
	}

	return &fraction
}

func getTeam(parsedProfile *ProfileParseResult, fraction *model.Fraction) *model.Team {
	db := database.GetDb()
	team := model.Team{}
	if (len(parsedProfile.Squad) == 2) {
		err := db.Unscoped().Where("Code = ?", parsedProfile.Squad).First(&team).Error

		if (err != nil) {
			team.Code = parsedProfile.Squad
			team.Fraction = *fraction
			db.Create(&team)
		}
	}

	return &team
}

func getTarget(parsedProfile *ProfileParseResult, userFraction *model.Fraction) *model.Fraction {
	if (len(parsedProfile.Target) == 0) {
		var fraction model.Fraction
		return &fraction
	}

	if (parsedProfile.Target == "🔐") {
		return userFraction
	}

	fraction, _ := repository.FindOneFractionByIcon(parsedProfile.Target)

	return fraction
}

func insertProfile(
	parsedProfile *ProfileParseResult,
	user *model.User,
	fraction *model.Fraction,
	team *model.Team,
	target *model.Fraction,
	messageTime time.Time,
) (model.Profile, bool) {
	var (
		profile model.Profile
		val uint64
	)

	val, _ = strconv.ParseUint(parsedProfile.Level, 10, 64)
	profile.Level = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Prestige, 10, 64)
	profile.Prestige = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Experience, 10, 64)
	profile.Experience = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.ExperienceLimit, 10, 64)
	profile.ExperienceLimit = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Money, 10, 64)
	profile.Money = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Points, 10, 64)
	profile.Points = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Bitcoins, 10, 64)
	profile.Bitcoins = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Disks, 10, 64)
	profile.Disks = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Pages, 10, 64)
	profile.Pages = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Chips, 10, 64)
	profile.Chips = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Instructions, 10, 64)
	profile.Instructions = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Stocks, 10, 64)
	profile.Stocks = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Motivation, 10, 64)
	profile.Motivation = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.MotivationLimit, 10, 64)
	profile.MotivationLimit = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Practice, 10, 64)
	profile.Practice = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Theory, 10, 64)
	profile.Theory = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Cunning, 10, 64)
	profile.Cunning = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Wisdom, 10, 64)
	profile.Wisdom = uint(val)
	val, _ = strconv.ParseUint(parsedProfile.Stamina, 10, 64)
	profile.Stamina = uint(val)

	profile.Fraction = *fraction
	profile.User = *user
	profile.MessageDate = messageTime

	profile.Username = parsedProfile.Username
	profile.Vkcoin, _ = strconv.ParseFloat(parsedProfile.Vkcoin, 10)
	profile.Lead = getLeadType(parsedProfile.Lead)
	profile.Way = getWayType(parsedProfile.Way)

	if (team.Id > 0) {
		profile.Team = *team
	} else {
		profile.TeamId = nil
	}

	if (len(parsedProfile.BeforeSleepHour) > 0 || len(parsedProfile.BeforeSleepMinute) > 0) {
		hours, _ := strconv.ParseUint(parsedProfile.BeforeSleepHour, 10, 64)
		minutes, _ := strconv.ParseUint(parsedProfile.BeforeSleepMinute, 10, 64)
		sleepAt := messageTime.Add(time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute)
		profile.SleepAt = &sleepAt
	}

	if (target.Id > 0) {
		profile.Target = *target
	} else {
		profile.TargetId = nil
	}

	db := database.GetDb()
	db.Create(&profile)

	return profile, profile.Id != 0
}

func getLeadType(icon string) uint {
	if (icon == "🔸") {
		return 2
	}

	if (icon == "▫") {
		return 1
	}

	return 0
}

func getWayType(icon string) uint {
	if (len(icon) == 0) {
		return 0
	}

	if (icon == "⬛") {
		return 1
	}

	if (icon == "⬜") {
		return 2
	}

	if (icon == "🔲") {
		return 3
	}

	if (icon == "🔳") {
		return 4
	}

	return 0
}

func parseProfile(messageText string) *ProfileParseResult {
	var parsedProfile ProfileParseResult
	reflection := reflect.ValueOf(&parsedProfile)

	for _, regular := range getCompiledRegulars() {
		allMatches := regular.FindAllStringSubmatch(messageText, -1)
		if (len(allMatches) == 0) {
			continue
		}
		match := allMatches[len(allMatches) - 1][1:]
		groups := regular.SubexpNames()[1:]

		for key, _ := range groups {
			field := reflection.Elem().FieldByName(groups[key])
			if !field.IsValid() {
				log.Printf("invalid field %s, %s, %s", key, groups[key], match[key])
			}

			value := match[key]
			if (len(value) == 0) {
				continue
			}

			field.SetString(value)
		}
	}

	return &parsedProfile
}

func validateInsertedProfile(isProfileInserted bool, parsedProfile *ProfileParseResult, profile model.Profile, team *model.Team, fraction *model.Fraction) *ProfileResponse {
	response := ProfileResponse{
		IsInserted: isProfileInserted,
		Messages: make([]string, 0, 5),
	}

	thresholds := readiness.ThresholdsFor(fraction)

	if (profile.TargetId == nil) {
		response.Messages = append(response.Messages, "‼️Не выбрана цель на следующую битву! Не забудьте выбрать цель перед битвой.")
	} else if (!readiness.TargetMatches(profile, team, fraction)) {
		response.Messages = append(response.Messages, "❗Выбрана неправильная цель на следующую битву! Не забудьте сменить цель перед битвой.")
	}

	if (len(parsedProfile.Money) > 0) {
		money, _ := strconv.ParseUint(parsedProfile.Money, 10, 64)
		if (money >= uint64(thresholds.MoneyVerySevere)) {
			response.Messages = append(response.Messages, "‼️Очень много денег! Обязательно слейте деньги в акции перед битвой.")
		} else if (money >= uint64(thresholds.MoneySevere)) {
			response.Messages = append(response.Messages, "❗Много денег! Не забудьте слить деньги в акции перед битвой.")
		} else if (money > uint64(thresholds.MoneyRecommended)) {
			response.Messages = append(response.Messages, fmt.Sprintf("⚠️Рекомендуется слить деньги до значения не выше %d перед битвой.", thresholds.MoneyRecommended))
		}
	}

	if (len(parsedProfile.Stamina) > 0) {
		stamina, _ := strconv.ParseUint(parsedProfile.Stamina, 10, 64)
		if (stamina < uint64(thresholds.StaminaVerySevere)) {
			response.Messages = append(response.Messages, "‼️Очень мало выносливости! Обязательно пополните выносливость перед битвой.")
		} else if (stamina < uint64(thresholds.StaminaSevere)) {
			response.Messages = append(response.Messages, "❗Мало выносливости! Не забудьте пополнить выносливость перед битвой.")
		} else if (stamina < uint64(thresholds.StaminaRecommended)) {
			response.Messages = append(response.Messages, fmt.Sprintf("⚠️Рекомендуется пополнить выносливость до %d перед битвой.", thresholds.StaminaIdeal))
		}
	}

	if (len(parsedProfile.BeforeSleepHour) + len(parsedProfile.BeforeSleepMinute) > 0) {
		hour, _ := strconv.ParseUint(parsedProfile.BeforeSleepHour, 10, 64)
		if (hour < 24) {
			response.Messages = append(response.Messages, "ℹ️Меньше 24 часов до сна, не забудьте отправить персонажа спать после битвы.")
		}
	}

	if (len(response.Messages) == 0) {
		response.Messages = append(response.Messages, "✅Вы отлично подготовлены к битве!😎")
	}

	return &response
}

func getCompiledRegulars () []*regexp.Regexp {

	if (areRegularsCompiled) {
		return compiledRegulars
	}

	for i, regular := range regulars {
		compiledRegular, err := regexp.Compile(regular)
		if (err != nil) {
			fmt.Println(err)
		} else {
			compiledRegulars[i] = compiledRegular
		}
	}

	areRegularsCompiled = true
	return compiledRegulars
}

type ProfileParseResult struct {
	Username string
	Squad string
	Fraction string
	Lead string
	Way string
	Level string
	Prestige string
	Experience string
	ExperienceLimit string
	Money string
	Vkcoin string
	Points string
	Bitcoins string
	Disks string
	Pages string
	Chips string
	Instructions string
	Stocks string
	Motivation string
	MotivationLimit string
	Practice string
	Theory string
	Cunning string
	Wisdom string
	Stamina string
	Target string
	BeforeSleepHour string
	BeforeSleepMinute string
}

type ProfileResponse struct {
	IsInserted bool
	Messages []string
}
