// Package mailtest fournit un faux expéditeur pour les tests.
package mailtest

import (
	"context"
	"errors"
	"regexp"
	"sync"

	"github.com/balsakup/frigg/internal/mail"
)

// Sent est un message capturé.
type Sent struct {
	To  string
	Msg mail.Message
}

// Sender capture les messages au lieu de les envoyer ; Fail simule une panne SMTP.
type Sender struct {
	mu   sync.Mutex
	sent []Sent
	Fail bool
}

func (s *Sender) Send(_ context.Context, to string, msg mail.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Fail {
		return errors.New("serveur SMTP indisponible")
	}
	s.sent = append(s.sent, Sent{To: to, Msg: msg})
	return nil
}

// Messages renvoie une copie des messages capturés.
func (s *Sender) Messages() []Sent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Sent(nil), s.sent...)
}

// Last renvoie le dernier message envoyé à to.
func (s *Sender) Last(to string) (Sent, bool) {
	msgs := s.Messages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].To == to {
			return msgs[i], true
		}
	}
	return Sent{}, false
}

var tokenRe = regexp.MustCompile(`#([A-Za-z0-9_-]{43})`)

// Token extrait le jeton du lien contenu dans le message.
func Token(msg mail.Message) string {
	m := tokenRe.FindStringSubmatch(msg.Text)
	if m == nil {
		return ""
	}
	return m[1]
}
