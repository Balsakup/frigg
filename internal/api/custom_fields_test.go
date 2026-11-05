package api_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

const customFieldsPath = "/api/admin/custom-fields"

func customFieldPath(id int64, suffix string) string {
	return fmt.Sprintf("%s/%d%s", customFieldsPath, id, suffix)
}

func fieldBody(scope, label, typ string, extra ...any) map[string]any {
	b := map[string]any{"scope": scope, "label": label, "type": typ, "visibility": "admin", "required": false}
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

func expectField(t *testing.T, res apitest.Response, status int) api.FieldDefinition {
	t.Helper()
	if res.Status != status {
		t.Fatalf("statut %d, attendu %d : %s", res.Status, status, res.Body)
	}
	var fd api.FieldDefinition
	res.JSON(t, &fd)
	return fd
}

func createField(t *testing.T, admin *apitest.Client, body map[string]any) api.FieldDefinition {
	t.Helper()
	return expectField(t, admin.Do("POST", customFieldsPath, body), http.StatusCreated)
}

func listFields(t *testing.T, admin *apitest.Client) api.CustomFieldList {
	t.Helper()
	res := admin.Do("GET", customFieldsPath, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste : %d %s", res.Status, res.Body)
	}
	var l api.CustomFieldList
	res.JSON(t, &l)
	return l
}

func fieldIDs(fds []api.FieldDefinition) []int64 {
	out := make([]int64, len(fds))
	for i, fd := range fds {
		out[i] = fd.ID
	}
	return out
}

// updateBody reprend une définition pour update-custom-field.
func updateBody(fd api.FieldDefinition, changes ...any) map[string]any {
	choices := make([]map[string]any, len(fd.Choices))
	for i, c := range fd.Choices {
		choices[i] = map[string]any{"id": c.ID, "label": c.Label}
	}
	b := map[string]any{"label": fd.Label, "type": fd.Type, "visibility": fd.Visibility, "required": fd.Required, "choices": choices}
	if fd.Help != nil {
		b["help"] = *fd.Help
	}
	for i := 0; i+1 < len(changes); i += 2 {
		b[changes[i].(string)] = changes[i+1]
	}
	return b
}

func insertMemberValue(t *testing.T, env *apitest.Env, memberID, fieldID int64, text string) {
	t.Helper()
	p := sqlc.UpsertMemberFieldValueParams{OwnerID: memberID, FieldID: fieldID, Now: "2026-11-01T10:00:00Z"}
	p.ValueText.String, p.ValueText.Valid = text, true
	if err := env.Store.Write.UpsertMemberFieldValue(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

// TestCustomFieldDefinitions couvre la section « Champs personnalisés » (US1).
func TestCustomFieldDefinitions(t *testing.T) {
	f := newMsFixture(t)
	admin := f.admin
	if l := listFields(t, admin); len(l.Member) != 0 || len(l.Membership) != 0 {
		t.Fatalf("liste initiale : %+v", l)
	}

	licence := createField(t, admin, fieldBody("member", "Numéro de licence fédérale", "short_text",
		"visibility", "editable", "help", "  7 chiffres "))
	if licence.Visibility != "editable" || *licence.Help != "7 chiffres" || licence.Archived || licence.ValueCount != 0 {
		t.Fatalf("champ créé : %+v", licence)
	}
	byDefault := createField(t, admin, map[string]any{"scope": "member", "label": "Remarque", "type": "long_text", "visibility": "admin", "required": false})
	if byDefault.Visibility != "admin" {
		t.Fatalf("visibilité : %s", byDefault.Visibility)
	}
	for _, typ := range []string{"number", "date", "bool"} {
		createField(t, admin, fieldBody("member", "Champ "+typ, typ))
	}
	size := createField(t, admin, fieldBody("membership", "Taille de t-shirt", "choice", "choices", []string{"S", "M", "L"}, "visibility", "editable", "required", true))
	paid := createField(t, admin, fieldBody("membership", "A payé sa licence fédérale", "bool"))
	if len(size.Choices) != 3 || size.Choices[1].Label != "M" {
		t.Fatalf("choix : %+v", size.Choices)
	}
	l := listFields(t, admin)
	if len(l.Member) != 5 || len(l.Membership) != 2 || l.Member[0].ID != licence.ID || l.Membership[1].ID != paid.ID {
		t.Fatalf("ordre : %v %v", fieldIDs(l.Member), fieldIDs(l.Membership))
	}

	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "numero de LICENCE federale", "date")), "body.label", "Un champ porte déjà ce libellé.")
	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "Liste", "choice", "choices", []string{"A"})), "body.choices", "Indiquez de 2 à 50 choix.")
	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "Liste", "choice", "choices", []string{"A", "a"})), "body.choices[1]", "Ce choix figure déjà dans la liste.")
	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "Liste", "choice", "choices", []string{"A", " "})), "body.choices[1]", "Indiquez le libellé du choix.")
	many := make([]string, 51)
	for i := range many {
		many[i] = fmt.Sprintf("C%d", i)
	}
	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "Liste", "choice", "choices", many)), "body.choices", "Indiquez de 2 à 50 choix.")
	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "Coché", "bool", "required", true)), "body.required", "Un champ Oui/non ne peut pas être obligatoire.")

	// Modification : libellé, aide, visibilité, obligatoire, choix.
	size = expectField(t, admin.Do("PUT", customFieldPath(size.ID, ""), updateBody(size,
		"label", "Taille du t-shirt", "visibility", "visible",
		"choices", []map[string]any{{"id": size.Choices[0].ID, "label": "Small"}, {"id": size.Choices[1].ID, "label": "M"}, {"id": size.Choices[2].ID, "label": "L"}, {"label": "XL"}},
	)), http.StatusOK)
	if size.Label != "Taille du t-shirt" || size.Visibility != "visible" || len(size.Choices) != 4 || size.Choices[0].Label != "Small" || size.Choices[3].Label != "XL" {
		t.Fatalf("modification : %+v", size)
	}
	// Choix utilisé : non retirable ; choix inutilisé : retirable.
	c, _, m := f.member(t, "Lucas")
	ms := request(t, c, m.ID, f.season.ID, f.adult.ID, "first")
	p := sqlc.UpsertMembershipFieldValueParams{OwnerID: ms.ID, FieldID: size.ID, Now: "2026-11-01T10:00:00Z"}
	p.ValueChoiceID.Int64, p.ValueChoiceID.Valid = size.Choices[2].ID, true
	if err := f.env.Store.Write.UpsertMembershipFieldValue(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	expectFieldError(t, admin.Do("PUT", customFieldPath(size.ID, ""), updateBody(size, "choices", []map[string]any{
		{"id": size.Choices[0].ID, "label": "Small"}, {"id": size.Choices[1].ID, "label": "M"},
	})), "body.choices", "Le choix « L » est utilisé : renommez-le plutôt que de le retirer.")
	size = expectField(t, admin.Do("PUT", customFieldPath(size.ID, ""), updateBody(size, "choices", []map[string]any{
		{"id": size.Choices[0].ID, "label": "Small"}, {"id": size.Choices[1].ID, "label": "M"}, {"id": size.Choices[2].ID, "label": "Large"},
	})), http.StatusOK)
	if len(size.Choices) != 3 || size.Choices[2].Label != "Large" {
		t.Fatalf("choix inutilisé retiré, choix utilisé renommé : %+v", size.Choices)
	}

	// Type : modifiable sans valeur, figé dès qu'une valeur existe.
	num := listFields(t, admin).Member[2]
	num = expectField(t, admin.Do("PUT", customFieldPath(num.ID, ""), updateBody(num, "type", "short_text")), http.StatusOK)
	insertMemberValue(t, f.env, m.ID, num.ID, "abc")
	if res := admin.Do("PUT", customFieldPath(num.ID, ""), updateBody(num, "type", "date")); res.Status != http.StatusConflict {
		t.Fatalf("changement de type avec valeurs : %d %s", res.Status, res.Body)
	}
	if fd := listFields(t, admin).Member[2]; fd.ValueCount != 1 {
		t.Fatalf("valueCount : %d", fd.ValueCount)
	}

	// Ordre.
	l = listFields(t, admin)
	ids := fieldIDs(l.Member)
	reversed := []int64{ids[4], ids[3], ids[2], ids[1], ids[0]}
	res := admin.Do("PUT", customFieldsPath+"/order", map[string]any{"scope": "member", "ids": reversed})
	if res.Status != http.StatusOK {
		t.Fatalf("ordre : %d %s", res.Status, res.Body)
	}
	if got := fieldIDs(listFields(t, admin).Member); fmt.Sprint(got) != fmt.Sprint(reversed) {
		t.Fatalf("nouvel ordre : %v", got)
	}
	for name, bad := range map[string][]int64{
		"incomplète":      reversed[1:],
		"autre portée":    append(reversed[1:], paid.ID),
		"identifiant fou": append(reversed[1:], 9999),
	} {
		if res := admin.Do("PUT", customFieldsPath+"/order", map[string]any{"scope": "member", "ids": bad}); res.Status != http.StatusConflict {
			t.Errorf("ordre %s : %d", name, res.Status)
		}
	}

	// Archivage et rétablissement : valeurs conservées, champ en fin de liste.
	fd := expectField(t, admin.Do("POST", customFieldPath(num.ID, "/archive"), nil), http.StatusOK)
	if !fd.Archived {
		t.Fatal("champ non archivé")
	}
	if res := admin.Do("POST", customFieldPath(num.ID, "/archive"), nil); res.Status != http.StatusConflict {
		t.Errorf("double archivage : %d", res.Status)
	}
	if res := admin.Do("POST", customFieldPath(licence.ID, "/restore"), nil); res.Status != http.StatusConflict {
		t.Errorf("rétablissement d'un actif : %d", res.Status)
	}
	l = listFields(t, admin)
	if last := l.Member[len(l.Member)-1]; last.ID != num.ID || !last.Archived {
		t.Fatalf("archivés en fin de liste : %v", fieldIDs(l.Member))
	}
	if res := admin.Do("PUT", customFieldsPath+"/order", map[string]any{"scope": "member", "ids": fieldIDs(l.Member)}); res.Status != http.StatusConflict {
		t.Errorf("ordre avec un archivé : %d", res.Status)
	}
	fd = expectField(t, admin.Do("POST", customFieldPath(num.ID, "/restore"), nil), http.StatusOK)
	l = listFields(t, admin)
	if fd.Archived || fd.ValueCount != 1 || l.Member[len(l.Member)-1].ID != num.ID {
		t.Fatalf("rétablissement : %+v %v", fd, fieldIDs(l.Member))
	}
	if res := admin.Do("POST", customFieldPath(999, "/archive"), nil); res.Status != http.StatusNotFound {
		t.Errorf("champ inconnu : %d", res.Status)
	}

	// Journaux sans libellé.
	logs := f.env.Logs.String()
	for _, action := range []string{"custom_field.create", "custom_field.update", "custom_field.reorder", "custom_field.archive", "custom_field.restore"} {
		if !strings.Contains(logs, action) {
			t.Errorf("journal %s absent", action)
		}
	}
	if strings.Contains(logs, "licence") || strings.Contains(logs, "t-shirt") {
		t.Error("libellé de champ dans les journaux")
	}
}

// TestCustomFieldLimitAndCounters couvre la limite de 30 champs actifs et les
// compteurs de fiches et d'adhésions sans valeur (FR-010, FR-026).
func TestCustomFieldLimitAndCounters(t *testing.T) {
	f := newMsFixture(t)
	admin := f.admin
	var last api.FieldDefinition
	for i := range 30 {
		last = createField(t, admin, fieldBody("member", fmt.Sprintf("Champ %d", i), "short_text"))
	}
	expectFieldError(t, admin.Do("POST", customFieldsPath, fieldBody("member", "Champ 31", "short_text")), "body.scope", api.MsgCustomFieldsFull)
	createField(t, admin, fieldBody("membership", "Autre portée", "short_text"))
	expectField(t, admin.Do("POST", customFieldPath(last.ID, "/archive"), nil), http.StatusOK)
	createField(t, admin, fieldBody("member", "Champ 31", "short_text"))
	if res := admin.Do("POST", customFieldPath(last.ID, "/restore"), nil); res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("rétablissement au-delà de 30 : %d %s", res.Status, res.Body)
	}

	// Compteurs.
	f2 := newMsFixture(t)
	c, _, lucas := f2.member(t, "Lucas")
	f2.member(t, "Léa")
	required := createField(t, f2.admin, fieldBody("member", "Licence", "short_text", "required", true))
	insertMemberValue(t, f2.env, lucas.ID, required.ID, "123")
	msField := createField(t, f2.admin, fieldBody("membership", "Certificat", "date", "required", true))
	request(t, c, lucas.ID, f2.season.ID, f2.adult.ID, "first")
	l := listFields(t, f2.admin)
	if l.Member[0].MissingCount == nil || *l.Member[0].MissingCount != 1 {
		t.Fatalf("fiches sans valeur : %v", l.Member[0].MissingCount)
	}
	if l.Membership[0].ID != msField.ID || l.Membership[0].MissingCount == nil || *l.Membership[0].MissingCount != 1 {
		t.Fatalf("adhésions sans valeur : %+v", l.Membership[0])
	}
	f2.env.SetToday("2030-01-01")
	if l := listFields(t, f2.admin); l.Membership[0].MissingCount != nil {
		t.Fatalf("sans saison en cours : %v", *l.Membership[0].MissingCount)
	}
}

// TestMyCustomFields vérifie que l'espace membre ne reçoit que les champs
// actifs visibles ou remplis par le membre.
func TestMyCustomFields(t *testing.T) {
	f := newMsFixture(t)
	editable := createField(t, f.admin, fieldBody("member", "Licence", "short_text", "visibility", "editable"))
	visible := createField(t, f.admin, fieldBody("member", "Groupe", "short_text", "visibility", "visible"))
	createField(t, f.admin, fieldBody("member", "Remarque", "short_text"))
	archived := createField(t, f.admin, fieldBody("member", "Ancien", "short_text", "visibility", "editable"))
	expectField(t, f.admin.Do("POST", customFieldPath(archived.ID, "/archive"), nil), http.StatusOK)
	size := createField(t, f.admin, fieldBody("membership", "Taille", "choice", "visibility", "editable", "choices", []string{"S", "M"}))

	c, _, _ := f.member(t, "Lucas")
	res := c.Do("GET", "/api/me/custom-fields", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("%d %s", res.Status, res.Body)
	}
	var l api.MyCustomFieldList
	res.JSON(t, &l)
	if len(l.Member) != 2 || l.Member[0].ID != editable.ID || !l.Member[0].Editable || l.Member[1].ID != visible.ID || l.Member[1].Editable {
		t.Fatalf("champs de membre : %+v", l.Member)
	}
	if len(l.Membership) != 1 || l.Membership[0].ID != size.ID || len(l.Membership[0].Choices) != 2 {
		t.Fatalf("champs d'adhésion : %+v", l.Membership)
	}
	if strings.Contains(res.Body, "Remarque") || strings.Contains(res.Body, "Ancien") {
		t.Fatalf("champ réservé ou archivé transmis : %s", res.Body)
	}
}

// TestCustomFieldDelete couvre la suppression définitive d'un champ (US5).
func TestCustomFieldDelete(t *testing.T) {
	f := newMsFixture(t)
	admin := f.admin
	licence := createField(t, admin, fieldBody("member", "Licence", "short_text"))
	size := createField(t, admin, fieldBody("membership", "Taille", "choice", "choices", []string{"S", "M"}, "visibility", "editable"))
	old := createField(t, admin, fieldBody("member", "Ancien", "date"))
	c, _, m := f.member(t, "Lucas")
	insertMemberValue(t, f.env, m.ID, licence.ID, "1234567")
	expectMine(t, c.Do("POST", "/api/me/memberships", map[string]any{
		"memberId": m.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "declaredKind": "first",
		"customFields": []map[string]any{{"fieldId": size.ID, "choiceId": size.Choices[0].ID}},
	}), http.StatusCreated)
	if l := listFields(t, admin); l.Member[0].ValueCount != 1 || l.Membership[0].ValueCount != 1 {
		t.Fatalf("compteurs avant suppression : %+v", l)
	}

	for _, id := range []int64{licence.ID, size.ID} {
		if res := admin.Do("DELETE", customFieldPath(id, ""), nil); res.Status != http.StatusNoContent {
			t.Fatalf("suppression de %d : %d %s", id, res.Status, res.Body)
		}
	}
	expectField(t, admin.Do("POST", customFieldPath(old.ID, "/archive"), nil), http.StatusOK)
	if res := admin.Do("DELETE", customFieldPath(old.ID, ""), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression d'un champ archivé : %d", res.Status)
	}
	if res := admin.Do("DELETE", customFieldPath(licence.ID, ""), nil); res.Status != http.StatusNotFound {
		t.Fatalf("champ déjà supprimé : %d", res.Status)
	}
	if l := listFields(t, admin); len(l.Member) != 0 || len(l.Membership) != 0 {
		t.Fatalf("champs restants : %+v", l)
	}
	if got := expectMember(t, admin.Do("GET", adminMemberPath(m.ID), nil), http.StatusOK); len(got.CustomFields) != 0 {
		t.Fatalf("champs sur la fiche : %+v", got.CustomFields)
	}
	var n int
	if err := f.env.Store.WriteDB.QueryRow(`SELECT (SELECT count(*) FROM member_field_values) + (SELECT count(*) FROM membership_field_values) + (SELECT count(*) FROM custom_field_choices)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("lignes restantes : %d (%v)", n, err)
	}
	if _, err := f.env.Store.WriteDB.Exec(`INSERT INTO member_field_values_fts (member_field_values_fts) VALUES ('integrity-check')`); err != nil {
		t.Fatalf("index FTS : %v", err)
	}
	if !strings.Contains(f.env.Logs.String(), "custom_field.delete") {
		t.Error("journal de suppression absent")
	}
}
