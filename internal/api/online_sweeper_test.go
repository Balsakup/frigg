package api_test

import (
	"context"
	"testing"
	"time"
)

func TestSweeperExpiresAndRecords(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	st := f.start(t)
	f.env.Advance(30 * time.Minute)
	f.env.Sweeper.Tick(ctx)
	if got := f.mine(t); got.CardPayment == nil || got.CardPayment.Status != "pending" {
		t.Fatalf("tentative récente balayée : %+v", got.CardPayment)
	}
	f.env.Advance(31 * time.Minute)
	f.env.Sweeper.Tick(ctx)
	if got := f.mine(t); got.CardPayment.Status != "expired" {
		t.Fatalf("tentative non expirée : %+v", got.CardPayment)
	}
	_ = st

	// Paiement autorisé découvert par le balayage.
	g := newCardFixture(t)
	g.start(t)
	g.env.HelloAsso.Authorize(g.env.HelloAsso.LastIntent().ID, 3000, 0)
	g.env.Advance(61 * time.Minute)
	g.env.Sweeper.Tick(ctx)
	if got := g.mine(t); got.Payment.Status != "paid" {
		t.Fatalf("paiement non constaté par le balayage : %+v", got.Payment)
	}
}

func TestSweeperUnverifiedAlert(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	f.start(t)
	f.env.HelloAsso.Down(true)
	f.env.Advance(2 * time.Hour)
	f.env.Sweeper.Tick(ctx)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), 200)
	if len(got.Alerts) != 0 || got.OnlinePayments[0].Status != "pending" {
		t.Fatalf("signalement prématuré : %+v %+v", got.Alerts, got.OnlinePayments)
	}
	f.env.Advance(24 * time.Hour)
	f.env.Sweeper.Tick(ctx)
	got = expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), 200)
	if len(got.Alerts) != 1 || got.Alerts[0].Kind != "unverified" {
		t.Fatalf("signalement unverified : %+v", got.Alerts)
	}
}

func TestSweeperOtherEnvironment(t *testing.T) {
	ctx := context.Background()
	f := newCardFixture(t)
	f.start(t)
	if _, err := f.env.Store.WriteDB.ExecContext(ctx, `UPDATE online_payments SET environment = 'production'`); err != nil {
		t.Fatal(err)
	}
	calls := f.env.HelloAsso.TotalCalls()
	f.env.Advance(61 * time.Minute)
	f.env.Sweeper.Tick(ctx)
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(f.ms.ID, ""), nil), 200)
	op := got.OnlinePayments[0]
	if op.Status != "expired" || op.ErrorCode == nil || *op.ErrorCode != "environment_changed" || len(got.Alerts) != 0 {
		t.Fatalf("tentative d'un autre environnement : %+v %+v", op, got.Alerts)
	}
	if f.env.HelloAsso.TotalCalls() != calls {
		t.Fatal("HelloAsso appelé pour un autre environnement")
	}
}
