package model

import "time"

type AuditAction string

const (
	AuditActionLoginRequest        AuditAction = "login_request"
	AuditActionLogin               AuditAction = "login"
	AuditActionSetTarget           AuditAction = "set_target"
	AuditActionSetSquadToken       AuditAction = "set_squad_token"
	AuditActionSetFractionToken    AuditAction = "set_fraction_token"
	AuditActionSetFractionMessage  AuditAction = "set_fraction_message"
	AuditActionSetFractionSettings AuditAction = "set_fraction_settings"
	AuditActionRenameSquad         AuditAction = "rename_squad"
	AuditActionCreateSquad         AuditAction = "create_squad"
	AuditActionDeleteSquad         AuditAction = "delete_squad"
	AuditActionCreateManager       AuditAction = "create_manager"
	AuditActionUpdateManager       AuditAction = "update_manager"
	AuditActionDeleteManager       AuditAction = "delete_manager"
	AuditActionViewBalance         AuditAction = "view_balance"
	AuditActionViewSquad           AuditAction = "view_squad"
	AuditActionViewPlayer          AuditAction = "view_player"
	AuditActionViewActualBudget    AuditAction = "view_actual_budget"
	AuditActionViewBudgetChart     AuditAction = "view_budget_chart"
	AuditActionBindSquadChat       AuditAction = "bind_squad_chat"
	AuditActionCreateSchedule      AuditAction = "create_schedule"
	AuditActionUpdateSchedule      AuditAction = "update_schedule"
	AuditActionDeleteSchedule      AuditAction = "delete_schedule"
	AuditActionSetSquadOverride    AuditAction = "set_squad_override"
	AuditActionClearSquadOverride  AuditAction = "clear_squad_override"
	AuditActionFreezePlayer        AuditAction = "freeze_player"
	AuditActionCreatePlannedTarget AuditAction = "create_planned_target"
	AuditActionUpdatePlannedTarget AuditAction = "update_planned_target"
	AuditActionDeletePlannedTarget AuditAction = "delete_planned_target"
	AuditActionApplyPlannedTarget       AuditAction = "apply_planned_target"
	AuditActionApplyPlannedTargetFailed AuditAction = "apply_planned_target_failed"
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
