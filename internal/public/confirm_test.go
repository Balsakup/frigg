package public

import "testing"

func TestDecide(t *testing.T) {
	open := State{TargetOpen: true}
	account := State{TargetOpen: true, AccountFound: true, AccountActive: true, AccountManaged: 3}
	tests := []struct {
		name string
		st   State
		want Outcome
	}{
		{"cible fermée", State{TargetOpen: false, ClosedReason: "complet", AccountFound: true, AccountActive: true}, Closed},
		{"cible fermée et doublon actif", State{ExistingMember: true, ExistingActive: true}, Closed},
		{"sans compte", open, Tracking},
		{"compte actif", account, Account},
		{"compte désactivé", State{TargetOpen: true, AccountFound: true, AccountManaged: 3}, Tracking},
		{"compte à 20 fiches", State{TargetOpen: true, AccountFound: true, AccountActive: true, AccountManaged: 20}, Tracking},
		{"compte à 19 fiches", State{TargetOpen: true, AccountFound: true, AccountActive: true, AccountManaged: 19}, Account},
		{"fiche existante dans un compte plein", State{TargetOpen: true, AccountFound: true, AccountActive: true, AccountManaged: 20, ExistingMember: true, ExistingInAccount: true}, Account},
		{"fiche suivie existante, compte plein", State{TargetOpen: true, AccountFound: true, AccountActive: true, AccountManaged: 20, ExistingMember: true}, Tracking},
		{"fiche suivie existante, compte désactivé", State{TargetOpen: true, AccountFound: true, AccountManaged: 3, ExistingMember: true}, Tracking},
		{"fiche existante sans demande active", State{TargetOpen: true, ExistingMember: true}, Tracking},
		{"fiche existante avec demande active", State{TargetOpen: true, ExistingMember: true, ExistingActive: true}, Duplicate},
		{"doublon actif dans un compte", State{TargetOpen: true, AccountFound: true, AccountActive: true, ExistingMember: true, ExistingInAccount: true, ExistingActive: true}, Duplicate},
	}
	for _, tt := range tests {
		if got := Decide(tt.st); got != tt.want {
			t.Errorf("%s : Decide = %q, attendu %q", tt.name, got, tt.want)
		}
	}
}
