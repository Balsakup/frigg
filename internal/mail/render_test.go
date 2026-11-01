package mail

import (
	"strings"
	"testing"
)

func TestRenderMembershipTemplates(t *testing.T) {
	data := Data{
		Association: "Club de Frigg",
		Link:        "https://frigg.example/adhesions/42",
		Member:      "Léo Martin",
		Season:      "2026-2027",
		Amount:      "40,00 €",
		Reference:   "ADH-2627-0001",
		Reason:      "Doublon de fiche",
		Method:      "virement",
	}
	cases := []struct {
		name     string
		data     Data
		subject  string
		contains []string
		absent   []string
	}{
		{"membership-validated", data, "Adhésion de Léo Martin validée",
			[]string{"2026-2027", "ADH-2627-0001", data.Link, "40,00"}, []string{"Aucun paiement"}},
		{"membership-validated", Data{Association: "Club", Member: "Léo Martin", Season: "2026-2027"},
			"Adhésion de Léo Martin validée", []string{"Aucun paiement n'est demandé"}, []string{"Référence"}},
		{"membership-refused", data, "Demande d'adhésion de Léo Martin refusée", []string{"Doublon de fiche"}, nil},
		{"membership-cancelled", data, "Adhésion de Léo Martin annulée", []string{"Doublon de fiche"}, nil},
		{"membership-paid", data, "Paiement de l'adhésion de Léo Martin reçu", []string{"virement", "ADH-2627-0001"}, nil},
	}
	for _, c := range cases {
		msg, err := Render(c.name, c.data)
		if err != nil {
			t.Fatalf("%s : %v", c.name, err)
		}
		if msg.Subject != c.subject {
			t.Errorf("%s : objet %q", c.name, msg.Subject)
		}
		for _, want := range c.contains {
			if !strings.Contains(msg.Text, want) || !strings.Contains(msg.HTML, want) {
				t.Errorf("%s : %q absent du texte ou du HTML", c.name, want)
			}
		}
		for _, no := range c.absent {
			if strings.Contains(msg.Text, no) {
				t.Errorf("%s : %q ne devrait pas figurer", c.name, no)
			}
		}
	}
}
