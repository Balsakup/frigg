// Package mail rend les e-mails, les met en file d'attente et les envoie via un
// worker périodique (constitution, E-mails).
package mail

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	htmltemplate "html/template"
	"regexp"
	"strings"
	texttemplate "text/template"

	"github.com/balsakup/frigg/internal/brand"
)

//go:embed templates
var templates embed.FS

// Message est un e-mail rendu, prêt à être mis en file.
type Message struct {
	Subject string
	Text    string
	HTML    string
	// Footer remplace la mention de pied par défaut ; appliqué par Wrap.
	Footer string
	// FromName est le nom d'affichage de l'expéditeur ; vide : adresse seule.
	FromName string
	// Attachments sont les pièces jointes, chargées par le worker au moment de
	// l'envoi (communications, specs/009-bulk-email).
	Attachments []Attachment
	// Inline sont les images embarquées (logo, cid:logo), ajoutées par Wrap.
	Inline []Attachment
}

// Attachment est une pièce jointe prête à être envoyée.
type Attachment struct {
	Name        string
	ContentType string
	Data        []byte
}

// Data est commune à tous les gabarits ; Link est le lien éventuel (à jeton
// pour l'authentification). Les autres champs servent aux e-mails d'adhésion
// (research.md 006 › R8) : ils sont déjà formatés.
type Data struct {
	Association string
	Link        string
	Member      string
	Season      string
	Amount      string
	Reference   string
	Reason      string
	Method      string
	// PaidOn : date de réception du paiement, « 3 novembre 2026 ».
	PaidOn string
	// Subject et Body : objet et texte saisis d'une communication.
	Subject string
	Body    string
	// Footer remplace la mention « Merci de ne pas y répondre » du pied.
	Footer string
	// Événements (012) : titre, date et heure en français, lieu, anciennes
	// valeurs d'un changement, inscrits concernés, message de l'association.
	Event            string
	When             string
	Location         string
	PreviousWhen     string
	PreviousLocation string
	Members          []string
	Message          string
	// Paid : au moins une des inscriptions concernées est payée.
	Paid bool
}

var htmlFuncs = htmltemplate.FuncMap{"paragraphs": Paragraphs}

// Render rend le contenu du gabarit name : name.txt.tmpl définit "subject"
// et "body", name.html.tmpl définit "body". Le gabarit commun (en-tête aux
// couleurs de l'association, logo, pied) est appliqué au moment de l'envoi par
// Wrap (specs/010-general-settings › R6).
func Render(name string, data Data) (Message, error) {
	txt, err := texttemplate.ParseFS(templates, "templates/"+name+".txt.tmpl")
	if err != nil {
		return Message{}, fmt.Errorf("gabarit texte %s : %w", name, err)
	}
	html, err := htmltemplate.New(name+".html.tmpl").Funcs(htmlFuncs).ParseFS(templates, "templates/partials.html.tmpl", "templates/"+name+".html.tmpl")
	if err != nil {
		return Message{}, fmt.Errorf("gabarit HTML %s : %w", name, err)
	}

	var subject, text, body bytes.Buffer
	if err := txt.ExecuteTemplate(&subject, "subject", data); err != nil {
		return Message{}, err
	}
	if err := txt.ExecuteTemplate(&text, "body", data); err != nil {
		return Message{}, err
	}
	if err := html.ExecuteTemplate(&body, "body", data); err != nil {
		return Message{}, err
	}
	return Message{
		Subject: strings.TrimSpace(subject.String()),
		Text:    strings.TrimSpace(text.String()),
		HTML:    strings.TrimSpace(body.String()),
		Footer:  data.Footer,
	}, nil
}

// Identity est l'identité visuelle appliquée aux e-mails à l'envoi.
type Identity struct {
	Name    string
	Palette brand.Palette
	// Logo est la variante e-mail du logo, nil sans logo.
	Logo *Attachment
}

// LogoContentID est l'identifiant de l'image embarquée du logo.
const LogoContentID = "logo"

var (
	layoutText = texttemplate.Must(texttemplate.ParseFS(templates, "templates/layout.txt.tmpl"))
	layoutHTML = htmltemplate.Must(htmltemplate.ParseFS(templates, "templates/layout.html.tmpl"))
)

type layoutData struct {
	Association string
	Footer      string
	Body        any
	HasLogo     bool
	LogoSrc     htmltemplate.URL
}

// Wrap applique le gabarit commun au contenu rendu par Render : nom de
// l'association en en-tête et en expéditeur, couleurs de la palette, logo
// embarqué (cid:logo) s'il existe, mention de pied.
func Wrap(msg Message, id Identity) (Message, error) {
	return wrap(msg, id, "cid:"+LogoContentID)
}

// WrapPreview est Wrap pour un aperçu affiché dans le navigateur : le logo est
// intégré en data: URI au lieu d'une image embarquée.
func WrapPreview(msg Message, id Identity) (Message, error) {
	src := ""
	if id.Logo != nil {
		src = "data:" + id.Logo.ContentType + ";base64," + base64.StdEncoding.EncodeToString(id.Logo.Data)
	}
	out, err := wrap(msg, id, src)
	out.Inline = nil
	return out, err
}

func wrap(msg Message, id Identity, logoSrc string) (Message, error) {
	d := layoutData{Association: id.Name, Footer: msg.Footer, HasLogo: id.Logo != nil, LogoSrc: htmltemplate.URL(logoSrc)}
	var text, html bytes.Buffer
	d.Body = msg.Text
	if err := layoutText.ExecuteTemplate(&text, "layout", d); err != nil {
		return Message{}, err
	}
	// Le contenu HTML a déjà été produit et échappé par Render.
	d.Body = htmltemplate.HTML(msg.HTML)
	if err := layoutHTML.ExecuteTemplate(&html, "layout", d); err != nil {
		return Message{}, err
	}
	p := id.Palette
	colors := strings.NewReplacer(
		"<<brand>>", p.Brand.Hex(), "<<on-brand>>", p.OnBrand.Hex(), "<<brand-ink>>", p.BrandInk.Hex(),
	)
	out := msg
	out.Text = strings.TrimSpace(text.String()) + "\n"
	out.HTML = colors.Replace(html.String())
	out.FromName = id.Name
	out.Inline = nil
	if id.Logo != nil {
		out.Inline = []Attachment{{Name: LogoContentID, ContentType: id.Logo.ContentType, Data: id.Logo.Data}}
	}
	return out, nil
}

var urlRe = regexp.MustCompile(`https?://[^\s<>"']+`)

// Paragraphs convertit un texte simple en HTML : chaque segment est échappé,
// les lignes vides séparent les paragraphes, les retours à la ligne
// deviennent des <br> et seules les URL http(s) deviennent des liens. Aucun
// balisage du texte n'est interprété (research.md 009 › R9).
func Paragraphs(text string) htmltemplate.HTML {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var b strings.Builder
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.Trim(para, "\n")
		if strings.TrimSpace(para) == "" {
			continue
		}
		b.WriteString(`<p style="margin:0 0 12px">`)
		for i, line := range strings.Split(para, "\n") {
			if i > 0 {
				b.WriteString("<br>")
			}
			writeLinked(&b, line)
		}
		b.WriteString("</p>\n")
	}
	return htmltemplate.HTML(b.String())
}

func writeLinked(b *strings.Builder, line string) {
	last := 0
	for _, m := range urlRe.FindAllStringIndex(line, -1) {
		start, end := m[0], m[1]
		// La ponctuation finale appartient à la phrase, pas au lien.
		for end > start && strings.ContainsRune(".,;:!?)]", rune(line[end-1])) {
			end--
		}
		b.WriteString(htmltemplate.HTMLEscapeString(line[last:start]))
		u := htmltemplate.HTMLEscapeString(line[start:end])
		b.WriteString(`<a href="` + u + `" rel="noopener" style="color:<<brand-ink>>">` + u + `</a>`)
		last = end
	}
	b.WriteString(htmltemplate.HTMLEscapeString(line[last:]))
}
