package api_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

func countRows(t *testing.T, env *apitest.Env, query string, args ...any) int {
	t.Helper()
	var n int
	if err := env.Store.ReadDB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestSendCommunication(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	c := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(key, f.season.ID, unpaid, 2)), http.StatusCreated)
	if c.TargetLabel != "2026-2027 · En impayé" || c.MembersCount != 4 || c.UnreachableCount != 1 || c.RecipientsCount != 2 ||
		c.Preset != "unpaid" || c.Delivery.Pending != 2 || len(c.Recipients) != 2 || c.Author == nil || c.Season.Label != "2026-2027" {
		t.Fatalf("communication : %+v", c)
	}
	if c.Subject != "Rappel : cotisation 2026-2027" || !slices.Equal(c.Target.Situations, []string{"announced", "due"}) {
		t.Errorf("objet ou cible : %q %v", c.Subject, c.Target.Situations)
	}
	if n := countRows(t, f.env, `SELECT count(*) FROM outbound_emails WHERE communication_id = ? AND from_name = 'Club de Frigg'`, c.ID); n != 2 {
		t.Errorf("%d e-mail(s) en file", n)
	}

	f.env.FlushMail()
	msgs := f.env.Mail.Messages()
	if len(msgs) != 2 {
		t.Fatalf("%d message(s) envoyé(s)", len(msgs))
	}
	var to []string
	for _, m := range msgs {
		to = append(to, m.To)
		if m.Msg.Subject != "Rappel : cotisation 2026-2027" || m.Msg.FromName != "Club de Frigg" ||
			!strings.Contains(m.Msg.Text, "pour la saison 2026-2027") || strings.Contains(m.Msg.Text, "@") {
			t.Errorf("message à %s : %+v", m.To, m.Msg)
		}
	}
	slices.Sort(to)
	want := []string{f.claire, f.paul}
	slices.Sort(want)
	if !slices.Equal(to, want) {
		t.Errorf("destinataires %v, attendus %v (un exemplaire par adresse)", to, want)
	}
	got := expectCommunication(t, f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d", c.ID), nil), http.StatusOK)
	if got.Delivery.Sent != 2 || got.Recipients[0].Status != "sent" || got.Recipients[0].SentAt == nil {
		t.Errorf("état d'envoi : %+v", got.Delivery)
	}

	// Journal : aucune adresse ni objet.
	logs := f.env.Logs.String()
	if !strings.Contains(logs, "communication envoyée") {
		t.Error("journal d'envoi absent")
	}
	for _, no := range []string{f.claire, f.paul, "Rappel"} {
		if strings.Contains(logs, no) {
			t.Errorf("journal : %q présent", no)
		}
	}
}

func TestSendCommunicationIdempotent(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	body := sendBody(key, f.season.ID, unpaid, 2)
	var wg sync.WaitGroup
	statuses := make([]int, 2)
	ids := make([]int64, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := f.admin.Do("POST", "/api/admin/communications", body)
			statuses[i] = res.Status
			var c api.Communication
			res.JSON(t, &c)
			ids[i] = c.ID
		}()
	}
	wg.Wait()
	slices.Sort(statuses)
	if !slices.Equal(statuses, []int{http.StatusOK, http.StatusCreated}) || ids[0] != ids[1] {
		t.Fatalf("double soumission : statuts %v, identifiants %v", statuses, ids)
	}
	// Une troisième soumission, même avec un autre corps, renvoie l'existante.
	body["subject"] = "Autre"
	again := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", body), http.StatusOK)
	if again.ID != ids[0] || again.Subject == "Autre" {
		t.Errorf("renvoi : %+v", again)
	}
	if n := countRows(t, f.env, `SELECT count(*) FROM communications`); n != 1 {
		t.Errorf("%d communications", n)
	}
	if n := countRows(t, f.env, `SELECT count(*) FROM outbound_emails`); n != 2 {
		t.Errorf("%d e-mails en file", n)
	}
}

func TestSendCommunicationAudienceChanged(t *testing.T) {
	f := newCmFixture(t)
	// Le paiement de Paul est constaté entre l'aperçu et l'envoi.
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE memberships SET payment_status = 'paid', paid_method = 'cheque', paid_on = '2026-11-02'
		WHERE member_id = ?`, f.members["Paul"]); err != nil {
		t.Fatal(err)
	}
	res := f.admin.Do("POST", "/api/admin/communications", sendBody(newDraftKey(), f.season.ID, unpaid, 2))
	if res.Status != http.StatusConflict {
		t.Fatalf("statut %d : %s", res.Status, res.Body)
	}
	var p struct {
		Detail   string       `json:"detail"`
		Audience api.Audience `json:"audience"`
	}
	res.JSON(t, &p)
	if p.Detail != api.MsgCommunicationChanged || p.Audience.RecipientsCount != 1 {
		t.Errorf("409 : %+v", p)
	}
	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/problem+json") {
		t.Errorf("Content-Type %q", res.Header.Get("Content-Type"))
	}
	if n := countRows(t, f.env, `SELECT count(*) FROM communications`); n != 0 {
		t.Error("communication enregistrée malgré le conflit")
	}
}

func TestSendCommunicationValidation(t *testing.T) {
	f := newCmFixture(t)
	empty := sendBody(newDraftKey(), f.season.ID, []string{"due"}, 1)
	empty["target"] = map[string]any{"seasonId": f.season.ID, "situations": []string{"due"}, "feeIds": []int64{f.adult.ID}}
	res := f.admin.Do("POST", "/api/admin/communications", empty)
	if res.Status != http.StatusConflict {
		t.Fatalf("décompte 1 attendu, 0 réel : %d %s", res.Status, res.Body)
	}
	for name, tc := range map[string]struct {
		field    string
		value    any
		location string
	}{
		"objet vide":            {"subject", "   ", "body.subject"},
		"objet sur deux lignes": {"subject", "Ligne 1\nLigne 2", "body.subject"},
		"objet trop long":       {"subject", strings.Repeat("a", 151), "body.subject"},
		"texte trop long":       {"body", strings.Repeat("a", 10001), "body.body"},
		"clé mal formée":        {"draftKey", "../../etc", "body.draftKey"},
	} {
		body := sendBody(newDraftKey(), f.season.ID, unpaid, 2)
		body[tc.field] = tc.value
		expectCmFieldError(t, f.admin.Do("POST", "/api/admin/communications", body), http.StatusUnprocessableEntity, tc.location, "")
		_ = name
	}

	// Aperçu : rendu échappé, aucune écriture.
	res = f.admin.Do("POST", "/api/admin/communications/preview", map[string]any{
		"seasonId": f.season.ID, "subject": "Objet", "body": "<script>x</script>\nhttps://exemple.test",
	})
	if res.Status != http.StatusOK {
		t.Fatalf("aperçu : %d %s", res.Status, res.Body)
	}
	var p api.Preview
	res.JSON(t, &p)
	if !strings.Contains(p.HTML, "&lt;script&gt;") || strings.Contains(p.HTML, "<script>") ||
		!strings.Contains(p.HTML, `<a href="https://exemple.test"`) || !strings.Contains(p.Text, "Club de Frigg") {
		t.Errorf("aperçu : %+v", p)
	}
	if n := countRows(t, f.env, `SELECT count(*) FROM communications`) + countRows(t, f.env, `SELECT count(*) FROM outbound_emails`); n != 0 {
		t.Error("l'aperçu a écrit en base")
	}
}

func TestSendCommunicationNoRecipient(t *testing.T) {
	f := newCmFixture(t)
	// Seul Hugo, sans adresse, est à payer au tarif Jeune hors Léo et Inès :
	// on les retire de la cible en marquant leurs adhésions payées.
	for _, m := range []string{"Léo", "Inès"} {
		if _, err := f.env.Store.WriteDB.Exec(`UPDATE memberships SET payment_status = 'paid', paid_method = 'cash', paid_on = '2026-10-01'
			WHERE member_id = ?`, f.members[m]); err != nil {
			t.Fatal(err)
		}
	}
	body := sendBody(newDraftKey(), f.season.ID, []string{"due"}, 1)
	res := f.admin.Do("POST", "/api/admin/communications", body)
	if res.Status != http.StatusConflict {
		t.Fatalf("statut %d", res.Status)
	}
	// Pour atteindre le 422, le décompte attendu doit correspondre (0) : le
	// schéma impose au moins 1, la cible vide est donc toujours refusée.
	body["expectedRecipients"] = 0
	expectCmFieldError(t, f.admin.Do("POST", "/api/admin/communications", body), http.StatusUnprocessableEntity, "body.expectedRecipients", "")
}

// SC-004 : un envoi à 500 destinataires est accepté immédiatement.
func TestSendCommunicationVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("volume")
	}
	f := newCmFixture(t)
	tx, _ := f.env.Store.WriteDB.Begin()
	for i := range 498 {
		res, err := tx.Exec(`INSERT INTO members (first_name, last_name, birth_date, email, created_at, updated_at)
			VALUES ('Prénom', ?, '2000-01-01', ?, '2026-09-15T10:00:00Z', '2026-09-15T10:00:00Z')`, fmt.Sprintf("Nom%04d", i), fmt.Sprintf("m%d@exemple.fr", i))
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		if _, err := tx.Exec(`INSERT INTO memberships (member_id, season_id, fee_id, status, kind, amount_cents, requested_at, payment_reference, payment_status, updated_at)
			VALUES (?, ?, ?, 'validated', 'first', 3000, '2026-09-15T10:00:00Z', ?, 'due', '2026-09-15T10:00:00Z')`, id, f.season.ID, f.young.ID, fmt.Sprintf("ADH-VOL-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	c := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(newDraftKey(), f.season.ID, unpaid, 500)), http.StatusCreated)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("envoi accepté en %v", d)
	}
	if c.RecipientsCount != 500 || countRows(t, f.env, `SELECT count(*) FROM communication_recipients`) != 500 ||
		countRows(t, f.env, `SELECT count(*) FROM outbound_emails`) != 500 {
		t.Errorf("destinataires : %d", c.RecipientsCount)
	}
}
