package mail_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/brand"
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/database"
	"github.com/balsakup/frigg/internal/database/sqlc"
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

// communicationFixture insère une communication portant une pièce jointe
// enregistrée dans files, et renvoie son identifiant.
func communicationFixture(t *testing.T, store *database.Store, files *communication.FileStore, data []byte) int64 {
	t.Helper()
	ctx := context.Background()
	const ts = "2026-09-01T10:00:00Z"
	stored, size, sum, err := files.Save(bytes.NewReader(data), communication.MaxFileBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO seasons (id, starts_on, ends_on, created_at) VALUES (1, '2026-09-01', '2027-08-31', '` + ts + `')`,
		`INSERT INTO communications (id, draft_key, season_id, situations, target_label, subject, body, members_count, unreachable_count, recipients_count, created_at)
		 VALUES (1, '00000001-0000-4000-8000-000000000000', 1, 'due', 'Cible', 'Objet', 'Texte', 1, 0, 1, '` + ts + `')`,
	} {
		if _, err := store.WriteDB.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.WriteDB.ExecContext(ctx, `INSERT INTO communication_attachments (draft_key, communication_id, stored_name, original_name, content_type, size_bytes, sha256, created_at)
		VALUES ('00000001-0000-4000-8000-000000000000', 1, ?, 'Convocation été.pdf', 'application/pdf', ?, ?, ?)`, stored, size, sum, ts); err != nil {
		t.Fatal(err)
	}
	return 1
}

func enqueueCommunication(t *testing.T, store *database.Store, id int64, to string) {
	t.Helper()
	if _, err := store.Write.EnqueueCommunicationEmail(context.Background(), sqlc.EnqueueCommunicationEmailParams{
		Recipient: to, Subject: "Objet", BodyText: "t", BodyHtml: "h", Now: "2026-09-01T10:00:00Z",
		CommunicationID: sql.NullInt64{Int64: id, Valid: true}, FromName: sql.NullString{String: "Club de Frigg", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkerSendsCommunicationAttachments(t *testing.T) {
	ctx := context.Background()
	store, sender, w, _ := setup(t)
	files, _ := communication.NewFileStore(t.TempDir())
	w.Files = files
	data := []byte("%PDF-1.7 convocation")
	id := communicationFixture(t, store, files, data)
	enqueueCommunication(t, store, id, "claire@exemple.fr")
	enqueueCommunication(t, store, id, "paul@exemple.fr")
	msg, _ := mail.Render("test", mail.Data{Association: "A"})
	_ = mail.Enqueue(ctx, store.Write, "autre@exemple.fr", msg)
	w.Tick(ctx)
	for _, to := range []string{"claire@exemple.fr", "paul@exemple.fr"} {
		got, ok := sender.Last(to)
		if !ok || got.Msg.FromName != "Club de Frigg" || len(got.Msg.Attachments) != 1 {
			t.Fatalf("%s : %+v", to, got)
		}
		a := got.Msg.Attachments[0]
		if a.Name != "Convocation été.pdf" || a.ContentType != "application/pdf" || !bytes.Equal(a.Data, data) {
			t.Errorf("%s : pièce %q %q %q", to, a.Name, a.ContentType, a.Data)
		}
	}
	if got, _ := sender.Last("autre@exemple.fr"); got.Msg.FromName != "" || len(got.Msg.Attachments) != 0 {
		t.Errorf("e-mail ordinaire modifié : %+v", got.Msg)
	}
}

func TestWorkerFailsOnMissingAttachment(t *testing.T) {
	ctx := context.Background()
	store, sender, w, _ := setup(t)
	files, _ := communication.NewFileStore(t.TempDir())
	w.Files = files
	id := communicationFixture(t, store, files, []byte("%PDF-1.7"))
	var stored string
	_ = store.ReadDB.QueryRow(`SELECT stored_name FROM communication_attachments`).Scan(&stored)
	if err := files.Remove(stored); err != nil {
		t.Fatal(err)
	}
	enqueueCommunication(t, store, id, "claire@exemple.fr")
	w.Tick(ctx)
	if s, attempts, _ := status(t, store); s != "failed" || attempts != 1 {
		t.Fatalf("statut %q après %d tentative(s), attendu failed immédiat", s, attempts)
	}
	if len(sender.Messages()) != 0 {
		t.Error("e-mail envoyé sans sa pièce jointe")
	}
}

// SC-004 : 500 e-mails portant une pièce de 5 Mio partent en 25 lots de 20.
func TestWorkerCommunicationThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("volume")
	}
	ctx := context.Background()
	store, sender, w, _ := setup(t)
	files, _ := communication.NewFileStore(t.TempDir())
	w.Files = files
	id := communicationFixture(t, store, files, bytes.Repeat([]byte("x"), communication.MaxFileBytes))
	for i := range 500 {
		enqueueCommunication(t, store, id, fmt.Sprintf("adherent%d@exemple.fr", i))
	}
	for range 25 {
		w.Tick(ctx)
	}
	var pending int
	_ = store.ReadDB.QueryRow(`SELECT count(*) FROM outbound_emails WHERE status <> 'sent'`).Scan(&pending)
	if pending != 0 || len(sender.Messages()) != 500 {
		t.Fatalf("%d e-mail(s) non envoyé(s), %d envoyés", pending, len(sender.Messages()))
	}
}

// L'habillage (nom, couleurs, logo, expéditeur) est celui en vigueur au moment
// du passage du worker ; les lignes antérieures (wrap = 0) partent telles
// quelles (specs/010-general-settings › R6).
func TestWorkerWrapsWithCurrentIdentity(t *testing.T) {
	ctx := context.Background()
	store, sender, w, _ := setup(t)
	id := mail.Identity{Name: "Ancien nom", Palette: brand.Default}
	w.Identity = func(context.Context) (mail.Identity, error) { return id, nil }

	msg, err := mail.Render("invitation", mail.Data{Association: "Ancien nom", Link: "http://x/#abc"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mail.Enqueue(ctx, store.Write, "claire@exemple.fr", msg); err != nil {
		t.Fatal(err)
	}
	if _, err := store.WriteDB.Exec(`INSERT INTO outbound_emails (recipient, subject, body_text, body_html, status, next_attempt_at, created_at, from_name)
		VALUES ('ancien@exemple.fr', 'Ancien', 'texte complet', '<html>complet</html>', 'pending', ?, ?, 'Expéditeur figé')`,
		time.Now().UTC().Format(mail.TimeFormat), time.Now().UTC().Format(mail.TimeFormat)); err != nil {
		t.Fatal(err)
	}

	pale := brand.MustHex("#fff3b0")
	id = mail.Identity{Name: "Nouveau nom", Palette: brand.Derive(&pale, nil),
		Logo: &mail.Attachment{Name: "logo.png", ContentType: "image/png", Data: []byte("PNG")}}
	w.Tick(ctx)

	got, ok := sender.Last("claire@exemple.fr")
	if !ok {
		t.Fatal("message non envoyé")
	}
	if got.Msg.FromName != "Nouveau nom" || !bytes.Contains([]byte(got.Msg.HTML), []byte("Nouveau nom")) ||
		!bytes.Contains([]byte(got.Msg.HTML), []byte("background:#fff3b0")) || len(got.Msg.Inline) != 1 {
		t.Errorf("habillage : expéditeur %q, %d image(s)\n%s", got.Msg.FromName, len(got.Msg.Inline), got.Msg.HTML)
	}
	old, ok := sender.Last("ancien@exemple.fr")
	if !ok || old.Msg.HTML != "<html>complet</html>" || old.Msg.FromName != "Expéditeur figé" || len(old.Msg.Inline) != 0 {
		t.Errorf("ligne antérieure modifiée : %+v", old.Msg)
	}
}
