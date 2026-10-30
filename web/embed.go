// Package web intègre le frontend compilé (SPA React Router) dans le binaire.
//
// Le répertoire build/client doit exister au moment de la compilation Go :
// lancer `bun run build` dans web/ avant `go build`.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:build/client
var dist embed.FS

// Dist renvoie les fichiers du SPA, enracinés dans build/client
// (index.html à la racine).
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "build/client")
	if err != nil {
		panic(err) // chemin constant, vérifié par la directive go:embed
	}
	return sub
}
