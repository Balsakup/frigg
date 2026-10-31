package textfold

import "testing"

func TestFold(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"accents et casse", "Tarif réduit", "tarif reduit"},
		{"déjà replié", "tarif reduit", "tarif reduit"},
		{"espaces de bord et répétés", "  TARIF  RÉDUIT ", "tarif reduit"},
		{"majuscule accentuée", "Élève", "eleve"},
		{"cédille", "Ça", "ca"},
		{"vide", "", ""},
		{"espaces seuls", " \t ", ""},
		{"espace insécable", "Tarif réduit", "tarif reduit"},
		{"espace fine insécable", "Tarif réduit", "tarif reduit"},
		{"tiret conservé", "Plein-tarif", "plein-tarif"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Fold(tt.in); got != tt.want {
				t.Errorf("Fold(%q) = %q, attendu %q", tt.in, got, tt.want)
			}
		})
	}
	if Fold("Plein tarif") == Fold("Plein-tarif") {
		t.Error("« Plein tarif » et « Plein-tarif » ne doivent pas être confondus")
	}
}
