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

func TestRenderCommunication(t *testing.T) {
	body := "Bonjour,\r\n\r\n<script>alert(1)</script> et <b>gras</b>\nVoir https://exemple.test/a?b=1&c=2.\n\n\njavascript:alert(1)"
	msg, err := Render("communication", Data{
		Association: "Club de Frigg", Subject: "Rappel : cotisation", Body: body,
		Footer: "Vous recevez ce message en tant qu'adhérent de Club de Frigg pour la saison 2026-2027.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject != "Rappel : cotisation" {
		t.Errorf("objet %q", msg.Subject)
	}
	for _, want := range []string{"<script>alert(1)</script>", "https://exemple.test/a?b=1&c=2.", "pour la saison 2026-2027"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("texte : %q absent", want)
		}
	}
	for _, want := range []string{
		"&lt;script&gt;alert(1)&lt;/script&gt;", "&lt;b&gt;gras&lt;/b&gt;<br>Voir ",
		`<a href="https://exemple.test/a?b=1&amp;c=2" rel="noopener"`, "c=2</a>.",
		`<p style="margin:0 0 12px">Bonjour,</p>`, "javascript:alert(1)</p>", "pour la saison 2026-2027",
	} {
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("HTML : %q absent\n%s", want, msg.HTML)
		}
	}
	for _, no := range []string{"<script>", "<b>", `href="javascript`, "Merci de ne pas y répondre"} {
		if strings.Contains(msg.HTML, no) || (no == "Merci de ne pas y répondre" && strings.Contains(msg.Text, no)) {
			t.Errorf("%q ne devrait pas figurer", no)
		}
	}
	other, err := Render("membership-paid", Data{Association: "Club", Member: "Léo"})
	if err != nil || !strings.Contains(other.Text, "Merci de ne pas y répondre") || !strings.Contains(other.HTML, "Merci de ne pas y répondre") {
		t.Errorf("pied par défaut absent des autres gabarits (%v)", err)
	}
}
