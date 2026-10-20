package audit

import (
	"log"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

// Record never fails the caller — a write error is only logged, since
// audit logging must not block the action it's recording.
func Record(fractionId uint, managerId uint, action model.AuditAction, details string) {
	var vkId uint
	if manager, err := repository.FindOneManagerById(managerId); err == nil {
		vkId = manager.VkId
	}

	entry := &model.AuditLog{
		FractionId: fractionId,
		ManagerId:  managerId,
		VkId:       vkId,
		Action:     action,
		Details:    details,
	}

	if err := repository.CreateAuditLog(entry); err != nil {
		log.Printf("audit: failed to record %s for manager %d: %v", action, managerId, err)
	}
}
