// Commande frigg : binaire unique de l'application (contracts/cli.md).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/balsakup/frigg/internal/cli"
)

const usage = `Utilisation : frigg [commande]

Commandes :
  serve         démarre le serveur (commande par défaut)
  openapi       écrit le schéma OpenAPI sur la sortie standard
  create-admin  crée un administrateur (et le nom de l'association si besoin)
  backup        sauvegarde la base : frigg backup <fichier>
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
