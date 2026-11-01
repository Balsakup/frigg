package api_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

func dashboard(t *testing.T, f msFixture) api.MembershipDashboard {
	t.Helper()
	var d api.MembershipDashboard
	res := f.admin.Do("GET", "/api/admin/memberships/dashboard", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("tableau de bord : %d %s", res.Status, res.Body)
	}
	res.JSON(t, &d)
	return d
}

// TestMembershipDecisions couvre la validation et le refus (US2).
func TestMembershipDecisions(t *testing.T) {
	f := newMsFixture(t)
	c, email, leo := f.member(t, "Léo")
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	theo := createMine(t, c, memberBody("Théo", "Martin", "2014-02-01", "confirmDuplicate", true))
	r1 := request(t, c, leo.ID, f.season.ID, f.adult.ID, "renewal")
	r2 := request(t, c, ines.ID, f.season.ID, f.young.ID, "first")
	r3 := request(t, c, theo.ID, f.season.ID, f.young.ID, "first")

	if d := dashboard(t, f); d.PendingCount != 3 || !d.PaymentSetupNeeded {
		t.Errorf("tableau de bord : %+v", d)
	}
	page := adminMemberships(t, f.admin, "?season=all&status=pending")
	if page.Total != 3 || page.Items[0].ID != r1.ID || page.Items[2].ID != r3.ID {
		t.Fatalf("demandes en attente : %+v", page)
	}

	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(r1.ID, ""), nil), http.StatusOK)
	if got.PreviousSeason != "unknown" || got.Account == nil || got.Account.Email != email ||
		!got.Actions.Validate || !got.Actions.Refuse || got.Actions.Update || len(got.SeasonFees) != 3 {
		t.Fatalf("détail : %+v", got)
	}
	if got.SeasonFees[0].Amounts != (api.FeeAmounts{First: 6000, Renewal: 4000}) || got.SeasonFees[1].Amounts != (api.FeeAmounts{First: 3000, Renewal: 3000}) {
		t.Errorf("montants des tarifs : %+v", got.SeasonFees)
	}

	// Validation telle que déclarée.
	v1 := validate(t, f.admin, r1.ID, f.adult.ID, "renewal")
	if v1.Status != "validated" || *v1.Kind != "renewal" || v1.AmountCents != 4000 || !v1.AmountFixed ||
		v1.Payment == nil || v1.Payment.Reference != "ADH-2627-0001" || v1.Payment.Status != "due" || v1.DecidedBy == nil {
		t.Fatalf("validation : %+v", v1)
	}
	// Validation avec un autre tarif et un autre type.
	v2 := validate(t, f.admin, r2.ID, f.adult.ID, "first")
	if v2.AmountCents != 6000 || *v2.DeclaredKind != "first" || v2.Fee.ID != f.adult.ID || v2.Payment.Reference != "ADH-2627-0002" {
		t.Fatalf("validation corrigée : %+v", v2)
	}
	// Tarif d'une autre saison.
	other := createSeason(t, f.admin, "2027-09-01", "2028-08-31")
	otherFee := createFee(t, f.admin, other.ID, map[string]string{"label": "Adulte", "amount": "65"})
	res := f.admin.Do("POST", adminMembershipPath(r3.ID, "/validate"), map[string]any{"feeId": otherFee.ID, "kind": "first"})
	expectFieldError(t, res, "body.feeId", api.MsgFeeNotInSeason)

	// Refus.
	for _, reason := range []string{"", "   "} {
		expectFieldError(t, f.admin.Do("POST", adminMembershipPath(r3.ID, "/refuse"), map[string]any{"reason": reason}), "body.reason", api.MsgReasonRequired)
	}
	refused := expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(r3.ID, "/refuse"), map[string]any{"reason": "Fiche en double"}), http.StatusOK)
	if refused.Status != "refused" || refused.Reason == nil || *refused.Reason != "Fiche en double" || refused.Payment != nil {
		t.Errorf("refus : %+v", refused)
	}
	// Décision sur une adhésion déjà décidée.
	expectProblem(t, f.admin.Do("POST", adminMembershipPath(r3.ID, "/validate"), map[string]any{"feeId": f.young.ID, "kind": "first"}), http.StatusConflict, api.MsgMembershipChanged)
	expectProblem(t, f.admin.Do("POST", adminMembershipPath(r1.ID, "/refuse"), map[string]any{"reason": "x"}), http.StatusConflict, api.MsgMembershipChanged)

	if d := dashboard(t, f); d.PendingCount != 0 {
		t.Errorf("demandes en attente : %d", d.PendingCount)
	}

	// Modifier le tarif après validation ne change pas le montant figé.
	f.admin.Do("PUT", feePath(f.season.ID, f.adult.ID), map[string]any{"label": "Adulte", "amount": "70", "hasRenewalAmount": true, "renewalAmount": "50"})
	if got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(r1.ID, ""), nil), http.StatusOK); got.AmountCents != 4000 {
		t.Errorf("montant figé : %d", got.AmountCents)
	}

	// E-mails.
	f.env.FlushMail()
	var validated, refusedMail int
	for _, m := range f.env.Mail.Messages() {
		if m.To != email {
			t.Errorf("destinataire inattendu %s", m.To)
		}
		switch {
		case strings.Contains(m.Msg.Subject, "validée"):
			validated++
			if !strings.Contains(m.Msg.Text, "http://frigg.test/adhesions/") || !strings.Contains(m.Msg.Text, "ADH-2627-") {
				t.Errorf("e-mail de validation : %s", m.Msg.Text)
			}
		case strings.Contains(m.Msg.Subject, "refusée"):
			refusedMail++
			if !strings.Contains(m.Msg.Text, "Fiche en double") {
				t.Errorf("e-mail de refus sans motif : %s", m.Msg.Text)
			}
		}
	}
	if validated != 2 || refusedMail != 1 {
		t.Errorf("e-mails : %d validations, %d refus", validated, refusedMail)
	}

	if n := membershipLogs(t, f.env, "membership.validate", "Léo", "Martin"); n != 2 {
		t.Errorf("journal membership.validate : %d", n)
	}
	if n := membershipLogs(t, f.env, "membership.refuse", "Fiche en double"); n != 1 {
		t.Errorf("journal membership.refuse : %d", n)
	}
}

// TestMembershipDecisionsMail : adhésion gratuite, fiche sans compte et compte
// désactivé (FR-023).
func TestMembershipDecisionsMail(t *testing.T) {
	f := newMsFixture(t)
	c, email, leo := f.member(t, "Léo")
	free := request(t, c, leo.ID, f.season.ID, f.free.ID, "first")
	if v := validate(t, f.admin, free.ID, f.free.ID, "first"); v.Payment != nil || v.AmountCents != 0 {
		t.Errorf("gratuite : %+v", v)
	}
	f.env.FlushMail()
	last, ok := f.env.Mail.Last(email)
	if !ok || strings.Contains(last.Msg.Text, "/adhesions/") || !strings.Contains(last.Msg.Text, "Aucun paiement") {
		t.Errorf("e-mail d'une adhésion gratuite : %+v", last)
	}

	// Fiche sans compte : aucun e-mail.
	orphan := insertOrphan(t, f.env, "Jeanne", "Petit", "2010-01-01")
	before := len(f.env.Mail.Messages())
	expectAdminMembership(t, f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": orphan, "seasonId": f.season.ID, "feeId": f.young.ID, "kind": "first",
	}), http.StatusCreated)
	f.env.FlushMail()
	if n := len(f.env.Mail.Messages()); n != before {
		t.Errorf("e-mail envoyé pour une fiche sans compte")
	}

	// Compte désactivé : aucun e-mail.
	c2, email2, zoe := f.member(t, "Zoé")
	r := request(t, c2, zoe.ID, f.season.ID, f.young.ID, "first")
	if res := f.admin.Do("POST", accountPath(accountID(t, f.env, email2), "/disable"), nil); res.Status != http.StatusOK {
		t.Fatalf("désactivation : %d %s", res.Status, res.Body)
	}
	validate(t, f.admin, r.ID, f.young.ID, "first")
	f.env.FlushMail()
	if _, ok := f.env.Mail.Last(email2); ok {
		t.Errorf("e-mail envoyé à un compte désactivé")
	}
}

// TestMembershipPreviousSeasonHint couvre l'aide « saison précédente » (R6).
func TestMembershipPreviousSeasonHint(t *testing.T) {
	f := newMsFixture(t)
	next := createSeason(t, f.admin, "2027-09-01", "2028-08-31")
	expectSeason(t, f.admin.Do("POST", seasonPath(next.ID, "/open"), nil), http.StatusOK)
	nextFee := createFee(t, f.admin, next.ID, map[string]string{"label": "Adulte", "amount": "60"})

	c, _, leo := f.member(t, "Léo")
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	r := request(t, c, leo.ID, next.ID, nextFee.ID, "renewal")
	hint := func() string {
		return expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(r.ID, ""), nil), http.StatusOK).PreviousSeason
	}
	if h := hint(); h != "unknown" {
		t.Errorf("saison précédente sans adhésion validée : %s", h)
	}
	validate(t, f.admin, request(t, c, ines.ID, f.season.ID, f.young.ID, "first").ID, f.young.ID, "first")
	if h := hint(); h != "no" {
		t.Errorf("autre membre adhérent : %s", h)
	}
	validate(t, f.admin, request(t, c, leo.ID, f.season.ID, f.adult.ID, "first").ID, f.adult.ID, "first")
	if h := hint(); h != "yes" {
		t.Errorf("membre adhérent : %s", h)
	}
}

// TestMembershipClosedSeason : décisions et constatation admises sur une
// saison close, correction et annulation refusées (FR-017).
func TestMembershipClosedSeason(t *testing.T) {
	f := newMsFixture(t)
	c, _, leo := f.member(t, "Léo")
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	r1 := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")
	r2 := request(t, c, ines.ID, f.season.ID, f.young.ID, "first")
	f.env.SetToday("2027-09-01")

	v := validate(t, f.admin, r1.ID, f.adult.ID, "first")
	if v.Season.State != "closed" || v.Actions.Update || v.Actions.Cancel || !v.Actions.RecordPayment {
		t.Fatalf("saison close : %+v", v)
	}
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(r2.ID, "/refuse"), map[string]any{"reason": "Hors délai"}), http.StatusOK)
	expectProblem(t, f.admin.Do("PUT", adminMembershipPath(r1.ID, ""), map[string]any{"feeId": f.young.ID, "kind": "first"}), http.StatusConflict, api.MsgSeasonClosedAction)
	expectProblem(t, f.admin.Do("POST", adminMembershipPath(r1.ID, "/cancel"), map[string]any{"reason": "Départ"}), http.StatusConflict, api.MsgSeasonClosedAction)
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(r1.ID, "/payment"), map[string]any{"method": "cash", "receivedOn": "2027-09-01"}), http.StatusOK)
	// Création sur une saison close : 422.
	res := f.admin.Do("POST", "/api/admin/memberships", map[string]any{"memberId": ines.ID, "seasonId": f.season.ID, "feeId": f.young.ID, "kind": "first"})
	expectFieldError(t, res, "body.seasonId", api.MsgSeasonNotOffered)
	if o := offer(t, c); len(o.Seasons) != 0 {
		t.Errorf("saison close proposée : %+v", o.Seasons)
	}
}

// TestMembershipPaymentRecording couvre la constatation du paiement (US4).
func TestMembershipPaymentRecording(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, email, leo := f.member(t, "Léo")
	m := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, m.ID, f.adult.ID, "first")
	path := adminMembershipPath(m.ID, "/payment")

	expectFieldError(t, f.admin.Do("POST", path, map[string]any{"method": "cash", "receivedOn": "2026-11-02"}), "body.receivedOn", api.MsgReceivedFuture)
	expectFieldError(t, f.admin.Do("POST", path, map[string]any{"method": "cash", "receivedOn": "2026-10-31"}), "body.receivedOn", api.MsgReceivedBeforeValidation)
	if res := f.admin.Do("POST", path, map[string]any{"method": "cash", "receivedOn": msToday, "note": strings.Repeat("a", 201)}); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("commentaire trop long : %d", res.Status)
	}

	// Constatation d'une adhésion due, puis annulation : retour à « due ».
	paid := expectAdminMembership(t, f.admin.Do("POST", path, map[string]any{"method": "cheque", "receivedOn": msToday, "note": "Chèque n° 1234"}), http.StatusOK)
	if paid.Payment.Status != "paid" || *paid.Payment.PaidMethod != "cheque" || *paid.Payment.PaidOn != msToday ||
		paid.PaidBy == nil || paid.PaymentNote == nil || !paid.Actions.CancelPayment || paid.Actions.RecordPayment {
		t.Fatalf("constatation : %+v", paid)
	}
	expectProblem(t, f.admin.Do("POST", path, map[string]any{"method": "cheque", "receivedOn": msToday}), http.StatusConflict, api.MsgMembershipChanged)
	back := expectAdminMembership(t, f.admin.Do("DELETE", path, nil), http.StatusOK)
	if back.Payment.Status != "due" || back.Payment.PaidMethod != nil || back.PaymentNote != nil {
		t.Errorf("annulation de la constatation : %+v", back.Payment)
	}

	// Moyen annoncé : l'annulation y revient.
	expectMine(t, c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "transfer"}), http.StatusOK)
	expectAdminMembership(t, f.admin.Do("POST", path, map[string]any{"method": "transfer", "receivedOn": msToday}), http.StatusOK)
	back = expectAdminMembership(t, f.admin.Do("DELETE", path, nil), http.StatusOK)
	if back.Payment.Status != "announced" || *back.Payment.AnnouncedMethod != "transfer" {
		t.Errorf("retour au moyen annoncé : %+v", back.Payment)
	}
	expectProblem(t, f.admin.Do("DELETE", path, nil), http.StatusConflict, api.MsgMembershipChanged)

	f.env.FlushMail()
	paidMails := 0
	for _, msg := range f.env.Mail.Messages() {
		if msg.To == email && strings.Contains(msg.Msg.Subject, "reçu") {
			paidMails++
		}
	}
	if paidMails != 2 {
		t.Errorf("e-mails de paiement : %d", paidMails)
	}
	membershipLogs(t, f.env, "membership.payment_record", "1234")
	if n := membershipLogs(t, f.env, "membership.payment_cancel"); n != 2 {
		t.Errorf("journal membership.payment_cancel : %d", n)
	}
}

// TestMembershipDirectCreation couvre la création directe (US6).
func TestMembershipDirectCreation(t *testing.T) {
	f := newMsFixture(t)
	c, email, leo := f.member(t, "Léo")
	create := func(body map[string]any) apitest.Response {
		return f.admin.Do("POST", "/api/admin/memberships", body)
	}
	body := func(memberID, feeID int64, paid map[string]any) map[string]any {
		b := map[string]any{"memberId": memberID, "seasonId": f.season.ID, "feeId": feeID, "kind": "first"}
		if paid != nil {
			b["paid"] = paid
		}
		return b
	}

	m := expectAdminMembership(t, create(body(leo.ID, f.adult.ID, nil)), http.StatusCreated)
	if m.Status != "validated" || m.DeclaredKind != nil || m.CreatedBy == nil || m.Payment == nil || m.Payment.Status != "due" {
		t.Fatalf("création : %+v", m)
	}
	f.env.FlushMail()
	if last, ok := f.env.Mail.Last(email); !ok || !strings.Contains(last.Msg.Text, "/adhesions/") {
		t.Errorf("e-mail de validation absent ou sans lien")
	}
	expectProblem(t, create(body(leo.ID, f.young.ID, nil)), http.StatusConflict, api.MsgMembershipActive)
	expectProblem(t, create(body(99999, f.young.ID, nil)), http.StatusNotFound, api.MsgMemberGone)

	// Règlement déjà reçu : date dans la saison, éventuellement passée.
	orphan := insertOrphan(t, f.env, "Jeanne", "Petit", "2010-01-01")
	expectFieldError(t, create(body(orphan, f.young.ID, map[string]any{"method": "cheque", "receivedOn": "2026-08-31"})), "body.paid.receivedOn", api.MsgReceivedBeforeSeason)
	expectFieldError(t, create(body(orphan, f.young.ID, map[string]any{"method": "cheque", "receivedOn": "2026-11-02"})), "body.paid.receivedOn", api.MsgReceivedFuture)
	p := expectAdminMembership(t, create(body(orphan, f.young.ID, map[string]any{"method": "cheque", "receivedOn": "2026-10-20", "note": "Bulletin papier"})), http.StatusCreated)
	if p.Payment == nil || p.Payment.Status != "paid" || *p.Payment.PaidOn != "2026-10-20" || p.Account != nil {
		t.Fatalf("création payée : %+v", p)
	}

	// La fiche d'administration liste les adhésions ; la suppression les emporte.
	got := expectMember(t, f.admin.Do("GET", adminMemberPath(orphan), nil), http.StatusOK)
	if len(got.Memberships) != 1 || got.Memberships[0].ID != p.ID || got.Memberships[0].Payment.Status != "paid" {
		t.Fatalf("adhésions de la fiche : %+v", got.Memberships)
	}
	if res := f.admin.Do("DELETE", adminMemberPath(orphan), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, f.admin.Do("GET", adminMembershipPath(p.ID, ""), nil), http.StatusNotFound, api.MsgMembershipGone)
	if !strings.Contains(f.env.Logs.String(), `"memberships_deleted":1`) {
		t.Errorf("journal member.delete sans memberships_deleted")
	}

	// Suppression d'un compte en conservant ses fiches : adhésions conservées,
	// demandeur effacé.
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	r := request(t, c, ines.ID, f.season.ID, f.young.ID, "first")
	accID := accountID(t, f.env, email)
	ids := itoa(leo.ID) + "," + itoa(ines.ID)
	if res := f.admin.Do("DELETE", accountPath(accID, "?members=keep&memberIds="+ids), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression du compte : %d %s", res.Status, res.Body)
	}
	kept := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(r.ID, ""), nil), http.StatusOK)
	if kept.RequestedBy != nil || kept.Account != nil || kept.Status != "pending" {
		t.Errorf("adhésion conservée : %+v", kept)
	}
}

// TestMembershipCorrectionAndCancellation couvre US7.
func TestMembershipCorrectionAndCancellation(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, email, leo := f.member(t, "Léo")
	m := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, m.ID, f.adult.ID, "first")
	put := func(feeID int64, kind string) apitest.Response {
		return f.admin.Do("PUT", adminMembershipPath(m.ID, ""), map[string]any{"feeId": feeID, "kind": kind})
	}

	got := expectAdminMembership(t, put(f.adult.ID, "renewal"), http.StatusOK)
	if got.AmountCents != 4000 || *got.Kind != "renewal" || got.Payment.Reference != "ADH-2627-0001" {
		t.Fatalf("correction : %+v", got)
	}
	expectMine(t, c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "cheque"}), http.StatusOK)
	got = expectAdminMembership(t, put(f.young.ID, "first"), http.StatusOK)
	if got.AmountCents != 3000 || got.Payment.Status != "announced" || *got.Payment.AnnouncedMethod != "cheque" {
		t.Errorf("moyen annoncé conservé : %+v", got.Payment)
	}
	// Payant → gratuit → payant : la référence est conservée.
	if got = expectAdminMembership(t, put(f.free.ID, "first"), http.StatusOK); got.Payment != nil || got.AmountCents != 0 {
		t.Errorf("devenue gratuite : %+v", got)
	}
	if got = expectAdminMembership(t, put(f.young.ID, "first"), http.StatusOK); got.Payment == nil || got.Payment.Reference != "ADH-2627-0001" || got.Payment.Status != "due" {
		t.Errorf("redevenue payante : %+v", got.Payment)
	}
	// Payée : correction refusée.
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(m.ID, "/payment"), map[string]any{"method": "cash", "receivedOn": msToday}), http.StatusOK)
	expectProblem(t, put(f.adult.ID, "first"), http.StatusConflict, api.MsgPaidCorrection)

	// Annulation.
	expectFieldError(t, f.admin.Do("POST", adminMembershipPath(m.ID, "/cancel"), map[string]any{"reason": " "}), "body.reason", api.MsgReasonRequired)
	cancelled := expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(m.ID, "/cancel"), map[string]any{"reason": "Départ de l'association"}), http.StatusOK)
	if cancelled.Status != "cancelled" || cancelled.CancelledBy == nil || *cancelled.Reason != "Départ de l'association" {
		t.Fatalf("annulation : %+v", cancelled)
	}
	f.env.FlushMail()
	if last, ok := f.env.Mail.Last(email); !ok || !strings.Contains(last.Msg.Subject, "annulée") || !strings.Contains(last.Msg.Text, "Départ") {
		t.Errorf("e-mail d'annulation : %+v", last)
	}
	// La fiche peut de nouveau adhérer.
	request(t, c, leo.ID, f.season.ID, f.young.ID, "first")
	var st api.MembershipStats
	f.admin.Do("GET", "/api/admin/memberships/stats?season="+itoa(f.season.ID), nil).JSON(t, &st)
	if st.Validated != 0 || st.Pending != 1 || st.PaidCents != 0 {
		t.Errorf("chiffres après annulation : %+v", st)
	}
	membershipLogs(t, f.env, "membership.cancel", "Départ")
	if n := membershipLogs(t, f.env, "membership.update"); n != 4 {
		t.Errorf("journal membership.update : %d", n)
	}
}

// TestMembershipListAndStats couvre la liste, la recherche et les chiffres (US8).
func TestMembershipListAndStats(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, _, leo := f.member(t, "Léo")
	add := func(first string) api.Member {
		return createMine(t, c, memberBody(first, "Durand", "2011-01-01", "confirmDuplicate", true))
	}
	a, b, d, e, g, h := add("Anna"), add("Basile"), add("Denis"), add("Emma"), add("Gaspard"), add("Hugo")

	// 2 premières adhésions (une payée par chèque), 1 renouvellement annoncé
	// par virement, 1 gratuite, 1 en attente, 1 refusée, 1 annulée payée.
	m1 := request(t, c, a.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, m1.ID, f.adult.ID, "first")
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(m1.ID, "/payment"), map[string]any{"method": "cheque", "receivedOn": msToday}), http.StatusOK)
	m2 := request(t, c, b.ID, f.season.ID, f.young.ID, "first")
	validate(t, f.admin, m2.ID, f.young.ID, "first")
	m3 := request(t, c, d.ID, f.season.ID, f.adult.ID, "renewal")
	validate(t, f.admin, m3.ID, f.adult.ID, "renewal")
	expectMine(t, c.Do("PUT", myMembershipPath(m3.ID)+"/payment", map[string]any{"method": "transfer"}), http.StatusOK)
	m4 := request(t, c, e.ID, f.season.ID, f.free.ID, "first")
	validate(t, f.admin, m4.ID, f.free.ID, "first")
	request(t, c, g.ID, f.season.ID, f.young.ID, "first")
	m6 := request(t, c, h.ID, f.season.ID, f.young.ID, "first")
	f.admin.Do("POST", adminMembershipPath(m6.ID, "/refuse"), map[string]any{"reason": "Non"})
	m7 := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, m7.ID, f.adult.ID, "first")
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(m7.ID, "/payment"), map[string]any{"method": "cash", "receivedOn": msToday}), http.StatusOK)
	f.admin.Do("POST", adminMembershipPath(m7.ID, "/cancel"), map[string]any{"reason": "Départ"})

	var st api.MembershipStats
	res := f.admin.Do("GET", "/api/admin/memberships/stats?season="+itoa(f.season.ID), nil)
	res.JSON(t, &st)
	if st.Validated != 4 || st.First != 3 || st.Renewal != 1 || st.Pending != 1 {
		t.Errorf("compteurs : %+v", st)
	}
	if st.TotalCents != 13000 || st.PaidCents != 6000 || st.DueCents != 7000 ||
		st.DueByMethod != (api.MembershipDueByMethod{None: 3000, Transfer: 4000}) {
		t.Errorf("montants : %+v", st)
	}
	if len(st.ByFee) != 3 || st.ByFee[0].Label != "Adulte" || st.ByFee[0].Count != 2 || st.ByFee[0].AmountCents != 10000 {
		t.Errorf("répartition : %+v", st.ByFee)
	}
	expectProblem(t, f.admin.Do("GET", "/api/admin/memberships/stats?season=99999", nil), http.StatusNotFound, api.MsgSeasonGone)

	// Liste : saison par défaut, demandes en attente d'abord.
	page := adminMemberships(t, f.admin, "")
	if page.SeasonID == nil || *page.SeasonID != f.season.ID || page.Total != 7 || page.Items[0].Status != "pending" {
		t.Fatalf("liste par défaut : %+v", page)
	}
	for q, want := range map[string]int64{
		"?payment=due": 1, "?payment=announced": 1, "?payment=paid": 1, "?payment=free": 1,
		"?status=refused": 1, "?q=adh-2627-000": 4, "?q=0002": 1, "?q=Durand": 6, "?q=leo": 1,
	} {
		if p := adminMemberships(t, f.admin, q); p.Total != want {
			t.Errorf("%s : %d, attendu %d", q, p.Total, want)
		}
	}
	if p := adminMemberships(t, f.admin, "?q=0002"); p.Items[0].Payment == nil || p.Items[0].Payment.Reference != "ADH-2627-0002" {
		t.Errorf("recherche par référence : %+v", p.Items)
	}
	expectFieldError(t, f.admin.Do("GET", "/api/admin/memberships?q=ab", nil), "query.q", api.MsgSearchTooShort)
	if p := adminMemberships(t, f.admin, "?pageSize=2&page=2"); len(p.Items) != 2 || p.Total != 7 {
		t.Errorf("pagination : %+v", p)
	}

	// Une adhésion supprimée avec sa fiche n'est plus trouvée par sa référence.
	if res := f.admin.Do("DELETE", adminMemberPath(b.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d", res.Status)
	}
	if p := adminMemberships(t, f.admin, "?q=0002"); p.Total != 0 {
		t.Errorf("référence d'une adhésion supprimée encore trouvée : %+v", p)
	}
	ensureNoReferenceReuse(t, f)
}

// ensureNoReferenceReuse vérifie qu'aucune référence n'est attribuée deux fois.
func ensureNoReferenceReuse(t *testing.T, f msFixture) {
	t.Helper()
	rows, err := f.env.Store.Read.ListPaymentReferences(context.Background(), "ADH-2627-")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, r := range rows {
		if seen[r.String] {
			t.Errorf("référence en double : %s", r.String)
		}
		seen[r.String] = true
	}
}

// TestMembershipAdminAccess : un membre n'accède à aucune opération
// d'administration des adhésions, même pour ses propres fiches.
func TestMembershipAdminAccess(t *testing.T) {
	f := newMsFixture(t)
	c, _, leo := f.member(t, "Léo")
	m := request(t, c, leo.ID, f.season.ID, f.young.ID, "first")
	for _, op := range []struct{ method, path string }{
		{"GET", "/api/admin/memberships"},
		{"GET", adminMembershipPath(m.ID, "")},
		{"POST", adminMembershipPath(m.ID, "/validate")},
		{"GET", "/api/admin/settings/payment"},
	} {
		if res := c.Do(op.method, op.path, map[string]any{"feeId": f.young.ID, "kind": "first"}); res.Status != http.StatusForbidden {
			t.Errorf("%s %s : %d", op.method, op.path, res.Status)
		}
	}
}

// TestMembershipLogsContainNoPersonalData : après un parcours complet, les
// journaux ne contiennent ni nom, ni motif, ni commentaire, ni coordonnées
// bancaires (FR-026, principe VII).
func TestMembershipLogsContainNoPersonalData(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, _, leo := f.member(t, "Léonard")
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	m := request(t, c, leo.ID, f.season.ID, f.adult.ID, "renewal")
	validate(t, f.admin, m.ID, f.adult.ID, "renewal")
	expectMine(t, c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "transfer"}), http.StatusOK)
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(m.ID, "/payment"), map[string]any{
		"method": "transfer", "receivedOn": msToday, "note": "Virement reçu sur Crédit Mutuel",
	}), http.StatusOK)
	r := request(t, c, ines.ID, f.season.ID, f.young.ID, "first")
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(r.ID, "/refuse"), map[string]any{"reason": "Motif confidentiel"}), http.StatusOK)

	logs := f.env.Logs.String()
	for _, secret := range []string{"Léonard", "Inès", "Martin", "Motif confidentiel", "Crédit Mutuel", "FR76", "BNPAFRPP", "samedi", "rue du Port"} {
		if strings.Contains(logs, secret) {
			t.Errorf("%q journalisé", secret)
		}
	}
}
