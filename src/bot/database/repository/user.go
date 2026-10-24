package repository

import (
	"fmt"

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