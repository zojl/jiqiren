package auth

import (
	"net/http"

	"jiqiren/bot/database/model"
)

// Permissions is the single place permission logic lives — handlers call its
// methods instead of re-deriving role checks. Zero-trust: every method
// defaults to false unless explicitly granted below. It wraps the user and
// manager rows fetched fresh from the database for this request (see
// RequireAuth/ViewerFromContext) rather than anything read out of the
// session cookie, so a revoked or changed role takes effect immediately.
// A plain player (nil manager) gets every role-gated permission denied.
type Permissions struct {
	userId uint
	manager *model.Manager
	hasProfile bool
}

func NewPermissions(viewer *Viewer) *Permissions {
	if (viewer == nil) {
		return &Permissions{}
	}

	permissions := &Permissions{manager: viewer.Manager, hasProfile: viewer.HasProfile}
	if (viewer.User != nil) {
		permissions.userId = viewer.User.Id
	}

	return permissions
}

func PermissionsFromRequest(r *http.Request) *Permissions {
	return NewPermissions(ViewerFromContext(r))
}

// ManagerId and FractionId are 0 for a plain player — only ever meaningful
// after a role-gated Can* check has already passed.
func (p *Permissions) ManagerId() uint {
	if (p.manager == nil) {
		return 0
	}

	return p.manager.Id
}

func (p *Permissions) FractionId() uint {
	if (p.manager == nil) {
		return 0
	}

	return p.manager.FractionId
}

// CanViewOwnProfile gates "Мой профиль" — open to anyone who has ever sent
// a profile, regardless of role. It only ever shows the viewer's own
// profile; other players stay behind CanViewPlayer.
func (p *Permissions) CanViewOwnProfile() bool {
	return p.hasProfile
}

// CanHideProfile gates hiding/restoring a profile ("не мой профиль") — only
// ever the sender's own, regardless of role; managers can't do it for
// their players.
func (p *Permissions) CanHideProfile(profileUserId uint) bool {
	return p.userId != 0 && p.userId == profileUserId
}

func (p *Permissions) hasRole(accountType model.AccountType) bool {
	return p.manager != nil && p.manager.AccountType != nil && *p.manager.AccountType == accountType
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
	return p.manager != nil && p.manager.SquadId != nil && *p.manager.SquadId == squadId
}

// CanViewDashboard gates the "Отряды" page (the fraction-wide squad list) —
// a plain squad lead has no fraction-wide view, they land on "Мой отряд"
// instead (see HomeSquadId).
func (p *Permissions) CanViewDashboard() bool {
	return p.isTeamlead() || p.isDeputy()
}

// CanViewFraction gates the "Команда" page. Viewing is bundled for
// teamlead/deputy alike; within the page, editing the target vs. the token
// is still gated separately (CanSetFractionTarget / CanSetFractionToken).
func (p *Permissions) CanViewFraction() bool {
	return p.isTeamlead() || p.isDeputy()
}

// OwnSquadId returns the manager's personally assigned squad, if any,
// regardless of role — unlike HomeSquadId this isn't restricted to plain
// squad leads, since a deputy can also be assigned to a specific squad.
func (p *Permissions) OwnSquadId() (uint, bool) {
	if (p.manager == nil || p.manager.SquadId == nil) {
		return 0, false
	}

	return *p.manager.SquadId, true
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

// CanViewPlayer gates the player page. Teamlead/deputy see every player in
// their own fraction; a squad lead sees a player only if their own squad is
// either the natural (profile-reported) or effective (override) squad —
// covering both captains while a player is mid-transfer between squads.
func (p *Permissions) CanViewPlayer(fractionId uint, naturalSquadId *uint, effectiveSquadId *uint) bool {
	if (p.manager == nil || p.manager.FractionId != fractionId) {
		return false
	}

	if (p.isTeamlead() || p.isDeputy()) {
		return true
	}

	if (!p.isSquadLead()) {
		return false
	}

	return (naturalSquadId != nil && p.ownsSquad(*naturalSquadId)) ||
		(effectiveSquadId != nil && p.ownsSquad(*effectiveSquadId))
}

func (p *Permissions) CanViewBalance() bool {
	return p.isTeamlead() || p.isDeputy()
}

func (p *Permissions) CanSetFractionTarget() bool {
	return p.isTeamlead() || p.isDeputy()
}

func (p *Permissions) CanSetFractionMessage() bool {
	return p.isTeamlead() || p.isDeputy()
}

func (p *Permissions) CanSetFractionSettings() bool {
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

// CanViewSchedule is fraction-wide for deputy (like CanViewSquad);
// CanManageSchedule below scopes deputy to their own squad instead.
func (p *Permissions) CanViewSchedule(squadId uint) bool {
	if (p.isTeamlead() || p.isDeputy()) {
		return true
	}

	return p.isSquadLead() && p.ownsSquad(squadId)
}

func (p *Permissions) CanManageSchedule(squadId uint) bool {
	if (p.isTeamlead()) {
		return true
	}

	return (p.isDeputy() || p.isSquadLead()) && p.ownsSquad(squadId)
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
// full dashboard, even if a SquadId happens to be set on their row.
func (p *Permissions) HomeSquadId() (uint, bool) {
	if (p.isTeamlead() || p.isDeputy() || !p.isSquadLead()) {
		return 0, false
	}

	if (p.manager.SquadId == nil) {
		return 0, false
	}

	return *p.manager.SquadId, true
}
