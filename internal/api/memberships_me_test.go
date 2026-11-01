package api_test

import (
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
)

// TestMembershipRequest couvre la demande d'adhésion (US1).
func TestMembershipRequest(t *testing.T) {
	f := newMsFixture(t)

	// Compte sans fiche : rien à demander.
	empty := f.env.LoggedIn(f.env.Seed(auth.RoleMember, true))
	if o := offer(t, empty); len(o.Members) != 0 || o.CanRequest || len(o.Seasons) != 1 {
		t.Fatalf("offre sans fiche : %+v", o)
	}

	c, _, leo := f.member(t, "Léo")
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	o := offer(t, c)
	if len(o.Seasons) != 1 || o.Seasons[0].ID != f.season.ID || len(o.Seasons[0].Fees) != 3 || !o.CanRequest {
		t.Fatalf("offre : %+v", o)
	}
	if adult := o.Seasons[0].Fees[0]; adult.Label != "Adulte" || adult.Amounts.First != 6000 || adult.Amounts.Renewal != 4000 {
		t.Errorf("tarif Adulte : %+v", adult)
	}
	if len(o.Members) != 2 || len(o.Members[0].AvailableSeasonIDs) != 1 {
		t.Fatalf("fiches de l'offre : %+v", o.Members)
	}

	m := request(t, c, leo.ID, f.season.ID, f.adult.ID, "renewal")
	if m.Status != "pending" || m.AmountCents != 4000 || m.AmountFixed || m.Payment != nil || !m.Actions.Cancel || m.Actions.Pay {
		t.Fatalf("demande : %+v", m)
	}
	o = offer(t, c)
	for _, om := range o.Members {
		if om.ID == leo.ID && len(om.AvailableSeasonIDs) != 0 {
			t.Errorf("Léo encore proposé : %+v", om)
		}
		if om.ID == ines.ID && len(om.AvailableSeasonIDs) != 1 {
			t.Errorf("Inès non proposée : %+v", om)
		}
	}

	// Seconde demande pour la même fiche et la même saison.
	res := c.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": leo.ID, "seasonId": f.season.ID, "feeId": f.young.ID, "declaredKind": "first",
	})
	expectProblem(t, res, http.StatusConflict, api.MsgMembershipActive)

	// Tarif d'une autre saison, saison inexistante.
	other := createSeason(t, f.admin, "2027-09-01", "2028-08-31")
	otherFee := createFee(t, f.admin, other.ID, map[string]string{"label": "Adulte", "amount": "65"})
	res = c.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": ines.ID, "seasonId": f.season.ID, "feeId": otherFee.ID, "declaredKind": "first",
	})
	expectFieldError(t, res, "body.feeId", api.MsgFeeNotInSeason)
	// Saison en brouillon (avec tarif) : non proposée.
	res = c.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": ines.ID, "seasonId": other.ID, "feeId": otherFee.ID, "declaredKind": "first",
	})
	expectFieldError(t, res, "body.seasonId", api.MsgSeasonNotOffered)
	if o := offer(t, c); len(o.Seasons) != 1 {
		t.Errorf("saison en brouillon proposée : %+v", o.Seasons)
	}

	// Fiche d'un autre compte : 404.
	other2, _, _ := f.member(t, "Zoé")
	res = other2.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": leo.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "declaredKind": "first",
	})
	expectProblem(t, res, http.StatusNotFound, api.MsgMemberGone)

	// Le montant d'une demande en attente suit le tarif actuel.
	res = f.admin.Do("PUT", feePath(f.season.ID, f.adult.ID), map[string]any{
		"label": "Adulte", "amount": "62", "hasRenewalAmount": true, "renewalAmount": "42",
	})
	if res.Status != http.StatusOK {
		t.Fatalf("modification du tarif : %d %s", res.Status, res.Body)
	}
	if got := expectMine(t, c.Do("GET", myMembershipPath(m.ID), nil), http.StatusOK); got.AmountCents != 4200 {
		t.Errorf("montant de la demande : %d", got.AmountCents)
	}

	// Après refus, nouvelle demande acceptée.
	res = f.admin.Do("POST", adminMembershipPath(m.ID, "/refuse"), map[string]any{"reason": "Doublon de fiche"})
	if res.Status != http.StatusOK {
		t.Fatalf("refus : %d %s", res.Status, res.Body)
	}
	request(t, c, leo.ID, f.season.ID, f.young.ID, "first")

	if n := membershipLogs(t, f.env, "membership.request", "Léo", "Martin"); n != 2 {
		t.Errorf("journal membership.request : %d", n)
	}
}

// TestMembershipRequestConcurrent : deux demandes simultanées pour la même
// fiche et la même saison, une seule aboutit (SC-003).
func TestMembershipRequestConcurrent(t *testing.T) {
	f := newMsFixture(t)
	c, _, leo := f.member(t, "Léo")
	var wg sync.WaitGroup
	statuses := make([]int, 2)
	for i := range statuses {
		wg.Go(func() {
			statuses[i] = c.Do("POST", "/api/me/memberships", map[string]any{
				"memberId": leo.ID, "seasonId": f.season.ID, "feeId": f.young.ID, "declaredKind": "first",
			}).Status
		})
	}
	wg.Wait()
	slices.Sort(statuses)
	if statuses[0] != http.StatusCreated || statuses[1] != http.StatusConflict {
		t.Fatalf("statuts : %v", statuses)
	}
}

// TestMyMembershipsFollowAndCancel couvre le suivi et l'annulation (US3).
func TestMyMembershipsFollowAndCancel(t *testing.T) {
	f := newMsFixture(t)
	c, _, leo := f.member(t, "Léo")
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	pending := request(t, c, ines.ID, f.season.ID, f.young.ID, "first")
	validated := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")
	validate(t, f.admin, validated.ID, f.adult.ID, "first")

	res := c.Do("GET", "/api/me/memberships", nil)
	var list api.MyMembershipList
	res.JSON(t, &list)
	if len(list.Seasons) != 1 || len(list.Seasons[0].Items) != 2 {
		t.Fatalf("liste : %+v", list)
	}
	if items := list.Seasons[0].Items; items[0].Member.FirstName != "Inès" || items[1].Member.FirstName != "Léo" {
		t.Errorf("ordre : %s, %s", items[0].Member.FirstName, items[1].Member.FirstName)
	}

	// Filtre par fiche, fiche d'un autre compte.
	res = c.Do("GET", "/api/me/memberships?member="+itoa(leo.ID), nil)
	res.JSON(t, &list)
	if len(list.Seasons) != 1 || len(list.Seasons[0].Items) != 1 || list.Seasons[0].Items[0].ID != validated.ID {
		t.Errorf("filtre par fiche : %+v", list)
	}
	other, _, zoe := f.member(t, "Zoé")
	expectProblem(t, other.Do("GET", "/api/me/memberships?member="+itoa(leo.ID), nil), http.StatusNotFound, api.MsgMemberGone)
	_ = zoe

	// Annulation d'une demande en attente.
	got := expectMine(t, c.Do("POST", myMembershipPath(pending.ID)+"/cancel", nil), http.StatusOK)
	if got.Status != "cancelled" || got.CancelledAt == nil || got.Actions.Cancel {
		t.Errorf("annulation : %+v", got)
	}
	// Une adhésion validée ne s'annule pas depuis l'espace membre.
	expectProblem(t, c.Do("POST", myMembershipPath(validated.ID)+"/cancel", nil), http.StatusConflict, api.MsgMembershipChanged)
	if n := membershipLogs(t, f.env, "membership.cancel"); n != 1 {
		t.Errorf("journal membership.cancel : %d", n)
	}
}

// TestMyMembershipPayment couvre le choix du moyen de paiement (US4).
func TestMyMembershipPayment(t *testing.T) {
	f := newMsFixture(t)
	c, _, leo := f.member(t, "Léo")
	m := request(t, c, leo.ID, f.season.ID, f.adult.ID, "renewal")

	// En attente : rien à payer.
	if got := expectMine(t, c.Do("GET", myMembershipPath(m.ID), nil), http.StatusOK); got.Actions.Pay || len(got.PaymentMethods) != 0 {
		t.Errorf("demande en attente : %+v", got)
	}
	res := c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "cash"})
	expectProblem(t, res, http.StatusConflict, api.MsgMembershipChanged)

	validate(t, f.admin, m.ID, f.adult.ID, "renewal")
	// Aucun moyen activé.
	got := expectMine(t, c.Do("GET", myMembershipPath(m.ID), nil), http.StatusOK)
	if !got.Actions.Pay || len(got.PaymentMethods) != 0 || got.Payment == nil || got.Payment.Status != "due" {
		t.Fatalf("à payer sans moyen : %+v", got)
	}

	setPayment(t, f.admin, map[string]any{
		"cash":     map[string]any{"enabled": false, "instructions": "Au local"},
		"cheque":   map[string]any{"enabled": true, "payee": "Association Frigg"},
		"transfer": map[string]any{"enabled": true, "holder": "Association Frigg", "iban": "FR76 3000 6000 0112 3456 7890 189"},
	})
	got = expectMine(t, c.Do("GET", myMembershipPath(m.ID), nil), http.StatusOK)
	if len(got.PaymentMethods) != 2 || got.PaymentMethods[0].Method != "cheque" || got.PaymentMethods[1].Method != "transfer" {
		t.Fatalf("moyens : %+v", got.PaymentMethods)
	}
	if tr := got.PaymentMethods[1]; tr.IBAN == nil || *tr.IBAN != "FR7630006000011234567890189" || tr.Holder == nil {
		t.Errorf("virement : %+v", tr)
	}

	res = c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "cash"})
	expectFieldError(t, res, "body.method", api.MsgMethodUnavailable)

	got = expectMine(t, c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "transfer"}), http.StatusOK)
	if got.Payment.Status != "announced" || got.Payment.AnnouncedMethod == nil || *got.Payment.AnnouncedMethod != "transfer" {
		t.Fatalf("annonce : %+v", got.Payment)
	}
	got = expectMine(t, c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "cheque"}), http.StatusOK)
	if *got.Payment.AnnouncedMethod != "cheque" {
		t.Errorf("changement de moyen : %+v", got.Payment)
	}

	// Payée : plus rien à choisir.
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(m.ID, "/payment"), map[string]any{
		"method": "cheque", "receivedOn": msToday,
	}), http.StatusOK)
	got = expectMine(t, c.Do("GET", myMembershipPath(m.ID), nil), http.StatusOK)
	if got.Actions.Pay || len(got.PaymentMethods) != 0 {
		t.Errorf("payée : %+v", got)
	}
	expectProblem(t, c.Do("PUT", myMembershipPath(m.ID)+"/payment", map[string]any{"method": "cheque"}), http.StatusConflict, api.MsgMembershipChanged)

	// Adhésion gratuite : pas de paiement.
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	free := request(t, c, ines.ID, f.season.ID, f.free.ID, "first")
	validate(t, f.admin, free.ID, f.free.ID, "first")
	if got := expectMine(t, c.Do("GET", myMembershipPath(free.ID), nil), http.StatusOK); got.Payment != nil || got.Actions.Pay {
		t.Errorf("gratuite : %+v", got)
	}

	if n := membershipLogs(t, f.env, "membership.payment_announce", "FR76", "3000"); n != 2 {
		t.Errorf("journal membership.payment_announce : %d", n)
	}
}

// TestMembershipRulesOnFeesAndMembers couvre les règles ajoutées à 004 et 005
// (research.md 006 › R9).
func TestMembershipRulesOnFeesAndMembers(t *testing.T) {
	f := newMsFixture(t)
	c, _, leo := f.member(t, "Léo")
	m := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")

	// Tarif utilisé : non supprimable, même après annulation.
	expectMine(t, c.Do("POST", myMembershipPath(m.ID)+"/cancel", nil), http.StatusOK)
	expectProblem(t, f.admin.Do("DELETE", feePath(f.season.ID, f.adult.ID), nil), http.StatusConflict, api.MsgFeeInUse)
	for _, fee := range listFees(t, f.admin, f.season.ID).Items {
		if fee.InUse != (fee.ID == f.adult.ID) {
			t.Errorf("inUse de %s : %v", fee.Label, fee.InUse)
		}
	}
	if res := f.admin.Do("DELETE", feePath(f.season.ID, f.young.ID), nil); res.Status != http.StatusNoContent {
		t.Errorf("suppression d'un tarif libre : %d %s", res.Status, res.Body)
	}

	// Fiche avec une adhésion en cours : non supprimable par le compte.
	active := request(t, c, leo.ID, f.season.ID, f.adult.ID, "first")
	expectProblem(t, c.Do("DELETE", myMemberPath(leo.ID), nil), http.StatusConflict, api.MsgMemberHasMembership)
	validate(t, f.admin, active.ID, f.adult.ID, "first")
	expectProblem(t, c.Do("DELETE", myMemberPath(leo.ID), nil), http.StatusConflict, api.MsgMemberHasMembership)

	// Avec seulement des adhésions refusées ou annulées : supprimable.
	ines := createMine(t, c, memberBody("Inès", "Martin", "2012-06-01", "confirmDuplicate", true))
	r := request(t, c, ines.ID, f.season.ID, f.free.ID, "first")
	expectMine(t, c.Do("POST", myMembershipPath(r.ID)+"/cancel", nil), http.StatusOK)
	if res := c.Do("DELETE", myMemberPath(ines.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, c.Do("GET", myMembershipPath(r.ID), nil), http.StatusNotFound, api.MsgMembershipGone)
}
