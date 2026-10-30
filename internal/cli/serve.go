// Package cli implémente les sous-commandes du binaire frigg (contracts/cli.md).
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/server"
	"github.com/balsakup/frigg/web"
)

// Serve démarre le serveur : migrations, worker d'e-mails, HTTP.
func Serve(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.RequireServe(); err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	store, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := database.Migrate(ctx, store.WriteDB); err != nil {
		return err
	}

	limits := auth.NewLimits()
	go limits.RunPurge(ctx)

	worker := &mail.Worker{
		Queries: store.Write,
		Sender: mail.NewSMTPSender(mail.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword, TLS: cfg.SMTPTLS, From: cfg.MailFrom,
		}),
		Interval: cfg.MailPollInterval,
		Logger:   logger,
	}
	go worker.Run(ctx)

	h, err := server.NewHandler(api.Deps{
		Store:    store,
		Sessions: auth.NewSessionManager(store.WriteDB, cfg.CookieSecure),
		Limits:   limits,
		Config:   cfg,
		Logger:   logger,
	}, web.Dist())
	if err != nil {
		return fmt.Errorf("préparation du serveur : %w", err)
	}
	return server.Run(ctx, cfg.ListenAddr, h, logger)
}
