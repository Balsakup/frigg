package cli

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/balsakup/frigg/internal/api"
)

// OpenAPI écrit le schéma OpenAPI 3.1 sur w, sans base ni configuration SMTP.
func OpenAPI(w io.Writer) error {
	humaAPI := api.New(http.NewServeMux(), api.Deps{})
	b, err := json.MarshalIndent(humaAPI.OpenAPI(), "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}
