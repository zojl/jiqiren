package database

import (
	"fmt"
	"log"
	"os"

	"jiqiren/bot/database/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init() {
	dsn := fmt.Sprintf(
		"host=db user=%s password=%s dbname=%s port=5432 sslmode=disable TimeZone=Europe/Moscow",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if (err != nil) {
		log.Fatal(err)
	}

	// A failed migration used to pass silently — and with it, missing
	// indexes that turn every profile lookup into a full table scan. Keep
	// starting (the schema is usually still workable), but say so loudly.
	for _, entity := range model.GetAllModels() {
		if err := db.AutoMigrate(entity); (err != nil) {
			log.Printf("database: AutoMigrate %T failed: %v", entity, err)
		}
	}

	model.CreateInitialValues(db)
}

func GetDb() *gorm.DB {
	return db
}
