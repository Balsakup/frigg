package api_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
)

func listCommunications(t *testing.T, f cmFixture, query string) api.CommunicationPage {
	t.Helper()
	res := f.admin.Do("GET", "/api/admin/communications"+query, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("historique : %d %s", res.Status, res.Body)
	}
	var p api.CommunicationPage
	res.JSON(t, &p)
	return p
}

func TestCommunicationHistory(t *testing.T) {
	f := newCmFixture(t)
	if p := listCommunications(t, f, ""); p.Total != 0 || p.Items == nil || len(p.Items) != 0 {
		t.Fatalf("historique vide : %+v", p)
	}
	first := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(newDraftKey(), f.season.ID, unpaid, 2)), http.StatusCreated)
	f.env.Advance(time.Minute)
	all := sendBody(newDraftKey(), f.season.ID, []string{"up_to_date", "announced", "due"}, 4)
	all["subject"] = "Assemblée générale"
	second := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", all), http.StatusCreated)

	// Le serveur d'envoi échoue définitivement pour Paul.
	if _, err := f.env.Store.WriteDB.Exec(`UPDATE outbound_emails SET status = 'failed' WHERE recipient = ? AND communication_id = ?`, f.paul, second.ID); err != nil {
		t.Fatal(err)
	}
	f.env.FlushMail()

	p := listCommunications(t, f, "")
	if p.Total != 2 || p.Items[0].ID != second.ID || p.Items[1].ID != first.ID {
		t.Fatalf("ordre : %+v", p.Items)
	}
	s := p.Items[0]
	if s.Subject != "Assemblée générale" || s.Preset != "all" || s.RecipientsCount != 4 || s.Author == nil ||
		s.Delivery != (api.Delivery{Sent: 3, Failed: 1}) || s.AttachmentsCount != 0 {
		t.Errorf("ligne : %+v", s)
	}
	if page := listCommunications(t, f, "?page=2&perPage=1"); page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != first.ID {
		t.Errorf("pagination : %+v", page)
	}
	if res := f.admin.Do("GET", "/api/admin/communications?perPage=101", nil); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("perPage=101 : %d", res.Status)
	}

	c := expectCommunication(t, f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d", second.ID), nil), http.StatusOK)
	if c.Recipients[0].Status != "failed" || *c.Recipients[0].Email != f.paul || c.Recipients[1].Status != "sent" {
		t.Errorf("destinataires : échecs en tête attendus : %+v", c.Recipients)
	}

	// La purge de la file (30 jours) ne fait pas perdre l'état.
	if _, err := f.env.Store.WriteDB.Exec(`DELETE FROM outbound_emails WHERE status = 'sent'`); err != nil {
		t.Fatal(err)
	}
	c = expectCommunication(t, f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d", second.ID), nil), http.StatusOK)
	if c.Delivery != (api.Delivery{Sent: 3, Failed: 1}) {
		t.Errorf("après purge : %+v", c.Delivery)
	}
	expectProblem(t, f.admin.Do("GET", "/api/admin/communications/9999", nil), http.StatusNotFound, api.MsgCommunicationGone)

	// FR-020 : aucune opération ne modifie ni ne supprime une communication.
	item := f.env.API.OpenAPI().Paths["/admin/communications/{id}"]
	if item.Put != nil || item.Patch != nil || item.Delete != nil {
		t.Error("une communication envoyée ne doit pas être modifiable")
	}
}

func TestCommunicationAttachmentSweeper(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	old := expectAttachment(t, upload(t, f.admin, key, "ancien.pdf", samplePDF(100)))
	f.env.Advance(2 * time.Hour)
	recent := expectAttachment(t, upload(t, f.admin, newDraftKey(), "recent.pdf", samplePDF(100)))
	sentKey := newDraftKey()
	expectAttachment(t, upload(t, f.admin, sentKey, "envoye.pdf", samplePDF(100)))
	expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(sentKey, f.season.ID, unpaid, 2)), http.StatusCreated)

	orphanOld := filepath.Join(f.env.Files.Dir, strings.Repeat("a", 32))
	orphanNew := filepath.Join(f.env.Files.Dir, strings.Repeat("b", 32))
	for _, p := range []string{orphanOld, orphanNew} {
		_ = os.WriteFile(p, []byte("x"), 0o640)
	}
	past := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(orphanOld, past, past)

	// 23 h après le premier dépôt : rien n'expire.
	f.env.Advance(21 * time.Hour)
	f.env.AttachmentSweeper.Tick(context.Background())
	if n := countRows(t, f.env, `SELECT count(*) FROM communication_attachments`); n != 3 {
		t.Fatalf("%d pièces après 23 h", n)
	}
	// 25 h : la pièce abandonnée disparaît, la récente et l'envoyée restent.
	f.env.Advance(2 * time.Hour)
	f.env.AttachmentSweeper.Tick(context.Background())
	if n := countRows(t, f.env, `SELECT count(*) FROM communication_attachments WHERE id = ?`, old.ID); n != 0 {
		t.Error("pièce abandonnée conservée")
	}
	if n := countRows(t, f.env, `SELECT count(*) FROM communication_attachments WHERE id = ? OR communication_id IS NOT NULL`, recent.ID); n != 2 {
		t.Errorf("pièces conservées : %d", n)
	}
	files := storedFiles(t, f)
	if len(files) != 3 { // récente, envoyée, orphelin récent
		t.Errorf("fichiers : %v", files)
	}
	if _, err := os.Stat(orphanOld); err == nil {
		t.Error("orphelin ancien conservé")
	}
	if _, err := os.Stat(orphanNew); err != nil {
		t.Error("orphelin récent supprimé")
	}
}

func TestDeleteAccountErasesCommunicationAddress(t *testing.T) {
	f := newCmFixture(t)
	// Claire reçoit une communication déjà envoyée, puis une en attente.
	sent := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(newDraftKey(), f.season.ID, unpaid, 2)), http.StatusCreated)
	f.env.FlushMail()
	pending := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(newDraftKey(), f.season.ID, unpaid, 2)), http.StatusCreated)

	id := accountID(t, f.env, f.claire)
	if res := f.admin.Do("DELETE", accountPath(id, fmt.Sprintf("?members=keep&memberIds=%d,%d", f.members["Léo"], f.members["Inès"])), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	for _, c := range []api.Communication{sent, pending} {
		got := expectCommunication(t, f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d", c.ID), nil), http.StatusOK)
		erased := 0
		for _, r := range got.Recipients {
			if r.Email == nil {
				erased++
				if c.ID == pending.ID && r.Status != "failed" {
					t.Errorf("e-mail en attente supprimé : état %s", r.Status)
				}
			} else if strings.EqualFold(*r.Email, f.claire) {
				t.Error("adresse du compte supprimé conservée")
			}
		}
		if erased != 1 {
			t.Errorf("communication %d : %d adresse(s) effacée(s)", c.ID, erased)
		}
	}
}

func TestDeleteAuthorKeepsCommunication(t *testing.T) {
	f := newCmFixture(t)
	author := f.env.Seed("admin", true)
	authorClient := f.env.LoggedIn(author)
	key := newDraftKey()
	a := expectAttachment(t, upload(t, authorClient, key, "a.pdf", samplePDF(500)))
	c := expectCommunication(t, authorClient.Do("POST", "/api/admin/communications", sendBody(key, f.season.ID, unpaid, 2)), http.StatusCreated)
	if res := f.admin.Do("DELETE", accountPath(accountID(t, f.env, author), ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression de l'auteur : %d %s", res.Status, res.Body)
	}
	got := expectCommunication(t, f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d", c.ID), nil), http.StatusOK)
	if got.Author != nil || len(got.Attachments) != 1 {
		t.Fatalf("communication après suppression de l'auteur : auteur %v, %d pièce(s)", got.Author, len(got.Attachments))
	}
	if res := f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d/attachments/%d", c.ID, a.ID), nil); res.Status != http.StatusOK {
		t.Errorf("téléchargement après suppression de l'auteur : %d", res.Status)
	}
}
