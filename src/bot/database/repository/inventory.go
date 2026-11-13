package repository

import (
	"fmt"
	"time"

	"jiqiren/bot/database"
	"jiqiren/bot/database/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func FindInventoryDraft(userId uint) (*model.InventoryDraft, error) {
	db := database.GetDb()
	draft := model.InventoryDraft{}
	err := db.Where("user_id = ?", userId).First(&draft).Error
	if (err != nil) {
		return nil, fmt.Errorf("No inventory draft for user id: %d", userId)
	}

	return &draft, nil
}

// StartInventoryDraft discards any draft the user already had and starts
// a fresh one.
func StartInventoryDraft(userId uint, startedAt time.Time) (*model.InventoryDraft, error) {
	db := database.GetDb()
	draft := model.InventoryDraft{UserId: userId, StartedAt: startedAt}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := deleteInventoryDraftsOf(tx, userId); (err != nil) {
			return err
		}

		return tx.Create(&draft).Error
	})
	if (err != nil) {
		return nil, err
	}

	return &draft, nil
}

func SetInventoryDraftConfirming(draft *model.InventoryDraft, confirming bool) error {
	db := database.GetDb()
	if err := db.Model(draft).Update("confirming", confirming).Error; (err != nil) {
		return err
	}

	draft.Confirming = confirming
	return nil
}

func DeleteInventoryDraft(draft *model.InventoryDraft) error {
	db := database.GetDb()
	return db.Transaction(func(tx *gorm.DB) error {
		return deleteInventoryDraftsOf(tx, draft.UserId)
	})
}

func deleteInventoryDraftsOf(tx *gorm.DB, userId uint) error {
	if err := tx.Exec("DELETE FROM inventory_draft_messages WHERE draft_id IN (SELECT id FROM inventory_drafts WHERE user_id = ?)", userId).Error; (err != nil) {
		return err
	}

	return tx.Where("user_id = ?", userId).Delete(&model.InventoryDraft{}).Error
}

// SaveInventoryDraftMessage stores a collected screen; the same (kind,
// page) sent again replaces the earlier copy.
func SaveInventoryDraftMessage(message *model.InventoryDraftMessage) error {
	db := database.GetDb()
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "draft_id"}, {Name: "kind"}, {Name: "page"}},
		DoUpdates: clause.AssignmentColumns([]string{"text", "sent_at"}),
	}).Create(message).Error
}

func FindInventoryDraftMessages(draftId uint) ([]model.InventoryDraftMessage, error) {
	db := database.GetDb()
	messages := []model.InventoryDraftMessage{}
	err := db.Where("draft_id = ?", draftId).Order("sent_at ASC, id ASC").Find(&messages).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load inventory draft messages for draft id: %d", draftId)
	}

	return messages, nil
}

// ApplyInventoryDraft replaces the user's whole inventory with items and
// discards the draft, atomically.
func ApplyInventoryDraft(draft *model.InventoryDraft, items []model.InventoryItem, hasBackpack bool, analyzedAt time.Time) error {
	db := database.GetDb()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", draft.UserId).Delete(&model.InventoryItem{}).Error; (err != nil) {
			return err
		}

		if (len(items) > 0) {
			if err := tx.Create(&items).Error; (err != nil) {
				return err
			}
		}

		analysis := model.InventoryAnalysis{UserId: draft.UserId, AnalyzedAt: analyzedAt, HasBackpack: hasBackpack}
		err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"analyzed_at", "has_backpack"}),
		}).Create(&analysis).Error
		if (err != nil) {
			return err
		}

		return deleteInventoryDraftsOf(tx, draft.UserId)
	})
}

func FindInventoryItems(userId uint) ([]model.InventoryItem, error) {
	db := database.GetDb()
	items := []model.InventoryItem{}
	err := db.Where("user_id = ?", userId).Order("id ASC").Find(&items).Error
	if (err != nil) {
		return nil, fmt.Errorf("Could not load inventory for user id: %d", userId)
	}

	return items, nil
}

func FindInventoryAnalysis(userId uint) (*model.InventoryAnalysis, error) {
	db := database.GetDb()
	analysis := model.InventoryAnalysis{}
	err := db.Where("user_id = ?", userId).First(&analysis).Error
	if (err != nil) {
		return nil, fmt.Errorf("No inventory analysis for user id: %d", userId)
	}

	return &analysis, nil
}
