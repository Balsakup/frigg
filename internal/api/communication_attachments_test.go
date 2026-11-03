package api_test

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/communication"
)

func storedFiles(t *testing.T, f cmFixture) []string {
	t.Helper()
	entries, err := os.ReadDir(f.env.Files.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestUploadAttachment(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	a := expectAttachment(t, upload(t, f.admin, key, "../Convocation AG été.pdf", samplePDF(2048)))
	if a.Name != "Convocation AG été.pdf" || a.ContentType != communication.TypePDF || a.SizeBytes != 2048 {
		t.Fatalf("pièce : %+v", a)
	}
	if files := storedFiles(t, f); len(files) != 1 || !communication.ValidStoredName(files[0]) {
		t.Fatalf("fichiers : %v", files)
	}

	exe := append([]byte("MZ\x90\x00"), make([]byte, 100)...)
	expectCmFieldError(t, upload(t, f.admin, key, "facture.pdf", exe), http.StatusUnprocessableEntity, "form.file", "Ce type de fichier n'est pas accepté")
	expectCmFieldError(t, upload(t, f.admin, key, "vide.pdf", nil), http.StatusUnprocessableEntity, "form.file", api.MsgAttachmentEmpty)
	res := upload(t, f.admin, key, "gros.pdf", samplePDF(communication.MaxFileBytes+1))
	if res.Status != http.StatusRequestEntityTooLarge && res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, api.MsgAttachmentSize) {
		t.Errorf("fichier de 5 Mio + 1 : %d %s", res.Status, res.Body)
	}
	if files := storedFiles(t, f); len(files) != 1 {
		t.Errorf("fichiers refusés conservés : %v", files)
	}

	// Total : 4 Mio déjà joints + deux fichiers de 4 Mio → le second dépasse 10 Mio.
	key2 := newDraftKey()
	expectAttachment(t, upload(t, f.admin, key2, "a.pdf", samplePDF(4<<20)))
	expectAttachment(t, upload(t, f.admin, key2, "b.pdf", samplePDF(4<<20)))
	expectCmFieldError(t, upload(t, f.admin, key2, "c.pdf", samplePDF(4<<20)), http.StatusUnprocessableEntity, "form.file",
		"Les pièces jointes dépassent 10 Mo au total (8,0 Mo déjà joints).")

	// Nombre : un sixième fichier est refusé.
	key3 := newDraftKey()
	for i := range communication.MaxFiles {
		expectAttachment(t, upload(t, f.admin, key3, fmt.Sprintf("%d.pdf", i), samplePDF(100)))
	}
	expectCmFieldError(t, upload(t, f.admin, key3, "6.pdf", samplePDF(100)), http.StatusUnprocessableEntity, "form.file", api.MsgAttachmentCount)

	expectCmFieldError(t, upload(t, f.admin, "mauvaise-cle", "a.pdf", samplePDF(100)), http.StatusUnprocessableEntity, "form.draftKey", "")
	if strings.Contains(f.env.Logs.String(), "Convocation") {
		t.Error("nom de fichier dans les journaux")
	}
}

func TestDeleteAttachment(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	a := expectAttachment(t, upload(t, f.admin, key, "a.pdf", samplePDF(100)))
	other := f.env.LoggedIn(f.env.Seed(auth.RoleAdmin, true))
	path := fmt.Sprintf("/api/admin/communications/attachments/%d", a.ID)
	expectProblem(t, other.Do("DELETE", path, nil), http.StatusNotFound, api.MsgAttachmentGone)
	if res := f.admin.Do("DELETE", path, nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	if files := storedFiles(t, f); len(files) != 0 {
		t.Errorf("fichier conservé : %v", files)
	}
	expectProblem(t, f.admin.Do("DELETE", path, nil), http.StatusNotFound, api.MsgAttachmentGone)
}

func TestSendCommunicationWithAttachments(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	pdf := samplePDF(3000)
	a := expectAttachment(t, upload(t, f.admin, key, "Convocation AG été.pdf", pdf))
	// Les pièces d'un autre administrateur ne sont pas emportées.
	other := f.env.LoggedIn(f.env.Seed(auth.RoleAdmin, true))
	expectAttachment(t, upload(t, other, key, "autre.pdf", samplePDF(100)))

	c := expectCommunication(t, f.admin.Do("POST", "/api/admin/communications", sendBody(key, f.season.ID, unpaid, 2)), http.StatusCreated)
	if len(c.Attachments) != 1 || c.Attachments[0].ID != a.ID || c.AttachmentsCount != 1 {
		t.Fatalf("pièces : %+v", c.Attachments)
	}
	expectProblem(t, f.admin.Do("DELETE", fmt.Sprintf("/api/admin/communications/attachments/%d", a.ID), nil), http.StatusNotFound, api.MsgAttachmentGone)
	expectCmFieldError(t, upload(t, f.admin, key, "b.pdf", samplePDF(100)), http.StatusUnprocessableEntity, "form.draftKey", api.MsgCommunicationSent)

	f.env.FlushMail()
	for _, m := range f.env.Mail.Messages() {
		if len(m.Msg.Attachments) != 1 || !bytes.Equal(m.Msg.Attachments[0].Data, pdf) || m.Msg.Attachments[0].Name != "Convocation AG été.pdf" {
			t.Errorf("pièce reçue par %s : %+v", m.To, m.Msg.Attachments)
		}
	}

	res := f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d/attachments/%d", c.ID, a.ID), nil)
	if res.Status != http.StatusOK || res.Body != string(pdf) {
		t.Fatalf("téléchargement : %d (%d octets)", res.Status, len(res.Body))
	}
	for header, want := range map[string]string{
		"Content-Type":           communication.TypePDF,
		"Content-Disposition":    `attachment; filename="Convocation AG _t_.pdf"; filename*=UTF-8''Convocation%20AG%20%C3%A9t%C3%A9.pdf`,
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, no-store",
		"Content-Length":         "3000",
	} {
		if got := res.Header.Get(header); got != want {
			t.Errorf("%s = %q, attendu %q", header, got, want)
		}
	}
	if res := f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d/attachments/%d", c.ID+1, a.ID), nil); res.Status != http.StatusNotFound {
		t.Errorf("pièce d'une autre communication : %d", res.Status)
	}
	var draftID int64
	_ = f.env.Store.ReadDB.QueryRow(`SELECT id FROM communication_attachments WHERE communication_id IS NULL`).Scan(&draftID)
	if res := f.admin.Do("GET", fmt.Sprintf("/api/admin/communications/%d/attachments/%d", c.ID, draftID), nil); res.Status != http.StatusNotFound {
		t.Errorf("pièce de préparation téléchargeable : %d", res.Status)
	}
}

func TestSendCommunicationCorruptedAttachment(t *testing.T) {
	f := newCmFixture(t)
	key := newDraftKey()
	expectAttachment(t, upload(t, f.admin, key, "a.pdf", samplePDF(100)))
	files := storedFiles(t, f)
	if err := os.WriteFile(filepath.Join(f.env.Files.Dir, files[0]), samplePDF(101), 0o640); err != nil {
		t.Fatal(err)
	}
	expectCmFieldError(t, f.admin.Do("POST", "/api/admin/communications", sendBody(key, f.season.ID, unpaid, 2)),
		http.StatusUnprocessableEntity, "body.draftKey", api.MsgCommunicationAttachment)
	if n := countRows(t, f.env, `SELECT count(*) FROM communications`); n != 0 {
		t.Error("communication enregistrée avec une pièce altérée")
	}
}
