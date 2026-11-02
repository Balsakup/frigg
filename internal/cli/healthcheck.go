package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/config"
)

// healthcheckTimeout reste sous le délai de 5 s du HEALTHCHECK de l'image.
const healthcheckTimeout = 3 * time.Second

// HealthcheckCommand est la commande `frigg healthcheck` : elle interroge le
// point de santé de l'instance locale et sert de vérification de santé à
// l'image, qui n'a ni shell ni curl (contracts/cli.md de la spec 007).
func HealthcheckCommand(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	url, err := healthURL(cfg.ListenAddr)
	if err != nil {
		return err
	}
	return Healthcheck(ctx, url, os.Stderr)
}

// healthURL déduit l'URL du point de santé de l'adresse d'écoute : une adresse
// joker (hôte vide, 0.0.0.0, ::) est jointe par la boucle locale.
func healthURL(listenAddr string) (string, error) {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return "", fmt.Errorf("configuration invalide : FRIGG_LISTEN_ADDR invalide : %w", err)
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + api.HealthPath, nil
}

// Healthcheck renvoie nil si url répond 200 dans le délai imparti. stderr est
// réservé aux diagnostics : rien n'y est écrit en cas de succès.
func Healthcheck(ctx context.Context, url string, stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, healthcheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("instance injoignable : %w", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("instance en mauvaise santé : statut HTTP %d", res.StatusCode)
	}
	return nil
}
