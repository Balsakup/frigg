package api

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
	"github.com/balsakup/frigg/internal/textfold"
)

// Espace membre (« Adhésions »). Le compte vient uniquement de la session et
// chaque requête est conditionnée par le compte qui gère la fiche
// (research.md 006 › R5) : une adhésion d'une autre fiche est indiscernable
// d'une adhésion inexistante.

func (s *server) registerMyMemberships(api huma.API) {
	register(api, auth.Member, huma.Operation{
		OperationID: "get-membership-offer",
		Method:      http.MethodGet,
		Path:        "/me/memberships/offer",
		Summary:     "Saisons et tarifs ouverts à l'adhésion pour mes membres",
		Tags:        []string{tagMyMemberships},
	}, s.getMembershipOffer)

	register(api, auth.Member, huma.Operation{
		OperationID: "list-my-memberships",
		Method:      http.MethodGet,
		Path:        "/me/memberships",
		Summary:     "Adhésions de mes membres",
		Tags:        []string{tagMyMemberships},
	}, s.listMyMemberships)

	register(api, auth.Member, huma.Operation{
		OperationID: "get-my-membership",
		Method:      http.MethodGet,
		Path:        "/me/memberships/{id}",
		Summary:     "Une adhésion de mes membres",
		Tags:        []string{tagMyMemberships},
	}, s.getMyMembership)

	register(api, auth.Member, huma.Operation{
		OperationID:   "create-my-membership",
		Method:        http.MethodPost,
		Path:          "/me/memberships",
		Summary:       "Demander l'adhésion d'un de mes membres",
		Tags:          []string{tagMyMemberships},
		DefaultStatus: http.StatusCreated,
	}, s.createMyMembership)

	register(api, auth.Member, huma.Operation{
		OperationID: "cancel-my-membership",
		Method:      http.MethodPost,
		Path:        "/me/memberships/{id}/cancel",
		Summary:     "Annuler une demande d'adhésion en attente",
		Tags:        []string{tagMyMemberships},
	}, s.cancelMyMembership)

	register(api, auth.Member, huma.Operation{
		OperationID: "announce-my-payment",
		Method:      http.MethodPut,
		Path:        "/me/memberships/{id}/payment",
		Summary:     "Choisir le moyen de paiement d'une adhésion",
		Tags:        []string{tagMyMemberships},
	}, s.announceMyPayment)
}

// OfferSeason est une saison ouverte à l'adhésion, avec ses tarifs.
type OfferSeason struct {
	ID    int64              `json:"id"`
	Label string             `json:"label"`
	Fees  []MembershipFeeRef `json:"fees" doc:"Du montant le plus élevé au plus faible"`
}

// OfferMember est une fiche du compte et les saisons où elle peut adhérer.
type OfferMember struct {
	ID                 int64   `json:"id"`
	FirstName          string  `json:"firstName"`
	LastName           string  `json:"lastName"`
	IsSelf             bool    `json:"isSelf"`
	AvailableSeasonIDs []int64 `json:"availableSeasonIds" doc:"Saisons où la fiche n'a ni demande en attente ni adhésion validée"`
}

// MembershipOffer est ce que le compte peut demander.
type MembershipOffer struct {
	Seasons    []OfferSeason `json:"seasons" doc:"Saisons ouvertes, non closes, avec au moins un tarif ; la plus récente d'abord"`
	Members    []OfferMember `json:"members" doc:"Fiches du compte, « Vous » d'abord"`
	CanRequest bool          `json:"canRequest" doc:"Au moins une fiche peut adhérer à au moins une saison"`
}

// MyMembershipActions sont les actions proposées au compte.
type MyMembershipActions struct {
	Cancel    bool `json:"cancel" doc:"Annuler la demande en attente"`
	Pay       bool `json:"pay" doc:"Choisir ou changer le moyen de paiement"`
	PayByCard bool `json:"payByCard" doc:"Payer en ligne par carte bancaire (HelloAsso)"`
}

// CardPaymentState est la dernière tentative de paiement par carte.
type CardPaymentState struct {
	Status    string    `json:"status" enum:"pending,succeeded,failed,abandoned,expired,refunded"`
	StartedAt time.Time `json:"startedAt"`
}

// PaymentMethodInfo est un moyen de paiement activé et ses informations.
type PaymentMethodInfo struct {
	Method       string  `json:"method" enum:"card,cash,cheque,transfer"`
	AmountCents  *int64  `json:"amountCents" doc:"Carte : montant à payer"`
	Test         bool    `json:"test" doc:"Carte : environnement de test HelloAsso, aucun débit réel"`
	Instructions *string `json:"instructions"`
	Payee        *string `json:"payee" doc:"Chèque : ordre"`
	Address      *string `json:"address" doc:"Chèque : adresse d'envoi"`
	Holder       *string `json:"holder" doc:"Virement : titulaire du compte"`
	IBAN         *string `json:"iban" doc:"Virement : IBAN normalisé"`
	BIC          *string `json:"bic" doc:"Virement : BIC"`
}

// MyMembership est une adhésion vue par le compte qui gère la fiche.
type MyMembership struct {
	Membership
	Actions        MyMembershipActions `json:"actions"`
	PaymentMethods []PaymentMethodInfo `json:"paymentMethods" doc:"Moyens activés ; vide s'il n'y a rien à payer ; la carte en premier"`
	CardPayment    *CardPaymentState   `json:"cardPayment" doc:"Dernière tentative de paiement par carte"`
}

// MyMembershipSeason regroupe les adhésions d'une saison.
type MyMembershipSeason struct {
	Season SeasonRef      `json:"season"`
	Items  []MyMembership `json:"items" doc:"Par prénom"`
}

// MyMembershipList est la liste des adhésions des fiches du compte.
type MyMembershipList struct {
	Seasons []MyMembershipSeason `json:"seasons" doc:"La saison la plus récente d'abord"`
}

// MyMembershipBody est une demande d'adhésion.
type MyMembershipBody struct {
	MemberID     int64  `json:"memberId"`
	SeasonID     int64  `json:"seasonId"`
	FeeID        int64  `json:"feeId"`
	DeclaredKind string `json:"declaredKind" enum:"first,renewal"`
}

// AnnounceBody est le moyen de paiement choisi.
type AnnounceBody struct {
	Method string `json:"method" enum:"card,cash,cheque,transfer" doc:"La carte se paie en ligne : elle n'est jamais annoncée"`
}

type MembershipOfferOutput struct{ Body MembershipOffer }
type MyMembershipListOutput struct{ Body MyMembershipList }
type MyMembershipOutput struct{ Body MyMembership }

type ListMyMembershipsInput struct {
	Member int64 `query:"member" minimum:"0" doc:"Adhésions d'une seule fiche du compte"`
}

type MembershipIDInput struct {
	ID int64 `path:"id" doc:"Identifiant de l'adhésion"`
}

type CreateMyMembershipInput struct{ Body MyMembershipBody }

type AnnounceMyPaymentInput struct {
	ID   int64 `path:"id" doc:"Identifiant de l'adhésion"`
	Body AnnounceBody
}

func (s *server) getMembershipOffer(ctx context.Context, _ *struct{}) (*MembershipOfferOutput, error) {
	q, account, today := s.Store.Read, ownAccount(ctx), s.today()
	seasons, order, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	offer := MembershipOffer{Seasons: []OfferSeason{}, Members: []OfferMember{}}
	for i := len(order) - 1; i >= 0; i-- {
		info := seasons[order[i]]
		if info.ref.State != string(season.Open) {
			continue
		}
		fees, err := q.ListMembershipFees(ctx, info.ref.ID)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des tarifs impossible", err)
		}
		if len(fees) > 0 {
			offer.Seasons = append(offer.Seasons, OfferSeason{ID: info.ref.ID, Label: info.ref.Label, Fees: toFeeRefs(fees)})
		}
	}
	rows, err := q.ListMembersOfAccount(ctx, account)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches impossible", err)
	}
	active, err := q.ListActiveSeasonsOfAccount(ctx, account)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	taken := map[[2]int64]bool{}
	for _, a := range active {
		taken[[2]int64{a.MemberID, a.SeasonID}] = true
	}
	byID := make(map[int64]sqlc.Member, len(rows))
	sorted := make([]member.Sortable, len(rows))
	for i, r := range rows {
		byID[r.ID] = r
		sorted[i] = sortable(r)
	}
	member.SortMine(sorted)
	for _, o := range sorted {
		r := byID[o.ID]
		om := OfferMember{ID: r.ID, FirstName: r.FirstName, LastName: r.LastName, IsSelf: r.IsSelf == 1, AvailableSeasonIDs: []int64{}}
		for _, os := range offer.Seasons {
			if !taken[[2]int64{r.ID, os.ID}] {
				om.AvailableSeasonIDs = append(om.AvailableSeasonIDs, os.ID)
				offer.CanRequest = true
			}
		}
		offer.Members = append(offer.Members, om)
	}
	return &MembershipOfferOutput{Body: offer}, nil
}

// myMembership complète une adhésion pour l'espace membre : actions et
// moyens de paiement activés si elle est à payer.
func myMembership(m Membership, snap membership.Snapshot, settings membership.Settings, latest *sqlc.OnlinePayment) MyMembership {
	a := membership.MemberActionsFor(snap, settings.CardAvailable)
	out := MyMembership{
		Membership:     m,
		Actions:        MyMembershipActions{Cancel: a.Cancel, Pay: a.Pay, PayByCard: a.PayByCard},
		PaymentMethods: []PaymentMethodInfo{},
	}
	if a.PayByCard {
		amount := snap.AmountCents
		out.PaymentMethods = append(out.PaymentMethods, PaymentMethodInfo{Method: string(membership.Card), AmountCents: &amount, Test: settings.CardTest})
	}
	if a.Pay {
		out.PaymentMethods = append(out.PaymentMethods, paymentMethodInfos(settings)...)
	}
	if latest != nil {
		out.CardPayment = &CardPaymentState{Status: latest.Status, StartedAt: parseTime(latest.CreatedAt)}
	}
	return out
}

// latestCardPayment lit la dernière tentative d'une adhésion à paiement.
func latestCardPayment(ctx context.Context, q *sqlc.Queries, m sqlc.Membership) (*sqlc.OnlinePayment, error) {
	if !m.PaymentStatus.Valid {
		return nil, nil
	}
	op, err := q.LatestOnlinePaymentOfMembership(ctx, m.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

// paymentMethodInfos renvoie les informations des moyens activés.
func paymentMethodInfos(st membership.Settings) []PaymentMethodInfo {
	out := []PaymentMethodInfo{}
	for _, m := range st.EnabledMethods() {
		info := PaymentMethodInfo{Method: string(m)}
		switch m {
		case membership.Cash:
			info.Instructions = optionalString(st.CashInstructions)
		case membership.Cheque:
			info.Payee = optionalString(st.ChequePayee)
			info.Address = optionalString(st.ChequeAddress)
			info.Instructions = optionalString(st.ChequeInstructions)
		case membership.Transfer:
			info.Holder = optionalString(st.TransferHolder)
			info.IBAN = optionalString(st.TransferIBAN)
			info.BIC = optionalString(st.TransferBIC)
			info.Instructions = optionalString(st.TransferInstructions)
		}
		out = append(out, info)
	}
	return out
}

func (s *server) listMyMemberships(ctx context.Context, in *ListMyMembershipsInput) (*MyMembershipListOutput, error) {
	q, account, today := s.Store.Read, ownAccount(ctx), s.today()
	params := sqlc.ListMembershipsOfAccountParams{AccountID: account}
	if in.Member != 0 {
		if _, err := q.GetMemberOfAccount(ctx, sqlc.GetMemberOfAccountParams{ID: in.Member, AccountID: account}); errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error404NotFound(MsgMemberGone)
		} else if err != nil {
			return nil, s.internalError(ctx, "lecture de la fiche impossible", err)
		}
		params.MemberID = sql.NullInt64{Int64: in.Member, Valid: true}
	}
	rows, err := q.ListMembershipsOfAccount(ctx, params)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des adhésions impossible", err)
	}
	seasons, order, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	bySeason := map[int64][]MyMembership{}
	for _, r := range rows {
		m := toMembership(r.Membership, r.MembershipFee, r.FirstName, r.LastName, r.BirthDate, seasons, today)
		if err := withOnline(ctx, q, &m, r.Membership); err != nil {
			return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
		}
		latest, err := latestCardPayment(ctx, q, r.Membership)
		if err != nil {
			return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
		}
		bySeason[r.Membership.SeasonID] = append(bySeason[r.Membership.SeasonID], myMembership(m, snapshotOf(r.Membership, seasons), settings, latest))
	}
	list := MyMembershipList{Seasons: []MyMembershipSeason{}}
	for i := len(order) - 1; i >= 0; i-- {
		items := bySeason[order[i]]
		if len(items) == 0 {
			continue
		}
		slices.SortFunc(items, func(a, b MyMembership) int {
			return cmp.Or(
				cmp.Compare(textfold.Fold(a.Member.FirstName), textfold.Fold(b.Member.FirstName)),
				cmp.Compare(textfold.Fold(a.Member.LastName), textfold.Fold(b.Member.LastName)),
				cmp.Compare(a.ID, b.ID),
			)
		})
		list.Seasons = append(list.Seasons, MyMembershipSeason{Season: seasons[order[i]].ref, Items: items})
	}
	return &MyMembershipListOutput{Body: list}, nil
}

// myMembershipOutput relit une adhésion d'une fiche du compte.
func (s *server) myMembershipOutput(ctx context.Context, q *sqlc.Queries, id int64) (*MyMembershipOutput, error) {
	r, err := q.GetMembershipOfAccount(ctx, sqlc.GetMembershipOfAccountParams{ID: id, AccountID: ownAccount(ctx)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgMembershipGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture de l'adhésion impossible", err)
	}
	today := s.today()
	seasons, _, err := seasonIndex(ctx, q, today)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	settings, err := s.paymentSettings(ctx, q)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	m := toMembership(r.Membership, r.MembershipFee, r.FirstName, r.LastName, r.BirthDate, seasons, today)
	if err := withOnline(ctx, q, &m, r.Membership); err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	latest, err := latestCardPayment(ctx, q, r.Membership)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paiements en ligne impossible", err)
	}
	return &MyMembershipOutput{Body: myMembership(m, snapshotOf(r.Membership, seasons), settings, latest)}, nil
}

func (s *server) getMyMembership(ctx context.Context, in *MembershipIDInput) (*MyMembershipOutput, error) {
	return s.myMembershipOutput(ctx, s.Store.Read, in.ID)
}

func (s *server) createMyMembership(ctx context.Context, in *CreateMyMembershipInput) (*MyMembershipOutput, error) {
	account := ownAccount(ctx)
	var created sqlc.Membership
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		if _, err := q.GetMemberOfAccount(ctx, sqlc.GetMemberOfAccountParams{ID: in.Body.MemberID, AccountID: account}); errors.Is(err, sql.ErrNoRows) {
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
		if _, err := feeOfSeason(fees, in.Body.FeeID, "body.feeId"); err != nil {
			return err
		}
		now := s.timestamp()
		id, err := q.CreateMembership(ctx, sqlc.CreateMembershipParams{
			MemberID:     in.Body.MemberID,
			SeasonID:     in.Body.SeasonID,
			FeeID:        in.Body.FeeID,
			Status:       string(membership.Pending),
			DeclaredKind: sql.NullString{String: in.Body.DeclaredKind, Valid: true},
			RequestedBy:  account,
			Now:          now,
		})
		if err != nil {
			return activeConflict(err)
		}
		created = sqlc.Membership{ID: id, MemberID: in.Body.MemberID, SeasonID: in.Body.SeasonID}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "demande d'adhésion impossible", err)
	}
	s.logMembershipAction(ctx, "membership.request", created, "me", "kind", in.Body.DeclaredKind)
	return s.myMembershipOutput(ctx, s.Store.Read, created.ID)
}

// myMembershipWrite exécute fn sur une adhésion d'une fiche du compte, relue
// dans la transaction avec son état.
func (s *server) myMembershipWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, m sqlc.Membership, snap membership.Snapshot) error) (sqlc.Membership, error) {
	var current sqlc.Membership
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		r, err := q.GetMembershipOfAccount(ctx, sqlc.GetMembershipOfAccountParams{ID: id, AccountID: ownAccount(ctx)})
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
		return fn(q, r.Membership, snapshotOf(r.Membership, seasons))
	})
	if err != nil {
		return current, s.txError(ctx, "écriture de l'adhésion impossible", err)
	}
	return current, nil
}

// changed traduit une écriture conditionnelle sans effet en conflit.
func changed(n int64, err error) error {
	if err == nil && n == 0 {
		return huma.Error409Conflict(MsgMembershipChanged)
	}
	return err
}

func (s *server) cancelMyMembership(ctx context.Context, in *MembershipIDInput) (*MyMembershipOutput, error) {
	m, err := s.myMembershipWrite(ctx, in.ID, func(q *sqlc.Queries, m sqlc.Membership, snap membership.Snapshot) error {
		if !membership.CanCancelRequest(snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		now := s.timestamp()
		return changed(q.CancelMembershipOfAccount(ctx, sqlc.CancelMembershipOfAccountParams{
			ActorID: nid(CurrentAccount(ctx).ID), Now: ns(now), ID: m.ID, AccountID: ownAccount(ctx),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.cancel", m, "me")
	return s.myMembershipOutput(ctx, s.Store.Read, in.ID)
}

func (s *server) announceMyPayment(ctx context.Context, in *AnnounceMyPaymentInput) (*MyMembershipOutput, error) {
	m, err := s.myMembershipWrite(ctx, in.ID, func(q *sqlc.Queries, m sqlc.Membership, snap membership.Snapshot) error {
		if !membership.ManualMethod(membership.Method(in.Body.Method)) {
			return FieldErrors("body.method", MsgCardIsOnline)
		}
		if !membership.CanPay(snap) {
			return huma.Error409Conflict(MsgMembershipChanged)
		}
		settings, err := s.paymentSettings(ctx, q)
		if err != nil {
			return err
		}
		if !settings.Enabled(membership.Method(in.Body.Method)) {
			return FieldErrors("body.method", MsgMethodUnavailable)
		}
		return changed(q.AnnouncePaymentOfAccount(ctx, sqlc.AnnouncePaymentOfAccountParams{
			Method: sql.NullString{String: in.Body.Method, Valid: true}, Now: s.timestamp(), ID: m.ID, AccountID: ownAccount(ctx),
		}))
	})
	if err != nil {
		return nil, err
	}
	s.logMembershipAction(ctx, "membership.payment_announce", m, "me", "method", in.Body.Method)
	return s.myMembershipOutput(ctx, s.Store.Read, in.ID)
}
