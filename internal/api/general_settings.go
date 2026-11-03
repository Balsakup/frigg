package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/association"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Paramètres généraux de l'association : section « Général »
// (specs/010-general-settings, contracts/api.md).

// Address est l'adresse du siège.
type Address struct {
	Line1      *string `json:"line1"`
	Line2      *string `json:"line2"`
	PostalCode *string `json:"postalCode"`
	City       *string `json:"city"`
}

// SocialLinks sont les liens des réseaux sociaux (liste fermée).
type SocialLinks struct {
	Facebook  *string `json:"facebook"`
	Instagram *string `json:"instagram"`
	LinkedIn  *string `json:"linkedin"`
	YouTube   *string `json:"youtube"`
	Mastodon  *string `json:"mastodon"`
	Bluesky   *string `json:"bluesky"`
}

// GeneralSettings sont les paramètres de la section « Général ».
type GeneralSettings struct {
	Name         string      `json:"name"`
	Tagline      *string     `json:"tagline"`
	ShortName    *string     `json:"shortName"`
	Purpose      *string     `json:"purpose"`
	RNA          *string     `json:"rna"`
	SIRET        *string     `json:"siret"`
	FoundedOn    *string     `json:"foundedOn" doc:"Date de création, AAAA-MM-JJ"`
	Address      Address     `json:"address"`
	ContactEmail *string     `json:"contactEmail"`
	Phone        *string     `json:"phone" doc:"Forme nationale à 10 chiffres"`
	Website      *string     `json:"website"`
	Social       SocialLinks `json:"social"`
}

// AddressBody est l'adresse saisie ; chaque élément est facultatif.
type AddressBody struct {
	Line1      *string `json:"line1,omitempty" required:"false" nullable:"true" maxLength:"100"`
	Line2      *string `json:"line2,omitempty" required:"false" nullable:"true" maxLength:"100"`
	PostalCode *string `json:"postalCode,omitempty" required:"false" nullable:"true" maxLength:"10"`
	City       *string `json:"city,omitempty" required:"false" nullable:"true" maxLength:"100"`
}

// SocialLinksBody sont les liens saisis.
type SocialLinksBody struct {
	Facebook  *string `json:"facebook,omitempty" required:"false" nullable:"true" maxLength:"300"`
	Instagram *string `json:"instagram,omitempty" required:"false" nullable:"true" maxLength:"300"`
	LinkedIn  *string `json:"linkedin,omitempty" required:"false" nullable:"true" maxLength:"300"`
	YouTube   *string `json:"youtube,omitempty" required:"false" nullable:"true" maxLength:"300"`
	Mastodon  *string `json:"mastodon,omitempty" required:"false" nullable:"true" maxLength:"300"`
	Bluesky   *string `json:"bluesky,omitempty" required:"false" nullable:"true" maxLength:"300"`
}

// GeneralSettingsBody remplace les paramètres de la section « Général ». Une
// chaîne vide ou faite d'espaces vaut « non renseigné », sauf pour le nom.
type GeneralSettingsBody struct {
	Name         string          `json:"name" maxLength:"120"`
	Tagline      *string         `json:"tagline,omitempty" required:"false" nullable:"true" maxLength:"150"`
	ShortName    *string         `json:"shortName,omitempty" required:"false" nullable:"true" maxLength:"20"`
	Purpose      *string         `json:"purpose,omitempty" required:"false" nullable:"true" maxLength:"2000"`
	RNA          *string         `json:"rna,omitempty" required:"false" nullable:"true" maxLength:"20"`
	SIRET        *string         `json:"siret,omitempty" required:"false" nullable:"true" maxLength:"30"`
	FoundedOn    *string         `json:"foundedOn,omitempty" required:"false" nullable:"true" maxLength:"10" doc:"AAAA-MM-JJ"`
	Address      AddressBody     `json:"address,omitempty" required:"false"`
	ContactEmail *string         `json:"contactEmail,omitempty" required:"false" nullable:"true" maxLength:"254"`
	Phone        *string         `json:"phone,omitempty" required:"false" nullable:"true" maxLength:"30"`
	Website      *string         `json:"website,omitempty" required:"false" nullable:"true" maxLength:"300"`
	Social       SocialLinksBody `json:"social,omitempty" required:"false"`
}

type GeneralSettingsOutput struct{ Body GeneralSettings }

type UpdateGeneralSettingsInput struct{ Body GeneralSettingsBody }

func (s *server) registerGeneralSettings(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "get-general-settings",
		Method:      http.MethodGet,
		Path:        "/admin/settings/general",
		Summary:     "Paramètres généraux de l'association",
		Tags:        []string{tagSettings},
	}, s.getGeneralSettings)

	register(api, auth.Admin, huma.Operation{
		OperationID: "update-general-settings",
		Method:      http.MethodPut,
		Path:        "/admin/settings/general",
		Summary:     "Modifier les paramètres généraux",
		Tags:        []string{tagSettings},
	}, s.updateGeneralSettings)
}

func nsPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func ptrNS(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func toGeneralSettings(a sqlc.Association) GeneralSettings {
	return GeneralSettings{
		Name:      a.Name.String,
		Tagline:   nsPtr(a.Tagline),
		ShortName: nsPtr(a.ShortName),
		Purpose:   nsPtr(a.Purpose),
		RNA:       nsPtr(a.Rna),
		SIRET:     nsPtr(a.Siret),
		FoundedOn: nsPtr(a.FoundedOn),
		Address: Address{
			Line1: nsPtr(a.AddressLine1), Line2: nsPtr(a.AddressLine2),
			PostalCode: nsPtr(a.PostalCode), City: nsPtr(a.City),
		},
		ContactEmail: nsPtr(a.ContactEmail),
		Phone:        nsPtr(a.Phone),
		Website:      nsPtr(a.Website),
		Social:       toSocialLinks(a),
	}
}

func toSocialLinks(a sqlc.Association) SocialLinks {
	return SocialLinks{
		Facebook: nsPtr(a.FacebookUrl), Instagram: nsPtr(a.InstagramUrl), LinkedIn: nsPtr(a.LinkedinUrl),
		YouTube: nsPtr(a.YoutubeUrl), Mastodon: nsPtr(a.MastodonUrl), Bluesky: nsPtr(a.BlueskyUrl),
	}
}

func (s *server) getGeneralSettings(ctx context.Context, _ *struct{}) (*GeneralSettingsOutput, error) {
	a, err := s.Store.Read.GetAssociation(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'association impossible", err)
	}
	return &GeneralSettingsOutput{Body: toGeneralSettings(a)}, nil
}

func (s *server) updateGeneralSettings(ctx context.Context, in *UpdateGeneralSettingsInput) (*GeneralSettingsOutput, error) {
	b := in.Body
	st, errs := association.Normalize(association.Settings{
		Name: b.Name, Tagline: b.Tagline, ShortName: b.ShortName, Purpose: b.Purpose,
		RNA: b.RNA, SIRET: b.SIRET, FoundedOn: b.FoundedOn,
		Address: association.Address{
			Line1: b.Address.Line1, Line2: b.Address.Line2, PostalCode: b.Address.PostalCode, City: b.Address.City,
		},
		ContactEmail: b.ContactEmail, Phone: b.Phone, Website: b.Website,
		Social: association.Social{
			Facebook: b.Social.Facebook, Instagram: b.Social.Instagram, LinkedIn: b.Social.LinkedIn,
			YouTube: b.Social.YouTube, Mastodon: b.Social.Mastodon, Bluesky: b.Social.Bluesky,
		},
	}, s.today())
	if len(errs) > 0 {
		details := make([]*huma.ErrorDetail, len(errs))
		for i, e := range errs {
			details[i] = &huma.ErrorDetail{Location: "body." + e.Field, Message: e.Message}
		}
		return nil, FieldErrorDetails(details...)
	}

	params := sqlc.UpdateGeneralSettingsParams{
		Name: sql.NullString{String: st.Name, Valid: true}, Tagline: ptrNS(st.Tagline), ShortName: ptrNS(st.ShortName),
		Purpose: ptrNS(st.Purpose), Rna: ptrNS(st.RNA), Siret: ptrNS(st.SIRET), FoundedOn: ptrNS(st.FoundedOn),
		AddressLine1: ptrNS(st.Address.Line1), AddressLine2: ptrNS(st.Address.Line2),
		PostalCode: ptrNS(st.Address.PostalCode), City: ptrNS(st.Address.City),
		ContactEmail: ptrNS(st.ContactEmail), Phone: ptrNS(st.Phone), Website: ptrNS(st.Website),
		FacebookUrl: ptrNS(st.Social.Facebook), InstagramUrl: ptrNS(st.Social.Instagram),
		LinkedinUrl: ptrNS(st.Social.LinkedIn), YoutubeUrl: ptrNS(st.Social.YouTube),
		MastodonUrl: ptrNS(st.Social.Mastodon), BlueskyUrl: ptrNS(st.Social.Bluesky),
		UpdatedAt: nowString(),
	}
	var changed []string
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		before, err := q.GetAssociation(ctx)
		if err != nil {
			return err
		}
		changed = changedFields(before, params)
		return q.UpdateGeneralSettings(ctx, params)
	})
	if err != nil {
		return nil, s.internalError(ctx, "enregistrement des paramètres généraux impossible", err)
	}
	// Noms des champs seulement, jamais leurs valeurs (FR-031).
	s.Logger.InfoContext(ctx, "paramètres généraux modifiés",
		"action", "settings.general_update", "actor_id", CurrentAccount(ctx).ID, "fields", changed)
	return s.getGeneralSettings(ctx, nil)
}

// changedFields renvoie les noms JSON des champs dont la valeur change.
func changedFields(a sqlc.Association, p sqlc.UpdateGeneralSettingsParams) []string {
	pairs := []struct {
		name     string
		old, new sql.NullString
	}{
		{"name", a.Name, p.Name}, {"tagline", a.Tagline, p.Tagline}, {"shortName", a.ShortName, p.ShortName},
		{"purpose", a.Purpose, p.Purpose}, {"rna", a.Rna, p.Rna}, {"siret", a.Siret, p.Siret},
		{"foundedOn", a.FoundedOn, p.FoundedOn}, {"address.line1", a.AddressLine1, p.AddressLine1},
		{"address.line2", a.AddressLine2, p.AddressLine2}, {"address.postalCode", a.PostalCode, p.PostalCode},
		{"address.city", a.City, p.City}, {"contactEmail", a.ContactEmail, p.ContactEmail},
		{"phone", a.Phone, p.Phone}, {"website", a.Website, p.Website},
		{"social.facebook", a.FacebookUrl, p.FacebookUrl}, {"social.instagram", a.InstagramUrl, p.InstagramUrl},
		{"social.linkedin", a.LinkedinUrl, p.LinkedinUrl}, {"social.youtube", a.YoutubeUrl, p.YoutubeUrl},
		{"social.mastodon", a.MastodonUrl, p.MastodonUrl}, {"social.bluesky", a.BlueskyUrl, p.BlueskyUrl},
	}
	changed := []string{}
	for _, c := range pairs {
		if c.old != c.new {
			changed = append(changed, c.name)
		}
	}
	return changed
}
