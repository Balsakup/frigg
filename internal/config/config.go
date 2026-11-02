// Package config charge la configuration de Frigg depuis l'environnement,
// éventuellement complété par un fichier .env.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config regroupe toute la configuration de l'instance (research.md › R10).
type Config struct {
	DatabasePath     string
	DataDir          string
	ListenAddr       string
	BaseURL          string
	CookieSecure     bool
	TrustedProxies   []string
	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPPassword     string
	SMTPTLS          string
	MailFrom         string
	MailPollInterval time.Duration
	LogLevel         slog.Level
	APIDocs          bool
	HelloAsso        HelloAsso
}

// Environnements HelloAsso (research.md 008 › R2).
const (
	HelloAssoSandbox    = "sandbox"
	HelloAssoProduction = "production"
)

// HelloAsso regroupe la configuration facultative du paiement par carte. Le
// secret ne doit jamais être journalisé, stocké en base ni renvoyé au
// navigateur (constitution, principe V).
type HelloAsso struct {
	ClientID     string
	ClientSecret string
	Organization string
	Environment  string
	// APIURL remplace l'hôte HelloAsso ; réservée aux tests (faux HelloAsso).
	APIURL string
}

// HelloAssoConfigured indique si les identifiants HelloAsso sont renseignés.
func (c Config) HelloAssoConfigured() bool {
	return c.HelloAsso.ClientID != "" && c.HelloAsso.ClientSecret != "" && c.HelloAsso.Organization != ""
}

// Load lit le fichier .env s'il existe (sans écraser l'environnement), puis les
// variables FRIGG_*, en appliquant les valeurs par défaut.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("lecture du fichier .env : %w", err)
	}

	var errs []error
	cfg := Config{
		DatabasePath: env("FRIGG_DATABASE_PATH", "./data/frigg.db"),
		DataDir:      env("FRIGG_DATA_DIR", "./data/files"),
		ListenAddr:   env("FRIGG_LISTEN_ADDR", ":8080"),
		BaseURL:      strings.TrimRight(os.Getenv("FRIGG_BASE_URL"), "/"),
		SMTPHost:     os.Getenv("FRIGG_SMTP_HOST"),
		SMTPUsername: os.Getenv("FRIGG_SMTP_USERNAME"),
		SMTPPassword: os.Getenv("FRIGG_SMTP_PASSWORD"),
		SMTPTLS:      env("FRIGG_SMTP_TLS", "starttls"),
		MailFrom:     os.Getenv("FRIGG_MAIL_FROM"),
	}

	var err error
	if cfg.CookieSecure, err = envBool("FRIGG_COOKIE_SECURE", true); err != nil {
		errs = append(errs, err)
	}
	if cfg.APIDocs, err = envBool("FRIGG_API_DOCS", false); err != nil {
		errs = append(errs, err)
	}
	if cfg.SMTPPort, err = strconv.Atoi(env("FRIGG_SMTP_PORT", "587")); err != nil {
		errs = append(errs, fmt.Errorf("FRIGG_SMTP_PORT doit être un nombre"))
	}
	if cfg.MailPollInterval, err = time.ParseDuration(env("FRIGG_MAIL_POLL_INTERVAL", "30s")); err != nil || cfg.MailPollInterval <= 0 {
		errs = append(errs, fmt.Errorf("FRIGG_MAIL_POLL_INTERVAL doit être une durée positive (ex. 30s)"))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(env("FRIGG_LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("FRIGG_LOG_LEVEL doit valoir debug, info, warn ou error"))
	}
	switch cfg.SMTPTLS {
	case "starttls", "tls", "none":
	default:
		errs = append(errs, fmt.Errorf("FRIGG_SMTP_TLS doit valoir starttls, tls ou none"))
	}
	cfg.HelloAsso = HelloAsso{
		ClientID:     strings.TrimSpace(os.Getenv("FRIGG_HELLOASSO_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(os.Getenv("FRIGG_HELLOASSO_CLIENT_SECRET")),
		Organization: strings.TrimSpace(os.Getenv("FRIGG_HELLOASSO_ORGANIZATION")),
		Environment:  env("FRIGG_HELLOASSO_ENVIRONMENT", HelloAssoSandbox),
		APIURL:       strings.TrimRight(strings.TrimSpace(os.Getenv("FRIGG_HELLOASSO_API_URL")), "/"),
	}
	errs = append(errs, cfg.HelloAsso.validate()...)
	for p := range strings.SplitSeq(os.Getenv("FRIGG_TRUSTED_PROXIES"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			cfg.TrustedProxies = append(cfg.TrustedProxies, p)
		}
	}

	if len(errs) > 0 {
		return Config{}, invalid(errs)
	}
	return cfg, nil
}

// RequireServe vérifie les variables obligatoires pour la commande serve.
func (c Config) RequireServe() error {
	var errs []error
	for _, v := range []struct{ name, value string }{
		{"FRIGG_BASE_URL", c.BaseURL},
		{"FRIGG_SMTP_HOST", c.SMTPHost},
		{"FRIGG_MAIL_FROM", c.MailFrom},
	} {
		if v.value == "" {
			errs = append(errs, fmt.Errorf("%s est obligatoire", v.name))
		}
	}
	if c.HelloAssoConfigured() && c.HelloAsso.APIURL == "" && c.BaseURL != "" && !strings.HasPrefix(c.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("FRIGG_BASE_URL doit commencer par https:// lorsque HelloAsso est configuré"))
	}
	if len(errs) > 0 {
		return invalid(errs)
	}
	return nil
}

// validate applique la règle « tout ou rien » des identifiants HelloAsso.
func (h HelloAsso) validate() []error {
	var errs []error
	switch h.Environment {
	case HelloAssoSandbox, HelloAssoProduction:
	default:
		errs = append(errs, fmt.Errorf("FRIGG_HELLOASSO_ENVIRONMENT doit valoir sandbox ou production"))
	}
	vars := []struct{ name, value string }{
		{"FRIGG_HELLOASSO_CLIENT_ID", h.ClientID},
		{"FRIGG_HELLOASSO_CLIENT_SECRET", h.ClientSecret},
		{"FRIGG_HELLOASSO_ORGANIZATION", h.Organization},
	}
	var set string
	for _, v := range vars {
		if v.value != "" {
			set = v.name
			break
		}
	}
	if set == "" {
		return errs
	}
	for _, v := range vars {
		if v.value == "" {
			errs = append(errs, fmt.Errorf("%s est obligatoire lorsque %s est renseigné", v.name, set))
		}
	}
	return errs
}

func invalid(errs []error) error {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return fmt.Errorf("configuration invalide : %s", strings.Join(msgs, " ; "))
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func envBool(name string, fallback bool) (bool, error) {
	v := os.Getenv(name)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s doit valoir true ou false", name)
	}
	return b, nil
}
