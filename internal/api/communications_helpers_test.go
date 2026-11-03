package api_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

// cmFixture est le jeu de données de quickstart.md › validation manuelle :
// saison 2026-2027 ouverte, tarifs « Jeune » et « Adulte », et les adhérents
// Léo et Inès (compte Claire, à payer), Paul (annoncé), Anne (sans compte,
// payée), Hugo (sans compte ni adresse, à payer), Zoé (gratuite) et Marc
// (demande en attente).
type cmFixture struct {
	env          *apitest.Env
	admin        *apitest.Client
	season       api.Season
	young, adult api.MembershipFee
	claire, paul string
	members      map[string]int64
}

const cmToday = "2026-11-03"

func newCmFixture(t *testing.T) cmFixture {
	t.Helper()
	env := apitest.New(t)
	env.SetToday(cmToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	if _, err := env.Store.WriteDB.Exec(`UPDATE association SET name = 'Club de Frigg' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	s := createSeason(t, admin, "2026-09-01", "2027-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(s.ID, "/open"), nil), http.StatusOK)
	f := cmFixture{env: env, admin: admin, season: s, members: map[string]int64{}}
	f.young = createFee(t, admin, s.ID, map[string]string{"label": "Jeune", "amount": "30"})
	f.adult = createFee(t, admin, s.ID, map[string]string{"label": "Adulte", "amount": "60"})

	f.claire = env.Seed(auth.RoleMember, true)
	f.paul = env.Seed(auth.RoleMember, true)
	zoe := env.Seed(auth.RoleMember, true)
	marc := env.Seed(auth.RoleMember, true)
	type row struct {
		first, last, account, email string
		fee                         int64
		status, payment             string
		amount                      int64
	}
	for _, r := range []row{
		{"Léo", "Martin", f.claire, "", f.young.ID, "validated", "due", 3000},
		{"Inès", "Martin", f.claire, "ines@exemple.fr", f.young.ID, "validated", "due", 3000},
		{"Paul", "Durand", f.paul, "", f.adult.ID, "validated", "announced", 6000},
		{"Anne", "Petit", "", "anne@exemple.fr", f.adult.ID, "validated", "paid", 6000},
		{"Hugo", "Roux", "", "", f.young.ID, "validated", "due", 3000},
		{"Zoé", "Blanc", zoe, "", f.young.ID, "validated", "", 0},
		{"Marc", "Noir", marc, "", f.adult.ID, "pending", "", 0},
	} {
		var account any
		if r.account != "" {
			account = accountID(t, env, r.account)
		}
		f.members[r.first] = f.insertMembership(t, r.first, r.last, account, r.email, r.fee, r.status, r.payment, r.amount)
	}
	return f
}

var cmSeq int

// insertMembership crée une fiche et son adhésion directement en base, dans
// l'état de paiement voulu.
func (f cmFixture) insertMembership(t *testing.T, first, last string, account any, email string, fee int64, status, payment string, amount int64) int64 {
	t.Helper()
	db := f.env.Store.WriteDB
	const ts = "2026-09-15T10:00:00Z"
	var mail any
	if email != "" {
		mail = email
	}
	res, err := db.Exec(`INSERT INTO members (first_name, last_name, birth_date, email, account_id, created_at, updated_at)
		VALUES (?, ?, '2000-01-01', ?, ?, ?, ?)`, first, last, mail, account, ts, ts)
	if err != nil {
		t.Fatal(err)
	}
	memberID, _ := res.LastInsertId()
	cmSeq++
	var kind, amt, ref, pay, announced, paidMethod, paidOn any
	if status == "validated" {
		kind, amt = "first", amount
		if payment != "" {
			ref, pay = fmt.Sprintf("ADH-2627-%04d", cmSeq), payment
		}
		switch payment {
		case "announced":
			announced = "cheque"
		case "paid":
			paidMethod, paidOn = "cheque", "2026-09-20"
		}
	}
	if _, err := db.Exec(`INSERT INTO memberships (member_id, season_id, fee_id, status, kind, amount_cents, requested_at,
		payment_reference, payment_status, announced_method, paid_method, paid_on, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		memberID, f.season.ID, fee, status, kind, amt, ts, ref, pay, announced, paidMethod, paidOn, ts); err != nil {
		t.Fatal(err)
	}
	return memberID
}

func audiencePath(seasonID int64, situations []string, feeIDs ...int64) string {
	v := url.Values{}
	v.Set("seasonId", fmt.Sprint(seasonID))
	for _, s := range situations {
		v.Add("situations", s)
	}
	for _, id := range feeIDs {
		v.Add("feeIds", fmt.Sprint(id))
	}
	return "/api/admin/communications/audience?" + v.Encode()
}

var unpaid = []string{"announced", "due"}

func (f cmFixture) audience(t *testing.T, situations []string, feeIDs ...int64) api.Audience {
	t.Helper()
	res := f.admin.Do("GET", audiencePath(f.season.ID, situations, feeIDs...), nil)
	if res.Status != http.StatusOK {
		t.Fatalf("audience : %d %s", res.Status, res.Body)
	}
	var a api.Audience
	res.JSON(t, &a)
	return a
}

// newDraftKey génère une clé de préparation (UUID v4).
func newDraftKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func sendBody(key string, seasonID int64, situations []string, expected int) map[string]any {
	return map[string]any{
		"draftKey":           key,
		"target":             map[string]any{"seasonId": seasonID, "situations": situations, "feeIds": []int64{}},
		"subject":            "Rappel : cotisation 2026-2027",
		"body":               "Bonjour,\n\nMerci de régler votre cotisation.",
		"expectedRecipients": expected,
	}
}

func expectCommunication(t *testing.T, res apitest.Response, status int) api.Communication {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var c api.Communication
	res.JSON(t, &c)
	return c
}

func expectCmFieldError(t *testing.T, res apitest.Response, status int, location, message string) {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var p apitest.Problem
	res.JSON(t, &p)
	for _, e := range p.Errors {
		if e.Location == location && (message == "" || strings.HasPrefix(e.Message, message)) {
			return
		}
	}
	t.Fatalf("erreur %s %q absente : %s", location, message, res.Body)
}

// upload dépose un fichier pour une préparation.
func upload(t *testing.T, c *apitest.Client, key, name string, data []byte) apitest.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("draftKey", key)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	return c.Do("POST", "/api/admin/communications/attachments", apitest.Raw(buf.Bytes()), "Content-Type", w.FormDataContentType())
}

func expectAttachment(t *testing.T, res apitest.Response) api.Attachment {
	t.Helper()
	if res.Status != http.StatusCreated {
		t.Fatalf("dépôt : %d %s", res.Status, res.Body)
	}
	var a api.Attachment
	res.JSON(t, &a)
	return a
}

// samplePDF est un PDF minimal de n octets.
func samplePDF(n int) []byte {
	head := []byte("%PDF-1.7\n")
	if n < len(head) {
		n = len(head)
	}
	return append(head, bytes.Repeat([]byte("x"), n-len(head))...)
}
