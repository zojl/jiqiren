package battletime

import "time"

// Moscow is a fixed +3 offset rather than a tzdata-backed location: Moscow
// hasn't observed DST since 2014, and runtime images aren't guaranteed to
// ship tzdata.
var Moscow = time.FixedZone("MSK", 3*60*60)

const hourUTC = 15 // 18:00 MSK

// Next returns the next battle's moment — today's 18:00 MSK if it hasn't
// happened yet, otherwise tomorrow's.
func Next(now time.Time) time.Time {
	nowUTC := now.UTC()
	battle := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), hourUTC, 0, 0, 0, time.UTC)
	if (!nowUTC.Before(battle)) {
		battle = battle.Add(24 * time.Hour)
	}

	return battle
}

// Last returns the most recent battle that has already happened.
func Last(now time.Time) time.Time {
	return Next(now).Add(-24 * time.Hour)
}

// Today returns the current MSK calendar day as a date-only instant (UTC
// midnight) — the representation callers persist as "which day something
// last happened" (e.g. ScheduledMessage.LastSentDate).
func Today(now time.Time) time.Time {
	nowMsk := now.In(Moscow)
	return time.Date(nowMsk.Year(), nowMsk.Month(), nowMsk.Day(), 0, 0, 0, 0, time.UTC)
}

// IsPastToday reports whether an hour:minute (MSK) has already gone by
// today.
func IsPastToday(hour uint, minute uint, now time.Time) bool {
	nowMsk := now.In(Moscow)
	sendMoment := time.Date(nowMsk.Year(), nowMsk.Month(), nowMsk.Day(), int(hour), int(minute), 0, 0, Moscow)

	return nowMsk.After(sendMoment)
}
