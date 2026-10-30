package auth

// Access est le niveau d'accès minimal déclaré par chaque opération (principe III).
type Access int

const (
	Public Access = iota
	Member
	Admin
)

func (a Access) String() string {
	switch a {
	case Public:
		return "public"
	case Member:
		return "member"
	case Admin:
		return "admin"
	}
	return "inconnu"
}

// Rôles des comptes (valeurs stockées en base).
const (
	RoleMember = "member"
	RoleAdmin  = "admin"
)

// Allows est la fonction d'autorisation unique : admin ⊇ member.
func Allows(role string, a Access) bool {
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
