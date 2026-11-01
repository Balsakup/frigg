// Package mail rend les e-mails, les met en file d'attente et les envoie via un
// worker périodique (constitution, E-mails).
package mail

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
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
}

// Render rend le gabarit name : name.txt.tmpl définit "subject" et "body",
// name.html.tmpl définit "body" ; tous deux sont insérés dans le layout.
func Render(name string, data Data) (Message, error) {
	txt, err := texttemplate.ParseFS(templates, "templates/layout.txt.tmpl", "templates/"+name+".txt.tmpl")
	if err != nil {
		return Message{}, fmt.Errorf("gabarit texte %s : %w", name, err)
	}
	html, err := htmltemplate.ParseFS(templates, "templates/layout.html.tmpl", "templates/"+name+".html.tmpl")
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
