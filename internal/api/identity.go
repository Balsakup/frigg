package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/brand"
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
)

// Identité visuelle de l'association (specs/010-general-settings › R5, R8).
// get-identity et get-identity-logo sont publiques, dans le seul périmètre de
// l'amendement 1.8.0 du principe II : nom, couleurs, fichier du logo.

const tagAssociation = "association"

// Palette est la palette dérivée des couleurs de l'association.
type Palette struct {
	Brand     string `json:"brand" doc:"Fond plein (boutons, élément actif)"`
	BrandInk  string `json:"brandInk" doc:"Texte et repères dans la teinte principale"`
	OnBrand   string `json:"onBrand" doc:"Texte posé sur la couleur principale"`
	Accent    string `json:"accent"`
	AccentInk string `json:"accentInk"`
	OnAccent  string `json:"onAccent"`
}

// Identity est l'identité visuelle publique : exactement ces trois champs.
type Identity struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logoUrl"`
	Palette Palette `json:"palette"`
}

// AssociationContact sont les coordonnées présentées aux membres (sans
// identité légale).
type AssociationContact struct {
	Address      Address     `json:"address"`
	ContactEmail *string     `json:"contactEmail"`
	Phone        *string     `json:"phone"`
	Website      *string     `json:"website"`
	Social       SocialLinks `json:"social"`
}

type IdentityOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         Identity
}

type AssociationContactOutput struct{ Body AssociationContact }

// LogoInput désigne la variante interface du logo par son nom généré.
type LogoInput struct {
	Name string `path:"name" maxLength:"64"`
}

func (s *server) registerIdentity(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID: "get-identity",
		Method:      http.MethodGet,
		Path:        "/identity",
		Summary:     "Identité visuelle de l'association (nom, couleurs, logo)",
		Tags:        []string{tagAssociation},
	}, s.getIdentity)

	logo := huma.Operation{
		OperationID: "get-identity-logo",
		Method:      http.MethodGet,
		Path:        "/identity/logo/{name}",
		Summary:     "Logo de l'association",
		Tags:        []string{tagAssociation},
	}
	logo.Responses = map[string]*huma.Response{"200": {
		Description: "Le logo (PNG ou JPEG)",
		Content: map[string]*huma.MediaType{
			"image/png":  {Schema: &huma.Schema{Type: "string", Format: "binary"}},
			"image/jpeg": {Schema: &huma.Schema{Type: "string", Format: "binary"}},
		},
	}}
	register(api, auth.Public, logo, s.getIdentityLogo)

	register(api, auth.Member, huma.Operation{
		OperationID: "get-association-contact",
		Method:      http.MethodGet,
		Path:        "/association/contact",
		Summary:     "Coordonnées et réseaux sociaux de l'association",
		Tags:        []string{tagAssociation},
	}, s.getAssociationContact)
}

func toPalette(p brand.Palette) Palette {
	return Palette{
		Brand: p.Brand.Hex(), BrandInk: p.BrandInk.Hex(), OnBrand: p.OnBrand.Hex(),
		Accent: p.Accent.Hex(), AccentInk: p.AccentInk.Hex(), OnAccent: p.OnAccent.Hex(),
	}
}

// colorPtr lit une couleur stockée ; nil si absente ou invalide.
func colorPtr(v sql.NullString) *brand.RGB {
	if !v.Valid {
		return nil
	}
	c, err := brand.ParseHex(v.String)
	if err != nil {
		return nil
	}
	return &c
}

func paletteOf(a sqlc.Association) brand.Palette {
	return brand.Derive(colorPtr(a.PrimaryColor), colorPtr(a.AccentColor))
}

func logoURL(name string) *string {
	u := "/api/identity/logo/" + name
	return &u
}

// currentLogo lit le logo ; nil sans logo.
func currentLogo(ctx context.Context, q *sqlc.Queries) (*sqlc.AssociationLogo, error) {
	l, err := q.GetAssociationLogo(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (s *server) getIdentity(ctx context.Context, _ *struct{}) (*IdentityOutput, error) {
	a, err := s.Store.Read.GetAssociation(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'association impossible", err)
	}
	logo, err := currentLogo(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture du logo impossible", err)
	}
	id := Identity{Name: displayName(a), Palette: toPalette(paletteOf(a))}
	if logo != nil {
		id.LogoURL = logoURL(logo.WebName)
	}
	return &IdentityOutput{CacheControl: "no-cache", Body: id}, nil
}

func (s *server) getIdentityLogo(ctx context.Context, in *LogoInput) (*huma.StreamResponse, error) {
	notFound := huma.Error404NotFound("Logo introuvable.")
	if !communication.ValidStoredName(in.Name) || s.LogoFiles == nil {
		return nil, notFound
	}
	logo, err := currentLogo(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture du logo impossible", err)
	}
	// Seule la variante interface courante est servie : ni la variante e-mail,
	// ni un ancien logo.
	if logo == nil || logo.WebName != in.Name {
		return nil, notFound
	}
	data, err := s.LogoFiles.Read(logo.WebName, logo.WebSizeBytes, logo.WebSha256)
	if err != nil {
		return nil, s.internalError(ctx, "fichier du logo illisible", err)
	}
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		hctx.SetHeader("Content-Type", logo.WebType)
		hctx.SetHeader("Content-Length", strconv.Itoa(len(data)))
		hctx.SetHeader("Content-Disposition", "inline")
		hctx.SetHeader("X-Content-Type-Options", "nosniff")
		hctx.SetHeader("Cache-Control", "public, max-age=31536000, immutable")
		hctx.SetStatus(http.StatusOK)
		_, _ = hctx.BodyWriter().Write(data)
	}}, nil
}

func (s *server) getAssociationContact(ctx context.Context, _ *struct{}) (*AssociationContactOutput, error) {
	a, err := s.Store.Read.GetAssociation(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'association impossible", err)
	}
	g := toGeneralSettings(a)
	return &AssociationContactOutput{Body: AssociationContact{
		Address: g.Address, ContactEmail: g.ContactEmail, Phone: g.Phone, Website: g.Website, Social: g.Social,
	}}, nil
}

// MailIdentity renvoie la fonction de lecture de l'identité visuelle utilisée
// par le worker d'e-mails (nom, palette, variante e-mail du logo).
func MailIdentity(deps Deps) func(ctx context.Context) (mail.Identity, error) {
	s := &server{Deps: deps}
	return func(ctx context.Context) (mail.Identity, error) { return s.mailIdentity(ctx, s.Store.Read) }
}

func (s *server) mailIdentity(ctx context.Context, q *sqlc.Queries) (mail.Identity, error) {
	a, err := q.GetAssociation(ctx)
	if err != nil {
		return mail.Identity{}, err
	}
	id := mail.Identity{Name: displayName(a), Palette: paletteOf(a)}
	logo, err := currentLogo(ctx, q)
	if err != nil {
		return mail.Identity{}, err
	}
	if logo != nil && s.LogoFiles != nil {
		data, err := s.LogoFiles.Read(logo.EmailName, logo.EmailSizeBytes, logo.EmailSha256)
		if err != nil {
			// Sans logo plutôt que sans e-mail.
			s.Logger.ErrorContext(ctx, "variante e-mail du logo illisible", "error", err)
			return id, nil
		}
		id.Logo = &mail.Attachment{Name: mail.LogoContentID, ContentType: logo.EmailType, Data: data}
	}
	return id, nil
}
