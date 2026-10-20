package model

const TargetNone uint = 0
const TargetDefense uint = 7

func IsValidTarget(target uint) bool {
	return target == TargetNone || target == TargetDefense || (target >= 1 && target <= 6)
}
