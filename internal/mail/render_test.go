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
		{"membership-refused", data, "Demande d'adhésion de Léo Martin refusée", []string{"Doublon de fiche", data.Link}, nil},
		{"membership-refused", Data{Association: "Club", Member: "Léo Martin", Reason: "Motif"},
			"Demande d'adhésion de Léo Martin refusée", []string{"Motif"}, []string{"lien personnel"}},
		// Fiche suivie par lien (016 FR-035) : adhésion gratuite avec un lien de suivi.
		{"membership-validated", Data{Association: "Club", Member: "Léo Martin", Season: "2026-2027", Link: "https://frigg.example/suivi#jeton=abc"},
			"Adhésion de Léo Martin validée", []string{"Aucun paiement n'est demandé", "https://frigg.example/suivi#jeton=abc", "lien personnel"}, []string{"Référence"}},
		{"membership-cancelled", data, "Adhésion de Léo Martin annulée", []string{"Doublon de fiche"}, nil},
		{"membership-paid", data, "Paiement de l'adhésion de Léo Martin reçu", []string{"virement", "ADH-2627-0001"}, []string{"de don"}},
		{"membership-paid", Data{Association: "Club", Member: "Léo Martin", Season: "2026-2027", Amount: "80,00 €", Donation: "35,00 €", Method: "virement"},
			"Paiement de l'adhésion de Léo Martin reçu", []string{"80,00", "Dont 35,00 € de don : merci pour votre soutien !"}, nil},
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
	"public-confirm", "tracking-link", "public-registration-confirmed",
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
	for _, donation := range []string{"", "5,00 €"} {
		d := data
		d.Donation = donation
		msg, err := Render("registration-paid", d)
		if err != nil {
			t.Fatal(err)
		}
		has := strings.Contains(msg.Text, "de don") && strings.Contains(msg.HTML, "de don")
		if has != (donation != "") || (donation != "" && !strings.Contains(msg.Text, "Dont 5,00 € de don")) {
			t.Errorf("registration-paid, don %q : %q", donation, msg.Text)
		}
	}
	free, err := Render("registration-confirmed", Data{Association: "Club", Event: "AG", When: "w", Location: "l", Members: []string{"Lucas"}})
	if err != nil || strings.Contains(free.Text, "Montant") {
		t.Fatalf("inscription gratuite : %v %q", err, free.Text)
	}
}

// Parcours publics (016) : confirmation d'un dépôt selon l'état du compte de
// l'adresse (research.md › R7) et lien personnel de suivi.
func TestRenderPublicTemplates(t *testing.T) {
	const (
		exists = "Un compte Frigg existe déjà pour cette adresse : après confirmation, votre demande apparaîtra dans votre espace membre."
		full   = "Votre compte gère déjà 20 fiches : cette demande ne pourra pas y être ajoutée et sera suivie par un lien personnel."
	)
	link := "https://frigg.example/confirmer-demande#jeton=abc"
	base := Data{Association: "Club de Frigg", Link: link, Member: "Léo Martin", Season: "2026-2027"}
	withAccount, fullAccount := base, base
	withAccount.AccountExists = true
	fullAccount.AccountFull = true
	event := Data{Association: "Club de Frigg", Link: link, Member: "Léo Martin", Event: "Stage d'escalade"}
	cases := []struct {
		name, label string
		data        Data
		subject     string
		contains    []string
		absent      []string
	}{
		{"public-confirm", "sans compte", base, "Confirmez votre demande",
			[]string{link, "48 heures", "adhésion de Léo Martin pour la saison 2026-2027", "Club de Frigg"}, []string{"compte", "20 fiches"}},
		{"public-confirm", "compte existant", withAccount, "Confirmez votre demande", []string{link, exists}, []string{"20 fiches"}},
		{"public-confirm", "compte plein", fullAccount, "Confirmez votre demande", []string{link, full}, []string{"existe déjà"}},
		{"public-confirm", "inscription", event, "Confirmez votre demande",
			[]string{"inscription de Léo Martin à « Stage d'escalade »"}, []string{"adhésion"}},
		{"tracking-link", "lien", Data{Association: "Club de Frigg", Link: "https://frigg.example/suivi#jeton=abc"}, "Votre lien de suivi",
			[]string{"https://frigg.example/suivi#jeton=abc", "30 jours", "Club de Frigg"}, nil},
		{"public-registration-confirmed", "payante", Data{Association: "Club de Frigg", Link: "https://frigg.example/suivi/inscriptions/7#jeton=abc",
			Member: "Léo Martin", Event: "Stage d'escalade", When: "samedi 12 décembre 2026 de 14 h à 17 h", Location: "Gymnase",
			Amount: "35,00 €", Reference: "EVT-2627-0001"}, "Inscription confirmée : Stage d'escalade",
			[]string{"https://frigg.example/suivi/inscriptions/7#jeton=abc", "Léo Martin", "samedi 12 décembre 2026 de 14 h à 17 h", "Gymnase",
				"35,00 €", "EVT-2627-0001", "30 jours"}, nil},
		{"public-registration-confirmed", "gratuite", Data{Association: "Club de Frigg", Link: "https://frigg.example/suivi/inscriptions/7#jeton=abc",
			Member: "Léo Martin", Event: "Portes ouvertes", When: "w", Location: "l"}, "Inscription confirmée : Portes ouvertes",
			[]string{"https://frigg.example/suivi/inscriptions/7#jeton=abc", "30 jours", "annuler"}, []string{"Montant", "Référence"}},
	}
	for _, c := range cases {
		msg, err := Render(c.name, c.data)
		if err != nil {
			t.Fatalf("%s (%s) : %v", c.name, c.label, err)
		}
		if msg.Subject != c.subject {
			t.Errorf("%s (%s) : objet %q", c.name, c.label, msg.Subject)
		}
		for _, want := range c.contains {
			if !strings.Contains(msg.Text, want) || !strings.Contains(msg.HTML, strings.ReplaceAll(want, "'", "&#39;")) {
				t.Errorf("%s (%s) : %q absent du texte ou du HTML\n%s\n%s", c.name, c.label, want, msg.Text, msg.HTML)
			}
		}
		for _, no := range c.absent {
			if strings.Contains(msg.Text, no) || strings.Contains(msg.HTML, no) {
				t.Errorf("%s (%s) : %q ne devrait pas figurer", c.name, c.label, no)
			}
		}
	}
}
