package public

// State est l'état relu dans la transaction de confirmation d'un dépôt
// (research.md › R7).
type State struct {
	// AccountFound : un compte existe pour l'adresse du dépôt.
	AccountFound bool
	// AccountActive : ce compte n'est pas désactivé.
	AccountActive bool
	// AccountManaged est le nombre de fiches gérées par ce compte.
	AccountManaged int
	// ExistingMember : une fiche identique (nom, prénom, date de naissance)
	// existe déjà parmi les fiches du compte ou celles suivies par lien.
	ExistingMember bool
	// ExistingInAccount : cette fiche est gérée par le compte de l'adresse ;
	// sinon, c'est une fiche suivie par lien.
	ExistingInAccount bool
	// ExistingActive : cette fiche a déjà une adhésion en attente ou validée
	// sur la saison, ou une inscription confirmée à l'événement.
	ExistingActive bool
	// TargetOpen : la cible accepte encore la demande (FR-010).
	TargetOpen bool
	// ClosedReason explique pourquoi la cible n'accepte plus la demande.
	ClosedReason string
}

// Outcome est l'issue d'une confirmation.
type Outcome string

const (
	// Closed : la cible n'accepte plus la demande ; rien n'est créé.
	Closed Outcome = "closed"
	// Duplicate : la demande existe déjà ; renvoi au suivi.
	Duplicate Outcome = "duplicate"
	// Account : la fiche est rattachée au compte de l'adresse (FR-033).
	Account Outcome = "account"
	// Tracking : la fiche reste sans compte, suivie par lien personnel.
	Tracking Outcome = "tracking"
)

// Decide choisit l'issue d'une confirmation. Un compte désactivé ou qui gère
// déjà MaxManagedMembers fiches ne reçoit pas de nouvelle fiche ; une fiche
// existante du compte peut toujours y recevoir la demande. Une fiche suivie
// par lien réutilisée reste suivie par lien.
func Decide(st State) Outcome {
	switch {
	case !st.TargetOpen:
		return Closed
	case st.ExistingMember && st.ExistingActive:
		return Duplicate
	case st.AccountFound && st.AccountActive &&
		(st.ExistingInAccount || (!st.ExistingMember && st.AccountManaged < MaxManagedMembers)):
		return Account
	}
	return Tracking
}
