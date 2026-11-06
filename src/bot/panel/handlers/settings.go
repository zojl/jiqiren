package handlers

import (
	"net/http"
	"strconv"

	"jiqiren/bot/audit"
	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
	"jiqiren/bot/panel/auth"
	"jiqiren/bot/panel/templates"
	"jiqiren/bot/readiness"
)

type settingsFormData struct {
	Action string

	StaminaVerySevere  string
	StaminaSevere      string
	StaminaRecommended string
	StaminaIdeal       string
	MoneyRecommended   string
	MoneySevere        string

	DefaultStaminaVerySevere  uint
	DefaultStaminaSevere      uint
	DefaultStaminaRecommended uint
	DefaultStaminaIdeal       uint
	DefaultMoneyRecommended   uint
	DefaultMoneySevere        uint
}

func uintLabel(value *uint) string {
	if (value == nil) {
		return ""
	}

	return strconv.FormatUint(uint64(*value), 10)
}

func FractionSettingsForm(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionSettings()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, err := repository.FindOneFractionById(permissions.FractionId())
	if (err != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	templates.Render(w, r, "settings_form", settingsFormData{
		Action:             "/panel/fraction/settings",
		StaminaVerySevere:  uintLabel(fraction.StaminaVerySevereThreshold),
		StaminaSevere:      uintLabel(fraction.StaminaSevereThreshold),
		StaminaRecommended: uintLabel(fraction.StaminaRecommendedThreshold),
		StaminaIdeal:       uintLabel(fraction.StaminaIdealThreshold),
		MoneyRecommended:   uintLabel(fraction.MoneyRecommendedThreshold),
		MoneySevere:        uintLabel(fraction.MoneySevereThreshold),

		DefaultStaminaVerySevere:  readiness.DefaultStaminaVerySevereThreshold,
		DefaultStaminaSevere:      readiness.DefaultStaminaSevereThreshold,
		DefaultStaminaRecommended: readiness.DefaultStaminaRecommendedThreshold,
		DefaultStaminaIdeal:       readiness.DefaultStaminaIdealThreshold,
		DefaultMoneyRecommended:   readiness.DefaultMoneyRecommendedThreshold,
		DefaultMoneySevere:        readiness.DefaultMoneySevereThreshold,
	})
}

// parseOptionalThreshold treats an empty field as "clear the override, use
// the default" rather than an error.
func parseOptionalThreshold(raw string) (*uint, error) {
	if (raw == "") {
		return nil, nil
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if (err != nil) {
		return nil, err
	}

	result := uint(value)
	return &result, nil
}

func effectiveOrDefault(override *uint, fallback uint) uint {
	if (override == nil) {
		return fallback
	}

	return *override
}

func SetFractionReadinessSettings(w http.ResponseWriter, r *http.Request) {
	permissions := auth.PermissionsFromRequest(r)
	if (!permissions.CanSetFractionSettings()) {
		http.Error(w, "Access denied / 访问被拒绝", http.StatusForbidden)
		return
	}

	fraction, fractionErr := repository.FindOneFractionById(permissions.FractionId())
	if (fractionErr != nil) {
		http.Error(w, "Фракция не найдена", http.StatusInternalServerError)
		return
	}

	staminaVerySevere, err1 := parseOptionalThreshold(r.FormValue("stamina_very_severe"))
	staminaSevere, err2 := parseOptionalThreshold(r.FormValue("stamina_severe"))
	staminaRecommended, err3 := parseOptionalThreshold(r.FormValue("stamina_recommended"))
	staminaIdeal, err4 := parseOptionalThreshold(r.FormValue("stamina_ideal"))
	moneyRecommended, err5 := parseOptionalThreshold(r.FormValue("money_recommended"))
	moneySevere, err6 := parseOptionalThreshold(r.FormValue("money_severe"))
	if (err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil) {
		http.Error(w, "Некорректное значение порога", http.StatusBadRequest)
		return
	}

	effStaminaVerySevere := effectiveOrDefault(staminaVerySevere, readiness.DefaultStaminaVerySevereThreshold)
	effStaminaSevere := effectiveOrDefault(staminaSevere, readiness.DefaultStaminaSevereThreshold)
	effStaminaRecommended := effectiveOrDefault(staminaRecommended, readiness.DefaultStaminaRecommendedThreshold)
	effStaminaIdeal := effectiveOrDefault(staminaIdeal, readiness.DefaultStaminaIdealThreshold)
	if (!(effStaminaVerySevere < effStaminaSevere && effStaminaSevere < effStaminaRecommended && effStaminaRecommended < effStaminaIdeal)) {
		http.Error(w, "Пороги выносливости должны идти по возрастанию: очень мало < мало < рекомендовано < идеально", http.StatusBadRequest)
		return
	}

	effMoneyRecommended := effectiveOrDefault(moneyRecommended, readiness.DefaultMoneyRecommendedThreshold)
	effMoneySevere := effectiveOrDefault(moneySevere, readiness.DefaultMoneySevereThreshold)
	if (effMoneyRecommended >= effMoneySevere) {
		http.Error(w, "Порог «рекомендовано слить» должен быть меньше порога «много денег»", http.StatusBadRequest)
		return
	}

	if setErr := repository.SetFractionReadinessSettings(fraction, staminaVerySevere, staminaSevere, staminaRecommended, staminaIdeal, moneyRecommended, moneySevere); (setErr != nil) {
		http.Error(w, "Не удалось сохранить настройки", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionSetFractionSettings, "обновлены настройки профиля")
	http.Redirect(w, r, "/panel/fraction", http.StatusFound)
}
