package api_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Confirmation d'un dépôt d'adhésion publique (016, US1 ; research.md › R7).

func expectOutcome(t *testing.T, res apitest.Response, outcome string) api.ConfirmOutcome {
	t.Helper()
	if res.Status != http.StatusOK {
		t.Fatalf("confirmation : %d %s", res.Status, res.Body)
	}
	var o api.ConfirmOutcome
	res.JSON(t, &o)
	if o.Outcome != outcome || o.Kind != "membership" {
		t.Fatalf("issue %+v, attendu %s", o, outcome)
	}
	return o
}

func expectGone(t *testing.T, res apitest.Response) {
	t.Helper()
	if res.Status != http.StatusGone || !strings.Contains(res.Body, "Ce lien n'est plus valable.") {
		t.Fatalf("lien : %d %s, attendu 410", res.Status, res.Body)
	}
}

func TestConfirmPublicMembership(t *testing.T) {
	f := newPubFixture(t, true)
	club := createField(t, f.admin, fieldBody("member", "Club précédent", "short_text", "visibility", "editable"))
	size := createField(t, f.admin, fieldBody("membership", "Taille", "choice", "choices", []string{"S", "M"}, "visibility", "editable", "required", true))
	email := "claire@exemple.fr"
	token := f.deposit(t, f.publicBody(email, "createAccount", true, "phone", "06 12 34 56 78", "customFields", []map[string]any{
		{"fieldId": club.ID, "text": "AS Lyon"}, {"fieldId": size.ID, "choiceId": size.Choices[1].ID},
	}))

	c := f.env.Client()
	o := expectOutcome(t, f.confirm(c, token), "tracking")
	if !o.CreateAccount || o.Reason != nil {
		t.Fatalf("issue : %+v", o)
	}

	// Fiche publique sans compte, adresse confirmée, consentement daté.
	var (
		memberID                 int64
		origin                   string
		account                  sql.NullInt64
		confirmed, consented     sql.NullString
		memberEmail, memberPhone sql.NullString
	)
	if err := f.env.Store.ReadDB.QueryRow(`SELECT id, origin, account_id, email_confirmed_at, consented_at, email, phone FROM members`).
		Scan(&memberID, &origin, &account, &confirmed, &consented, &memberEmail, &memberPhone); err != nil {
		t.Fatal(err)
	}
	if origin != "public" || account.Valid || !confirmed.Valid || !consented.Valid || memberEmail.String != email || memberPhone.String != "0612345678" {
		t.Fatalf("fiche : %s %v %v %v %v %v", origin, account, confirmed, consented, memberEmail, memberPhone)
	}
	// Adhésion en attente, auteur « Demande publique ».
	var (
		membershipID           int64
		status, kind           string
		viaPublic              int
		requestedBy, createdBy sql.NullInt64
	)
	if err := f.env.Store.ReadDB.QueryRow(`SELECT id, status, declared_kind, via_public, requested_by, created_by FROM memberships WHERE member_id = ?`, memberID).
		Scan(&membershipID, &status, &kind, &viaPublic, &requestedBy, &createdBy); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || kind != "first" || viaPublic != 1 || requestedBy.Valid || createdBy.Valid {
		t.Fatalf("adhésion : %s %s %d %v %v", status, kind, viaPublic, requestedBy, createdBy)
	}
	got := expectAdminMembership(t, f.admin.Do("GET", adminMembershipPath(membershipID, ""), nil), http.StatusOK)
	if !got.ViaPublic || got.RequestedBy != nil || *findValue(t, got.CustomFields, size.ID).ChoiceID != size.Choices[1].ID {
		t.Fatalf("adhésion vue par l'administration : %+v", got)
	}
	m := expectMember(t, f.admin.Do("GET", adminMemberPath(memberID), nil), http.StatusOK)
	if v := findValue(t, m.CustomFields, club.ID); v.Text == nil || *v.Text != "AS Lyon" {
		t.Fatalf("champ de membre : %+v", m.CustomFields)
	}
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions`); n != 0 {
		t.Fatalf("dépôt conservé après confirmation : %d", n)
	}
	mails := f.env.Mails(email)
	if len(mails) != 2 || mails[1].Subject != "Votre lien de suivi" || !strings.HasPrefix(mails[1].Link, "http://frigg.test/suivi#jeton=") {
		t.Fatalf("e-mail du lien personnel : %+v", mails)
	}
	// L'accès par lien est ouvert dans la session du client qui confirme.
	if res := c.Do("POST", "/api/tracking/close", nil); res.Status != http.StatusNoContent {
		t.Fatalf("accès par lien absent après confirmation : %d %s", res.Status, res.Body)
	}

	// Lien déjà utilisé, puis lien expiré (48 h).
	expectGone(t, f.confirm(f.env.Client(), token))
	expectGone(t, f.confirm(f.env.Client(), "inconnu"))
	late := f.deposit(t, f.publicBody("tard@exemple.fr", "customFields", []map[string]any{
		{"fieldId": size.ID, "choiceId": size.Choices[0].ID},
	}))
	f.env.Advance(49 * time.Hour)
	expectGone(t, f.confirm(f.env.Client(), late))

	if n := publicLogs(t, f.env, "public_confirm"); n != 1 {
		t.Errorf("journal public_confirm : %d", n)
	}
	for _, secret := range []string{token, "Claire", "Martin"} {
		if strings.Contains(f.env.Logs.String(), secret) {
			t.Errorf("journaux : %q présent", secret)
		}
	}
}

// insertSubmission insère directement un dépôt valide et renvoie son jeton.
func (f *pubFixture) insertSubmission(t *testing.T, email, first string) string {
	t.Helper()
	raw, hash := auth.NewToken()
	now := time.Now().UTC()
	if _, err := f.env.Store.Write.CreatePublicSubmission(context.Background(), sqlc.CreatePublicSubmissionParams{
		Kind: "membership", SeasonID: sql.NullInt64{Int64: f.season.ID, Valid: true},
		FeeID: sql.NullInt64{Int64: f.adult.ID, Valid: true}, DeclaredKind: sql.NullString{String: "first", Valid: true},
		FirstName: first, LastName: "Martin", BirthDate: "1985-03-02", Email: email, CustomValues: "[]",
		ConsentedAt: now.Format(time.RFC3339), TokenHash: hash,
		ExpiresAt: now.Add(240 * time.Hour).Format("2006-01-02T15:04:05Z"), CreatedAt: now.Format("2006-01-02T15:04:05Z"),
	}); err != nil {
		t.Fatal(err)
	}
	return raw
}

// Limite par adresse (R9) : 10 confirmations par heure pour une adresse,
// même réparties sur plusieurs IP.
func TestConfirmPublicMembershipEmailLimit(t *testing.T) {
	f := newPubFixture(t, true)
	for i := range 10 {
		res := f.confirm(f.env.Client(), f.insertSubmission(t, "limite@exemple.fr", fmt.Sprintf("Claire%c", 'A'+i)))
		expectOutcome(t, res, "tracking")
	}
	res := f.confirm(f.env.Client(), f.insertSubmission(t, "limite@exemple.fr", "Claire"))
	if res.Status != http.StatusTooManyRequests {
		t.Fatalf("11e confirmation : %d %s", res.Status, res.Body)
	}
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions`); n != 1 {
		t.Fatalf("le dépôt refusé doit être conservé : %d", n)
	}
}

// Conditions revérifiées à la confirmation (FR-010).
func TestConfirmPublicMembershipClosed(t *testing.T) {
	t.Run("accueil désactivé", func(t *testing.T) {
		f := newPubFixture(t, true)
		token := f.deposit(t, f.publicBody("claire@exemple.fr"))
		f.setPublic(t, false)
		o := expectOutcome(t, f.confirm(f.env.Client(), token), "closed")
		if o.Reason == nil || *o.Reason != "Les demandes d'adhésion ne sont pas ouvertes." {
			t.Fatalf("raison : %v", o.Reason)
		}
		if n := count(t, f.env, `SELECT count(*) FROM members`) + count(t, f.env, `SELECT count(*) FROM public_submissions`); n != 0 {
			t.Fatalf("fiche ou dépôt après fermeture : %d", n)
		}
	})
	t.Run("saison close", func(t *testing.T) {
		f := newPubFixture(t, true)
		f.env.SetToday("2027-08-31")
		token := f.deposit(t, f.publicBody("claire@exemple.fr"))
		f.env.Advance(24 * time.Hour)
		o := expectOutcome(t, f.confirm(f.env.Client(), token), "closed")
		if o.Reason == nil || *o.Reason != "Cette saison n'est pas ouverte à l'adhésion." {
			t.Fatalf("raison : %v", o.Reason)
		}
		if n := count(t, f.env, `SELECT count(*) FROM memberships`); n != 0 {
			t.Fatalf("adhésion créée sur une saison close : %d", n)
		}
	})
}

// Adresse d'un compte (FR-033) : actif, rattachement ; désactivé ou plein,
// suivi par lien. La réponse du dépôt ne change pas.
func TestConfirmPublicMembershipAccount(t *testing.T) {
	f := newPubFixture(t, true)
	const (
		exists = "Un compte Frigg existe déjà pour cette adresse"
		full   = "Votre compte gère déjà 20 fiches"
	)

	t.Run("compte actif", func(t *testing.T) {
		email := f.env.Seed(auth.RoleMember, true)
		token := f.deposit(t, f.publicBody(email, "createAccount", true))
		if !strings.Contains(f.env.Mails(email)[0].Text, exists) {
			t.Fatalf("e-mail de dépôt sans mention du compte : %s", f.env.Mails(email)[0].Text)
		}
		c := f.env.Client()
		o := expectOutcome(t, f.confirm(c, token), "account")
		if o.CreateAccount {
			t.Error("la case « Créer un compte » doit être ignorée")
		}
		mine := myMembers(t, f.env.LoggedIn(email))
		if len(mine.Items) != 1 || mine.Items[0].FirstName != "Claire" {
			t.Fatalf("fiche non rattachée au compte : %+v", mine)
		}
		if n := len(f.env.Mails(email)); n != 1 {
			t.Fatalf("%d e-mails : aucun lien personnel attendu", n)
		}
		if n := count(t, f.env, `SELECT count(*) FROM tracking_links WHERE email = ?`, email); n != 0 {
			t.Fatalf("lien personnel émis : %d", n)
		}
		if res := c.Do("POST", "/api/tracking/close", nil); res.Status != http.StatusUnauthorized {
			t.Fatalf("accès par lien ouvert : %d", res.Status)
		}
	})

	t.Run("compte désactivé", func(t *testing.T) {
		email := f.env.Seed(auth.RoleMember, true)
		if _, err := f.env.Store.WriteDB.Exec(`UPDATE accounts SET disabled_at = '2026-10-01T10:00:00Z' WHERE email = ?`, email); err != nil {
			t.Fatal(err)
		}
		token := f.deposit(t, f.publicBody(email))
		if text := f.env.Mails(email)[0].Text; strings.Contains(text, exists) || strings.Contains(text, full) {
			t.Fatalf("e-mail de dépôt mentionnant le compte : %s", text)
		}
		expectOutcome(t, f.confirm(f.env.Client(), token), "tracking")
		if n := count(t, f.env, `SELECT count(*) FROM members WHERE email = ? AND account_id IS NULL`, email); n != 1 {
			t.Fatalf("fiche sans compte : %d", n)
		}
	})

	t.Run("compte plein", func(t *testing.T) {
		email := f.env.Seed(auth.RoleMember, true)
		a, err := f.env.Store.Read.GetAccountByEmail(context.Background(), email)
		if err != nil {
			t.Fatal(err)
		}
		for i := range 20 {
			insertForAccount(t, f.env, a.ID, fmt.Sprintf("Fiche%c", 'A'+i), "Durand", "2000-01-01")
		}
		unknown := submit(f.env.Client(), f.nextIP(), f.publicBody("inconnu@exemple.fr"))
		res := submit(f.env.Client(), f.nextIP(), f.publicBody(email))
		if res.Status != unknown.Status || res.Body != unknown.Body {
			t.Fatalf("réponse du dépôt différente : %d %s / %d %s", res.Status, res.Body, unknown.Status, unknown.Body)
		}
		mails := f.env.Mails(email)
		if len(mails) != 1 || !strings.Contains(mails[0].Text, full) {
			t.Fatalf("e-mail de dépôt : %+v", mails)
		}
		expectOutcome(t, f.confirm(f.env.Client(), mails[0].Token), "tracking")
		if n := count(t, f.env, `SELECT count(*) FROM members WHERE account_id = ?`, a.ID); n != 20 {
			t.Fatalf("fiches du compte : %d", n)
		}
	})
}

// Même identité et même adresse (FR-011) : la fiche suivie est réutilisée,
// et une demande déjà en attente renvoie au suivi sans rien créer.
func TestConfirmPublicMembershipDuplicate(t *testing.T) {
	f := newPubFixture(t, true)
	email := "claire@exemple.fr"
	expectOutcome(t, f.confirm(f.env.Client(), f.deposit(t, f.publicBody(email))), "tracking")
	f.env.Advance(2 * time.Hour) // limite d'e-mails par adresse
	o := expectOutcome(t, f.confirm(f.env.Client(), f.deposit(t, f.publicBody(" CLAIRE@exemple.fr", "firstName", "claire"))), "duplicate")
	if o.Reason != nil {
		t.Fatalf("issue : %+v", o)
	}
	if n := count(t, f.env, `SELECT count(*) FROM members`); n != 1 {
		t.Fatalf("fiches : %d", n)
	}
	if n := count(t, f.env, `SELECT count(*) FROM memberships`); n != 1 {
		t.Fatalf("adhésions : %d", n)
	}
	if n := count(t, f.env, `SELECT count(*) FROM public_submissions`); n != 0 {
		t.Fatalf("dépôt conservé : %d", n)
	}
}

// Champ archivé ou ajouté entre le dépôt et la confirmation : la valeur est
// ignorée et le nouveau champ n'est pas exigé.
func TestConfirmPublicMembershipFieldsChanged(t *testing.T) {
	f := newPubFixture(t, true)
	old := createField(t, f.admin, fieldBody("membership", "Taille", "short_text", "visibility", "editable"))
	kept := createField(t, f.admin, fieldBody("member", "Club précédent", "short_text", "visibility", "editable"))
	token := f.deposit(t, f.publicBody("claire@exemple.fr", "customFields", []map[string]any{
		{"fieldId": old.ID, "text": "M"}, {"fieldId": kept.ID, "text": "AS Lyon"},
	}))
	expectField(t, f.admin.Do("POST", customFieldPath(old.ID, "/archive"), nil), http.StatusOK)
	createField(t, f.admin, fieldBody("membership", "Certificat", "date", "visibility", "editable", "required", true))

	expectOutcome(t, f.confirm(f.env.Client(), token), "tracking")
	if n := count(t, f.env, `SELECT count(*) FROM membership_field_values`); n != 0 {
		t.Fatalf("valeur d'un champ archivé enregistrée : %d", n)
	}
	if n := count(t, f.env, `SELECT count(*) FROM member_field_values WHERE field_id = ?`, kept.ID); n != 1 {
		t.Fatalf("valeur du champ conservé : %d", n)
	}
}
