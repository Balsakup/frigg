// Package mail rend les e-mails, les met en file d'attente et les envoie via un
// worker périodique (constitution, E-mails).
package mail

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"regexp"
	"strings"
	texttemplate "text/template"
)

//go:embed templates
var templates embed.FS

// Message est un e-mail rendu, prêt à être mis en file.
type Message struct {
	Subject string
	Text    string
	HTML    string
	// FromName est le nom d'affichage de l'expéditeur ; vide : adresse seule.
	FromName string
	// Attachments sont les pièces jointes, chargées par le worker au moment de
	// l'envoi (communications, specs/009-bulk-email).
	Attachments []Attachment
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
}

var htmlFuncs = htmltemplate.FuncMap{"paragraphs": Paragraphs}

// Render rend le gabarit name : name.txt.tmpl définit "subject" et "body",
// name.html.tmpl définit "body" ; tous deux sont insérés dans le layout.
func Render(name string, data Data) (Message, error) {
	txt, err := texttemplate.ParseFS(templates, "templates/layout.txt.tmpl", "templates/"+name+".txt.tmpl")
	if err != nil {
		return Message{}, fmt.Errorf("gabarit texte %s : %w", name, err)
	}
	html, err := htmltemplate.New("layout.html.tmpl").Funcs(htmlFuncs).ParseFS(templates, "templates/layout.html.tmpl", "templates/"+name+".html.tmpl")
	if err != nil {
		return Message{}, fmt.Errorf("gabarit HTML %s : %w", name, err)
	}

	var subject, text, body bytes.Buffer
	if err := txt.ExecuteTemplate(&subject, "subject", data); err != nil {
		return Message{}, err
	}
	if err := txt.ExecuteTemplate(&text, "layout", data); err != nil {
		return Message{}, err
	}
	if err := html.ExecuteTemplate(&body, "layout", data); err != nil {
		return Message{}, err
	}
	return Message{
		Subject: strings.TrimSpace(subject.String()),
		Text:    strings.TrimSpace(text.String()) + "\n",
		HTML:    body.String(),
	}, nil
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
		b.WriteString(`<a href="` + u + `" rel="noopener" style="color:#1d4ed8">` + u + `</a>`)
		last = end
	}
	b.WriteString(htmltemplate.HTMLEscapeString(line[last:]))
}
