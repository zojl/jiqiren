package plain


import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

var reportRegulars = [...]string {
	`Результаты битвы за (?P<Date>\d{2}\.\d{2}\.\d{4})`,
	`(?:Атака на |Команда )(?P<Target>[💠🚧🎭🈵🔱🇺🇸]{1,2})\S+ (?P<BattleResult>прошла успешно|отбила вашу атаку|отбила твою атаку)`, //this ones for attack
	`Твоя команда (?P<BattleResult>успешно отбила атаку.|не смогла отбить атаку.)`, //this ones for defence
	`Ты получил:(?:\nДеньги: ?(?P<RewardMoney>\d+)💵)?\nОпыт: (?P<RewardExperience>\d+)💡(?:\nVKCoin: (?P<RewardVkc>\d+\.?\d+)💸)?\nОсталось выносливости: 🔋(?P<Stamina>\d+)%`,
	`(?:Вы перехватили транзакцию \S+|Вы получили награду за защиту:)\nДеньги: (?P<TransactionMoney>\d+)💵\nОпыт: (?P<TransactionExperience>\d+)💡`,
	`(?:Твоя компания не отбила|Твоя команда не смогла отбить) атаку\.\nТы потерял:\nДеньги: (?P<LostMoney>\d+)💵`,
	`(?:Атака на |Команда )[💠🚧🎭🈵🔱🇺🇸]{1,2}\S+ (?:отбила вашу атаку|отбила твою атаку)\.\nТы потерял:\nДеньги: (?P<AttackLostMoney>\d+)💵`, //loss from a failed attack, distinct from LostMoney (a breached defense)
}

var compiledReportRegulars = make([]*regexp.Regexp, len(reportRegulars), len(reportRegulars))
var areReportRegularsCompiled = false

func HandleReport(messageText string, senderId int, messageTime time.Time) *ReportResponse {
	parsedReport := parseReport(messageText)
	response := ReportResponse{}

	user := repository.FindOneUserByVkId(senderId)
	if (user == nil) {
		response.IsUserExist = false
		response.IsStored = false
		return &response
	}

	response.IsUserExist = true

	date := parseDate(parsedReport.Date)

	reportsCount := repository.CountReportsByUserAndDate(senderId, date)
	if (reportsCount > 0) {
		response.IsFirst = false
		response.IsStored = false
		return &response
	}

	response.IsFirst = true

	target := getReportTarget(parsedReport)
	if (len(parsedReport.BattleResult) > 0) {
		response.IsParticipated = true
	}

	response.IsStored = storeReport(parsedReport, user, date, target, messageTime)
	if (!response.IsStored) {
		return &response
	}

	lastProfile, profileErr := repository.FindLastProfileByVkIdForBattleDate(uint(senderId), date)
	if (profileErr != nil) {
		return &response
	}
	response.IsProfileFound = true

	response.ContestResult = applyContest(parsedReport, lastProfile, senderId, date)
	return &response
}

func parseReport(messageText string) *ReportParseResult {
	var parsedReport ReportParseResult
	reflection := reflect.ValueOf(&parsedReport)

	for _, regular := range getCompiledReportRegulars() {
		allMatches := regular.FindAllStringSubmatch(messageText, -1)
		if (len(allMatches) == 0) {
			continue
		}
		match := allMatches[len(allMatches) - 1][1:]
		groups := regular.SubexpNames()[1:]

		for key, _ := range groups {
			field := reflection.Elem().FieldByName(groups[key])
			if !field.IsValid() {
				fmt.Println("invalid field " + groups[key])
				continue
			}

			value := match[key]
			if (len(value) == 0) {
				continue
			}

			field.SetString(value)
		}
	}

	return &parsedReport
}

func storeReport(parsedReport *ReportParseResult, user *model.User, date *time.Time, target *model.Fraction, messageTime time.Time) bool {
	var val uint64
	report := model.Report{}
	report.Target = target
	report.User = user
	report.BattleDate = date
	report.MessageDate = messageTime

	if len(parsedReport.RewardExperience) == 0 {
		report.IsSkipped = true
	}

	report.IsAttack = (target != nil)
	if (report.IsAttack) {
		report.IsSuccess = (parsedReport.BattleResult == "прошла успешно")
	} else {
		report.IsSuccess = (parsedReport.BattleResult == "успешно отбила атаку.")
	}

	val, _ = strconv.ParseUint(parsedReport.RewardMoney, 10, 64)
	report.RewardMoney = uint(val)
	val, _ = strconv.ParseUint(parsedReport.RewardExperience, 10, 64)
	report.RewardExperience = uint(val)
	val, _ = strconv.ParseUint(parsedReport.RewardVkc, 10, 64)
	report.Stamina = uint(val)
	val, _ = strconv.ParseUint(parsedReport.TransactionMoney, 10, 64)
	report.TransactionMoney = uint(val)
	val, _ = strconv.ParseUint(parsedReport.TransactionExperience, 10, 64)
	report.TransactionExperience = uint(val)
	val, _ = strconv.ParseUint(parsedReport.LostMoney, 10, 64)
	report.LostMoney = uint(val)
	val, _ = strconv.ParseUint(parsedReport.AttackLostMoney, 10, 64)
	report.AttackLostMoney = uint(val)
	report.RewardVkc, _ = strconv.ParseFloat(parsedReport.Stamina, 10)

	db := database.GetDb()
	db.Create(&report)

	return report.Id != 0
}

func parseDate(date string) *time.Time {
	layout := "02.01.2006"
	result, err := time.Parse(layout, date)
	if err != nil {
		return nil
	}
	return &result
}

func getReportTarget(report *ReportParseResult) *model.Fraction {
	if (len(report.Target) == 0) {
		return nil
	}

	target, _ := repository.FindOneFractionByIcon(report.Target)
	return target
}

func getCompiledReportRegulars () []*regexp.Regexp {

	if (areReportRegularsCompiled) {
		return compiledReportRegulars
	}

	for i, regular := range reportRegulars {
		compiledRegular, err := regexp.Compile(regular)
		if (err != nil) {
			fmt.Println(err)
		} else {
			compiledReportRegulars[i] = compiledRegular
		}
	}

	areReportRegularsCompiled = true
	return compiledReportRegulars
}

type ReportParseResult struct {
	Date string
	BattleResult string
	Target string
	RewardMoney string
	RewardExperience string
	RewardVkc string
	Stamina string
	TransactionMoney string
	TransactionExperience string
	LostMoney string
	AttackLostMoney string
}

type ReportResponse struct {
	IsFirst bool
	IsParticipated bool
	IsStored bool
	IsUserExist bool
	IsProfileFound bool
	ContestResult *ContestReport
}

func applyContest(parsedReport *ReportParseResult, lastProfile *model.Profile, senderId int, date *time.Time) *ContestReport {
	contestReport := ContestReport{}

	contest, contestErr := repository.FindOneContestByFractionIdTypeCodeAndDate(lastProfile.FractionId, "activity", date)
	if (contestErr != nil) {
		return nil;
	}

	pointsToday := repository.CountContestPointsByContestUserAndBattleDate(contest, &lastProfile.User, date)
	if (pointsToday > 0) {
		contestReport.Message = "Сегодня баллы за конкурс уже начислялись"
	}

	staminaVal, _ := strconv.ParseUint(parsedReport.Stamina, 10, 64)
	if (lastProfile.Stamina > uint(staminaVal)) {
		contestReport.Message = "Некорректный профиль до битвы, выносливость профиля ниже, чем выносливость отчёта, баллы не начислены"
	}

	points := uint64(lastProfile.Stamina * 4)
	lostMoney, _ := strconv.ParseUint(parsedReport.LostMoney, 10, 64)
	if (lostMoney >= points) {
		points = 0
	} else {
		points = points - lostMoney
	}

	contestReport.Points = uint(points)
	contestReport.Message = fmt.Sprintf("Начислены баллы в рамках конкурса «%s» за участие в битве: %d", contest.Name, points)
	storeContestPoints(lastProfile.UserId, contest, uint(points), date)

	return &contestReport
}

func storeContestPoints(userId uint, contest *model.Contest, points uint, date *time.Time) {
	contestPoints := model.ContestPoints{
		UserId: userId,
		Contest: contest,
		Points: int(points),
		BattleDate: date,
	}

	db := database.GetDb()
	db.Create(&contestPoints)
}

type ContestReport struct {
	Message string
	Points uint
}
