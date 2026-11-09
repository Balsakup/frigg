package mail

import (
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/brand"
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
	content, err := Render("communication", Data{
		Association: "Club de Frigg", Subject: "Rappel : cotisation", Body: body,
		Footer: "Vous recevez ce message en tant qu'adhérent de Club de Frigg pour la saison 2026-2027.",
	})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := Wrap(content, Identity{Name: "Club de Frigg", Palette: brand.Default})
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
	if err == nil {
		other, err = Wrap(other, Identity{Name: "Club", Palette: brand.Default})
	}
	if err != nil || !strings.Contains(other.Text, "Merci de ne pas y répondre") || !strings.Contains(other.HTML, "Merci de ne pas y répondre") {
		t.Errorf("pied par défaut absent des autres gabarits (%v)", err)
	}
}

var contentTemplates = []string{
	"invitation", "password-reset", "email-verification", "membership-validated",
	"membership-refused", "membership-cancelled", "membership-paid", "communication", "test",
}

// Render ne produit que le contenu ; Wrap applique l'en-tête, les couleurs de
// la palette et le logo (010 › R6, SC-008).
func TestWrapAllTemplates(t *testing.T) {
	pale := brand.MustHex("#fff3b0")
	p := brand.Derive(&pale, nil)
	logo := &Attachment{Name: "logo.png", ContentType: "image/png", Data: []byte("PNG")}
	data := Data{Association: "Club", Link: "https://frigg.example/x#t", Member: "Léo", Season: "2026-2027",
		Amount: "40,00 €", Reference: "R", Reason: "Motif", Method: "chèque", Subject: "Objet", Body: "Texte"}
	for _, name := range contentTemplates {
		content, err := Render(name, data)
		if err != nil {
			t.Fatalf("%s : %v", name, err)
		}
		if strings.Contains(content.HTML, "<html") || strings.Contains(content.HTML, "Merci de ne pas y répondre") {
			t.Errorf("%s : Render produit plus que le contenu", name)
		}
		for _, withLogo := range []bool{false, true} {
			id := Identity{Name: "Les Sentiers du Val", Palette: p}
			if withLogo {
				id.Logo = logo
			}
			msg, err := Wrap(content, id)
			if err != nil {
				t.Fatalf("%s : %v", name, err)
			}
			if !strings.Contains(msg.HTML, "background:"+p.Brand.Hex()) || !strings.Contains(msg.HTML, "color:"+p.OnBrand.Hex()) {
				t.Errorf("%s : bandeau sans les couleurs de la palette", name)
			}
			if strings.Contains(msg.HTML, "#1f5278") || strings.Contains(msg.HTML, "<<") || strings.Contains(msg.HTML, "ZgotmplZ") {
				t.Errorf("%s : couleur codée en dur ou jeton restant\n%s", name, msg.HTML)
			}
			if strings.Contains(content.HTML, "{{.Link}}") || (data.Link != "" && strings.Contains(msg.HTML, `href="https://frigg.example/x#t" style="display:inline-block`) &&
				!strings.Contains(msg.HTML, "background:"+p.Brand.Hex()+";color:"+p.OnBrand.Hex())) {
				t.Errorf("%s : bouton sans les couleurs de la palette", name)
			}
			if !strings.Contains(msg.HTML, "Les Sentiers du Val") || !strings.HasPrefix(msg.Text, "Les Sentiers du Val") {
				t.Errorf("%s : nom absent", name)
			}
			if msg.FromName != "Les Sentiers du Val" {
				t.Errorf("%s : expéditeur %q", name, msg.FromName)
			}
			hasImg := strings.Contains(msg.HTML, "<img")
			if withLogo != hasImg || (withLogo && (!strings.Contains(msg.HTML, `src="cid:logo"`) || !strings.Contains(msg.HTML, `alt=""`) || len(msg.Inline) != 1 || msg.Inline[0].Name != "logo")) {
				t.Errorf("%s (logo %v) : image %v, pièces embarquées %d", name, withLogo, hasImg, len(msg.Inline))
			}
		}
	}
}

func TestWrapFooterAndPreview(t *testing.T) {
	content, _ := Render("test", Data{Link: "https://x"})
	content.Footer = "Pied propre"
	msg, _ := Wrap(content, Identity{Name: "Club", Palette: brand.Default})
	if !strings.Contains(msg.HTML, "Pied propre") || strings.Contains(msg.HTML, "Merci de ne pas y répondre") ||
		!strings.Contains(msg.Text, "Pied propre") {
		t.Errorf("pied : %s", msg.Text)
	}
	prev, _ := WrapPreview(content, Identity{Name: "Club", Palette: brand.Default, Logo: &Attachment{ContentType: "image/png", Data: []byte("PNG")}})
	if !strings.Contains(prev.HTML, `src="data:image/png;base64,UE5H"`) || len(prev.Inline) != 0 {
		t.Errorf("aperçu : %s", prev.HTML)
	}
}

// Modèles des événements (012). Un titre contenant du balisage est échappé
// dans le HTML (spec 012, Edge Cases).
func TestRenderEventTemplates(t *testing.T) {
	data := Data{
		Association: "Club de Frigg", Event: "Stage <b>printemps</b>", When: "samedi 12 décembre 2026 de 14 h à 17 h",
		Location: "Gymnase", Members: []string{"Lucas Martin", "Claire Martin"}, Amount: "70,00 €",
		Reference: "EVT-2627-0001, EVT-2627-0002", Link: "https://frigg.example/a-regler", Method: "chèque",
		PreviousWhen: "vendredi 11 décembre 2026 de 14 h à 17 h", PreviousLocation: "Salle", Message: "Salle indisponible.",
		Paid: true, Reason: "Paiement remboursé dans HelloAsso.",
	}
	cases := []struct {
		name     string
		subject  string
		contains []string
	}{
		{"registration-confirmed", "Inscription confirmée : Stage <b>printemps</b>", []string{"Lucas Martin", "Claire Martin", "70,00 €", "EVT-2627-0002", data.Link}},
		{"registration-paid", "Paiement reçu : Stage <b>printemps</b>", []string{"chèque", "EVT-2627-0001"}},
		{"registration-removed", "Inscription annulée : Stage <b>printemps</b>", []string{"remboursement", "Paiement remboursé"}},
		{"event-changed", "Changement : Stage <b>printemps</b>", []string{"Gymnase", "auparavant : Salle", "vendredi 11 décembre"}},
		{"event-cancelled", "Événement annulé : Stage <b>printemps</b>", []string{"Salle indisponible.", "remboursement"}},
		{"event-restored", "Événement rétabli : Stage <b>printemps</b>", []string{"Gymnase", "Claire Martin"}},
	}
	for _, c := range cases {
		msg, err := Render(c.name, data)
		if err != nil {
			t.Fatalf("%s : %v", c.name, err)
		}
		if msg.Subject != c.subject {
			t.Errorf("%s : objet %q", c.name, msg.Subject)
		}
		for _, want := range c.contains {
			if !strings.Contains(msg.Text, want) || !strings.Contains(msg.HTML, strings.ReplaceAll(want, "€", "€")) {
				t.Errorf("%s : %q absent du texte ou du HTML", c.name, want)
			}
		}
		if strings.Contains(msg.HTML, "<b>printemps</b>") {
			t.Errorf("%s : balisage du titre interprété dans le HTML", c.name)
		}
	}
	free, err := Render("registration-confirmed", Data{Association: "Club", Event: "AG", When: "w", Location: "l", Members: []string{"Lucas"}})
	if err != nil || strings.Contains(free.Text, "Montant") {
		t.Fatalf("inscription gratuite : %v %q", err, free.Text)
	}
}
