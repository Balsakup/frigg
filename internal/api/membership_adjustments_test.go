package api_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

// Remises sur les adhésions (specs/014-payment-adjustments US1, US3).

// validateWith valide une demande avec un corps libre.
func validateWith(admin *apitest.Client, id int64, body map[string]any) apitest.Response {
	return admin.Do("POST", adminMembershipPath(id, "/validate"), body)
}

// pendingRequest dépose une demande de première adhésion « Adulte » (60 €).
func pendingRequest(t *testing.T, f msFixture) (*apitest.Client, string, api.MyMembership) {
	t.Helper()
	c, email, m := f.member(t, "Paul")
	return c, email, request(t, c, m.ID, f.season.ID, f.adult.ID, "first")
}

func TestValidateWithAdjustment(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)

	// Sans montant : le tarif s'applique, aucun ajustement.
	_, _, plain := pendingRequest(t, f)
	got := validate(t, f.admin, plain.ID, f.adult.ID, "first")
	if got.AmountCents != 6000 || got.BaseAmountCents != 6000 || got.AdjustmentCents != 0 || got.AdjustmentReason != nil {
		t.Fatalf("sans ajustement : %+v", got.Membership)
	}

	_, email, ms := pendingRequest(t, f)
	expectFieldError(t, validateWith(f.admin, ms.ID, map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "55"}), "body.adjustmentReason", api.MsgAdjustmentReasonRequired)
	expectFieldError(t, validateWith(f.admin, ms.ID, map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "abc"}), "body.amount", api.MsgAmountFormat)
	expectFieldError(t, validateWith(f.admin, ms.ID, map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "20000", "adjustmentReason": "x"}), "body.amount", api.MsgAmountTooHigh)

	const reason = "réduction de 5 € accordée par le bureau"
	got = expectAdminMembership(t, validateWith(f.admin, ms.ID, map[string]any{
		"feeId": f.adult.ID, "kind": "first", "amount": "55", "adjustmentReason": reason,
	}), http.StatusOK)
	if got.AmountCents != 5500 || got.BaseAmountCents != 6000 || got.AdjustmentCents != -500 ||
		got.AdjustmentReason == nil || *got.AdjustmentReason != reason || got.Payment == nil || got.Payment.Status != "due" {
		t.Fatalf("remise : %+v", got.Membership)
	}
	f.env.FlushMail()
	sent, ok := f.env.Mail.Last(email)
	if !ok || !strings.Contains(sent.Msg.Text, "55,00") || strings.Contains(sent.Msg.Text, "60,00") || strings.Contains(sent.Msg.Text, reason) {
		t.Fatalf("e-mail de validation : %q", sent.Msg.Text)
	}
	if n := membershipLogs(t, f.env, "membership.validate", reason); n != 2 {
		t.Fatalf("journal de validation : %d lignes", n)
	}
	if !strings.Contains(f.env.Logs.String(), `"adjusted":true`) {
		t.Fatal("ajustement non journalisé")
	}

	// Montant égal au théorique : motif ignoré.
	_, _, same := pendingRequest(t, f)
	got = expectAdminMembership(t, validateWith(f.admin, same.ID, map[string]any{
		"feeId": f.adult.ID, "kind": "first", "amount": "60,00", "adjustmentReason": "rien",
	}), http.StatusOK)
	if got.AdjustmentReason != nil || got.AdjustmentCents != 0 {
		t.Fatalf("montant égal : %+v", got.Membership)
	}

	// 0 € : gratuite, sans référence ni paiement.
	_, _, free := pendingRequest(t, f)
	got = expectAdminMembership(t, validateWith(f.admin, free.ID, map[string]any{
		"feeId": f.adult.ID, "kind": "first", "amount": "0", "adjustmentReason": "bénévole",
	}), http.StatusOK)
	if got.AmountCents != 0 || got.Payment != nil || got.AdjustmentCents != -6000 {
		t.Fatalf("gratuite par ajustement : %+v", got.Membership)
	}
}

func TestCreateMembershipWithAdjustment(t *testing.T) {
	f := newMsFixture(t)
	m := expectMember(t, f.admin.Do("POST", "/api/admin/members", adminBody("Zoé", "Durand", "1990-02-03", nil)), http.StatusCreated)
	res := f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": m.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "kind": "renewal",
		"amount": "50", "adjustmentReason": "soutien",
		"paid": map[string]any{"method": "cash", "receivedOn": msToday},
	})
	got := expectAdminMembership(t, res, http.StatusCreated)
	if got.AmountCents != 5000 || got.BaseAmountCents != 4000 || got.AdjustmentCents != 1000 ||
		got.Payment == nil || got.Payment.Status != "paid" || *got.Payment.PaidAmountCents != 5000 || got.Payment.DonationCents != 0 {
		t.Fatalf("création avec majoration : %+v %+v", got.Membership, got.Payment)
	}
	expectFieldError(t, f.admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": m.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "kind": "first", "amount": "50",
	}), "body.adjustmentReason", api.MsgAdjustmentReasonRequired)
}

func TestAdjustmentHiddenFromMember(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, _, ms := pendingRequest(t, f)
	expectAdminMembership(t, validateWith(f.admin, ms.ID, map[string]any{
		"feeId": f.adult.ID, "kind": "first", "amount": "55", "adjustmentReason": "motif-confidentiel",
	}), http.StatusOK)

	for _, path := range []string{myMembershipPath(ms.ID), "/api/me/memberships", "/api/me/dues"} {
		res := c.Do("GET", path, nil)
		body := string(res.Body)
		if res.Status != http.StatusOK || strings.Contains(body, "baseAmountCents") || strings.Contains(body, "adjustmentCents") ||
			strings.Contains(body, "adjustmentReason") || strings.Contains(body, "motif-confidentiel") {
			t.Fatalf("%s expose l'ajustement : %d %s", path, res.Status, body)
		}
		if !strings.Contains(body, `"amountCents":5500`) {
			t.Fatalf("%s : montant à payer absent : %s", path, body)
		}
	}

	// Une demande ne fixe pas de montant.
	c2, _, m2 := f.member(t, "Lou")
	res := c2.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": m2.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "declaredKind": "first", "amount": "1",
	})
	if res.Status == http.StatusCreated {
		var mine api.MyMembership
		res.JSON(t, &mine)
		if v := validate(t, f.admin, mine.ID, f.adult.ID, "first"); v.AmountCents != 6000 {
			t.Fatalf("montant fixé par le membre : %+v", v.Membership)
		}
	} else if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("demande avec montant : %d %s", res.Status, res.Body)
	}
}

func TestCardPaymentUsesAdjustedAmount(t *testing.T) {
	f := newCardFixture(t)
	// L'adhésion « Jeune » (30 €) de la fixture est validée : on la corrige à 25 €.
	res := f.admin.Do("PUT", adminMembershipPath(f.ms.ID, ""), map[string]any{
		"feeId": f.young.ID, "kind": "first", "amount": "25", "adjustmentReason": "réduction",
	})
	expectAdminMembership(t, res, http.StatusOK)
	st := f.start(t)
	in := f.env.HelloAsso.LastIntent()
	if in.Body["totalAmount"].(float64) != 2500 {
		t.Fatalf("montant envoyé à HelloAsso : %v", in.Body["totalAmount"])
	}
	f.env.HelloAsso.Authorize(in.ID, 2500, 0)
	f.env.Advance(10 * time.Second)
	code, out := f.confirm(t, f.member, st.OnlinePaymentID, "return")
	if code != http.StatusOK || out.Membership.Payment.Status != "paid" || *out.Membership.Payment.PaidAmountCents != 2500 {
		t.Fatalf("paiement : %d %+v", code, out)
	}
	if a := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), http.StatusOK); len(a.Alerts) != 0 {
		t.Fatalf("signalements : %+v", a.Alerts)
	}
}

// Correction d'une adhésion validée (US3).
func TestUpdateMembershipAdjustment(t *testing.T) {
	f := newMsFixture(t)
	setPayment(t, f.admin, nil)
	c, _, ms := pendingRequest(t, f)
	v := validate(t, f.admin, ms.ID, f.adult.ID, "first")
	ref := v.Payment.Reference
	put := func(body map[string]any) apitest.Response {
		return f.admin.Do("PUT", adminMembershipPath(ms.ID, ""), body)
	}

	got := expectAdminMembership(t, put(map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "50", "adjustmentReason": "remise"}), http.StatusOK)
	if got.AmountCents != 5000 || got.BaseAmountCents != 6000 || got.Payment.Reference != ref {
		t.Fatalf("ajustement : %+v", got.Membership)
	}
	// Changement de tarif avec le même montant : montant conservé, théorique recalculé.
	got = expectAdminMembership(t, put(map[string]any{"feeId": f.young.ID, "kind": "first", "amount": "50", "adjustmentReason": "remise"}), http.StatusOK)
	if got.AmountCents != 5000 || got.BaseAmountCents != 3000 || got.AdjustmentCents != 2000 {
		t.Fatalf("changement de tarif : %+v", got.Membership)
	}
	// Retour au théorique : ajustement et motif effacés.
	got = expectAdminMembership(t, put(map[string]any{"feeId": f.adult.ID, "kind": "first"}), http.StatusOK)
	if got.AmountCents != 6000 || got.AdjustmentReason != nil || got.AdjustmentCents != 0 {
		t.Fatalf("retour au théorique : %+v", got.Membership)
	}

	// Paiement annoncé, puis ajustement : l'annonce est conservée.
	if res := c.Do("PUT", myMembershipPath(ms.ID)+"/payment", map[string]any{"method": "cheque"}); res.Status != http.StatusOK {
		t.Fatalf("annonce : %d %s", res.Status, res.Body)
	}
	got = expectAdminMembership(t, put(map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "55", "adjustmentReason": "remise"}), http.StatusOK)
	if got.Payment.Status != "announced" || *got.Payment.AnnouncedMethod != "cheque" {
		t.Fatalf("annonce après ajustement : %+v", got.Payment)
	}

	// 0 € : gratuite, référence conservée ; retour à un montant positif : même référence.
	got = expectAdminMembership(t, put(map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "0", "adjustmentReason": "bénévole"}), http.StatusOK)
	if got.Payment != nil {
		t.Fatalf("gratuite : %+v", got.Payment)
	}
	got = expectAdminMembership(t, put(map[string]any{"feeId": f.adult.ID, "kind": "first"}), http.StatusOK)
	if got.Payment == nil || got.Payment.Reference != ref || got.Payment.Status != "due" {
		t.Fatalf("retour payant : %+v", got.Payment)
	}

	// Payée : correction refusée.
	expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(ms.ID, "/payment"), map[string]any{"method": "cash", "receivedOn": msToday}), http.StatusOK)
	if res := put(map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "50", "adjustmentReason": "x"}); res.Status != http.StatusConflict ||
		!strings.Contains(string(res.Body), api.MsgPaidCorrection) {
		t.Fatalf("correction d'une adhésion payée : %d %s", res.Status, res.Body)
	}

	// Saison close : ajustement refusé, constatation possible.
	_, _, other := pendingRequest(t, f)
	validate(t, f.admin, other.ID, f.adult.ID, "first")
	f.env.SetToday("2027-09-15")
	if res := f.admin.Do("PUT", adminMembershipPath(other.ID, ""), map[string]any{"feeId": f.adult.ID, "kind": "first", "amount": "50", "adjustmentReason": "x"}); res.Status != http.StatusConflict ||
		!strings.Contains(string(res.Body), api.MsgSeasonClosedAction) {
		t.Fatalf("saison close : %d %s", res.Status, res.Body)
	}
	got = expectAdminMembership(t, f.admin.Do("POST", adminMembershipPath(other.ID, "/payment"), map[string]any{
		"method": "cash", "receivedOn": "2027-09-15", "receivedAmount": "70",
	}), http.StatusOK)
	if got.Payment.DonationCents != 1000 {
		t.Fatalf("constatation sur saison close : %+v", got.Payment)
	}
}

// Ajustement pendant un paiement par carte en cours : la tentative n'est plus reprise.
func TestAdjustmentDuringCardPayment(t *testing.T) {
	f := newCardFixture(t)
	first := f.start(t)
	expectAdminMembership(t, f.admin.Do("PUT", adminMembershipPath(f.ms.ID, ""), map[string]any{
		"feeId": f.young.ID, "kind": "first", "amount": "20", "adjustmentReason": "réduction",
	}), http.StatusOK)
	again := f.start(t)
	if again.Resumed || again.OnlinePaymentID == first.OnlinePaymentID {
		t.Fatalf("tentative reprise malgré le nouveau montant : %+v", again)
	}
	if amount := f.env.HelloAsso.LastIntent().Body["totalAmount"].(float64); amount != 2000 {
		t.Fatalf("nouvelle intention : %v", amount)
	}
}
