package audit

import (
	"log"

	"jiqiren/bot/database/model"
	"jiqiren/bot/database/repository"
)

// Record writes one audit trail entry. VkId is resolved from the manager
// row at write time so the log stays self-contained even if the manager is
// later edited or deleted. A failure to write is logged but never bubbles
// up — audit logging must not block the action it's recording.
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
