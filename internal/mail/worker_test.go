package mail_test

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/mail/mailtest"
)

func setup(t *testing.T) (*database.Store, *mailtest.Sender, *mail.Worker, *time.Time) {
	t.Helper()
	ctx := context.Background()
	store, err := database.Open(filepath.Join(t.TempDir(), "frigg.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	if err := database.Migrate(ctx, store.WriteDB); err != nil {
		t.Fatal(err)
	}
	sender := &mailtest.Sender{}
	now := time.Now().UTC().Add(time.Minute)
	w := &mail.Worker{
		Queries:  store.Write,
		Sender:   sender,
		Interval: time.Second,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:      func() time.Time { return now },
	}
	return store, sender, w, &now
}

func status(t *testing.T, store *database.Store) (string, int64, string) {
	t.Helper()
	var s, next string
	var attempts int64
	if err := store.ReadDB.QueryRow("SELECT status, attempts, next_attempt_at FROM outbound_emails").Scan(&s, &attempts, &next); err != nil {
		t.Fatal(err)
	}
	return s, attempts, next
}

func TestWorkerDelivers(t *testing.T) {
	ctx := context.Background()
	store, sender, w, _ := setup(t)
	msg, err := mail.Render("test", mail.Data{Association: "Les Sentiers du Val", Link: "http://x/#abc"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mail.Enqueue(ctx, store.Write, "claire@exemple.fr", msg); err != nil {
		t.Fatal(err)
	}
	w.Tick(ctx)
	if s, _, _ := status(t, store); s != "sent" {
		t.Fatalf("statut %q, attendu sent", s)
	}
	got, ok := sender.Last("claire@exemple.fr")
	if !ok || got.Msg.Subject != "Message de test" {
		t.Fatalf("message non reçu : %+v", got)
	}
}

func TestWorkerRetriesThenFails(t *testing.T) {
	ctx := context.Background()
	store, sender, w, now := setup(t)
	sender.Fail = true
	msg, _ := mail.Render("test", mail.Data{Association: "A"})
	if err := mail.Enqueue(ctx, store.Write, "x@exemple.fr", msg); err != nil {
		t.Fatal(err)
	}

	for i := int64(1); i < 10; i++ {
		w.Tick(ctx)
		s, attempts, next := status(t, store)
		if s != "pending" || attempts != i {
			t.Fatalf("tentative %d : statut %q, tentatives %d", i, s, attempts)
		}
		want := now.Add(mail.Backoff(i)).Format(mail.TimeFormat)
		if next != want {
			t.Fatalf("tentative %d : prochaine %s, attendue %s", i, next, want)
		}
		*now = now.Add(mail.Backoff(i))
	}
	w.Tick(ctx)
	if s, attempts, _ := status(t, store); s != "failed" || attempts != 10 {
		t.Fatalf("statut %q après %d tentatives, attendu failed après 10", s, attempts)
	}
}

func TestBackoff(t *testing.T) {
	for attempts, want := range map[int64]time.Duration{1: time.Minute, 2: 2 * time.Minute, 3: 4 * time.Minute, 7: time.Hour, 20: time.Hour} {
		if got := mail.Backoff(attempts); got != want {
			t.Errorf("Backoff(%d) = %v, attendu %v", attempts, got, want)
		}
	}
}

func TestWorkerPurgesOldSentEmails(t *testing.T) {
	ctx := context.Background()
	store, _, w, now := setup(t)
	msg, _ := mail.Render("test", mail.Data{Association: "A"})
	_ = mail.Enqueue(ctx, store.Write, "x@exemple.fr", msg)
	w.Tick(ctx)
	*now = now.Add(31 * 24 * time.Hour)
	w.Tick(ctx)
	var n int
	_ = store.ReadDB.QueryRow("SELECT count(*) FROM outbound_emails").Scan(&n)
	if n != 0 {
		t.Fatalf("%d e-mail(s) restant(s) après purge", n)
	}
}
