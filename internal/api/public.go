package api

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/public"
	"github.com/balsakup/frigg/internal/season"
)

// Parcours publics (specs/016-public-journeys) : page d'adhésion publique et
// dépôt d'une demande, sans session. Seules les données énumérées par le
// principe II sont transmises (FR-001, FR-002).

const (
	// MsgPublicSubmitted est la réponse unique à un dépôt, quelle que soit
	// l'adresse saisie (FR-009, SC-005).
	MsgPublicSubmitted = "Si l'adresse est valide, un e-mail vient de vous être envoyé."
	// MsgPublicMembershipClosed : accueil désactivé ou aucune saison ouverte
	// avec un tarif (FR-005).
	MsgPublicMembershipClosed = "Les demandes d'adhésion ne sont pas ouvertes."
	// MsgPublicFeeGone : le tarif choisi a disparu avant la confirmation.
	MsgPublicFeeGone = "Ce tarif n'est plus proposé."
)

func (s *server) registerPublic(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID: "get-public-membership-page",
		Method:      http.MethodGet,
		Path:        "/public/membership",
		Summary:     "Page d'adhésion publique",
		Description: "Saisons ouvertes et leurs tarifs, champs personnalisés remplis par le membre, ou fermeture.",
		Tags:        []string{tagPublic},
	}, s.getPublicMembershipPage)

	register(api, auth.Public, huma.Operation{
		OperationID:   "submit-public-membership",
		Method:        http.MethodPost,
		Path:          "/public/membership",
		Summary:       "Déposer une demande d'adhésion publique",
		Description:   "La réponse est identique quelle que soit l'adresse. Un e-mail de confirmation est envoyé à l'adresse saisie.",
		Tags:          []string{tagPublic},
		DefaultStatus: http.StatusAccepted,
	}, s.submitPublicMembership)

	register(api, auth.Public, huma.Operation{
		OperationID: "confirm-public-submission",
		Method:      http.MethodPost,
		Path:        "/public/submissions/confirm",
		Summary:     "Confirmer l'adresse d'un dépôt public",
		Description: "Le jeton passe dans le corps, jamais dans l'URL. Ouvre l'accès par lien si la fiche reste sans compte.",
		Tags:        []string{tagPublic},
	}, s.confirmPublicSubmission)
}

// PublicSeason est une saison ouverte à l'adhésion publique, avec ses tarifs.
type PublicSeason struct {
	ID    int64              `json:"id"`
	Label string             `json:"label"`
	Fees  []MembershipFeeRef `json:"fees" doc:"Du montant le plus élevé au plus faible"`
}

// PublicMembershipPage est la page d'adhésion publique (FR-005, FR-006).
type PublicMembershipPage struct {
	Open             bool           `json:"open" doc:"Accueil activé et au moins une saison ouverte avec un tarif"`
	Seasons          []PublicSeason `json:"seasons" doc:"Vide si l'accueil est fermé ; la plus récente d'abord"`
	MemberFields     []FormField    `json:"memberFields" doc:"Champs de membre « Rempli par le membre » actifs"`
	MembershipFields []FormField    `json:"membershipFields" doc:"Champs d'adhésion « Rempli par le membre » actifs"`
}

// PublicSubmissionBody est la saisie commune aux dépôts publics : identité et
// coordonnées (005), consentement, création de compte et piège (R10).
type PublicSubmissionBody struct {
	FirstName     string `json:"firstName" maxLength:"80"`
	LastName      string `json:"lastName" maxLength:"80"`
	BirthDate     string `json:"birthDate" format:"date"`
	Email         string `json:"email" maxLength:"254" doc:"Obligatoire"`
	Phone         string `json:"phone,omitempty" required:"false" maxLength:"30" doc:"Absent ou vide : aucun"`
	AddressLine1  string `json:"addressLine1,omitempty" required:"false" maxLength:"100"`
	AddressLine2  string `json:"addressLine2,omitempty" required:"false" maxLength:"100"`
	PostalCode    string `json:"postalCode,omitempty" required:"false" maxLength:"10"`
	City          string `json:"city,omitempty" required:"false" maxLength:"100"`
	Consent       bool   `json:"consent" doc:"Consentement au traitement des données (obligatoire)"`
	CreateAccount bool   `json:"createAccount,omitempty" required:"false" doc:"Proposer la création d'un compte après confirmation"`
	Website       string `json:"website,omitempty" required:"false" maxLength:"500" doc:"Piège à robots : doit rester vide"`
}

// PublicMembershipBody est une demande d'adhésion publique.
type PublicMembershipBody struct {
	PublicSubmissionBody
	SeasonID     int64  `json:"seasonId"`
	FeeID        int64  `json:"feeId"`
	DeclaredKind string `json:"declaredKind" maxLength:"10" doc:"first ou renewal"`
	// CustomFields : champs de membre et d'adhésion remplis par le membre (011).
	CustomFields []FieldInput `json:"customFields,omitempty" required:"false" maxItems:"60" doc:"Champs personnalisés de membre et d'adhésion"`
}

// PublicMessage est la réponse neutre d'un dépôt.
type PublicMessage struct {
	Message string `json:"message"`
}

type PublicMembershipPageOutput struct{ Body PublicMembershipPage }
type PublicMessageOutput struct{ Body PublicMessage }
type SubmitPublicMembershipInput struct{ Body PublicMembershipBody }

// publicAccepted est la réponse 202 neutre, identique pour toute adresse,
// pour le piège et pour la limite par adresse (SC-005).
func publicAccepted() *PublicMessageOutput {
	return &PublicMessageOutput{Body: PublicMessage{Message: MsgPublicSubmitted}}
}

// publicSeasons renvoie les saisons ouvertes à l'adhésion publique : aucune
// si l'accueil est désactivé, sinon les saisons ouvertes, non closes, qui
// portent au moins un tarif (FR-005).
func (s *server) publicSeasons(ctx context.Context, q *sqlc.Queries) ([]PublicSeason, error) {
	out := []PublicSeason{}
	a, err := q.GetAssociation(ctx)
	if err != nil || a.PublicMembershipEnabled == 0 {
		return out, err
	}
	seasons, order, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return nil, err
	}
	for i := len(order) - 1; i >= 0; i-- {
		info := seasons[order[i]]
		if info.ref.State != string(season.Open) {
			continue
		}
		fees, err := q.ListMembershipFees(ctx, info.ref.ID)
		if err != nil {
			return nil, err
		}
		if len(fees) > 0 {
			out = append(out, PublicSeason{ID: info.ref.ID, Label: info.ref.Label, Fees: toFeeRefs(fees)})
		}
	}
	return out, nil
}

// ownerWritableDefinitions charge les champs actifs d'une portée que le
// membre remplit lui-même : ni réservés à l'administration, ni seulement
// visibles (FR-002).
func ownerWritableDefinitions(ctx context.Context, q *sqlc.Queries, scope customfield.Scope) ([]customfield.Definition, error) {
	defs, err := loadDefinitions(ctx, q, scope, customfield.Owner)
	if err != nil {
		return nil, err
	}
	out := make([]customfield.Definition, 0, len(defs))
	for _, d := range defs {
		if customfield.CanWrite(d, customfield.Owner) {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *server) getPublicMembershipPage(ctx context.Context, _ *struct{}) (*PublicMembershipPageOutput, error) {
	q := s.Store.Read
	seasons, err := s.publicSeasons(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de la page d'adhésion publique impossible", err)
	}
	page := PublicMembershipPage{Open: len(seasons) > 0, Seasons: seasons, MemberFields: []FormField{}, MembershipFields: []FormField{}}
	if !page.Open {
		return &PublicMembershipPageOutput{Body: page}, nil
	}
	memberDefs, err := ownerWritableDefinitions(ctx, q, customfield.ScopeMember)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	membershipDefs, err := ownerWritableDefinitions(ctx, q, customfield.ScopeMembership)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	page.MemberFields = formFields(memberDefs, customfield.Owner)
	page.MembershipFields = formFields(membershipDefs, customfield.Owner)
	return &PublicMembershipPageOutput{Body: page}, nil
}

// validatePublicIdentity nettoie l'identité et les coordonnées d'un dépôt et
// renvoie les erreurs de champ en détails 422 (FR-006).
func validatePublicIdentity(b PublicSubmissionBody, today season.Date) (member.Clean, []*huma.ErrorDetail) {
	birth, err := season.ParseDate(b.BirthDate)
	if err != nil {
		return member.Clean{}, []*huma.ErrorDetail{{Location: "body.birthDate", Message: MsgInvalidDate}}
	}
	c, errs := public.Validate(public.Submission{
		FirstName: b.FirstName, LastName: b.LastName, BirthDate: birth,
		Email: b.Email, Phone: b.Phone,
		AddressLine1: b.AddressLine1, AddressLine2: b.AddressLine2, PostalCode: b.PostalCode, City: b.City,
		Consent: b.Consent, CreateAccount: b.CreateAccount, Website: b.Website,
	}, today)
	details := make([]*huma.ErrorDetail, 0, len(errs))
	for _, fe := range errs {
		msg := public.Message(fe)
		if msg == "" {
			msgs := memberMessages[fe.Err]
			var ok bool
			if msg, ok = msgs[fe.Field]; !ok {
				msg = msgs[""]
			}
		}
		details = append(details, &huma.ErrorDetail{Location: "body." + fe.Field, Message: msg})
	}
	return c, details
}

// checkPublicFields vérifie les champs personnalisés d'un dépôt d'adhésion :
// champs de membre et d'adhésion que le membre remplit, obligatoires exigés
// (011 FR-015). Elle renvoie leur forme canonique, triée par champ, à ranger
// dans public_submissions.custom_values.
func checkPublicFields(ctx context.Context, q *sqlc.Queries, inputs []FieldInput) (string, []*huma.ErrorDetail, error) {
	memberDefs, err := ownerWritableDefinitions(ctx, q, customfield.ScopeMember)
	if err != nil {
		return "", nil, err
	}
	membershipDefs, err := ownerWritableDefinitions(ctx, q, customfield.ScopeMembership)
	if err != nil {
		return "", nil, err
	}
	isMember := make(map[int64]bool, len(memberDefs))
	for _, d := range memberDefs {
		isMember[d.ID] = true
	}
	var memberIn, membershipIn []customfield.Input
	for _, in := range toInputs(inputs) {
		if isMember[in.FieldID] {
			memberIn = append(memberIn, in)
		} else {
			membershipIn = append(membershipIn, in)
		}
	}
	_, memberErrs := customfield.Apply(memberDefs, nil, memberIn, customfield.Owner)
	_, membershipErrs := customfield.Apply(membershipDefs, nil, membershipIn, customfield.Owner)
	if details := append(customFieldDetails(memberErrs), customFieldDetails(membershipErrs)...); len(details) > 0 {
		return "", details, nil
	}
	sorted := slices.Clone(inputs)
	if sorted == nil {
		sorted = []FieldInput{}
	}
	slices.SortFunc(sorted, func(a, b FieldInput) int { return cmp.Compare(a.FieldID, b.FieldID) })
	raw, err := json.Marshal(sorted)
	return string(raw), nil, err
}

// emailAccount est le compte d'une adresse, relu pour un parcours public.
type emailAccount struct {
	ID      int64
	Active  bool
	Managed int64
}

// lookupEmailAccount cherche le compte d'une adresse normalisée ; nil s'il
// n'existe pas (research.md 016 › R7).
func lookupEmailAccount(ctx context.Context, q *sqlc.Queries, email string) (*emailAccount, error) {
	a, err := q.GetAccountByEmail(ctx, auth.NormalizeEmail(email))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n, err := q.CountMembersOfAccount(ctx, nid(a.ID))
	if err != nil {
		return nil, err
	}
	return &emailAccount{ID: a.ID, Active: !a.DisabledAt.Valid, Managed: n}, nil
}

// confirmNotice prépare l'e-mail de confirmation d'un dépôt selon l'état du
// compte de l'adresse : seule la personne qui lit la messagerie le voit
// (research.md 016 › R7). Un compte désactivé est traité comme une adresse
// sans compte.
func confirmNotice(acc *emailAccount) mail.Data {
	active := acc != nil && acc.Active
	return mail.Data{
		AccountExists: active && acc.Managed < public.MaxManagedMembers,
		AccountFull:   active && acc.Managed >= public.MaxManagedMembers,
	}
}

// logPublicRejected trace un dépôt ignoré ou refusé, sans aucune donnée
// saisie (research.md 016 › R16).
func (s *server) logPublicRejected(ctx context.Context, kind, reason string) {
	s.Logger.InfoContext(ctx, "dépôt public refusé", "action", "public_submit_rejected", "kind", kind, "reason", reason)
}

func (s *server) submitPublicMembership(ctx context.Context, in *SubmitPublicMembershipInput) (*PublicMessageOutput, error) {
	const kind = "membership"
	b := in.Body
	if !s.Limits.PublicSubmitIP.Allow(clientIP(ctx)) {
		s.logPublicRejected(ctx, kind, "rate_ip")
		return nil, TooManyAttempts()
	}
	if public.Honeypot(b.Website) {
		s.logPublicRejected(ctx, kind, "honeypot")
		return publicAccepted(), nil
	}
	q := s.Store.Read
	seasons, err := s.publicSeasons(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de la page d'adhésion publique impossible", err)
	}
	if len(seasons) == 0 {
		s.logPublicRejected(ctx, kind, "closed")
		return nil, huma.Error409Conflict(MsgPublicMembershipClosed)
	}

	clean, details := validatePublicIdentity(b.PublicSubmissionBody, s.today())
	if err := public.ValidateMembershipTarget(b.DeclaredKind); err != nil {
		details = append(details, &huma.ErrorDetail{Location: "body.declaredKind", Message: public.Message(err)})
	}
	i := slices.IndexFunc(seasons, func(ps PublicSeason) bool { return ps.ID == b.SeasonID })
	var target PublicSeason
	switch {
	case i < 0:
		details = append(details, &huma.ErrorDetail{Location: "body.seasonId", Message: MsgSeasonNotOffered})
	case !slices.ContainsFunc(seasons[i].Fees, func(f MembershipFeeRef) bool { return f.ID == b.FeeID }):
		details = append(details, &huma.ErrorDetail{Location: "body.feeId", Message: MsgFeeNotInSeason})
	default:
		target = seasons[i]
	}
	customValues, fieldDetails, err := checkPublicFields(ctx, q, b.CustomFields)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	if details = append(details, fieldDetails...); len(details) > 0 {
		s.logPublicRejected(ctx, kind, "invalid")
		return nil, FieldErrorDetails(details...)
	}

	// Plafond d'e-mails par adresse : au-delà, la réponse reste la même.
	email := *clean.Email
	if !s.Limits.PublicMailEmail.Allow(email) {
		s.logPublicRejected(ctx, kind, "rate_email")
		return publicAccepted(), nil
	}
	raw, hash := auth.NewToken()
	now := s.now().UTC()
	f := fieldsOf(clean)
	var id int64
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		// Le compte est toujours cherché et l'e-mail toujours mis en file, par
		// le même chemin : seule la personne qui lit la messagerie voit la
		// différence (SC-005).
		acc, err := lookupEmailAccount(ctx, q, email)
		if err != nil {
			return err
		}
		id, err = q.CreatePublicSubmission(ctx, sqlc.CreatePublicSubmissionParams{
			Kind:          kind,
			SeasonID:      nid(target.ID),
			FeeID:         nid(b.FeeID),
			DeclaredKind:  ns(b.DeclaredKind),
			FirstName:     f.FirstName,
			LastName:      f.LastName,
			BirthDate:     f.BirthDate,
			Email:         email,
			Phone:         f.Phone,
			AddressLine1:  f.AddressLine1,
			AddressLine2:  f.AddressLine2,
			PostalCode:    f.Postal,
			City:          f.City,
			CustomValues:  customValues,
			CreateAccount: boolInt(b.CreateAccount),
			ConsentedAt:   now.Format(timeFormat),
			TokenHash:     hash,
			ExpiresAt:     now.Add(public.ConfirmTTL).Format(timeFormat),
			CreatedAt:     now.Format(timeFormat),
		})
		if err != nil {
			return err
		}
		data := confirmNotice(acc)
		data.Member = clean.FirstName + " " + clean.LastName
		data.Season = target.Label
		return s.enqueueNotice(ctx, q, "public-confirm", email, data, "/confirmer-demande#jeton="+raw)
	})
	if err != nil {
		return nil, s.internalError(ctx, "enregistrement du dépôt public impossible", err)
	}
	s.Logger.InfoContext(ctx, "dépôt public enregistré", "action", "public_submit", "kind", kind,
		"submission_id", id, "season_id", target.ID)
	return publicAccepted(), nil
}
