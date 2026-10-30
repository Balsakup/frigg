package auth

import "testing"

func TestAllows(t *testing.T) {
	tests := []struct {
		role string
		a    Access
		want bool
	}{
		{"", Public, true},
		{"", Member, false},
		{RoleMember, Member, true},
		{RoleMember, Admin, false},
		{RoleAdmin, Member, true},
		{RoleAdmin, Admin, true},
		{"inconnu", Member, false},
	}
	for _, tt := range tests {
		if got := Allows(tt.role, tt.a); got != tt.want {
			t.Errorf("Allows(%q, %v) = %v, attendu %v", tt.role, tt.a, got, tt.want)
		}
	}
}
