package api_test

import (
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

// Chiffres avec remises et dons (specs/014-payment-adjustments US5).

func TestMembershipStatsWithAdjustments(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	validateBody := func(amount, reason string) map[string]any {
		return map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": amount, "adjustmentReason": reason}
	}
	pay := func(id int64, method, received string) {
		t.Helper()
		expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(id, "/payment"), map[string]any{
			"method": method, "receivedOn": msToday, "receivedAmount": received,
		}), http.StatusOK)
	}

	// Remise : 55 € au lieu de 60 €, payée 55 €.
	_, _, a := pendingRequest(t, f)
	expectAdminMembership(t, validateWith(f.admin, a.ID, validateBody("55", "remise")), http.StatusOK)
	pay(a.ID, "transfer", "55")
	// Majoration : 70 € au lieu de 60 €, annoncée par chèque.
	c, _, b := pendingRequest(t, f)
	expectAdminMembership(t, validateWith(f.admin, b.ID, validateBody("70", "soutien")), http.StatusOK)
	if res := c.Do("PUT", myMembershipPath(b.ID)+"/payment", map[string]any{"method": "cheque"}); res.Status != http.StatusOK {
		t.Fatalf("annonce : %d %s", res.Status, res.Body)
	}
	// Payée 95 € pour 60 € : 35 € de don.
	_, _, d := pendingRequest(t, f)
	validate(t, f.admin, d.ID, f.adult.ID, "first")
	pay(d.ID, "cash", "95")
	// Gratuite par ajustement.
	_, _, g := pendingRequest(t, f)
	expectAdminMembership(t, validateWith(f.admin, g.ID, validateBody("0", "bénévole")), http.StatusOK)
	// Annulée : hors chiffres.
	_, _, x := pendingRequest(t, f)
	expectAdminMembership(t, validateWith(f.admin, x.ID, validateBody("10", "x")), http.StatusOK)
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(x.ID, "/cancel"), map[string]any{"reason": "doublon"}), http.StatusOK)

	st := expectJSON[api.MembershipStats](t, f.admin.Do("GET", "/api/admin/memberships/stats?season="+itoa(f.season.ID), nil), http.StatusOK)
	want := api.MembershipStats{
		TheoreticalCents: 4 * 6000, DiscountCents: 500 + 6000, SurchargeCents: 1000,
		TotalCents: 5500 + 7000 + 6000 + 0, PaidCents: 5500 + 9500, DonationCents: 3500, DueCents: 7000,
	}
	if st.TheoreticalCents != want.TheoreticalCents || st.DiscountCents != want.DiscountCents || st.SurchargeCents != want.SurchargeCents ||
		st.TotalCents != want.TotalCents || st.PaidCents != want.PaidCents || st.DonationCents != want.DonationCents || st.DueCents != want.DueCents {
		t.Fatalf("chiffres : %+v", st)
	}
	if st.DueByMethod.Cheque != 7000 || st.CollectedByMethod.Transfer != 5500 || st.CollectedByMethod.Cash != 9500 {
		t.Fatalf("par moyen : dû %+v, encaissé %+v", st.DueByMethod, st.CollectedByMethod)
	}
	if len(st.ByFee) != 1 || st.ByFee[0].AmountCents != st.TotalCents {
		t.Fatalf("répartition : %+v", st.ByFee)
	}
	if st.TheoreticalCents-st.DiscountCents+st.SurchargeCents != st.TotalCents {
		t.Fatal("théorique − remises + majorations ≠ total à payer")
	}
}

func TestEventTotalsWithAdjustments(t *testing.T) {
	f := newEvFixture(t)
	setPayment(t, f.admin, nil)
	ev := f.publishedEvent(t, eventBody("Stage", paidEvent("35")))
	out := register(t, f.claire, ev.ID, http.StatusCreated, f.self.ID, f.lucas.ID)
	expectAdminRegistration(t, f.admin.Do("PUT", adminRegistrationPath(out.CreatedRegistrationIDs[0], "/amount"), map[string]any{
		"amount": "20", "adjustmentReason": "bénévole",
	}), http.StatusOK)
	expectAdminRegistration(t, f.admin.Do("POST", adminRegistrationPath(out.CreatedRegistrationIDs[1], "/payment"), map[string]any{
		"method": "cash", "receivedOn": msToday, "receivedAmount": "40",
	}), http.StatusOK)

	tot := expectAdminEvent(t, f.admin.Do("GET", eventPath(ev.ID, ""), nil), http.StatusOK).Totals
	if tot.TheoreticalCents != 7000 || tot.DiscountCents != 1500 || tot.SurchargeCents != 0 || tot.DueCents != 5500 ||
		tot.CollectedCents != 4000 || tot.DonationCents != 500 || tot.RemainingCents != 2000 || tot.PaidCount != 1 {
		t.Fatalf("totaux : %+v", tot)
	}
}
