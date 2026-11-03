package api

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/association/logo"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/brand"
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Section « Apparence » : couleurs et logo (specs/010-general-settings › R4, R7).

const (
	MsgColorInvalid = "Saisissez une couleur au format #1f5278."
	MsgLogoFormat   = "Déposez une image PNG, JPEG ou WebP de 1 Mo au plus."
	MsgLogoPixels   = "Cette image est trop grande : 8 000 pixels de côté au plus."
)

// LogoUploadMaxBody borne la requête de dépôt : 1 Mio et les en-têtes multipart.
const LogoUploadMaxBody = logo.MaxBytes + 16<<10

// Logo décrit le logo pour l'administrateur.
type Logo struct {
	URL          string    `json:"url"`
	OriginalName string    `json:"originalName"`
	Width        int64     `json:"width"`
	Height       int64     `json:"height"`
	UploadedAt   time.Time `json:"uploadedAt"`
}

// Appearance est l'état de la section « Apparence ».
type Appearance struct {
	PrimaryColor *string `json:"primaryColor" doc:"null : couleur par défaut de Frigg"`
	AccentColor  *string `json:"accentColor" doc:"null : couleur par défaut de Frigg"`
	Palette      Palette `json:"palette"`
	Logo         *Logo   `json:"logo"`
}

type AppearanceOutput struct{ Body Appearance }

type UpdateAppearanceInput struct {
	Body struct {
		PrimaryColor *string `json:"primaryColor" required:"true" nullable:"true" maxLength:"7"`
		AccentColor  *string `json:"accentColor" required:"true" nullable:"true" maxLength:"7"`
	}
}

type PalettePreviewInput struct {
	Primary string `query:"primary" maxLength:"7"`
	Accent  string `query:"accent" maxLength:"7"`
}

type PaletteOutput struct{ Body Palette }

type UploadLogoInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" required:"true"`
	}]
}

func (s *server) registerAppearance(api huma.API) {
	op := func(id, method, path, summary string) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagSettings}}
	}
	register(api, auth.Admin, op("get-appearance", http.MethodGet, "/admin/settings/appearance", "Couleurs et logo de l'association"), s.getAppearance)
	register(api, auth.Admin, op("update-appearance", http.MethodPut, "/admin/settings/appearance", "Modifier les couleurs"), s.updateAppearance)
	register(api, auth.Admin, op("preview-palette", http.MethodGet, "/admin/settings/appearance/palette", "Palette dérivée de deux couleurs, sans enregistrement"), s.previewPalette)
	upload := op("upload-logo", http.MethodPut, "/admin/settings/appearance/logo", "Déposer le logo")
	upload.MaxBodyBytes = LogoUploadMaxBody
	register(api, auth.Admin, upload, s.uploadLogo)
	del := op("delete-logo", http.MethodDelete, "/admin/settings/appearance/logo", "Retirer le logo")
	del.DefaultStatus = http.StatusNoContent
	register(api, auth.Admin, del, s.deleteLogo)
}

func (s *server) appearance(ctx context.Context) (*AppearanceOutput, error) {
	a, err := s.Store.Read.GetAssociation(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'association impossible", err)
	}
	l, err := currentLogo(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture du logo impossible", err)
	}
	out := Appearance{PrimaryColor: nsPtr(a.PrimaryColor), AccentColor: nsPtr(a.AccentColor), Palette: toPalette(paletteOf(a))}
	if l != nil {
		at, _ := time.Parse(timeFormat, l.UploadedAt)
		out.Logo = &Logo{URL: *logoURL(l.WebName), OriginalName: l.OriginalName, Width: l.WebWidth, Height: l.WebHeight, UploadedAt: at}
	}
	return &AppearanceOutput{Body: out}, nil
}

func (s *server) getAppearance(ctx context.Context, _ *struct{}) (*AppearanceOutput, error) {
	return s.appearance(ctx)
}

// normalizeColor renvoie la couleur en minuscules ; nil ou vide = défaut.
func normalizeColor(p *string) (sql.NullString, bool) {
	if p == nil || *p == "" {
		return sql.NullString{}, true
	}
	v, err := brand.NormalizeHex(*p)
	if err != nil {
		return sql.NullString{}, false
	}
	return sql.NullString{String: v, Valid: true}, true
}

func (s *server) updateAppearance(ctx context.Context, in *UpdateAppearanceInput) (*AppearanceOutput, error) {
	primary, okP := normalizeColor(in.Body.PrimaryColor)
	accent, okA := normalizeColor(in.Body.AccentColor)
	var details []*huma.ErrorDetail
	if !okP {
		details = append(details, &huma.ErrorDetail{Location: "body.primaryColor", Message: MsgColorInvalid})
	}
	if !okA {
		details = append(details, &huma.ErrorDetail{Location: "body.accentColor", Message: MsgColorInvalid})
	}
	if len(details) > 0 {
		return nil, FieldErrorDetails(details...)
	}
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		return q.UpdateAssociationColors(ctx, sqlc.UpdateAssociationColorsParams{
			PrimaryColor: primary, AccentColor: accent, UpdatedAt: nowString(),
		})
	})
	if err != nil {
		return nil, s.internalError(ctx, "enregistrement des couleurs impossible", err)
	}
	// Les codes couleur ne sont pas des données personnelles.
	s.Logger.InfoContext(ctx, "couleurs modifiées", "action", "settings.colors_update",
		"actor_id", CurrentAccount(ctx).ID, "primary", primary.String, "accent", accent.String)
	return s.appearance(ctx)
}

func (s *server) previewPalette(ctx context.Context, in *PalettePreviewInput) (*PaletteOutput, error) {
	read := func(loc, v string) (*brand.RGB, *huma.ErrorDetail) {
		if v == "" {
			return nil, nil
		}
		c, err := brand.ParseHex(v)
		if err != nil {
			return nil, &huma.ErrorDetail{Location: loc, Message: MsgColorInvalid}
		}
		return &c, nil
	}
	p, e1 := read("query.primary", in.Primary)
	a, e2 := read("query.accent", in.Accent)
	var details []*huma.ErrorDetail
	for _, e := range []*huma.ErrorDetail{e1, e2} {
		if e != nil {
			details = append(details, e)
		}
	}
	if len(details) > 0 {
		return nil, FieldErrorDetails(details...)
	}
	return &PaletteOutput{Body: toPalette(brand.Derive(p, a))}, nil
}

func (s *server) uploadLogo(ctx context.Context, in *UploadLogoInput) (*AppearanceOutput, error) {
	file := in.RawBody.Data().File
	defer file.Close()
	if s.LogoFiles == nil {
		return nil, s.internalError(ctx, "répertoire du logo non configuré", errors.New("Deps.LogoFiles nil"))
	}
	v, err := logo.Process(file)
	switch {
	case errors.Is(err, logo.ErrUnsupported), errors.Is(err, logo.ErrTooBig):
		return nil, FieldErrors("form.file", MsgLogoFormat)
	case errors.Is(err, logo.ErrTooLarge):
		return nil, FieldErrors("form.file", MsgLogoPixels)
	case err != nil:
		return nil, s.internalError(ctx, "lecture du logo impossible", err)
	}

	// Les deux variantes sont écrites avant la transaction ; elles sont
	// supprimées si elle échoue, et l'ancien logo l'est après sa validation.
	save := func(e logo.Encoded) (string, string, error) {
		name, _, sum, err := s.LogoFiles.Save(bytes.NewReader(e.Data), int64(len(e.Data)))
		return name, sum, err
	}
	webName, webSum, err := save(v.Web)
	if err != nil {
		return nil, s.internalError(ctx, "enregistrement du logo impossible", err)
	}
	emailName, emailSum, err := save(v.Email)
	if err != nil {
		_ = s.LogoFiles.Remove(webName)
		return nil, s.internalError(ctx, "enregistrement du logo impossible", err)
	}
	author := CurrentAccount(ctx).ID
	var previous *sqlc.AssociationLogo
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		var err error
		if previous, err = currentLogo(ctx, q); err != nil {
			return err
		}
		original := communication.SanitizeFilename(file.Filename)
		if original == "" {
			original = "logo"
		}
		return q.UpsertAssociationLogo(ctx, sqlc.UpsertAssociationLogoParams{
			OriginalName: original, SourceType: v.SourceType, SourceSizeBytes: v.SourceSize,
			WebName: webName, WebType: v.Web.Type, WebSizeBytes: int64(len(v.Web.Data)),
			WebWidth: int64(v.Web.Width), WebHeight: int64(v.Web.Height), WebSha256: webSum,
			EmailName: emailName, EmailType: v.Email.Type, EmailSizeBytes: int64(len(v.Email.Data)),
			EmailWidth: int64(v.Email.Width), EmailHeight: int64(v.Email.Height), EmailSha256: emailSum,
			UploadedAt: nowString(), UploadedBy: nid(author),
		})
	})
	if err != nil {
		_ = s.LogoFiles.Remove(webName)
		_ = s.LogoFiles.Remove(emailName)
		return nil, s.internalError(ctx, "enregistrement du logo impossible", err)
	}
	s.removeLogoFiles(ctx, previous)
	s.Logger.InfoContext(ctx, "logo déposé", "action", "settings.logo_upload", "actor_id", author,
		"type", v.SourceType, "width", v.Web.Width, "height", v.Web.Height,
		"web_bytes", len(v.Web.Data), "email_bytes", len(v.Email.Data))
	return s.appearance(ctx)
}

func (s *server) deleteLogo(ctx context.Context, _ *struct{}) (*struct{}, error) {
	var previous *sqlc.AssociationLogo
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		var err error
		if previous, err = currentLogo(ctx, q); err != nil || previous == nil {
			return err
		}
		return q.DeleteAssociationLogo(ctx)
	})
	if err != nil {
		return nil, s.internalError(ctx, "retrait du logo impossible", err)
	}
	if previous != nil {
		s.removeLogoFiles(ctx, previous)
		s.Logger.InfoContext(ctx, "logo retiré", "action", "settings.logo_delete", "actor_id", CurrentAccount(ctx).ID)
	}
	return nil, nil
}

// removeLogoFiles supprime les fichiers d'un logo remplacé ou retiré ; un
// échec est rattrapé au démarrage par le balayage des fichiers orphelins.
func (s *server) removeLogoFiles(ctx context.Context, l *sqlc.AssociationLogo) {
	if l == nil || s.LogoFiles == nil {
		return
	}
	for _, name := range []string{l.WebName, l.EmailName} {
		if err := s.LogoFiles.Remove(name); err != nil {
			s.Logger.WarnContext(ctx, "suppression d'un ancien fichier de logo impossible", "error", err)
		}
	}
}

// PurgeLogoOrphans supprime du répertoire du logo les fichiers qui ne sont pas
// ceux du logo courant (reprise après incident, au démarrage).
func PurgeLogoOrphans(ctx context.Context, deps Deps) (int, error) {
	if deps.LogoFiles == nil {
		return 0, nil
	}
	known := map[string]bool{}
	l, err := currentLogo(ctx, deps.Store.Read)
	if err != nil {
		return 0, err
	}
	if l != nil {
		known[l.WebName], known[l.EmailName] = true, true
	}
	return deps.LogoFiles.PurgeOrphans(known, time.Now())
}
