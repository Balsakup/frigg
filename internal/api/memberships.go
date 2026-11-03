package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Adhésions (research.md 006) : types, conversions et assistants communs aux
// deux espaces.

const (
	tagMyMemberships = "Mes adhésions"
	tagMemberships   = "Adhésions"
	tagSettings      = "Paramètres"

	MsgMembershipGone           = "Cette adhésion n'existe plus."
	MsgMembershipActive         = "Ce membre a déjà une adhésion en cours pour cette saison."
	MsgMembershipChanged        = "Cette adhésion a changé entre-temps."
	MsgPaidCorrection           = "Annulez d'abord la constatation du paiement."
	MsgSeasonClosedAction       = "Cette saison est close."
	MsgSeasonNotOffered         = "Cette saison n'est pas ouverte à l'adhésion."
	MsgFeeNotInSeason           = "Choisissez un tarif de cette saison."
	MsgReasonRequired           = "Indiquez le motif."
	MsgReasonTooLong            = "Le motif ne peut pas dépasser 500 caractères."
	MsgNoteTooLong              = "Le commentaire ne peut pas dépasser 200 caractères."
	MsgMethodUnavailable        = "Ce moyen de paiement n'est plus proposé. Choisissez-en un autre."
	MsgCardIsOnline             = "Le paiement par carte se fait en ligne."
	MsgReceivedFuture           = "La date de réception ne peut pas être dans le futur."
	MsgReceivedBeforeValidation = "La date de réception ne peut pas précéder la validation de l'adhésion."
	MsgReceivedBeforeSeason     = "La date de réception ne peut pas précéder le début de la saison."
	MsgFeeInUse                 = "Ce tarif est utilisé par au moins une adhésion : il ne peut pas être supprimé."
	MsgMemberHasMembership      = "Ce membre a une adhésion en cours. Annulez la demande ou contactez le bureau."
)

// FeeAmounts sont les montants d'un tarif selon le type d'adhésion, calculés
// par le serveur (principe VI : le frontend ne recalcule aucun montant).
type FeeAmounts struct {
	First   int64 `json:"first" doc:"Montant d'une première adhésion, en centimes"`
	Renewal int64 `json:"renewal" doc:"Montant d'un renouvellement, en centimes"`
}

// MembershipFeeRef est un tarif tel qu'il apparaît dans une adhésion.
type MembershipFeeRef struct {
	ID                 int64      `json:"id"`
	Label              string     `json:"label"`
	AmountCents        int64      `json:"amountCents"`
	RenewalAmountCents *int64     `json:"renewalAmountCents"`
	Description        *string    `json:"description"`
	Amounts            FeeAmounts `json:"amounts"`
}

// SeasonRef est la saison d'une adhésion.
type SeasonRef struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	State string `json:"state" enum:"draft,open,closed"`
}

// MemberRef est la fiche d'une adhésion.
type MemberRef struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Age       int    `json:"age"`
}

// AccountRef est un compte auteur d'une action ; null s'il a été supprimé.
type AccountRef struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

// Payment est le paiement d'une adhésion validée non gratuite.
type Payment struct {
	Reference       string  `json:"reference"`
	Status          string  `json:"status" enum:"due,announced,paid"`
	AnnouncedMethod *string `json:"announcedMethod" enum:"cash,cheque,transfer"`
	PaidMethod      *string `json:"paidMethod" enum:"card,cash,cheque,transfer"`
	PaidOn          *string `json:"paidOn" format:"date"`
	// PaidAmountCents : carte, montant reçu hors contribution volontaire.
	PaidAmountCents *int64            `json:"paidAmountCents"`
	Online          *OnlinePaymentRef `json:"online" doc:"Carte : paiement HelloAsso qui a constaté l'adhésion"`
}

// OnlinePaymentRef résume le paiement HelloAsso d'une adhésion payée par carte.
type OnlinePaymentRef struct {
	OrderID int64     `json:"orderId" doc:"Numéro de commande HelloAsso"`
	PaidAt  time.Time `json:"paidAt"`
	Test    bool      `json:"test" doc:"Payée dans l'environnement de test HelloAsso"`
}

// Membership est une adhésion.
type Membership struct {
	ID           int64            `json:"id"`
	Member       MemberRef        `json:"member"`
	Season       SeasonRef        `json:"season"`
	Fee          MembershipFeeRef `json:"fee" doc:"Tarif actuel"`
	Status       string           `json:"status" enum:"pending,validated,refused,cancelled"`
	DeclaredKind *string          `json:"declaredKind" enum:"first,renewal" doc:"Type déclaré par le compte ; null si créée par un administrateur"`
	Kind         *string          `json:"kind" enum:"first,renewal" doc:"Type retenu à la validation"`
	AmountCents  int64            `json:"amountCents" doc:"Montant appliqué s'il est figé, sinon montant du tarif actuel pour le type déclaré"`
	AmountFixed  bool             `json:"amountFixed" doc:"Le montant a été figé à la validation"`
	Payment      *Payment         `json:"payment" doc:"null : adhésion non validée ou gratuite"`
	Reason       *string          `json:"reason" doc:"Motif d'un refus ou d'une annulation"`
	RequestedAt  time.Time        `json:"requestedAt"`
	DecidedAt    *time.Time       `json:"decidedAt"`
	CancelledAt  *time.Time       `json:"cancelledAt"`
}

// seasonInfo est une saison avec son libellé et son état calculés.
type seasonInfo struct {
	ref      SeasonRef
	startsOn season.Date
	endsOn   season.Date
	current  bool
}

// seasonIndex charge la suite des saisons, indexée par identifiant, et leur
// ordre chronologique.
func seasonIndex(ctx context.Context, q *sqlc.Queries, today season.Date) (map[int64]seasonInfo, []int64, error) {
	rows, err := q.ListSeasons(ctx)
	if err != nil {
		return nil, nil, err
	}
	built, err := buildSeasons(rows, today)
	if err != nil {
		return nil, nil, err
	}
	index := make(map[int64]seasonInfo, len(built))
	order := make([]int64, len(built))
	for i, b := range built {
		start, _ := season.ParseDate(b.StartsOn)
		end, _ := season.ParseDate(b.EndsOn)
		index[b.ID] = seasonInfo{
			ref:      SeasonRef{ID: b.ID, Label: b.Label, State: b.Status},
			startsOn: start,
			endsOn:   end,
			current:  b.IsCurrent,
		}
		order[i] = b.ID
	}
	return index, order, nil
}

// offeredSeason vérifie qu'une saison accepte de nouvelles adhésions : ouverte,
// non close, au moins un tarif (FR-006, FR-014). Elle renvoie ses tarifs.
func offeredSeason(ctx context.Context, q *sqlc.Queries, seasons map[int64]seasonInfo, id int64) ([]sqlc.MembershipFee, error) {
	info, ok := seasons[id]
	if !ok || info.ref.State != string(season.Open) {
		return nil, FieldErrors("body.seasonId", MsgSeasonNotOffered)
	}
	fees, err := q.ListMembershipFees(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(fees) == 0 {
		return nil, FieldErrors("body.seasonId", MsgSeasonNotOffered)
	}
	return fees, nil
}

// feeOfSeason renvoie le tarif feeID parmi les tarifs d'une saison.
func feeOfSeason(fees []sqlc.MembershipFee, feeID int64, location string) (sqlc.MembershipFee, error) {
	for _, f := range fees {
		if f.ID == feeID {
			return f, nil
		}
	}
	return sqlc.MembershipFee{}, FieldErrors(location, MsgFeeNotInSeason)
}

func toFeeRef(r sqlc.MembershipFee) MembershipFeeRef {
	f := toFee(r)
	return MembershipFeeRef{
		ID:                 r.ID,
		Label:              r.Label,
		AmountCents:        r.AmountCents,
		RenewalAmountCents: f.RenewalAmountCents,
		Description:        f.Description,
		Amounts:            feeAmounts(f),
	}
}

func feeAmounts(f season.Fee) FeeAmounts {
	return FeeAmounts{
		First:   membership.AppliedAmount(f, membership.First),
		Renewal: membership.AppliedAmount(f, membership.Renewal),
	}
}

func toFeeRefs(rows []sqlc.MembershipFee) []MembershipFeeRef {
	fees := toFees(rows)
	season.SortFees(fees)
	byID := make(map[int64]sqlc.MembershipFee, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}
	out := make([]MembershipFeeRef, len(fees))
	for i, f := range fees {
		out[i] = toFeeRef(byID[f.ID])
	}
	return out
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeFormat, s)
	return t
}

func optionalTimeValue(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t := parseTime(v.String)
	return &t
}

func memberRef(id int64, first, last, birth string, today season.Date) MemberRef {
	ref := MemberRef{ID: id, FirstName: first, LastName: last}
	if b, err := season.ParseDate(birth); err == nil {
		ref.Age = member.Age(b, today)
	}
	return ref
}

func paymentOf(m sqlc.Membership) *Payment {
	if !m.PaymentStatus.Valid || !m.PaymentReference.Valid {
		return nil
	}
	return &Payment{
		Reference:       m.PaymentReference.String,
		Status:          m.PaymentStatus.String,
		AnnouncedMethod: optionalValue(m.AnnouncedMethod),
		PaidMethod:      optionalValue(m.PaidMethod),
		PaidOn:          optionalValue(m.PaidOn),
		PaidAmountCents: optionalInt(m.PaidAmountCents),
	}
}

func optionalInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

// withOnline complète le paiement d'une adhésion payée par carte avec la
// commande HelloAsso qui l'a constatée.
func withOnline(ctx context.Context, q *sqlc.Queries, out *Membership, m sqlc.Membership) error {
	if out.Payment == nil || !m.PaidOnlinePaymentID.Valid {
		return nil
	}
	op, err := q.GetOnlinePayment(ctx, m.PaidOnlinePaymentID.Int64)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	out.Payment.Online = &OnlinePaymentRef{
		OrderID: op.OrderID.Int64, PaidAt: parseTime(op.PaidAt.String), Test: op.Environment == config.HelloAssoSandbox,
	}
	return nil
}

// toMembership convertit une adhésion lue en base.
func toMembership(m sqlc.Membership, fee sqlc.MembershipFee, first, last, birth string, seasons map[int64]seasonInfo, today season.Date) Membership {
	out := Membership{
		ID:           m.ID,
		Member:       memberRef(m.MemberID, first, last, birth, today),
		Season:       seasons[m.SeasonID].ref,
		Fee:          toFeeRef(fee),
		Status:       m.Status,
		DeclaredKind: optionalValue(m.DeclaredKind),
		Kind:         optionalValue(m.Kind),
		Payment:      paymentOf(m),
		Reason:       optionalValue(m.Reason),
		RequestedAt:  parseTime(m.RequestedAt),
		DecidedAt:    optionalTimeValue(m.DecidedAt),
		CancelledAt:  optionalTimeValue(m.CancelledAt),
	}
	if m.AmountCents.Valid {
		out.AmountCents, out.AmountFixed = m.AmountCents.Int64, true
	} else {
		kind := membership.First
		if m.DeclaredKind.Valid {
			kind = membership.Kind(m.DeclaredKind.String)
		}
		out.AmountCents = membership.AppliedAmount(toFee(fee), kind)
	}
	return out
}

func snapshotOf(m sqlc.Membership, seasons map[int64]seasonInfo) membership.Snapshot {
	return membership.Snapshot{
		Status:        membership.Status(m.Status),
		PaymentStatus: membership.PaymentStatus(m.PaymentStatus.String),
		SeasonClosed:  seasons[m.SeasonID].ref.State == string(season.Closed),
		AmountCents:   m.AmountCents.Int64,
		PaidMethod:    membership.Method(m.PaidMethod.String),
	}
}

// assignReference renvoie la référence de paiement suivante pour la saison
// (research.md 006 › R4).
func assignReference(ctx context.Context, q *sqlc.Queries, info seasonInfo) (string, error) {
	prefix := membership.ReferencePrefix(info.startsOn, info.endsOn)
	rows, err := q.ListPaymentReferences(ctx, prefix)
	if err != nil {
		return "", err
	}
	existing := make([]string, len(rows))
	for i, r := range rows {
		existing[i] = r.String
	}
	return membership.NextReference(prefix, existing), nil
}

// paymentColumns calcule les colonnes de paiement après une validation ou une
// correction donnant le montant amount.
func paymentColumns(ctx context.Context, q *sqlc.Queries, info seasonInfo, amount int64, current sqlc.Membership) (ref, status, announced sql.NullString, err error) {
	ps, method := membership.PaymentAfter(amount, membership.PaymentStatus(current.PaymentStatus.String), membership.Method(current.AnnouncedMethod.String))
	ref = current.PaymentReference
	if ps != membership.NoPayment {
		status = sql.NullString{String: string(ps), Valid: true}
		if !ref.Valid {
			r, err := assignReference(ctx, q, info)
			if err != nil {
				return ref, status, announced, err
			}
			ref = sql.NullString{String: r, Valid: true}
		}
	}
	if method != "" {
		announced = sql.NullString{String: string(method), Valid: true}
	}
	return ref, status, announced, nil
}

// activeConflict traduit la violation de memberships_one_active (FR-003).
func activeConflict(err error) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: memberships.member_id") {
		return huma.Error409Conflict(MsgMembershipActive)
	}
	return err
}

// reasonError traduit une erreur de motif.
func reasonError(err error) error {
	if errors.Is(err, membership.ErrReasonTooLong) {
		return FieldErrors("body.reason", MsgReasonTooLong)
	}
	return FieldErrors("body.reason", MsgReasonRequired)
}

// timestamp est l'horodatage courant au format de la base, selon l'horloge
// injectable (research.md 003 › R3).
func (s *server) timestamp() string {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	return now().UTC().Format(timeFormat)
}

// dateOfTimestamp renvoie la date à Paris d'un horodatage stocké.
func dateOfTimestamp(v string) season.Date {
	return season.Today(parseTime(v))
}

var frenchMonths = [...]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet",
	"août", "septembre", "octobre", "novembre", "décembre"}

// frenchDate formate une date comme « 3 novembre 2026 ».
func frenchDate(d season.Date) string {
	t, _ := time.Parse("2006-01-02", d.String())
	day := fmt.Sprint(t.Day())
	if t.Day() == 1 {
		day = "1er"
	}
	return fmt.Sprintf("%s %s %d", day, frenchMonths[t.Month()-1], t.Year())
}

// formatEuros formate un montant en centimes comme le frontend : « 40,00 € »,
// « 1 250,00 € » (espaces fines insécables).
func formatEuros(cents int64) string {
	euros := fmt.Sprintf("%d", cents/100)
	var b strings.Builder
	for i, r := range euros {
		if i > 0 && (len(euros)-i)%3 == 0 {
			b.WriteString(" ")
		}
		b.WriteRune(r)
	}
	return fmt.Sprintf("%s,%02d €", b.String(), cents%100)
}

var methodLabels = map[string]string{"card": "carte bancaire", "cash": "espèces", "cheque": "chèque", "transfer": "virement"}

// noticeRecipient renvoie l'adresse du compte qui gère la fiche, s'il existe et
// n'est pas désactivé (FR-023).
func noticeRecipient(r sqlc.GetMembershipRow) string {
	if !r.MemberAccountID.Valid || !r.AccountEmail.Valid || r.AccountDisabledAt.Valid {
		return ""
	}
	return r.AccountEmail.String
}

// notifyMembership met en file un e-mail d'adhésion au compte qui gère la
// fiche, relu dans la transaction (research.md 006 › R8).
func (s *server) notifyMembership(ctx context.Context, q *sqlc.Queries, template string, id int64, withLink bool) error {
	r, err := q.GetMembership(ctx, id)
	if err != nil {
		return err
	}
	to := noticeRecipient(r)
	if to == "" {
		return nil
	}
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return err
	}
	m := r.Membership
	data := mail.Data{
		Member:    r.FirstName + " " + r.LastName,
		Season:    seasons[m.SeasonID].ref.Label,
		Reference: m.PaymentReference.String,
		Reason:    m.Reason.String,
		Method:    methodLabels[m.PaidMethod.String],
	}
	if m.AmountCents.Valid {
		data.Amount = formatEuros(m.AmountCents.Int64)
	}
	if m.PaidAmountCents.Valid {
		data.Amount = formatEuros(m.PaidAmountCents.Int64)
	}
	if d, err := season.ParseDate(m.PaidOn.String); err == nil && m.PaidOn.Valid {
		data.PaidOn = frenchDate(d)
	}
	path := ""
	if withLink && m.PaymentStatus.Valid {
		path = fmt.Sprintf("/adhesions/%d", m.ID)
	}
	return s.enqueueNotice(ctx, q, template, to, data, path)
}

// logMembershipAction trace une écriture réussie sur une adhésion, sans nom,
// motif ni commentaire (research.md 006 › R12).
func (s *server) logMembershipAction(ctx context.Context, action string, m sqlc.Membership, space string, attrs ...any) {
	base := []any{
		"action", action, "actor_id", CurrentAccount(ctx).ID, "membership_id", m.ID,
		"member_id", m.MemberID, "season_id", m.SeasonID, "space", space,
	}
	s.Logger.InfoContext(ctx, "écriture sur une adhésion", append(base, attrs...)...)
}

func (s *server) registerMemberships(api huma.API) {
	s.registerMyMemberships(api)
	s.registerAdminMemberships(api)
	s.registerSettings(api)
	s.registerGeneralSettings(api)
}
