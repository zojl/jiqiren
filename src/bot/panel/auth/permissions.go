package auth

import (
	"net/http"

	"jiqiren/bot/database/model"
)

// Permissions is the single place permission logic lives — handlers call its
// methods instead of re-deriving role checks. Zero-trust: every method
// defaults to false unless explicitly granted below.
type Permissions struct {
	claims *Claims
}

func NewPermissions(claims *Claims) *Permissions {
	return &Permissions{claims: claims}
}

func PermissionsFromRequest(r *http.Request) *Permissions {
	return NewPermissions(ClaimsFromContext(r))
}

func (p *Permissions) ManagerId() uint {
	return p.claims.ManagerId
}

func (p *Permissions) FractionId() uint {
	return p.claims.FractionId
}

func (p *Permissions) hasRole(accountType model.AccountType) bool {
	return p.claims != nil && p.claims.AccountType != nil && model.AccountType(*p.claims.AccountType) == accountType
}

func (p *Permissions) isTeamlead() bool {
	return p.hasRole(model.AccountTypeTeamlead)
}

func (p *Permissions) isDeputy() bool {
	return p.hasRole(model.AccountTypeDeputy)
}

func (p *Permissions) isSquadLead() bool {
	return p.hasRole(model.AccountTypeSquadLead)
}

func (p *Permissions) ownsSquad(squadId uint) bool {
	return p.claims != nil && p.claims.SquadId != nil && *p.claims.SquadId == squadId
}

func (p *Permissions) CanViewTargets() bool {
	if (p.isTeamlead() || p.isDeputy()) {
		return true
	}

	return p.isSquadLead() && p.claims.SquadId != nil
}

// CanViewSquad gates whether a specific squad's target is shown at all.
// Teamlead/deputy see every squad of the fraction; a plain squad lead only
// ever sees their own squad, never other squads' targets.
func (p *Permissions) CanViewSquad(squadId uint) bool {
	if (p.isTeamlead() || p.isDeputy()) {
		return true
	}

	return p.isSquadLead() && p.ownsSquad(squadId)
}

func (p *Permissions) CanViewBalance() bool {
	return p.isTeamlead() || p.isDeputy()
}

func (p *Permissions) CanSetFractionTarget() bool {
	return p.isTeamlead() || p.isDeputy()
}

func (p *Permissions) CanSetSquadTarget(squadId uint) bool {
	if (p.isTeamlead() || p.isDeputy()) {
		return true
	}

	return p.isSquadLead() && p.ownsSquad(squadId)
}

func (p *Permissions) CanSetFractionToken() bool {
	return p.isTeamlead()
}

// CanSetSquadToken and CanRenameSquad extend own-squad rights to a deputy
// whose SquadId happens to be set, layered on top of their fraction-wide
// deputy rights (user-confirmed amendment) — not just squad leads.
func (p *Permissions) CanSetSquadToken(squadId uint) bool {
	if (p.isTeamlead()) {
		return true
	}

	return (p.isSquadLead() || p.isDeputy()) && p.ownsSquad(squadId)
}

func (p *Permissions) CanManageManagers() bool {
	return p.isTeamlead()
}

func (p *Permissions) CanViewAuditLog() bool {
	return p.isTeamlead()
}

func (p *Permissions) CanManageSquads() bool {
	return p.isTeamlead()
}

func (p *Permissions) CanRenameSquad(squadId uint) bool {
	if (p.isTeamlead()) {
		return true
	}

	return (p.isSquadLead() || p.isDeputy()) && p.ownsSquad(squadId)
}

func (p *Permissions) CanManageSchedule(squadId uint) bool {
	if (p.isTeamlead()) {
		return true
	}

	return (p.isSquadLead() || p.isDeputy()) && p.ownsSquad(squadId)
}

func (p *Permissions) CanManageMembers(squadId uint) bool {
	if (p.isTeamlead()) {
		return true
	}

	return (p.isSquadLead() || p.isDeputy()) && p.ownsSquad(squadId)
}

// HomeSquadId returns the squad a plain squad lead should land on by
// default — they have no fraction-wide view, so their "home" is their own
// squad page rather than the dashboard. Teamlead/deputy always get the
// full dashboard, even if a SquadId happens to be set on their claims.
func (p *Permissions) HomeSquadId() (uint, bool) {
	if (p.isTeamlead() || p.isDeputy() || !p.isSquadLead()) {
		return 0, false
	}

	if (p.claims.SquadId == nil) {
		return 0, false
	}

	return *p.claims.SquadId, true
}
