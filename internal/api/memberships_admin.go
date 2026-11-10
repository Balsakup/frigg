package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
	"github.com/balsakup/frigg/internal/textfold"
)

// Administration (« Adhésions ») : liste, chiffres, décisions, création
// directe, corrections et constatation des paiements.

func (s *server) registerAdminMemberships(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagMemberships}, DefaultStatus: status}
	}
	register(api, auth.Admin, op("list-memberships", http.MethodGet, "/admin/memberships", "Lister les adhésions", 0), s.listMemberships)
	register(api, auth.Admin, op("get-membership-stats", http.MethodGet, "/admin/memberships/stats", "Chiffres de l'adhésion d'une saison", 0), s.getMembershipStats)
	register(api, auth.Admin, op("get-membership-dashboard", http.MethodGet, "/admin/memberships/dashboard", "Indicateurs d'adhésion du tableau de bord", 0), s.getMembershipDashboard)
	register(api, auth.Admin, op("get-membership", http.MethodGet, "/admin/memberships/{id}", "Une adhésion", 0), s.getMembership)
	register(api, auth.Admin, op("create-membership", http.MethodPost, "/admin/memberships", "Enregistrer une adhésion validée", http.StatusCreated), s.createMembership)
	register(api, auth.Admin, op("validate-membership", http.MethodPost, "/admin/memberships/{id}/validate", "Valider une demande d'adhésion", 0), s.validateMembership)
	register(api, auth.Admin, op("refuse-membership", http.MethodPost, "/admin/memberships/{id}/refuse", "Refuser une demande d'adhésion", 0), s.refuseMembership)
	register(api, auth.Admin, op("update-membership", http.MethodPut, "/admin/memberships/{id}", "Corriger une adhésion validée", 0), s.updateMembership)
	register(api, auth.Admin, op("cancel-membership", http.MethodPost, "/admin/memberships/{id}/cancel", "Annuler une adhésion validée", 0), s.cancelMembership)
	register(api, auth.Admin, op("record-membership-payment", http.MethodPost, "/admin/memberships/{id}/payment", "Constater le paiement d'une adhésion", 0), s.recordMembershipPayment)
	register(api, auth.Admin, op("cancel-membership-payment", http.MethodDelete, "/admin/memberships/{id}/payment", "Annuler la constatation du paiement", 0), s.cancelMembershipPayment)
	register(api, auth.Admin, op("update-membership-fields", http.MethodPut, "/admin/memberships/{id}/fields", "Modifier les champs personnalisés d'une adhésion", 0), s.updateMembershipFields)
}

// AdminMembershipActions sont les actions proposées à un administrateur.
type AdminMembershipActions struct {
	Validate      bool `json:"validate"`
	Refuse        bool `json:"refuse"`
	Update        bool `json:"update"`
	Cancel        bool `json:"cancel"`
	RecordPayment bool `json:"recordPayment"`
	CancelPayment bool `json:"cancelPayment"`
	// VerifyCardPayment : une tentative de paiement par carte peut être
	// vérifiée auprès de HelloAsso (research.md 008 › R5).
	VerifyCardPayment bool `json:"verifyCardPayment"`
}

// AdminMembership est une adhésion vue par l'administration.
type AdminMembership struct {
	Membership
	Account     *MemberAccount `json:"account" doc:"Compte qui gère la fiche ; null : sans compte"`
	RequestedBy *AccountRef    `json:"requestedBy"`
	CreatedBy   *AccountRef    `json:"createdBy"`
	DecidedBy   *AccountRef    `json:"decidedBy"`
	CancelledBy *AccountRef    `json:"cancelledBy"`
	PaidBy      *AccountRef    `json:"paidBy"`
	PaymentNote *string        `json:"paymentNote"`
	// Remises (014) : réservés aux administrateurs (research.md 014 › R9).
	BaseAmountCents  int64                  `json:"baseAmountCents" doc:"Montant théorique : tarif et type retenus (montant du tarif actuel tant que la demande est en attente)"`
	AdjustmentCents  int64                  `json:"adjustmentCents" doc:"Montant à payer moins montant théorique : négatif = remise, positif = majoration, 0 sans ajustement"`
	AdjustmentReason *string                `json:"adjustmentReason" doc:"Motif de l'ajustement"`
	PreviousSeason   string                 `json:"previousSeason" enum:"yes,no,unknown" doc:"Adhérent la saison précédente dans Frigg"`
	SeasonFees       []MembershipFeeRef     `json:"seasonFees" doc:"Tarifs de la saison, pour valider ou corriger"`
	Actions          AdminMembershipActions `json:"actions"`
	OnlinePayments   []OnlinePayment        `json:"onlinePayments" doc:"Tentatives de paiement par carte, la plus récente d'abord"`
	Alerts           []PaymentAlert         `json:"alerts" doc:"Signalements de paiement, non traités d'abord"`
	// CardPaymentPending : une tentative de moins de 60 minutes est en cours.
	CardPaymentPending bool `json:"cardPaymentPending"`
	// CustomFields : champs d'adhésion actifs ; non modifiables si l'adhésion
	// est refusée ou annulée (011, FR-016).
	CustomFields []FieldValue `json:"customFields" doc:"Champs personnalisés d'adhésion, dans l'ordre d'affichage"`
}

// AdminMembershipFee est le tarif d'une ligne de la liste.
type AdminMembershipFee struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
}

// AdminMembershipRow est une ligne de la liste des adhésions.
type AdminMembershipRow struct {
	ID           int64              `json:"id"`
	Member       MemberRef          `json:"member"`
	Season       SeasonRef          `json:"season"`
	Fee          AdminMembershipFee `json:"fee"`
	Status       string             `json:"status" enum:"pending,validated,refused,cancelled"`
	Kind         *string            `json:"kind" enum:"first,renewal"`
	DeclaredKind *string            `json:"declaredKind" enum:"first,renewal"`
	AmountCents  *int64             `json:"amountCents" doc:"Montant à payer ; null tant que l'adhésion n'est pas validée"`
	// BaseAmountCents et AdjustmentCents : montant théorique et ajustement (014).
	BaseAmountCents *int64   `json:"baseAmountCents" doc:"Montant théorique ; null tant que l'adhésion n'est pas validée"`
	AdjustmentCents int64    `json:"adjustmentCents" doc:"Montant à payer moins montant théorique ; 0 sans ajustement"`
	Payment         *Payment `json:"payment"`
	RequestedAt     string   `json:"requestedAt" format:"date-time"`
	OpenAlerts      int64    `json:"openAlerts" doc:"Signalements de paiement à traiter"`
}

// AdminMembershipPage est une page de la liste des adhésions.
type AdminMembershipPage struct {
	SeasonID *int64               `json:"seasonId" doc:"Saison retenue ; null : toutes"`
	Items    []AdminMembershipRow `json:"items" doc:"Demandes en attente d'abord, puis par nom et prénom"`
	Total    int64                `json:"total"`
	Page     int64                `json:"page"`
	PageSize int64                `json:"pageSize"`
}

// MembershipFeeStat est la répartition d'un tarif.
type MembershipFeeStat struct {
	FeeID       int64  `json:"feeId"`
	Label       string `json:"label"`
	Count       int    `json:"count"`
	AmountCents int64  `json:"amountCents"`
}

// MembershipDueByMethod détaille le restant à percevoir par moyen annoncé.
type MembershipDueByMethod struct {
	None     int64 `json:"none" doc:"Aucun moyen choisi"`
	Cash     int64 `json:"cash"`
	Cheque   int64 `json:"cheque"`
	Transfer int64 `json:"transfer"`
}

// MembershipStats sont les chiffres d'une saison (FR-018).
type MembershipStats struct {
	SeasonID    int64                 `json:"seasonId"`
	Validated   int                   `json:"validated"`
	First       int                   `json:"first"`
	Renewal     int                   `json:"renewal"`
	Pending     int                   `json:"pending"`
	ByFee       []MembershipFeeStat   `json:"byFee"`
	TotalCents  int64                 `json:"totalCents"`
	PaidCents   int64                 `json:"paidCents"`
	DueCents    int64                 `json:"dueCents"`
	DueByMethod MembershipDueByMethod `json:"dueByMethod"`
	// CollectedByMethod détaille paidCents ; collectedTest compte à part les
	// paiements par carte en mode test, exclus de paidCents.
	CollectedByMethod MembershipCollectedByMethod `json:"collectedByMethod"`
	CollectedTest     int64                       `json:"collectedTest" doc:"Paiements par carte en mode test (aucun débit réel)"`
	// Remises et dons (014). TotalCents est le montant total à payer,
	// PaidCents l'encaissé dons compris, DueCents le reste à percevoir.
	TheoreticalCents int64 `json:"theoreticalCents" doc:"Montant théorique total (tarifs et types retenus)"`
	DiscountCents    int64 `json:"discountCents" doc:"Total des remises"`
	SurchargeCents   int64 `json:"surchargeCents" doc:"Total des majorations"`
	DonationCents    int64 `json:"donationCents" doc:"Dons compris dans l'encaissé"`
}

// MembershipCollectedByMethod est le montant encaissé par moyen constaté.
type MembershipCollectedByMethod struct {
	Card     int64 `json:"card"`
	Cash     int64 `json:"cash"`
	Cheque   int64 `json:"cheque"`
	Transfer int64 `json:"transfer"`
}

// MembershipDashboard sont les indicateurs du tableau de bord.
type MembershipDashboard struct {
	PendingCount       int64 `json:"pendingCount" doc:"Demandes en attente, toutes saisons"`
	PaymentSetupNeeded bool  `json:"paymentSetupNeeded" doc:"Aucun moyen de paiement activé alors qu'une saison est ouverte à l'adhésion"`
	PaymentAlerts      int64 `json:"paymentAlerts" doc:"Signalements de paiement des adhésions à traiter"`
	// RegistrationPaymentAlerts : signalements des inscriptions (012).
	RegistrationPaymentAlerts int64 `json:"registrationPaymentAlerts" doc:"Signalements de paiement des inscriptions à traiter"`
	HelloAssoFailing          bool  `json:"helloassoFailing" doc:"Carte activée et connexion à HelloAsso en échec"`
}

// DecisionBody est le tarif et le type retenus, avec le montant à payer
// éventuellement ajusté (014).
type DecisionBody struct {
	FeeID            int64  `json:"feeId"`
	Kind             string `json:"kind" enum:"first,renewal"`
	Amount           string `json:"amount,omitempty" required:"false" maxLength:"12" doc:"Montant à payer en euros, par exemple « 55 » ; vide : montant théorique du tarif et du type"`
	AdjustmentReason string `json:"adjustmentReason,omitempty" required:"false" maxLength:"200" doc:"Motif de l'ajustement, obligatoire si le montant diffère du montant théorique"`
}

// ValidateBody ajoute à la décision les champs personnalisés saisis (011).
type ValidateBody struct {
	DecisionBody
	CustomFields []FieldInput `json:"customFields,omitempty" required:"false" maxItems:"30" doc:"Champs personnalisés d'adhésion ; un champ absent n'est pas modifié"`
}

// CustomFieldsBody est une saisie de champs personnalisés seule.
type CustomFieldsBody struct {
	CustomFields []FieldInput `json:"customFields" maxItems:"30"`
}

type ValidateInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body ValidateBody
}

type MembershipFieldsInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body CustomFieldsBody
}

// ReasonBody est un motif de refus ou d'annulation.
type ReasonBody struct {
	Reason string `json:"reason" maxLength:"500"`
}

// RecordPaymentBody est un paiement constaté.
type RecordPaymentBody struct {
	Method     string `json:"method" enum:"cash,cheque,transfer"`
	ReceivedOn string `json:"receivedOn" format:"date"`
	Note       string `json:"note,omitempty" required:"false" maxLength:"200"`
	// ReceivedAmount : montant reçu en euros (014) ; vide : montant à payer.
	ReceivedAmount string `json:"receivedAmount,omitempty" required:"false" maxLength:"12" doc:"Montant reçu en euros, par exemple « 50 » ; vide : montant à payer. L'excédent est un don."`
}

// AdminCreateBody est une adhésion enregistrée directement.
type AdminCreateBody struct {
	MemberID int64              `json:"memberId"`
	SeasonID int64              `json:"seasonId"`
	FeeID    int64              `json:"feeId"`
	Kind     string             `json:"kind" enum:"first,renewal"`
	Paid     *RecordPaymentBody `json:"paid,omitempty" required:"false" doc:"Règlement déjà reçu"`
	// Montant à payer ajusté (014).
	Amount           string `json:"amount,omitempty" required:"false" maxLength:"12" doc:"Montant à payer en euros ; vide : montant théorique du tarif et du type"`
	AdjustmentReason string `json:"adjustmentReason,omitempty" required:"false" maxLength:"200" doc:"Motif de l'ajustement, obligatoire si le montant diffère du montant théorique"`
	// CustomFields : champs personnalisés d'adhésion saisis (011).
	CustomFields []FieldInput `json:"customFields,omitempty" required:"false" maxItems:"30" doc:"Champs personnalisés d'adhésion"`
}

type ListMembershipsInput struct {
	Season  string `query:"season" pattern:"^(all|[0-9]+)$" doc:"Identifiant de saison ou all ; absent : saison courante, sinon la plus récente ouverte, sinon la plus récente"`
	Status  string `query:"status" enum:"pending,validated,refused,cancelled"`
	Payment string `query:"payment" enum:"due,announced,paid,free"`
	Member  int64  `query:"member" minimum:"0" doc:"Adhésions d'une fiche"`
	Q       string `query:"q" maxLength:"100" doc:"Fragments d'au moins 3 caractères cherchés dans la référence de paiement ou la fiche ; un nombre est aussi cherché parmi les commandes HelloAsso"`
	// PaidMethod filtre les adhésions payées par un moyen.
	PaidMethod string   `query:"paidMethod" enum:"card,cash,cheque,transfer"`
	Alerts     string   `query:"alerts" enum:"open" doc:"open : adhésions portant un signalement de paiement à traiter"`
	Field      []string `query:"field,explode" maxItems:"10" doc:"Filtre par champ personnalisé d'adhésion oui/non ou liste de choix : <champ>:true, <champ>:false, <champ>:empty ou <champ>:c<choix> ; répétable"`
	Page       int64    `query:"page" minimum:"1" default:"1"`
	PageSize   int64    `query:"pageSize" minimum:"1" maximum:"100" default:"25"`
}

type StatsInput struct {
	Season int64 `query:"season" required:"true" minimum:"1"`
}

type AdminMembershipPageOutput struct{ Body AdminMembershipPage }
type MembershipStatsOutput struct{ Body MembershipStats }
type MembershipDashboardOutput struct{ Body MembershipDashboard }
type AdminMembershipOutput struct{ Body AdminMembership }

type CreateMembershipInput struct{ Body AdminCreateBody }

type DecisionInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body DecisionBody
}

type ReasonInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body ReasonBody
}

type RecordPaymentInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body RecordPaymentBody
}

func ns(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }

func nid(v int64) sql.NullInt64 { return sql.NullInt64{Int64: v, Valid: true} }

// defaultSeason : saison courante, sinon la plus récente ouverte, sinon la plus
// récente (research.md 006 › R10).
func defaultSeason(seasons map[int64]seasonInfo, order []int64) *int64 {
	for _, id := range order {
		if seasons[id].current {
			return &id
		}
	}
	for i := len(order) - 1; i >= 0; i-- {
		if seasons[order[i]].ref.State == string(season.Open) {
			return &order[i]
		}
	}
	if len(order) > 0 {
		return &order[len(order)-1]
	}
	return nil
}

func (s *server) listMemberships(ctx context.Context, in *ListMembershipsInput) (*AdminMembershipPageOutput, error) {
	q, today := s.Store.Read, s.today()
	seasons, order, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	params := sqlc.ListMembershipsParams{Status: ns(in.Status), Payment: ns(in.Payment), PaidMethod: ns(in.PaidMethod)}
	if params.FieldFilters, err = fieldFilters(ctx, q, customfield.ScopeMembership, in.Field); err != nil {
		return nil, s.txError(ctx, "lecture des champs personnalisés impossible", err)
	}
	if in.Alerts == "open" {
		params.OpenAlerts = nid(1)
	}
	var seasonID *int64
	switch in.Season {
	case "all":
	case "":
		seasonID = defaultSeason(seasons, order)
	default:
		id, _ := strconv.ParseInt(in.Season, 10, 64)
		seasonID = &id
	}
	if seasonID != nil {
		params.SeasonID = nid(*seasonID)
	}
	if in.Member != 0 {
		params.MemberID = nid(in.Member)
	}
	if in.Q != "" {
		fts, ok := database.FTSTrigramQuery(in.Q)
		if !ok {
			return nil, FieldErrors("query.q", MsgSearchTooShort)
		}
		params.Query = ns(fts)
		if n, err := strconv.ParseInt(strings.TrimSpace(in.Q), 10, 64); err == nil {
			params.OrderID = nid(n)
		}
	}
	rows, err := q.ListMemberships(ctx, params)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	slices.SortFunc(rows, func(a, b sqlc.ListMembershipsRow) int {
		ap, bp := a.Membership.Status == string(membership.Pending), b.Membership.Status == string(membership.Pending)
		switch {
		case ap && !bp:
			return -1
		case bp && !ap:
			return 1
		case ap && bp:
			// Des plus anciennes aux plus récentes ; à date égale, ordre de dépôt.
			return cmp.Or(
				cmp.Compare(a.Membership.RequestedAt, b.Membership.RequestedAt),
				cmp.Compare(a.Membership.ID, b.Membership.ID),
			)
		}
		return cmp.Or(
			cmp.Compare(textfold.Fold(a.LastName), textfold.Fold(b.LastName)),
			cmp.Compare(textfold.Fold(a.FirstName), textfold.Fold(b.FirstName)),
			cmp.Compare(a.Membership.ID, b.Membership.ID),
		)
	})
	total := int64(len(rows))
	start := min((in.Page-1)*in.PageSize, total)
	end := min(start+in.PageSize, total)
	alertRows, err := q.ListOpenAlertCounts(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des signalements impossible", err)
	}
	openAlerts := make(map[int64]int64, len(alertRows))
	for _, a := range alertRows {
		openAlerts[a.MembershipID] = a.OpenAlerts
	}
	items := make([]AdminMembershipRow, 0, end-start)
	for _, r := range rows[start:end] {
		m := r.Membership
		row := AdminMembershipRow{
			ID:           m.ID,
			Member:       memberRef(m.MemberID, r.FirstName, r.LastName, r.BirthDate, today),
			Season:       seasons[m.SeasonID].ref,
			Fee:          AdminMembershipFee{ID: m.FeeID, Label: r.FeeLabel},
			Status:       m.Status,
			Kind:         optionalValue(m.Kind),
			DeclaredKind: optionalValue(m.DeclaredKind),
			Payment:      paymentOf(m),
			RequestedAt:  m.RequestedAt,
			OpenAlerts:   openAlerts[m.ID],
		}
		if m.AmountCents.Valid {
			row.AmountCents = &m.AmountCents.Int64
			theoretical := membership.Theoretical(m.AmountCents.Int64, basePtr(m.BaseAmountCents))
			row.BaseAmountCents = &theoretical
			row.AdjustmentCents = membership.AdjustmentOf(m.AmountCents.Int64, basePtr(m.BaseAmountCents))
		}
		var withRef Membership
		withRef.Payment = row.Payment
		if err := withOnline(ctx, q, &withRef, m); err != nil {
			return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
		}
		items = append(items, row)
	}
	return &AdminMembershipPageOutput{Body: AdminMembershipPage{SeasonID: seasonID, Items: items, Total: total, Page: in.Page, PageSize: in.PageSize}}, nil
}

func (s *server) getMembershipStats(ctx context.Context, in *StatsInput) (*MembershipStatsOutput, error) {
	q := s.Store.Read
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	if _, ok := seasons[in.Season]; !ok {
		return nil, huma.Error404NotFound(MsgSeasonGone)
	}
	rows, err := q.ListSeasonMembershipsForStats(ctx, in.Season)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	stat := make([]membership.StatRow, len(rows))
	for i, r := range rows {
		stat[i] = membership.StatRow{
			Status:          membership.Status(r.Status),
			Kind:            membership.Kind(r.Kind.String),
			AmountCents:     r.AmountCents.Int64,
			FeeID:           r.FeeID,
			FeeLabel:        r.FeeLabel,
			PaymentStatus:   membership.PaymentStatus(r.PaymentStatus.String),
			AnnouncedMethod: membership.Method(r.AnnouncedMethod.String),
			PaidMethod:      membership.Method(r.PaidMethod.String),
			PaidAmountCents: r.PaidAmountCents.Int64,
			BaseCents:       basePtr(r.BaseAmountCents),
			PaidTest:        r.PaidEnvironment.String == config.HelloAssoSandbox,
		}
	}
	st := membership.ComputeStats(stat)
	out := MembershipStats{
		SeasonID: in.Season, Validated: st.Validated, First: st.First, Renewal: st.Renewal, Pending: st.Pending,
		ByFee:      make([]MembershipFeeStat, len(st.ByFee)),
		TotalCents: st.TotalCents, PaidCents: st.PaidCents, DueCents: st.DueCents,
		DueByMethod: MembershipDueByMethod{
			None: st.DueByMethod.None, Cash: st.DueByMethod.Cash, Cheque: st.DueByMethod.Cheque, Transfer: st.DueByMethod.Transfer,
		},
		CollectedByMethod: MembershipCollectedByMethod{
			Card: st.PaidByMethod.Card, Cash: st.PaidByMethod.Cash, Cheque: st.PaidByMethod.Cheque, Transfer: st.PaidByMethod.Transfer,
		},
		CollectedTest:    st.TestPaidCents,
		TheoreticalCents: st.TheoreticalCents, DiscountCents: st.DiscountCents,
		SurchargeCents: st.SurchargeCents, DonationCents: st.DonationCents,
	}
	for i, f := range st.ByFee {
		out.ByFee[i] = MembershipFeeStat{FeeID: f.FeeID, Label: f.Label, Count: f.Count, AmountCents: f.AmountCents}
	}
	return &MembershipStatsOutput{Body: out}, nil
}

func (s *server) getMembershipDashboard(ctx context.Context, _ *struct{}) (*MembershipDashboardOutput, error) {
	q := s.Store.Read
	pending, err := q.CountPendingMemberships(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	alerts, err := q.CountOpenMembershipPaymentAlerts(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des signalements impossible", err)
	}
	row, err := q.GetPaymentSettings(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	regAlerts, err := q.CountOpenRegistrationPaymentAlerts(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des signalements impossible", err)
	}
	out := MembershipDashboard{PendingCount: pending, PaymentAlerts: alerts, RegistrationPaymentAlerts: regAlerts, HelloAssoFailing: s.helloAssoFailing(row)}
	if len(settings.EnabledMethods()) == 0 && !settings.CardAvailable {
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return nil, s.internalError(ctx, "lecture des saisons impossible", err)
		}
		for id, info := range seasons {
			if info.ref.State != string(season.Open) {
				continue
			}
			fees, err := q.ListMembershipFees(ctx, id)
			if err != nil {
				return nil, s.internalError(ctx, "lecture des tarifs impossible", err)
			}
			if len(fees) > 0 {
				out.PaymentSetupNeeded = true
				break
			}
		}
	}
	return &MembershipDashboardOutput{Body: out}, nil
}

func accountRef(id sql.NullInt64, email sql.NullString) *AccountRef {
	if !id.Valid || !email.Valid {
		return nil
	}
	return &AccountRef{ID: id.Int64, Email: email.String}
}

// adminMembership relit une adhésion complète pour l'administration.
func (s *server) adminMembership(ctx context.Context, q *sqlc.Queries, id int64) (*AdminMembershipOutput, error) {
	r, err := q.GetMembership(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMembershipGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'adhésion impossible", err)
	}
	today := s.today()
	seasons, order, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	fees, err := q.ListMembershipFees(ctx, r.Membership.SeasonID)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des tarifs impossible", err)
	}
	hint, err := previousSeasonHint(ctx, q, seasons, order, r.Membership)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	m := r.Membership
	a := membership.AdminActionsFor(snapshotOf(m, seasons))
	out := AdminMembership{
		Membership:       toMembership(m, r.MembershipFee, r.FirstName, r.LastName, r.BirthDate, seasons, today),
		RequestedBy:      accountRef(m.RequestedBy, r.RequestedByEmail),
		CreatedBy:        accountRef(m.CreatedBy, r.CreatedByEmail),
		DecidedBy:        accountRef(m.DecidedBy, r.DecidedByEmail),
		CancelledBy:      accountRef(m.CancelledBy, r.CancelledByEmail),
		PaidBy:           accountRef(m.PaidBy, r.PaidByEmail),
		PaymentNote:      optionalValue(m.PaymentNote),
		AdjustmentReason: optionalValue(m.AdjustmentReason),
		PreviousSeason:   string(hint),
		SeasonFees:       toFeeRefs(fees),
		Actions: AdminMembershipActions{
			Validate: a.Validate, Refuse: a.Refuse, Update: a.Update, Cancel: a.Cancel,
			RecordPayment: a.RecordPayment, CancelPayment: a.CancelPayment,
		},
	}
	out.BaseAmountCents = membership.Theoretical(out.AmountCents, basePtr(m.BaseAmountCents))
	out.AdjustmentCents = membership.AdjustmentOf(out.AmountCents, basePtr(m.BaseAmountCents))
	if r.MemberAccountID.Valid {
		out.Account = &MemberAccount{ID: r.MemberAccountID.Int64, Email: r.AccountEmail.String, Disabled: r.AccountDisabledAt.Valid}
	}
	if err := withOnline(ctx, q, &out.Membership, m); err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	if err := s.withOnlineAdmin(ctx, q, &out, m); err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	if out.CustomFields, err = readMembershipFields(ctx, q, id, customfield.Admin, fieldsWritable(m.Status)); err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	return &AdminMembershipOutput{Body: out}, nil
}

// previousSeasonHint calcule l'aide « Adhérent la saison précédente dans
// Frigg » (research.md 006 › R6).
func previousSeasonHint(ctx context.Context, q *sqlc.Queries, seasons map[int64]seasonInfo, order []int64, m sqlc.Membership) (membership.Hint, error) {
	current := seasons[m.SeasonID]
	var previous int64
	for _, id := range order {
		if seasons[id].endsOn.Equal(current.startsOn.AddDays(-1)) {
			previous = id
		}
	}
	if previous == 0 {
		return membership.PreviousSeasonHint(false, 0, false), nil
	}
	count, err := q.CountValidatedInSeason(ctx, previous)
	if err != nil {
		return "", err
	}
	mine, err := q.HasValidatedMembership(ctx, sqlc.HasValidatedMembershipParams{MemberID: m.MemberID, SeasonID: previous})
	if err != nil {
		return "", err
	}
	return membership.PreviousSeasonHint(true, count, mine > 0), nil
}

func (s *server) getMembership(ctx context.Context, in *MembershipIDInput) (*AdminMembershipOutput, error) {
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

// adminWriteContext est l'adhésion relue dans la transaction.
type adminWriteContext struct {
	q       *sqlc.Queries
	m       sqlc.Membership
	snap    membership.Snapshot
	seasons map[int64]seasonInfo
}

// membershipWrite exécute fn sur une adhésion relue dans une transaction
// d'écriture, puis renvoie l'adhésion à jour.
func (s *server) membershipWrite(ctx context.Context, id int64, fn func(w adminWriteContext) error) (sqlc.Membership, error) {
	var current sqlc.Membership
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetMembership(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMembershipGone)
		}
		if err != nil {
			return err
		}
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return err
		}
		current = r.Membership
		return fn(adminWriteContext{q: q, m: r.Membership, snap: snapshotOf(r.Membership, seasons), seasons: seasons})
	})
	if err != nil {
		return current, s.txError(ctx, "écriture de l'adhésion impossible", err)
	}
	return current, nil
}

// decide calcule le tarif, le type et le montant à payer retenus : le
// montant théorique du tarif, éventuellement ajusté (research.md 014 › R5).
func decide(ctx context.Context, w adminWriteContext, body DecisionBody) (sqlc.MembershipFee, membership.Adjustment, error) {
	fees, err := w.q.ListMembershipFees(ctx, w.m.SeasonID)
	if err != nil {
		return sqlc.MembershipFee{}, membership.Adjustment{}, err
	}
	fee, err := feeOfSeason(fees, body.FeeID, "body.feeId")
	if err != nil {
		return fee, membership.Adjustment{}, err
	}
	a, err := adjust(membership.AppliedAmount(toFee(fee), membership.Kind(body.Kind)), body.Amount, body.AdjustmentReason, "body")
	return fee, a, err
}

// adjustmentLog renvoie les attributs de journal d'un ajustement, sans motif
// (research.md 014 › R7).
func adjustmentLog(a membership.Adjustment) []any {
	return []any{"amount_cents", a.AmountCents, "base_amount_cents", membership.Theoretical(a.AmountCents, a.BaseCents), "adjusted", a.BaseCents != nil}
}

func (s *server) validateMembership(ctx context.Context, in *ValidateInput) (*AdminMembershipOutput, error) {
	var adj membership.Adjustment
	fields := 0
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		if !membership.CanDecide(w.snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		fee, a, err := decide(ctx, w, in.Body.DecisionBody)
		if err != nil {
			return err
		}
		adj = a
		ref, status, _, err := paymentColumns(ctx, w.q, w.seasons[w.m.SeasonID], adj.AmountCents, w.m)
		if err != nil {
			return err
		}
		base, reason := adjustmentColumns(adj)
		now := s.timestamp()
		if err := changed(w.q.ValidateMembership(ctx, sqlc.ValidateMembershipParams{
			FeeID: fee.ID, Kind: ns(in.Body.Kind), AmountCents: nid(adj.AmountCents),
			PaymentReference: ref, PaymentStatus: status, BaseAmountCents: base, AdjustmentReason: reason,
			ActorID: nid(CurrentAccount(ctx).ID), Now: ns(now), ID: w.m.ID,
		})); err != nil {
			return err
		}
		if fields, err = writeFields(ctx, w.q, customfield.ScopeMembership, w.m.ID, in.Body.CustomFields, customfield.Admin); err != nil {
			return err
		}
		return s.notifyMembership(ctx, w.q, "membership-validated", w.m.ID, true)
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.validate", m, "admin", append([]any{"kind", in.Body.Kind, "custom_fields", fields}, adjustmentLog(adj)...)...)
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

func (s *server) refuseMembership(ctx context.Context, in *ReasonInput) (*AdminMembershipOutput, error) {
	reason, err := membership.CleanReason(in.Body.Reason)
	if err != nil {
		return nil, reasonError(err)
	}
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		if !membership.CanDecide(w.snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		if err := changed(w.q.RefuseMembership(ctx, sqlc.RefuseMembershipParams{
			Reason: ns(reason), ActorID: nid(CurrentAccount(ctx).ID), Now: ns(s.timestamp()), ID: w.m.ID,
		})); err != nil {
			return err
		}
		return s.notifyMembership(ctx, w.q, "membership-refused", w.m.ID, false)
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.refuse", m, "admin")
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

func (s *server) updateMembership(ctx context.Context, in *DecisionInput) (*AdminMembershipOutput, error) {
	var adj membership.Adjustment
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		switch {
		case w.snap.Status == membership.Validated && w.snap.PaymentStatus == membership.Paid:
			return huma.Error409Conflict(MsgPaidCorrection)
		case w.snap.Status == membership.Validated && w.snap.SeasonClosed:
			return huma.Error409Conflict(MsgSeasonClosedAction)
		case !membership.CanUpdate(w.snap):
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		fee, a, err := decide(ctx, w, in.Body)
		if err != nil {
			return err
		}
		adj = a
		ref, status, announced, err := paymentColumns(ctx, w.q, w.seasons[w.m.SeasonID], adj.AmountCents, w.m)
		if err != nil {
			return err
		}
		base, reason := adjustmentColumns(adj)
		return changed(w.q.UpdateValidatedMembership(ctx, sqlc.UpdateValidatedMembershipParams{
			FeeID: fee.ID, Kind: ns(in.Body.Kind), AmountCents: nid(adj.AmountCents),
			PaymentReference: ref, PaymentStatus: status, AnnouncedMethod: announced,
			BaseAmountCents: base, AdjustmentReason: reason,
			Now: s.timestamp(), ID: w.m.ID,
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.update", m, "admin", append([]any{"kind", in.Body.Kind, "old_amount_cents", m.AmountCents.Int64}, adjustmentLog(adj)...)...)
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

func (s *server) cancelMembership(ctx context.Context, in *ReasonInput) (*AdminMembershipOutput, error) {
	reason, err := membership.CleanReason(in.Body.Reason)
	if err != nil {
		return nil, reasonError(err)
	}
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		switch {
		case w.snap.Status == membership.Validated && w.snap.SeasonClosed:
			return huma.Error409Conflict(MsgSeasonClosedAction)
		case !membership.CanCancel(w.snap):
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		if err := changed(w.q.CancelMembership(ctx, sqlc.CancelMembershipParams{
			Reason: ns(reason), ActorID: nid(CurrentAccount(ctx).ID), Now: ns(s.timestamp()), ID: w.m.ID,
		})); err != nil {
			return err
		}
		// Payée par carte : le remboursement se fait dans HelloAsso (R10).
		if w.m.PaidMethod.String == string(membership.Card) && w.m.PaidOnlinePaymentID.Valid {
			if err := w.q.CreatePaymentAlert(ctx, sqlc.CreatePaymentAlertParams{
				MembershipID: nid(w.m.ID), OnlinePaymentID: w.m.PaidOnlinePaymentID.Int64,
				Kind: string(membership.AlertRefundDue), Now: s.timestamp(),
			}); err != nil {
				return err
			}
		}
		return s.notifyMembership(ctx, w.q, "membership-cancelled", w.m.ID, false)
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.cancel", m, "admin")
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

// checkPayment valide un paiement saisi : date de réception entre notBefore
// et aujourd'hui, commentaire facultatif.
func (s *server) checkPayment(body RecordPaymentBody, location string, notBefore season.Date, beforeMsg string) (string, string, error) {
	received, err := season.ParseDate(body.ReceivedOn)
	if err != nil {
		return "", "", FieldErrors(location+".receivedOn", MsgInvalidDate)
	}
	if received.After(s.today()) {
		return "", "", FieldErrors(location+".receivedOn", MsgReceivedFuture)
	}
	if received.Before(notBefore) {
		return "", "", FieldErrors(location+".receivedOn", beforeMsg)
	}
	note, err := membership.CleanNote(body.Note)
	if err != nil {
		return "", "", FieldErrors(location+".note", MsgNoteTooLong)
	}
	return received.String(), note, nil
}

func (s *server) recordMembershipPayment(ctx context.Context, in *RecordPaymentInput) (*AdminMembershipOutput, error) {
	var received int64
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		if !membership.CanRecordPayment(w.snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		paidOn, note, err := s.checkPayment(in.Body, "body", dateOfTimestamp(w.m.DecidedAt.String), MsgReceivedBeforeValidation)
		if err != nil {
			return err
		}
		if received, err = receivedAmount(in.Body, "body", w.m.AmountCents.Int64); err != nil {
			return err
		}
		if err := changed(w.q.RecordMembershipPayment(ctx, sqlc.RecordMembershipPaymentParams{
			Method: ns(in.Body.Method), PaidOn: ns(paidOn), ActorID: nid(CurrentAccount(ctx).ID),
			Now: ns(s.timestamp()), Note: ns(note), ID: w.m.ID, PaidAmountCents: nid(received),
		})); err != nil {
			return err
		}
		return s.notifyMembership(ctx, w.q, "membership-paid", w.m.ID, false)
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.payment_record", m, "admin", "method", in.Body.Method,
		"received_cents", received, "donation_cents", membership.Donation(membership.Method(in.Body.Method), m.AmountCents.Int64, received))
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

func (s *server) cancelMembershipPayment(ctx context.Context, in *MembershipIDInput) (*AdminMembershipOutput, error) {
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		if !membership.CanCancelPayment(w.snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		return changed(w.q.CancelMembershipPayment(ctx, sqlc.CancelMembershipPaymentParams{Now: s.timestamp(), ID: w.m.ID}))
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.payment_cancel", m, "admin")
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}

func (s *server) createMembership(ctx context.Context, in *CreateMembershipInput) (*AdminMembershipOutput, error) {
	var created sqlc.Membership
	var adj membership.Adjustment
	var received int64
	paid := false
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetMember(ctx, in.Body.MemberID); errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgMemberGone)
		} else if err != nil {
			return err
		}
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return err
		}
		fees, err := offeredSeason(ctx, q, seasons, in.Body.SeasonID)
		if err != nil {
			return err
		}
		fee, err := feeOfSeason(fees, in.Body.FeeID, "body.feeId")
		if err != nil {
			return err
		}
		info := seasons[in.Body.SeasonID]
		adj, err = adjust(membership.AppliedAmount(toFee(fee), membership.Kind(in.Body.Kind)), in.Body.Amount, in.Body.AdjustmentReason, "body")
		if err != nil {
			return err
		}
		amount := adj.AmountCents
		var paidOn, note string
		if in.Body.Paid != nil && amount > 0 {
			// Règlement d'un bulletin papier : reçu au plus tôt au début de la
			// saison, éventuellement avant la saisie (FR-022d).
			paidOn, note, err = s.checkPayment(*in.Body.Paid, "body.paid", info.startsOn, MsgReceivedBeforeSeason)
			if err != nil {
				return err
			}
			if received, err = receivedAmount(*in.Body.Paid, "body.paid", amount); err != nil {
				return err
			}
		}
		ref, status, _, err := paymentColumns(ctx, q, info, amount, sqlc.Membership{})
		if err != nil {
			return err
		}
		base, reason := adjustmentColumns(adj)
		now := s.timestamp()
		actor := nid(CurrentAccount(ctx).ID)
		id, err := q.CreateMembership(ctx, sqlc.CreateMembershipParams{
			MemberID: in.Body.MemberID, SeasonID: in.Body.SeasonID, FeeID: fee.ID,
			Status: string(membership.Validated), Kind: ns(in.Body.Kind), AmountCents: nid(amount),
			CreatedBy: actor, Now: now, DecidedBy: actor, DecidedAt: ns(now),
			PaymentReference: ref, PaymentStatus: status, BaseAmountCents: base, AdjustmentReason: reason,
		})
		if err != nil {
			return activeConflict(err)
		}
		created = sqlc.Membership{ID: id, MemberID: in.Body.MemberID, SeasonID: in.Body.SeasonID}
		if _, err := writeFields(ctx, q, customfield.ScopeMembership, id, in.Body.CustomFields, customfield.Admin); err != nil {
			return err
		}
		if paidOn != "" {
			paid = true
			if err := changed(q.RecordMembershipPayment(ctx, sqlc.RecordMembershipPaymentParams{
				Method: ns(in.Body.Paid.Method), PaidOn: ns(paidOn), ActorID: actor, Now: ns(now), Note: ns(note), ID: id,
				PaidAmountCents: nid(received),
			})); err != nil {
				return err
			}
		}
		return s.notifyMembership(ctx, q, "membership-validated", id, true)
	})
	if err != nil {
		return nil, s.txError(ctx, "création de l'adhésion impossible", err)
	}
	s.logMembershipAction(ctx, "membership.create", created, "admin", append([]any{"kind", in.Body.Kind}, adjustmentLog(adj)...)...)
	if paid {
		s.logMembershipAction(ctx, "membership.payment_record", created, "admin", "method", in.Body.Paid.Method,
			"received_cents", received, "donation_cents", membership.Donation(membership.Method(in.Body.Paid.Method), adj.AmountCents, received))
	}
	out, err := s.adminMembership(ctx, s.Store.Read, created.ID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// fieldsWritable indique si les champs personnalisés d'une adhésion sont
// encore modifiables : en attente ou validée, payée ou non (research.md 011 › R8).
func fieldsWritable(status string) bool {
	return status == string(membership.Pending) || status == string(membership.Validated)
}

// MsgMembershipFieldsLocked : adhésion refusée ou annulée (FR-016).
const MsgMembershipFieldsLocked = "Cette adhésion ne peut plus être modifiée."

// updateMembershipFields enregistre les champs personnalisés d'une adhésion
// sans toucher à son état, son tarif ni son paiement (US3).
func (s *server) updateMembershipFields(ctx context.Context, in *MembershipFieldsInput) (*AdminMembershipOutput, error) {
	fields := 0
	m, err := s.membershipWrite(ctx, in.ID, func(w adminWriteContext) error {
		if !fieldsWritable(w.m.Status) {
			return huma.Error409Conflict(MsgMembershipFieldsLocked)
		}
		var err error
		fields, err = writeFields(ctx, w.q, customfield.ScopeMembership, w.m.ID, in.Body.CustomFields, customfield.Admin)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.fields", m, "admin", "custom_fields", fields)
	return s.adminMembership(ctx, s.Store.Read, in.ID)
}
