package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/season"
)

// Communications (specs/009-bulk-email) : e-mails groupés aux adhérents d'une
// saison. Toutes les opérations sont réservées aux administrateurs.

const tagCommunications = "Communications"

const (
	MsgCommunicationSeason      = "Choisissez une saison ouverte ou close."
	MsgCommunicationFee         = "Un tarif choisi n'appartient pas à cette saison."
	MsgCommunicationSituations  = "Choisissez au moins une situation de paiement."
	MsgCommunicationNoRecipient = "Aucun destinataire ne correspond à cette cible."
	MsgCommunicationChanged     = "Le nombre de destinataires a changé. Vérifiez la cible puis confirmez de nouveau."
	MsgCommunicationGone        = "Communication introuvable."
	MsgCommunicationSubject     = "Saisissez un objet de 150 caractères au plus, sur une seule ligne."
	MsgCommunicationBody        = "Saisissez un message de 10 000 caractères au plus."
	MsgCommunicationAttachment  = "Une pièce jointe est illisible ; retirez-la et ajoutez-la de nouveau."
	MsgCommunicationSent        = "Cette communication a déjà été envoyée."
)

// Target est la cible d'une communication.
type Target struct {
	SeasonID   int64    `json:"seasonId" minimum:"1"`
	Situations []string `json:"situations" minItems:"1" maxItems:"3" enum:"up_to_date,announced,due" doc:"Situations de paiement retenues"`
	FeeIDs     []int64  `json:"feeIds" maxItems:"50" doc:"Tarifs retenus ; vide : tous les tarifs de la saison"`
}

// AudienceMember est un adhérent ciblé.
type AudienceMember struct {
	MemberID     int64   `json:"memberId"`
	FirstName    string  `json:"firstName"`
	LastName     string  `json:"lastName"`
	Situation    string  `json:"situation" enum:"up_to_date,announced,due"`
	FeeLabel     string  `json:"feeLabel"`
	ContactEmail *string `json:"contactEmail" doc:"null : non joignable"`
}

// Audience est le décompte d'une cible et la liste des adhérents ciblés.
type Audience struct {
	Target           Target           `json:"target"`
	Preset           string           `json:"preset" enum:"all,up_to_date,unpaid,custom"`
	TargetLabel      string           `json:"targetLabel"`
	MembersCount     int              `json:"membersCount" doc:"Adhérents ciblés, non joignables compris"`
	UnreachableCount int              `json:"unreachableCount"`
	RecipientsCount  int              `json:"recipientsCount" doc:"Adresses distinctes"`
	Members          []AudienceMember `json:"members" doc:"Adhérents joignables, par nom puis prénom"`
	Unreachable      []AudienceMember `json:"unreachable"`
}

// Attachment est une pièce jointe.
type Attachment struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

// Delivery compte les e-mails d'une communication par état.
type Delivery struct {
	Pending int64 `json:"pending"`
	Sent    int64 `json:"sent"`
	Failed  int64 `json:"failed"`
}

// CommunicationSummary est une ligne de l'historique.
type CommunicationSummary struct {
	ID               int64       `json:"id"`
	Subject          string      `json:"subject"`
	TargetLabel      string      `json:"targetLabel"`
	Preset           string      `json:"preset" enum:"all,up_to_date,unpaid,custom"`
	Author           *AccountRef `json:"author" doc:"null : compte supprimé"`
	RecipientsCount  int64       `json:"recipientsCount"`
	AttachmentsCount int64       `json:"attachmentsCount"`
	Delivery         Delivery    `json:"delivery"`
	CreatedAt        time.Time   `json:"createdAt"`
}

// Recipient est une adresse destinataire d'une communication.
type Recipient struct {
	Email        *string    `json:"email" doc:"null : adresse effacée (compte supprimé)"`
	MembersCount int64      `json:"membersCount"`
	Status       string     `json:"status" enum:"pending,sent,failed"`
	SentAt       *time.Time `json:"sentAt"`
}

// CommunicationSeason est la saison d'une communication.
type CommunicationSeason struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

// Communication est une communication envoyée.
type Communication struct {
	CommunicationSummary
	Body             string              `json:"body"`
	Season           CommunicationSeason `json:"season"`
	Target           Target              `json:"target"`
	MembersCount     int64               `json:"membersCount"`
	UnreachableCount int64               `json:"unreachableCount"`
	Attachments      []Attachment        `json:"attachments"`
	Recipients       []Recipient         `json:"recipients" doc:"En échec, en attente puis envoyés"`
}

// Preview est le message rendu tel qu'il sera reçu.
type Preview struct {
	Subject string `json:"subject"`
	Text    string `json:"text"`
	HTML    string `json:"html"`
}

// AudienceInput est la cible lue dans l'adresse.
type AudienceInput struct {
	SeasonID   int64    `query:"seasonId" required:"true" minimum:"1"`
	Situations []string `query:"situations,explode" required:"true" minItems:"1" maxItems:"3" enum:"up_to_date,announced,due"`
	FeeIDs     []int64  `query:"feeIds,explode" maxItems:"50"`
}

// AudienceOutput est la réponse de get-communication-audience.
type AudienceOutput struct{ Body Audience }

// PreviewInput est le message à prévisualiser.
type PreviewInput struct {
	Body struct {
		SeasonID int64  `json:"seasonId" minimum:"1"`
		Subject  string `json:"subject" minLength:"1" maxLength:"150"`
		Body     string `json:"body" minLength:"1" maxLength:"10000"`
	}
}

// PreviewOutput est la réponse de preview-communication.
type PreviewOutput struct{ Body Preview }

// SendCommunicationInput est l'envoi d'une communication.
type SendCommunicationInput struct {
	Body struct {
		DraftKey           string `json:"draftKey" pattern:"^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$" patternDescription:"une clé de préparation (UUID v4)"`
		Target             Target `json:"target"`
		Subject            string `json:"subject" minLength:"1" maxLength:"150"`
		Body               string `json:"body" minLength:"1" maxLength:"10000"`
		ExpectedRecipients int    `json:"expectedRecipients" minimum:"1" doc:"Décompte présenté à l'administrateur"`
	}
}

// CommunicationOutput renvoie une communication ; Status vaut 200 lorsque la
// clé de préparation a déjà été envoyée.
type CommunicationOutput struct {
	Status int
	Body   Communication
}

// CommunicationIDInput désigne une communication.
type CommunicationIDInput struct {
	ID int64 `path:"id"`
}

// ListCommunicationsInput est la pagination de l'historique.
type ListCommunicationsInput struct {
	Page    int64 `query:"page" minimum:"1" default:"1"`
	PerPage int64 `query:"perPage" minimum:"1" maximum:"100" default:"25"`
}

// CommunicationPage est une page de l'historique.
type CommunicationPage struct {
	Items []CommunicationSummary `json:"items" doc:"La plus récente d'abord"`
	Total int64                  `json:"total"`
}

// CommunicationPageOutput est la réponse de list-communications.
type CommunicationPageOutput struct{ Body CommunicationPage }

// audienceChangedError est le 409 de send-communication : il porte la cible
// recalculée (FR-013).
type audienceChangedError struct {
	huma.ErrorModel
	Audience Audience `json:"audience"`
}

func (s *server) registerCommunications(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagCommunications}, DefaultStatus: status}
	}
	register(api, auth.Admin, op("get-communication-audience", http.MethodGet, "/admin/communications/audience", "Décompte d'une cible", 0), s.getCommunicationAudience)
	register(api, auth.Admin, op("preview-communication", http.MethodPost, "/admin/communications/preview", "Aperçu d'une communication", 0), s.previewCommunication)
	// 201 à l'envoi ; 200 (même corps) si la clé de préparation a déjà été envoyée.
	register(api, auth.Admin, op("send-communication", http.MethodPost, "/admin/communications", "Envoyer une communication", http.StatusCreated), s.sendCommunication)
	register(api, auth.Admin, op("list-communications", http.MethodGet, "/admin/communications", "Historique des communications", 0), s.listCommunications)
	register(api, auth.Admin, op("get-communication", http.MethodGet, "/admin/communications/{id}", "Une communication", 0), s.getCommunication)
	s.registerCommunicationAttachments(api, op)
}

// resolvedTarget est une cible vérifiée.
type resolvedTarget struct {
	season     seasonInfo
	situations []communication.Situation
	feeIDs     []int64
	feeLabels  []string
	audience   communication.Audience
	label      string
}

// audienceFor vérifie la cible et calcule son audience avec les requêtes q
// (research.md › R1, R2). location préfixe les emplacements d'erreur.
func (s *server) audienceFor(ctx context.Context, q *sqlc.Queries, t Target, location string) (resolvedTarget, error) {
	var r resolvedTarget
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return r, err
	}
	info, ok := seasons[t.SeasonID]
	if !ok || info.ref.State == string(season.Draft) {
		return r, FieldErrors(location+"seasonId", MsgCommunicationSeason)
	}
	r.season = info
	in := make([]communication.Situation, len(t.Situations))
	for i, v := range t.Situations {
		in[i] = communication.Situation(v)
	}
	if r.situations, err = communication.NormalizeSituations(in); err != nil {
		return r, FieldErrors(location+"situations", MsgCommunicationSituations)
	}
	r.feeIDs = communication.NormalizeFeeIDs(t.FeeIDs)
	fees, err := q.ListMembershipFees(ctx, t.SeasonID)
	if err != nil {
		return r, err
	}
	labels := map[int64]string{}
	for _, f := range fees {
		labels[f.ID] = f.Label
	}
	for _, id := range r.feeIDs {
		l, ok := labels[id]
		if !ok {
			return r, FieldErrors(location+"feeIds", MsgCommunicationFee)
		}
		r.feeLabels = append(r.feeLabels, l)
	}
	rows, err := q.ListAudienceCandidates(ctx, t.SeasonID)
	if err != nil {
		return r, err
	}
	cands := make([]communication.Candidate, len(rows))
	for i, row := range rows {
		cands[i] = communication.Candidate{
			MemberID: row.MemberID, FirstName: row.FirstName, LastName: row.LastName,
			FeeID: row.FeeID, FeeLabel: row.FeeLabel, Situation: communication.Situation(row.Situation),
			AccountEmail: row.AccountEmail.String, MemberEmail: row.MemberEmail.String,
		}
	}
	r.audience = communication.BuildAudience(communication.Filter(cands, r.situations, r.feeIDs))
	r.label = communication.TargetLabel(info.ref.Label, r.situations, r.feeLabels)
	return r, nil
}

func (r resolvedTarget) target() Target {
	situations := make([]string, len(r.situations))
	for i, v := range r.situations {
		situations[i] = string(v)
	}
	return Target{SeasonID: r.season.ref.ID, Situations: situations, FeeIDs: r.feeIDs}
}

func toAudienceMembers(targets []communication.Target) []AudienceMember {
	out := make([]AudienceMember, len(targets))
	for i, t := range targets {
		out[i] = AudienceMember{
			MemberID: t.MemberID, FirstName: t.FirstName, LastName: t.LastName,
			Situation: string(t.Situation), FeeLabel: t.FeeLabel,
		}
		if t.ContactEmail != "" {
			email := t.ContactEmail
			out[i].ContactEmail = &email
		}
	}
	return out
}

func (r resolvedTarget) toAudience() Audience {
	a := r.audience
	return Audience{
		Target:           r.target(),
		Preset:           string(communication.PresetOf(r.situations)),
		TargetLabel:      r.label,
		MembersCount:     a.MembersCount(),
		UnreachableCount: len(a.Unreachable),
		RecipientsCount:  len(a.Recipients),
		Members:          toAudienceMembers(a.Members),
		Unreachable:      toAudienceMembers(a.Unreachable),
	}
}

func (s *server) getCommunicationAudience(ctx context.Context, in *AudienceInput) (*AudienceOutput, error) {
	r, err := s.audienceFor(ctx, s.Store.Read, Target{SeasonID: in.SeasonID, Situations: in.Situations, FeeIDs: in.FeeIDs}, "query.")
	if err != nil {
		return nil, s.apiError(ctx, "calcul d'une cible impossible", err)
	}
	return &AudienceOutput{Body: r.toAudience()}, nil
}

// apiError laisse passer les erreurs huma et journalise les autres.
func (s *server) apiError(ctx context.Context, msg string, err error) error {
	var se huma.StatusError
	if errors.As(err, &se) {
		return err
	}
	return s.internalError(ctx, msg, err)
}

// checkMessage vérifie l'objet (une seule ligne) et le texte, espaces de bord
// retirés.
func checkMessage(subject, body *string) error {
	*subject = strings.TrimSpace(*subject)
	*body = strings.TrimSpace(*body)
	var details []*huma.ErrorDetail
	if *subject == "" || strings.ContainsAny(*subject, "\r\n") {
		details = append(details, &huma.ErrorDetail{Location: "body.subject", Message: MsgCommunicationSubject})
	}
	if *body == "" {
		details = append(details, &huma.ErrorDetail{Location: "body.body", Message: MsgCommunicationBody})
	}
	if len(details) > 0 {
		return FieldErrorDetails(details...)
	}
	return nil
}

// associationName lit le nom de l'association.
func associationName(ctx context.Context, q *sqlc.Queries) (string, error) {
	name, err := q.GetAssociation(ctx)
	if err != nil {
		return "", err
	}
	if name.Valid {
		return name.String, nil
	}
	return defaultAssociationName, nil
}

// renderCommunication rend le message d'une communication (research.md › R9).
func renderCommunication(association, seasonLabel, subject, body string) (mail.Message, error) {
	msg, err := mail.Render("communication", mail.Data{
		Association: association, Season: seasonLabel, Subject: subject, Body: body,
		Footer: "Vous recevez ce message en tant qu'adhérent de " + association + " pour la saison " + seasonLabel + ".",
	})
	msg.FromName = association
	return msg, err
}

func (s *server) previewCommunication(ctx context.Context, in *PreviewInput) (*PreviewOutput, error) {
	b := in.Body
	if err := checkMessage(&b.Subject, &b.Body); err != nil {
		return nil, err
	}
	seasons, _, err := seasonIndex(ctx, s.Store.Read, s.today())
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	info, ok := seasons[b.SeasonID]
	if !ok || info.ref.State == string(season.Draft) {
		return nil, FieldErrors("body.seasonId", MsgCommunicationSeason)
	}
	association, err := associationName(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'association impossible", err)
	}
	msg, err := renderCommunication(association, info.ref.Label, b.Subject, b.Body)
	if err != nil {
		return nil, s.internalError(ctx, "rendu d'une communication impossible", err)
	}
	return &PreviewOutput{Body: Preview{Subject: msg.Subject, Text: msg.Text, HTML: msg.HTML}}, nil
}

func (s *server) sendCommunication(ctx context.Context, in *SendCommunicationInput) (*CommunicationOutput, error) {
	b := in.Body
	if err := checkMessage(&b.Subject, &b.Body); err != nil {
		return nil, err
	}
	author := CurrentAccount(ctx).ID
	var (
		id       int64
		existing bool
		sent     resolvedTarget
		files    int
	)
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		// Double soumission : la communication existe déjà (FR-014).
		prev, err := q.GetCommunicationIDByDraftKey(ctx, b.DraftKey)
		if err == nil {
			id, existing = prev, true
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		r, err := s.audienceFor(ctx, q, b.Target, "body.target.")
		if err != nil {
			return err
		}
		if len(r.audience.Recipients) != b.ExpectedRecipients {
			return &audienceChangedError{
				ErrorModel: huma.ErrorModel{Status: http.StatusConflict, Title: errorTitles[http.StatusConflict], Detail: MsgCommunicationChanged},
				Audience:   r.toAudience(),
			}
		}
		if len(r.audience.Recipients) == 0 {
			return FieldErrors("body.target", MsgCommunicationNoRecipient)
		}
		drafts, err := q.ListDraftAttachments(ctx, sqlc.ListDraftAttachmentsParams{DraftKey: b.DraftKey, UploadedBy: nid(author)})
		if err != nil {
			return err
		}
		if err := s.checkDrafts(drafts); err != nil {
			return err
		}
		association, err := associationName(ctx, q)
		if err != nil {
			return err
		}
		msg, err := renderCommunication(association, r.season.ref.Label, b.Subject, b.Body)
		if err != nil {
			return err
		}
		now := s.timestamp()
		id, err = q.CreateCommunication(ctx, sqlc.CreateCommunicationParams{
			DraftKey: b.DraftKey, AuthorID: nid(author), SeasonID: r.season.ref.ID,
			Situations: communication.EncodeSituations(r.situations), FeeIds: communication.EncodeFeeIDs(r.feeIDs),
			TargetLabel: r.label, Subject: b.Subject, Body: b.Body,
			MembersCount: int64(r.audience.MembersCount()), UnreachableCount: int64(len(r.audience.Unreachable)),
			RecipientsCount: int64(len(r.audience.Recipients)), Now: now,
		})
		if err != nil {
			return err
		}
		if len(drafts) > 0 {
			if _, err := q.AttachDraftToCommunication(ctx, sqlc.AttachDraftToCommunicationParams{
				CommunicationID: nid(id), DraftKey: b.DraftKey, UploadedBy: nid(author),
			}); err != nil {
				return err
			}
		}
		for _, rc := range r.audience.Recipients {
			emailID, err := q.EnqueueCommunicationEmail(ctx, sqlc.EnqueueCommunicationEmailParams{
				Recipient: rc.Email, Subject: msg.Subject, BodyText: msg.Text, BodyHtml: msg.HTML, Now: now,
				CommunicationID: nid(id), FromName: ns(truncateRunes(association, 120)),
			})
			if err != nil {
				return err
			}
			if err := q.CreateCommunicationRecipient(ctx, sqlc.CreateCommunicationRecipientParams{
				CommunicationID: id, Email: ns(rc.Email), MembersCount: int64(len(rc.MemberIDs)), OutboundEmailID: nid(emailID),
			}); err != nil {
				return err
			}
		}
		sent, files = r, len(drafts)
		return nil
	})
	if err != nil {
		return nil, s.apiError(ctx, "envoi d'une communication impossible", err)
	}
	status := http.StatusCreated
	if existing {
		status = http.StatusOK
	} else {
		s.Logger.InfoContext(ctx, "communication envoyée", "communication_id", id, "author_id", author,
			"recipients", len(sent.audience.Recipients), "attachments", files)
	}
	out, err := s.communicationOut(ctx, s.Store.Write, id)
	if err != nil {
		return nil, s.apiError(ctx, "lecture d'une communication impossible", err)
	}
	return &CommunicationOutput{Status: status, Body: out}, nil
}

// checkDrafts revérifie, au moment de l'envoi, le nombre, la taille et
// l'intégrité des pièces de la préparation.
func (s *server) checkDrafts(drafts []sqlc.CommunicationAttachment) error {
	var total int64
	for i, d := range drafts {
		if err := communication.CheckLimits(i, total, d.SizeBytes); err != nil {
			return FieldErrors("body.draftKey", attachmentLimitMessage(err, total))
		}
		total += d.SizeBytes
		if s.Files == nil || s.Files.Verify(d.StoredName, d.SizeBytes, d.Sha256) != nil {
			return FieldErrors("body.draftKey", MsgCommunicationAttachment)
		}
	}
	return nil
}

func truncateRunes(v string, max int) string {
	r := []rune(v)
	if len(r) <= max {
		return v
	}
	return string(r[:max])
}

func toAttachments(rows []sqlc.CommunicationAttachment) []Attachment {
	out := make([]Attachment, len(rows))
	for i, r := range rows {
		out[i] = Attachment{ID: r.ID, Name: r.OriginalName, ContentType: r.ContentType, SizeBytes: r.SizeBytes}
	}
	return out
}

// communicationOut relit une communication complète.
func (s *server) communicationOut(ctx context.Context, q *sqlc.Queries, id int64) (Communication, error) {
	row, err := q.GetCommunication(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Communication{}, huma.Error404NotFound(MsgCommunicationGone)
	}
	if err != nil {
		return Communication{}, err
	}
	c := row.Communication
	attachments, err := q.ListCommunicationAttachments(ctx, nid(id))
	if err != nil {
		return Communication{}, err
	}
	recipients, err := q.ListCommunicationRecipients(ctx, id)
	if err != nil {
		return Communication{}, err
	}
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return Communication{}, err
	}
	out := Communication{
		CommunicationSummary: CommunicationSummary{
			ID: c.ID, Subject: c.Subject, TargetLabel: c.TargetLabel,
			Preset:           string(communication.PresetOf(communication.DecodeSituations(c.Situations))),
			Author:           accountRef(c.AuthorID, row.AuthorEmail),
			RecipientsCount:  c.RecipientsCount,
			AttachmentsCount: int64(len(attachments)),
			CreatedAt:        parseTime(c.CreatedAt),
		},
		Body:             c.Body,
		Season:           CommunicationSeason{ID: c.SeasonID, Label: seasons[c.SeasonID].ref.Label},
		Target:           Target{SeasonID: c.SeasonID, Situations: strings.Split(c.Situations, ","), FeeIDs: communication.DecodeFeeIDs(c.FeeIds)},
		MembersCount:     c.MembersCount,
		UnreachableCount: c.UnreachableCount,
		Attachments:      toAttachments(attachments),
		Recipients:       make([]Recipient, len(recipients)),
	}
	for i, r := range recipients {
		out.Recipients[i] = Recipient{Email: optionalValue(r.Email), MembersCount: r.MembersCount, Status: r.Status, SentAt: optionalTimeValue(r.SentAt)}
		switch r.Status {
		case "pending":
			out.Delivery.Pending++
		case "sent":
			out.Delivery.Sent++
		default:
			out.Delivery.Failed++
		}
	}
	return out, nil
}

func (s *server) getCommunication(ctx context.Context, in *CommunicationIDInput) (*CommunicationOutput, error) {
	out, err := s.communicationOut(ctx, s.Store.Read, in.ID)
	if err != nil {
		return nil, s.apiError(ctx, "lecture d'une communication impossible", err)
	}
	return &CommunicationOutput{Status: http.StatusOK, Body: out}, nil
}

func (s *server) listCommunications(ctx context.Context, in *ListCommunicationsInput) (*CommunicationPageOutput, error) {
	q := s.Store.Read
	total, err := q.CountCommunications(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des communications impossible", err)
	}
	rows, err := q.ListCommunications(ctx, sqlc.ListCommunicationsParams{MaxCount: in.PerPage, Skip: (in.Page - 1) * in.PerPage})
	if err != nil {
		return nil, s.internalError(ctx, "lecture des communications impossible", err)
	}
	items := make([]CommunicationSummary, len(rows))
	for i, r := range rows {
		items[i] = CommunicationSummary{
			ID: r.ID, Subject: r.Subject, TargetLabel: r.TargetLabel,
			Preset:           string(communication.PresetOf(communication.DecodeSituations(r.Situations))),
			Author:           accountRef(r.AuthorID, r.AuthorEmail),
			RecipientsCount:  r.RecipientsCount,
			AttachmentsCount: r.AttachmentsCount,
			Delivery:         Delivery{Pending: r.PendingCount, Sent: r.SentCount, Failed: r.FailedCount},
			CreatedAt:        parseTime(r.CreatedAt),
		}
	}
	return &CommunicationPageOutput{Body: CommunicationPage{Items: items, Total: total}}, nil
}
