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

	StaminaSevere      string
	StaminaRecommended string
	MoneyRecommended   string
	MoneySevere        string

	DefaultStaminaSevere      uint
	DefaultStaminaRecommended uint
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
		StaminaSevere:      uintLabel(fraction.StaminaSevereThreshold),
		StaminaRecommended: uintLabel(fraction.StaminaRecommendedThreshold),
		MoneyRecommended:   uintLabel(fraction.MoneyRecommendedThreshold),
		MoneySevere:        uintLabel(fraction.MoneySevereThreshold),

		DefaultStaminaSevere:      readiness.DefaultStaminaSevereThreshold,
		DefaultStaminaRecommended: readiness.DefaultStaminaRecommendedThreshold,
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

	staminaSevere, err1 := parseOptionalThreshold(r.FormValue("stamina_severe"))
	staminaRecommended, err2 := parseOptionalThreshold(r.FormValue("stamina_recommended"))
	moneyRecommended, err3 := parseOptionalThreshold(r.FormValue("money_recommended"))
	moneySevere, err4 := parseOptionalThreshold(r.FormValue("money_severe"))
	if (err1 != nil || err2 != nil || err3 != nil || err4 != nil) {
		http.Error(w, "Некорректное значение порога", http.StatusBadRequest)
		return
	}

	effStaminaSevere := effectiveOrDefault(staminaSevere, readiness.DefaultStaminaSevereThreshold)
	effStaminaRecommended := effectiveOrDefault(staminaRecommended, readiness.DefaultStaminaRecommendedThreshold)
	if (effStaminaSevere >= effStaminaRecommended) {
		http.Error(w, "Порог «мало выносливости» должен быть меньше порога «рекомендовано»", http.StatusBadRequest)
		return
	}

	effMoneyRecommended := effectiveOrDefault(moneyRecommended, readiness.DefaultMoneyRecommendedThreshold)
	effMoneySevere := effectiveOrDefault(moneySevere, readiness.DefaultMoneySevereThreshold)
	if (effMoneyRecommended >= effMoneySevere) {
		http.Error(w, "Порог «рекомендовано слить» должен быть меньше порога «много денег»", http.StatusBadRequest)
		return
	}

	if setErr := repository.SetFractionReadinessSettings(fraction, staminaSevere, staminaRecommended, moneyRecommended, moneySevere); (setErr != nil) {
		http.Error(w, "Не удалось сохранить настройки", http.StatusInternalServerError)
		return
	}

	audit.Record(permissions.FractionId(), permissions.ManagerId(), model.AuditActionSetFractionSettings, "обновлены настройки профиля")
	http.Redirect(w, r, "/panel/fraction", http.StatusFound)
}
