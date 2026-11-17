package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/event"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/member"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/public"
	"github.com/balsakup/frigg/internal/season"
)

// Confirmation d'un dépôt public (research.md 016 › R7) : une transaction qui
// revérifie les conditions, déduplique, rattache au compte de l'adresse ou
// émet un lien personnel.

// ConfirmOutcome est l'issue d'une confirmation de dépôt.
type ConfirmOutcome struct {
	Outcome       string  `json:"outcome" enum:"tracking,account,duplicate,closed" doc:"tracking : suivi par lien ; account : rattachée au compte de l'adresse ; duplicate : demande déjà enregistrée ; closed : demande impossible"`
	Kind          string  `json:"kind" enum:"membership,registration"`
	Reason        *string `json:"reason" doc:"closed : raison à afficher"`
	CreateAccount bool    `json:"createAccount" doc:"tracking : la personne a demandé à créer un compte"`
}

type ConfirmOutcomeOutput struct{ Body ConfirmOutcome }

// confirmResult est le résultat d'une confirmation, relu après la transaction.
type confirmResult struct {
	outcome        public.Outcome
	reason         string
	memberID       int64
	membershipID   int64
	eventID        int64
	registrationID int64
	linkID         int64
	createAccount  bool
}

func (s *server) confirmPublicSubmission(ctx context.Context, in *TokenInput) (*ConfirmOutcomeOutput, error) {
	if !s.Limits.PublicConfirmIP.Allow(clientIP(ctx)) {
		return nil, TooManyAttempts()
	}
	var (
		res confirmResult
		sub sqlc.PublicSubmission
	)
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		var err error
		sub, err = q.GetPublicSubmissionByTokenHash(ctx, auth.HashToken(in.Body.Token))
		if errors.Is(err, sql.ErrNoRows) {
			return huma.NewError(http.StatusGone, MsgLinkInvalid)
		}
		if err != nil {
			return err
		}
		if sub.ExpiresAt <= s.timestamp() {
			return huma.NewError(http.StatusGone, MsgLinkInvalid)
		}
		// Adresse connue seulement après la lecture du jeton (R9).
		if !s.Limits.PublicConfirmEmail.Allow(sub.Email) {
			return TooManyAttempts()
		}
		switch sub.Kind {
		case "membership":
			res, err = s.confirmMembershipSubmission(ctx, q, sub)
		case "registration":
			res, err = s.confirmRegistrationSubmission(ctx, q, sub)
		default:
			err = fmt.Errorf("dépôt public de nature inconnue : %q", sub.Kind)
		}
		if err != nil {
			return err
		}
		// Le dépôt est supprimé quelle que soit l'issue : le lien est à usage
		// unique (FR-008, FR-009).
		return q.DeletePublicSubmission(ctx, sub.ID)
	})
	if err != nil {
		return nil, s.txError(ctx, "confirmation du dépôt public impossible", err)
	}
	if res.linkID != 0 {
		if err := s.openTrackingInSession(ctx, res.linkID); err != nil {
			return nil, s.internalError(ctx, "ouverture de l'accès par lien impossible", err)
		}
	}
	attrs := []any{"action", "public_confirm", "kind", sub.Kind,
		"outcome", string(res.outcome), "submission_id", sub.ID, "member_id", res.memberID}
	if sub.Kind == "registration" {
		attrs = append(attrs, "event_id", res.eventID, "registration_id", res.registrationID)
	} else {
		attrs = append(attrs, "membership_id", res.membershipID)
	}
	s.Logger.InfoContext(ctx, "dépôt public confirmé", append(attrs, "tracking_link_id", res.linkID)...)
	out := ConfirmOutcome{Outcome: string(res.outcome), Kind: sub.Kind, CreateAccount: res.createAccount}
	if res.reason != "" {
		out.Reason = &res.reason
	}
	return &ConfirmOutcomeOutput{Body: out}, nil
}

// membershipTarget revérifie, dans la transaction, que la demande d'adhésion
// d'un dépôt est toujours possible (FR-010). Elle renvoie la raison du refus,
// ou "" si la cible est ouverte.
func (s *server) membershipTarget(ctx context.Context, q *sqlc.Queries, sub sqlc.PublicSubmission) (string, error) {
	a, err := q.GetAssociation(ctx)
	if err != nil {
		return "", err
	}
	if a.PublicMembershipEnabled == 0 {
		return MsgPublicMembershipClosed, nil
	}
	seasons, _, err := seasonIndex(ctx, q, s.today())
	if err != nil {
		return "", err
	}
	if info, ok := seasons[sub.SeasonID.Int64]; !ok || info.ref.State != string(season.Open) {
		return MsgSeasonNotOffered, nil
	}
	n, err := q.CountMembershipFeeForPublic(ctx, sqlc.CountMembershipFeeForPublicParams{
		FeeID: sub.FeeID.Int64, SeasonID: sub.SeasonID.Int64,
	})
	if err != nil {
		return "", err
	}
	if n == 0 {
		return MsgPublicFeeGone, nil
	}
	return "", nil
}

// sameIdentity renvoie la première fiche candidate de même prénom et nom
// (member.IdentityKey), ou 0.
func sameIdentity[R any](rows []R, key string, ident func(R) (int64, string, string)) int64 {
	for _, r := range rows {
		id, first, last := ident(r)
		if member.IdentityKey(first, last) == key {
			return id
		}
	}
	return 0
}

// existingMember cherche une fiche de même identité que le dépôt (FR-011) :
// parmi les fiches du compte de l'adresse si la demande peut y être
// rattachée, sinon parmi les fiches suivies par lien de l'adresse.
func existingMember(ctx context.Context, q *sqlc.Queries, sub sqlc.PublicSubmission, acc *emailAccount) (id int64, inAccount bool, err error) {
	key := member.IdentityKey(sub.FirstName, sub.LastName)
	if acc != nil && acc.Active {
		rows, err := q.FindAccountMemberByIdentity(ctx, sqlc.FindAccountMemberByIdentityParams{
			AccountID: nid(acc.ID), BirthDate: sub.BirthDate,
		})
		if err != nil {
			return 0, false, err
		}
		if id := sameIdentity(rows, key, func(r sqlc.FindAccountMemberByIdentityRow) (int64, string, string) {
			return r.ID, r.FirstName, r.LastName
		}); id != 0 {
			return id, true, nil
		}
		if acc.Managed < public.MaxManagedMembers {
			// La nouvelle fiche ira au compte.
			return 0, false, nil
		}
	}
	rows, err := q.FindTrackedMemberByIdentity(ctx, sqlc.FindTrackedMemberByIdentityParams{
		Email: ns(sub.Email), BirthDate: sub.BirthDate,
	})
	if err != nil {
		return 0, false, err
	}
	return sameIdentity(rows, key, func(r sqlc.FindTrackedMemberByIdentityRow) (int64, string, string) {
		return r.ID, r.FirstName, r.LastName
	}), false, nil
}

// confirmMembershipSubmission applique la décision de confirmation d'un dépôt
// d'adhésion, dans la transaction en cours (research.md 016 › R7).
func (s *server) confirmMembershipSubmission(ctx context.Context, q *sqlc.Queries, sub sqlc.PublicSubmission) (confirmResult, error) {
	var res confirmResult
	reason, err := s.membershipTarget(ctx, q, sub)
	if err != nil {
		return res, err
	}
	acc, err := lookupEmailAccount(ctx, q, sub.Email)
	if err != nil {
		return res, err
	}
	existing, inAccount, err := existingMember(ctx, q, sub, acc)
	if err != nil {
		return res, err
	}
	st := public.State{
		TargetOpen: reason == "", ClosedReason: reason,
		ExistingMember: existing != 0, ExistingInAccount: inAccount,
	}
	if acc != nil {
		st.AccountFound, st.AccountActive, st.AccountManaged = true, acc.Active, int(acc.Managed)
	}
	if existing != 0 && st.TargetOpen {
		n, err := q.CountActiveMembershipsOfMemberInSeason(ctx, sqlc.CountActiveMembershipsOfMemberInSeasonParams{
			MemberID: existing, SeasonID: sub.SeasonID.Int64,
		})
		if err != nil {
			return res, err
		}
		st.ExistingActive = n > 0
	}
	res.outcome = public.Decide(st)
	switch res.outcome {
	case public.Closed:
		res.reason = reason
		return res, nil
	case public.Duplicate:
		res.memberID = existing
		return res, nil
	}

	now := s.timestamp()
	if res.memberID, err = submissionMember(ctx, q, sub, existing, res.outcome, acc, now); err != nil {
		return res, err
	}
	res.membershipID, err = q.CreatePublicMembership(ctx, sqlc.CreatePublicMembershipParams{
		MemberID: res.memberID, SeasonID: sub.SeasonID.Int64, FeeID: sub.FeeID.Int64,
		DeclaredKind: sub.DeclaredKind, Now: now,
	})
	if err != nil {
		return res, activeConflict(err)
	}
	if err := applySubmissionFields(ctx, q, sub.CustomValues, res.memberID, res.membershipID, existing != 0); err != nil {
		return res, err
	}
	if res.outcome != public.Tracking {
		// Rattachée au compte : la demande apparaît dans l'espace membre, et
		// l'e-mail du dépôt invitait déjà à s'y connecter (FR-033).
		return res, nil
	}
	res.createAccount = sub.CreateAccount == 1
	raw, err := s.issueTrackingLink(ctx, q, sub.Email)
	if err != nil {
		return res, err
	}
	link, err := q.GetTrackingLinkByTokenHash(ctx, auth.HashToken(raw))
	if err != nil {
		return res, err
	}
	res.linkID = link.ID
	return res, s.enqueueNotice(ctx, q, "tracking-link", sub.Email, mail.Data{}, "/suivi#jeton="+raw)
}

// submissionMember renvoie la fiche d'un dépôt confirmé : la fiche existante
// réutilisée, sinon une fiche d'origine publique créée, rattachée au compte de
// l'adresse si l'issue est Account (FR-033), sans compte sinon.
func submissionMember(ctx context.Context, q *sqlc.Queries, sub sqlc.PublicSubmission, existing int64,
	outcome public.Outcome, acc *emailAccount, now string) (int64, error) {
	if existing != 0 {
		return existing, nil
	}
	var accountID sql.NullInt64
	if outcome == public.Account {
		accountID = nid(acc.ID)
	}
	return q.CreatePublicMember(ctx, sqlc.CreatePublicMemberParams{
		AccountID: accountID, FirstName: sub.FirstName, LastName: sub.LastName, BirthDate: sub.BirthDate,
		Email: ns(sub.Email), Phone: sub.Phone, AddressLine1: sub.AddressLine1, AddressLine2: sub.AddressLine2,
		PostalCode: sub.PostalCode, City: sub.City, Now: ns(now), ConsentedAt: ns(sub.ConsentedAt),
	})
}

// registrationTarget relit, dans la transaction, l'événement d'un dépôt
// d'inscription et revérifie qu'il l'accepte encore (FR-010, FR-027) :
// publié, ouvert au public, dans sa période et non complet, la capacité étant
// comptée dans la transaction comme pour 012 FR-015. Elle renvoie la raison
// du refus, ou "" si l'inscription reste possible.
func (s *server) registrationTarget(ctx context.Context, q *sqlc.Queries, sub sqlc.PublicSubmission) (sqlc.Event, string, error) {
	r, err := q.GetEvent(ctx, sub.EventID.Int64)
	if err != nil {
		return sqlc.Event{}, "", err
	}
	e := r.Event
	if event.OpennessOf(e.Openness, e.PublicSlug) != event.Public {
		return e, MsgRegistrationsClosed, nil
	}
	confirmed, err := q.CountConfirmedRegistrations(ctx, e.ID)
	if err != nil {
		return e, "", err
	}
	if reason := event.PublicClosedReason(eventState(e), int(confirmed), s.now()); reason != event.NotClosed {
		return e, publicClosedMessages[reason], nil
	}
	return e, "", nil
}

// confirmRegistrationSubmission applique la décision de confirmation d'un
// dépôt d'inscription, dans la transaction en cours (research.md 016 › R7,
// FR-027) : inscription confirmée sans auteur, prix figé, référence de
// paiement si l'événement est payant.
func (s *server) confirmRegistrationSubmission(ctx context.Context, q *sqlc.Queries, sub sqlc.PublicSubmission) (confirmResult, error) {
	res := confirmResult{eventID: sub.EventID.Int64}
	e, reason, err := s.registrationTarget(ctx, q, sub)
	if err != nil {
		return res, err
	}
	acc, err := lookupEmailAccount(ctx, q, sub.Email)
	if err != nil {
		return res, err
	}
	existing, inAccount, err := existingMember(ctx, q, sub, acc)
	if err != nil {
		return res, err
	}
	st := public.State{
		TargetOpen: reason == "", ClosedReason: reason,
		ExistingMember: existing != 0, ExistingInAccount: inAccount,
	}
	if acc != nil {
		st.AccountFound, st.AccountActive, st.AccountManaged = true, acc.Active, int(acc.Managed)
	}
	if existing != 0 {
		n, err := q.CountMemberConfirmedRegistration(ctx, sqlc.CountMemberConfirmedRegistrationParams{
			EventID: e.ID, MemberID: existing,
		})
		if err != nil {
			return res, err
		}
		st.ExistingActive = n > 0
	}
	res.outcome = public.Decide(st)
	switch res.outcome {
	case public.Closed:
		res.reason = reason
		return res, nil
	case public.Duplicate:
		res.memberID = existing
		return res, nil
	}

	now := s.timestamp()
	if res.memberID, err = submissionMember(ctx, q, sub, existing, res.outcome, acc, now); err != nil {
		return res, err
	}
	params := sqlc.CreatePublicRegistrationParams{EventID: e.ID, MemberID: res.memberID, Now: now, AmountCents: e.PriceCents}
	if e.PriceCents > 0 {
		seasons, _, err := seasonIndex(ctx, q, s.today())
		if err != nil {
			return res, err
		}
		info := seasons[e.SeasonID]
		prefix := event.ReferencePrefix(info.startsOn, info.endsOn)
		refs, err := q.ListRegistrationReferences(ctx, prefix)
		if err != nil {
			return res, err
		}
		params.PaymentReference = ns(membership.NextReference(prefix, refs))
		params.PaymentStatus = ns(string(membership.Due))
	}
	reg, err := q.CreatePublicRegistration(ctx, params)
	if err != nil {
		return res, err
	}
	res.registrationID = reg.ID
	if res.outcome != public.Tracking {
		// Rattachée au compte : confirmation de 012 FR-030 à l'adresse du
		// compte, avec le lien vers l'espace membre (FR-033).
		return res, s.notifyCreated(ctx, q, e, []sqlc.EventRegistration{reg})
	}
	res.createAccount = sub.CreateAccount == 1
	raw, err := s.issueTrackingLink(ctx, q, sub.Email)
	if err != nil {
		return res, err
	}
	link, err := q.GetTrackingLinkByTokenHash(ctx, auth.HashToken(raw))
	if err != nil {
		return res, err
	}
	res.linkID = link.ID
	start, end := eventTimes(e)
	data := mail.Data{
		Member: sub.FirstName + " " + sub.LastName, Event: e.Title, Location: e.Location,
		When: frenchDateTimeRange(start, end),
	}
	if e.PriceCents > 0 {
		data.Amount, data.Reference = formatEuros(e.PriceCents), reg.PaymentReference.String
	}
	return res, s.enqueueNotice(ctx, q, "public-registration-confirmed", sub.Email, data,
		fmt.Sprintf("/suivi/inscriptions/%d#jeton=%s", reg.ID, raw))
}

// applySubmissionFields enregistre les valeurs des champs personnalisés d'un
// dépôt, réappliquées aux définitions courantes : une valeur d'un champ
// archivé, supprimé ou devenu non modifiable par le membre est ignorée, et un
// champ ajouté depuis le dépôt n'est pas exigé. Sur une fiche réutilisée,
// une valeur déjà enregistrée n'est pas remplacée.
func applySubmissionFields(ctx context.Context, q *sqlc.Queries, raw string, memberID, membershipID int64, reused bool) error {
	var inputs []FieldInput
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return fmt.Errorf("valeurs des champs personnalisés du dépôt illisibles : %w", err)
	}
	if len(inputs) == 0 {
		return nil
	}
	defs := map[int64]customfield.Definition{}
	for _, scope := range []customfield.Scope{customfield.ScopeMember, customfield.ScopeMembership} {
		list, err := ownerWritableDefinitions(ctx, q, scope)
		if err != nil {
			return err
		}
		for _, d := range list {
			defs[d.ID] = d
		}
	}
	current := map[int64]customfield.Value{}
	if reused {
		var err error
		if current, err = memberValues(ctx, q, memberID); err != nil {
			return err
		}
	}
	now := nowString()
	for _, in := range toInputs(inputs) {
		d, ok := defs[in.FieldID]
		if !ok {
			continue
		}
		v, err := customfield.NormalizeValue(d, in)
		if err != nil || v.IsEmpty() {
			continue
		}
		if d.Scope == customfield.ScopeMember {
			if _, taken := current[d.ID]; taken {
				continue
			}
			err = q.UpsertMemberFieldValue(ctx, fieldValueParams(memberID, d.ID, v, now))
		} else {
			err = q.UpsertMembershipFieldValue(ctx, sqlc.UpsertMembershipFieldValueParams(fieldValueParams(membershipID, d.ID, v, now)))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
