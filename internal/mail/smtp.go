package mail

import (
	"bytes"
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"
)

// Sender envoie un e-mail déjà rendu.
type Sender interface {
	Send(ctx context.Context, to string, msg Message) error
}

// SMTPConfig décrit le serveur d'envoi.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      string // starttls, tls ou none
	From     string
}

// SMTPSender envoie via go-mail.
type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender { return &SMTPSender{cfg: cfg} }

func (s *SMTPSender) Send(ctx context.Context, to string, msg Message) error {
	m := gomail.NewMsg()
	from := m.From
	if msg.FromName != "" {
		// Nom de l'association en expéditeur des communications (FR-016 de 009).
		from = func(addr string) error { return m.FromFormat(msg.FromName, addr) }
	}
	if err := from(s.cfg.From); err != nil {
		return fmt.Errorf("expéditeur invalide : %w", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("destinataire invalide : %w", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(gomail.TypeTextPlain, msg.Text)
	m.AddAlternativeString(gomail.TypeTextHTML, msg.HTML)
	for _, a := range msg.Inline {
		// Content-ID dérivé du nom : <logo>, référencé par cid:logo.
		if err := m.EmbedReader(a.Name, bytes.NewReader(a.Data), gomail.WithFileContentType(gomail.ContentType(a.ContentType))); err != nil {
			return fmt.Errorf("image embarquée : %w", err)
		}
	}
	for _, a := range msg.Attachments {
		if err := m.AttachReader(a.Name, bytes.NewReader(a.Data), gomail.WithFileContentType(gomail.ContentType(a.ContentType))); err != nil {
			return fmt.Errorf("pièce jointe : %w", err)
		}
	}

	opts := []gomail.Option{gomail.WithPort(s.cfg.Port)}
	switch s.cfg.TLS {
	case "tls":
		opts = append(opts, gomail.WithSSL())
	case "none":
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	if s.cfg.Username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(s.cfg.Username),
			gomail.WithPassword(s.cfg.Password),
		)
	}
	c, err := gomail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("client SMTP : %w", err)
	}
	return c.DialAndSendWithContext(ctx, m)
}
