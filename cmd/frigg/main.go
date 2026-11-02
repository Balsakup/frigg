// Commande frigg : binaire unique de l'application (contracts/cli.md).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	// Base des fuseaux embarquée : la date du jour des saisons est celle
	// d'Europe/Paris, même sur un système sans tzdata (research.md 003 › R3).
	_ "time/tzdata"

	"github.com/balsakup/frigg/internal/cli"
)

const usage = `Utilisation : frigg [commande]

Commandes :
  serve         démarre le serveur (commande par défaut)
  openapi       écrit le schéma OpenAPI sur la sortie standard
  create-admin  crée un administrateur (et le nom de l'association si besoin)
  backup        sauvegarde la base : frigg backup <fichier>
  healthcheck   vérifie l'état de santé de l'instance locale (code 0 si saine)
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "serve":
		err = cli.Serve(ctx)
	case "openapi":
		err = cli.OpenAPI(os.Stdout)
	case "create-admin":
		err = cli.CreateAdminCommand(ctx, os.Args[2:], os.Stdout, os.Stderr)
	case "backup":
		err = cli.BackupCommand(ctx, os.Args[2:], os.Stdout)
	case "healthcheck":
		err = cli.HealthcheckCommand(ctx)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "commande inconnue : %s\n\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
