package auth

// Access est le niveau d'accès minimal déclaré par chaque opération (principe III).
type Access int

const (
	Public Access = iota
	Member
	Admin
	// Link est l'accès par lien personnel de suivi (amendement 1.9.0 de la
	// constitution). Il est hors de la hiérarchie des rôles : aucun rôle ne
	// l'accorde, seul accessMiddleware le fait quand la session porte un lien
	// valide (research.md 016 › R1).
	Link
)

func (a Access) String() string {
	switch a {
	case Public:
		return "public"
	case Member:
		return "member"
	case Admin:
		return "admin"
	case Link:
		return "link"
	}
	return "inconnu"
}

// Rôles des comptes (valeurs stockées en base).
const (
	RoleMember = "member"
	RoleAdmin  = "admin"
)

// Allows est la fonction d'autorisation unique : admin ⊇ member. Elle refuse
// toujours Link, qui ne dépend pas du rôle.
func Allows(role string, a Access) bool {
	if a == Link {
		return false
	}
	return rank(role) >= int(a)
}

func rank(role string) int {
	switch role {
	case RoleAdmin:
		return int(Admin)
	case RoleMember:
		return int(Member)
	}
	return int(Public)
}
