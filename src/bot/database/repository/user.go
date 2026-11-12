package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"
)

func FindUsersByIds(ids []uint) ([]model.User, error) {
	db := database.GetDb()
	users := []model.User{}
	err := db.Unscoped().Where("id IN ?", ids).Find(&users).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load users")
	}

	return users, nil
}

func FindOneUserByVkId(vkId int) *model.User {
	db := database.GetDb()
	user := model.User{}
	err := db.Unscoped().Where("vk_id = ?", vkId).First(&user).Error
	if (err != nil) {
		return nil
	}

	return &user
}

func SetUserSquadOverride(user *model.User, squadId *uint) error {
	db := database.GetDb()
	return db.Model(user).Update("SquadOverrideId", squadId).Error
}

func SetUserFrozen(user *model.User, frozen bool) error {
	db := database.GetDb()
	return db.Model(user).Update("Frozen", frozen).Error
}

func FindOrCreateUserByVkId(vkId int) *model.User  {
	user := FindOneUserByVkId(vkId)
	if (user != nil) {
		return user
	}

	newUser := model.User{}
	newUser.VkId = uint(vkId)

	db := database.GetDb()
	db.Create(&newUser)

	return &newUser
}

func FindOneUserById(id uint) (*model.User, error) {
	db := database.GetDb()
	user := model.User{}
	err := db.Unscoped().Where("id = ?", id).First(&user).Error
	if (err != nil) {
		return nil, fmt.Errorf("No user with id: %d", id)
	}

	return &user, nil
}

func FindOneUserByLoginCode(code string) (*model.User, error) {
	db := database.GetDb()
	user := model.User{}
	err := db.Unscoped().Where("login_code = ? AND login_code_expires_at > ?", code, time.Now()).First(&user).Error
	if (err != nil) {
		return nil, fmt.Errorf("Invalid or expired login code")
	}

	return &user, nil
}

// UserHasProfiles reports whether the user has ever sent at least one
// profile — the condition for the panel's "Мой профиль" section.
func UserHasProfiles(userId uint) bool {
	db := database.GetDb()
	var count int64
	err := db.Model(&model.Profile{}).Where("user_id = ?", userId).Limit(1).Count(&count).Error

	return err == nil && count > 0
}

func SetUserLoginCode(user *model.User, code string, expiresAt time.Time) error {
	db := database.GetDb()
	return db.Model(user).Updates(map[string]interface{}{
		"login_code": code,
		"login_code_expires_at": expiresAt,
	}).Error
}

func ClearUserLoginCode(user *model.User) error {
	db := database.GetDb()
	return db.Model(user).Updates(map[string]interface{}{
		"login_code": nil,
		"login_code_expires_at": nil,
	}).Error
}

func SetUserTheme(user *model.User, mode string, scheme string) error {
	db := database.GetDb()
	return db.Model(user).Updates(map[string]interface{}{
		"theme_mode": mode,
		"color_scheme": scheme,
	}).Error
}

// SetUserSessionId persists a new session identifier and, since JWT
// issuance needs it immediately after, also updates it on the in-memory
// user — a plain db.Model(...).Updates(map) call wouldn't write it back.
func SetUserSessionId(user *model.User, sessionId string) error {
	db := database.GetDb()
	if err := db.Model(user).Update("session_id", sessionId).Error; (err != nil) {
		return err
	}

	user.SessionId = &sessionId
	return nil
}
