package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func cfText(id int64, text string) map[string]any { return map[string]any{"fieldId": id, "text": text} }

func findValue(t *testing.T, fields []api.FieldValue, id int64) api.FieldValue {
	t.Helper()
	for _, f := range fields {
		if f.ID == id {
			return f
		}
	}
	t.Fatalf("champ %d absent de %+v", id, fields)
	return api.FieldValue{}
}

func hasFieldID(fields []api.FieldValue, id int64) bool {
	for _, f := range fields {
		if f.ID == id {
			return true
		}
	}
	return false
}

// adminMemberUpdate est le corps de update-member pour une fiche rattachée à accountID.
func adminMemberUpdate(m api.Member, accountID *int64, extra ...any) map[string]any {
	b := memberBody(m.FirstName, m.LastName, m.BirthDate, "accountId", accountID, "expectedAccountId", accountID)
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

type memberFieldsFixture struct {
	env                               *apitest.Env
	admin, owner                      *apitest.Client
	ownerID                           int64
	licence, group, note, n, d, b, ch api.FieldDefinition
}

func newMemberFieldsFixture(t *testing.T) memberFieldsFixture {
	t.Helper()
	env := apitest.New(t)
	env.SetToday(membersToday)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	email := env.Seed(auth.RoleMember, true)
	owner := env.LoggedIn(email)
	var me api.Session
	owner.Do("GET", "/api/session", nil).JSON(t, &me)
	f := memberFieldsFixture{env: env, admin: admin, owner: owner, ownerID: me.Account.ID}
	f.licence = createField(t, admin, fieldBody("member", "Numéro de licence fédérale", "short_text", "visibility", "editable"))
	f.group = createField(t, admin, fieldBody("member", "Groupe", "short_text", "visibility", "visible"))
	f.note = createField(t, admin, fieldBody("member", "Remarque du bureau", "long_text"))
	f.n = createField(t, admin, fieldBody("member", "Pointure", "number"))
	f.d = createField(t, admin, fieldBody("member", "Certificat médical", "date"))
	f.b = createField(t, admin, fieldBody("member", "Bénévole", "bool"))
	f.ch = createField(t, admin, fieldBody("member", "Niveau", "choice", "choices", []string{"Débutant", "Confirmé"}))
	return f
}

// TestMemberCustomFields couvre la saisie des champs de membre (US2).
func TestMemberCustomFields(t *testing.T) {
	f := newMemberFieldsFixture(t)
	admin, owner := f.admin, f.owner

	// Administration : chaque type, relu dans l'ordre, non renseignés compris.
	created := expectMember(t, admin.Do("POST", "/api/admin/members", memberBody("Lucas", "Moreau", "2012-05-20",
		"accountId", f.ownerID,
		"customFields", []map[string]any{
			cfText(f.licence.ID, "1234567"),
			cfText(f.note.ID, "Allergie\naux arachides"),
			cfText(f.n.ID, "42,5"),
			{"fieldId": f.d.ID, "date": "2026-09-15"},
			{"fieldId": f.b.ID, "checked": true},
			{"fieldId": f.ch.ID, "choiceId": f.ch.Choices[1].ID},
		},
	)), http.StatusCreated)
	if len(created.CustomFields) != 7 || created.CustomFields[0].ID != f.licence.ID {
		t.Fatalf("champs : %+v", created.CustomFields)
	}
	if v := findValue(t, created.CustomFields, f.n.ID); v.Number == nil || *v.Number != 42.5 {
		t.Errorf("nombre : %+v", v)
	}
	if v := findValue(t, created.CustomFields, f.note.ID); v.Text == nil || *v.Text != "Allergie\naux arachides" {
		t.Errorf("texte long : %+v", v)
	}
	if v := findValue(t, created.CustomFields, f.ch.ID); v.ChoiceID == nil || *v.ChoiceID != f.ch.Choices[1].ID {
		t.Errorf("choix : %+v", v)
	}
	if v := findValue(t, created.CustomFields, f.group.ID); v.Text != nil || !v.Editable {
		t.Errorf("non renseigné : %+v", v)
	}
	id := created.ID

	// Erreurs groupées avec celles de la fiche.
	res := admin.Do("PUT", adminMemberPath(id), adminMemberUpdate(created.Member, &f.ownerID,
		"firstName", "", "customFields", []map[string]any{cfText(f.n.ID, "beaucoup")}))
	expectFieldError(t, res, fmt.Sprintf("body.customFields.f%d", f.n.ID), "Saisissez un nombre.")
	expectFieldError(t, res, "body.firstName", "Saisissez le prénom.")

	// Espace membre : visible en lecture seule, rempli par le membre modifiable,
	// réservé absent.
	mine := expectMember(t, owner.Do("GET", myMemberPath(id), nil), http.StatusOK).Member
	if hasFieldID(mine.CustomFields, f.note.ID) || len(mine.CustomFields) != 2 {
		t.Fatalf("champs du membre : %+v", mine.CustomFields)
	}
	if v := findValue(t, mine.CustomFields, f.group.ID); v.Editable {
		t.Error("champ visible modifiable par le membre")
	}
	if v := findValue(t, mine.CustomFields, f.licence.ID); !v.Editable || *v.Text != "1234567" {
		t.Errorf("licence : %+v", v)
	}
	mine = expectMember(t, owner.Do("PUT", myMemberPath(id), memberBody("Lucas", "Moreau", "2012-05-20",
		"customFields", []map[string]any{cfText(f.licence.ID, "7654321")})), http.StatusOK).Member
	if v := findValue(t, mine.CustomFields, f.licence.ID); *v.Text != "7654321" {
		t.Errorf("licence modifiée : %+v", v)
	}
	for _, target := range []int64{f.group.ID, f.note.ID, 99999} {
		res := owner.Do("PUT", myMemberPath(id), memberBody("Lucas", "Moreau", "2012-05-20",
			"customFields", []map[string]any{cfText(target, "x")}))
		expectFieldError(t, res, fmt.Sprintf("body.customFields.f%d", target), "Ce champ n'existe pas ou n'est pas modifiable.")
	}

	// Un enregistrement du membre sans customFields ne touche à rien.
	expectMember(t, owner.Do("PUT", myMemberPath(id), memberBody("Lucas", "Moreau", "2012-05-20", "phone", "0612345678")), http.StatusOK)
	adminView := expectMember(t, admin.Do("GET", adminMemberPath(id), nil), http.StatusOK)
	if v := findValue(t, adminView.CustomFields, f.note.ID); v.Text == nil || *v.Text != "Allergie\naux arachides" {
		t.Fatalf("valeur réservée effacée : %+v", v)
	}

	// Champ archivé : refusé à l'écriture, absent des fiches.
	expectField(t, admin.Do("POST", customFieldPath(f.d.ID, "/archive"), nil), http.StatusOK)
	expectFieldError(t, admin.Do("PUT", adminMemberPath(id), adminMemberUpdate(adminView.Member, &f.ownerID,
		"customFields", []map[string]any{{"fieldId": f.d.ID, "date": "2026-01-01"}})),
		fmt.Sprintf("body.customFields.f%d", f.d.ID), "Ce champ n'existe pas ou n'est pas modifiable.")
	if adminView = expectMember(t, admin.Do("GET", adminMemberPath(id), nil), http.StatusOK); hasFieldID(adminView.CustomFields, f.d.ID) {
		t.Error("champ archivé affiché")
	}

	// Effacement : texte vide et case décochée.
	adminView = expectMember(t, admin.Do("PUT", adminMemberPath(id), adminMemberUpdate(adminView.Member, &f.ownerID,
		"customFields", []map[string]any{cfText(f.note.ID, ""), {"fieldId": f.b.ID, "checked": false}})), http.StatusOK)
	if v := findValue(t, adminView.CustomFields, f.note.ID); v.Text != nil {
		t.Errorf("texte non effacé : %+v", v)
	}
	if v := findValue(t, adminView.CustomFields, f.b.ID); v.Checked == nil || *v.Checked {
		t.Errorf("case non décochée : %+v", v)
	}

	// Journal sans valeur.
	logs := f.env.Logs.String()
	if !strings.Contains(logs, `"custom_fields":`) {
		t.Error("nombre de champs modifiés absent du journal")
	}
	for _, secret := range []string{"1234567", "7654321", "arachides"} {
		if strings.Contains(logs, secret) {
			t.Errorf("valeur %q dans les journaux", secret)
		}
	}
}

// TestMemberCustomFieldsRequired couvre les champs obligatoires (FR-015).
func TestMemberCustomFieldsRequired(t *testing.T) {
	f := newMemberFieldsFixture(t)
	admin, owner := f.admin, f.owner
	m := createMine(t, owner, memberBody("Lucas", "Moreau", "2012-05-20"))

	// Obligatoire réservé : ne bloque pas le membre, bloque l'administration.
	expectField(t, admin.Do("PUT", customFieldPath(f.note.ID, ""), updateBody(f.note, "required", true)), http.StatusOK)
	expectMember(t, owner.Do("PUT", myMemberPath(m.ID), memberBody("Lucas", "Moreau", "2012-05-20", "phone", "0612345678")), http.StatusOK)
	expectFieldError(t, admin.Do("PUT", adminMemberPath(m.ID), adminMemberUpdate(m, &f.ownerID)),
		fmt.Sprintf("body.customFields.f%d", f.note.ID), "Ce champ est obligatoire.")

	// Obligatoire rempli par le membre : exigé à la création.
	expectField(t, admin.Do("PUT", customFieldPath(f.licence.ID, ""), updateBody(f.licence, "required", true)), http.StatusOK)
	expectFieldError(t, owner.Do("POST", "/api/me/members", memberBody("Emma", "Moreau", "2014-01-01")),
		fmt.Sprintf("body.customFields.f%d", f.licence.ID), "Ce champ est obligatoire.")
	created := createMine(t, owner, memberBody("Emma", "Moreau", "2014-01-01",
		"customFields", []map[string]any{cfText(f.licence.ID, "1111111")}))
	if v := findValue(t, created.CustomFields, f.licence.ID); v.Text == nil || *v.Text != "1111111" {
		t.Fatalf("création : %+v", created.CustomFields)
	}

	// Suppression de la fiche : ses valeurs partent avec elle.
	if res := owner.Do("DELETE", myMemberPath(created.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	if l := listFields(t, admin); l.Member[0].ValueCount != 0 {
		t.Fatalf("valeurs restantes : %d", l.Member[0].ValueCount)
	}
}
