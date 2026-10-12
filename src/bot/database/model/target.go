package model

const TargetDefense uint = 7

func IsValidTarget(target uint) bool {
	return target == TargetDefense || (target >= 1 && target <= 6)
}
