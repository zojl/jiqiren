package auth

import (
	"testing"

	"jiqiren/bot/database/model"
)

const ownFraction = 4
const ownSquad = 5
const otherSquad = 6

func role(accountType model.AccountType) *model.AccountType {
	return &accountType
}

func squad(id uint) *uint {
	return &id
}

// viewers covers every kind of person who can be logged in: a plain player
// (no manager row), each role with and without an assigned squad, and a
// manager row with no role at all.
func viewers() map[string]*Viewer {
	user := &model.User{Id: 100}
	manager := func(accountType *model.AccountType, squadId *uint) *Viewer {
		return &Viewer{User: user, HasProfile: true, Manager: &model.Manager{Id: 7, FractionId: ownFraction, AccountType: accountType, SquadId: squadId}}
	}

	return map[string]*Viewer{
		"player":             {User: user, HasProfile: true},
		"teamlead":           manager(role(model.AccountTypeTeamlead), nil),
		"deputy":             manager(role(model.AccountTypeDeputy), nil),
		"deputy with squad":  manager(role(model.AccountTypeDeputy), squad(ownSquad)),
		"squad lead":         manager(role(model.AccountTypeSquadLead), squad(ownSquad)),
		"squad lead no squad": manager(role(model.AccountTypeSquadLead), nil),
		"no role":            manager(nil, squad(ownSquad)),
	}
}

// granted lists exactly who gets a permission; everyone else must be
// denied (zero-trust — see Permissions).
func checkMatrix(t *testing.T, name string, check func(*Permissions) bool, granted ...string) {
	t.Helper()

	allowed := map[string]bool{}
	for _, who := range granted {
		allowed[who] = true
	}

	for who, viewer := range viewers() {
		if got := check(NewPermissions(viewer)); (got != allowed[who]) {
			t.Errorf("%s for %s: got %v, want %v", name, who, got, allowed[who])
		}
	}
}

func TestFractionWidePermissions(t *testing.T) {
	leads := []string{"teamlead", "deputy", "deputy with squad"}

	checkMatrix(t, "CanViewDashboard", (*Permissions).CanViewDashboard, leads...)
	checkMatrix(t, "CanViewFraction", (*Permissions).CanViewFraction, leads...)
	checkMatrix(t, "CanViewBalance", (*Permissions).CanViewBalance, leads...)
	checkMatrix(t, "CanSetFractionTarget", (*Permissions).CanSetFractionTarget, leads...)
	checkMatrix(t, "CanSetFractionMessage", (*Permissions).CanSetFractionMessage, leads...)
	checkMatrix(t, "CanSetFractionSettings", (*Permissions).CanSetFractionSettings, leads...)

	checkMatrix(t, "CanSetFractionToken", (*Permissions).CanSetFractionToken, "teamlead")
	checkMatrix(t, "CanManageManagers", (*Permissions).CanManageManagers, "teamlead")
	checkMatrix(t, "CanViewAuditLog", (*Permissions).CanViewAuditLog, "teamlead")
	checkMatrix(t, "CanManageSquads", (*Permissions).CanManageSquads, "teamlead")
}

func TestSquadPermissions(t *testing.T) {
	onOwn := func(check func(*Permissions, uint) bool) func(*Permissions) bool {
		return func(p *Permissions) bool { return check(p, ownSquad) }
	}
	onOther := func(check func(*Permissions, uint) bool) func(*Permissions) bool {
		return func(p *Permissions) bool { return check(p, otherSquad) }
	}

	// Viewing and setting targets: fraction-wide for teamlead/deputy, own
	// squad only for a squad lead.
	for name, check := range map[string]func(*Permissions, uint) bool{
		"CanViewSquad": (*Permissions).CanViewSquad,
		"CanSetSquadTarget": (*Permissions).CanSetSquadTarget,
		"CanViewSchedule": (*Permissions).CanViewSchedule,
	} {
		checkMatrix(t, name+"(own)", onOwn(check), "teamlead", "deputy", "deputy with squad", "squad lead")
		checkMatrix(t, name+"(other)", onOther(check), "teamlead", "deputy", "deputy with squad")
	}

	// Managing a squad: teamlead anywhere, a deputy or squad lead only the
	// squad they're assigned to.
	for name, check := range map[string]func(*Permissions, uint) bool{
		"CanSetSquadToken": (*Permissions).CanSetSquadToken,
		"CanRenameSquad": (*Permissions).CanRenameSquad,
		"CanManageMembers": (*Permissions).CanManageMembers,
		"CanManageSchedule": (*Permissions).CanManageSchedule,
	} {
		checkMatrix(t, name+"(own)", onOwn(check), "teamlead", "deputy with squad", "squad lead")
		checkMatrix(t, name+"(other)", onOther(check), "teamlead")
	}
}

func TestCanViewPlayer(t *testing.T) {
	inOwnSquad := func(p *Permissions) bool { return p.CanViewPlayer(ownFraction, squad(ownSquad), squad(ownSquad)) }
	inOtherSquad := func(p *Permissions) bool { return p.CanViewPlayer(ownFraction, squad(otherSquad), squad(otherSquad)) }
	// Mid-transfer: the profile still says the other squad, an override
	// already puts them in ours — and the reverse.
	transferringIn := func(p *Permissions) bool { return p.CanViewPlayer(ownFraction, squad(otherSquad), squad(ownSquad)) }
	transferringOut := func(p *Permissions) bool { return p.CanViewPlayer(ownFraction, squad(ownSquad), squad(otherSquad)) }
	otherFraction := func(p *Permissions) bool { return p.CanViewPlayer(ownFraction+1, squad(ownSquad), squad(ownSquad)) }

	leads := []string{"teamlead", "deputy", "deputy with squad"}
	checkMatrix(t, "in own squad", inOwnSquad, append(leads, "squad lead")...)
	checkMatrix(t, "in other squad", inOtherSquad, leads...)
	checkMatrix(t, "transferring in", transferringIn, append(leads, "squad lead")...)
	checkMatrix(t, "transferring out", transferringOut, append(leads, "squad lead")...)
	checkMatrix(t, "other fraction", otherFraction)
}

func TestHomeAndOwnSquad(t *testing.T) {
	homeIsOwnSquad := func(p *Permissions) bool {
		squadId, ok := p.HomeSquadId()
		return ok && squadId == ownSquad
	}
	checkMatrix(t, "HomeSquadId", homeIsOwnSquad, "squad lead")

	hasOwnSquad := func(p *Permissions) bool {
		squadId, ok := p.OwnSquadId()
		return ok && squadId == ownSquad
	}
	checkMatrix(t, "OwnSquadId", hasOwnSquad, "deputy with squad", "squad lead", "no role")
}

func TestOwnProfilePermissions(t *testing.T) {
	everyone := []string{"player", "teamlead", "deputy", "deputy with squad", "squad lead", "squad lead no squad", "no role"}
	checkMatrix(t, "CanViewOwnProfile", (*Permissions).CanViewOwnProfile, everyone...)
	checkMatrix(t, "CanHideProfile(own)", func(p *Permissions) bool { return p.CanHideProfile(100) }, everyone...)
	checkMatrix(t, "CanHideProfile(someone else's)", func(p *Permissions) bool { return p.CanHideProfile(101) })

	noProfile := NewPermissions(&Viewer{User: &model.User{Id: 100}})
	if (noProfile.CanViewOwnProfile()) {
		t.Fatal("without any profile there's no «Мой профиль»")
	}
}

func TestNoViewer(t *testing.T) {
	permissions := NewPermissions(nil)
	if (permissions.CanHideProfile(0) || permissions.CanViewOwnProfile() || permissions.CanViewDashboard()) {
		t.Fatal("no viewer — nothing is granted")
	}

	if (permissions.ManagerId() != 0 || permissions.FractionId() != 0) {
		t.Fatal("ids must be zero, not panic, without a manager")
	}
}
