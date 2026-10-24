package model

import "time"

type AuditAction string

const (
	AuditActionLoginRequest    AuditAction = "login_request"
	AuditActionLogin           AuditAction = "login"
	AuditActionSetTarget       AuditAction = "set_target"
	AuditActionSetSquadToken   AuditAction = "set_squad_token"
	AuditActionSetFractionToken AuditAction = "set_fraction_token"
	AuditActionRenameSquad     AuditAction = "rename_squad"
	AuditActionCreateSquad     AuditAction = "create_squad"
	AuditActionDeleteSquad     AuditAction = "delete_squad"
	AuditActionCreateManager   AuditAction = "create_manager"
	AuditActionUpdateManager   AuditAction = "update_manager"
	AuditActionDeleteManager   AuditAction = "delete_manager"
	AuditActionViewBalance     AuditAction = "view_balance"
	AuditActionBindSquadChat   AuditAction = "bind_squad_chat"
	AuditActionCreateSchedule  AuditAction = "create_schedule"
	AuditActionUpdateSchedule  AuditAction = "update_schedule"
	AuditActionDeleteSchedule  AuditAction = "delete_schedule"
)

// VkId is stored directly (not joined from Manager) so it survives the
// manager being edited or deleted later.
type AuditLog struct {
	Id         uint `gorm:"primaryKey"`
	FractionId uint
	ManagerId  uint
	VkId       uint
	Action     AuditAction `gorm:"size:32"`
	Details    string      `gorm:"type:text"`
	CreatedAt  time.Time   `gorm:"autoCreateTime"`
}
