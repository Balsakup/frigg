package mail

import (
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
	if err := m.From(s.cfg.From); err != nil {
		return fmt.Errorf("expéditeur invalide : %w", err)
	}
	if err := m.To(to); err != nil {
		return fmt.Errorf("destinataire invalide : %w", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(gomail.TypeTextPlain, msg.Text)
	m.AddAlternativeString(gomail.TypeTextHTML, msg.HTML)

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
