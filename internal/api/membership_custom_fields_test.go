package api_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/balsakup/frigg/internal/api"
)

func cfError(id int64) string { return fmt.Sprintf("body.customFields.f%d", id) }

// TestMembershipCustomFields couvre la saisie des champs d'adhésion (US3).
func TestMembershipCustomFields(t *testing.T) {
	f := newMsFixture(t)
	admin := f.admin
	size := createField(t, admin, fieldBody("membership", "Taille de t-shirt", "choice",
		"choices", []string{"S", "M", "L"}, "visibility", "editable", "required", true))
	paid := createField(t, admin, fieldBody("membership", "A payé sa licence fédérale", "bool"))
	group := createField(t, admin, fieldBody("membership", "Groupe", "short_text", "visibility", "visible"))
	certificate := createField(t, admin, fieldBody("membership", "Certificat", "date"))
	c, _, m := f.member(t, "Lucas")
	body := func(extra ...any) map[string]any {
		b := map[string]any{"memberId": m.ID, "seasonId": f.season.ID, "feeId": f.adult.ID, "declaredKind": "first"}
		for i := 0; i+1 < len(extra); i += 2 {
			b[extra[i].(string)] = extra[i+1]
		}
		return b
	}

	// Demande : champ rempli par le membre exigé, champ réservé refusé.
	expectFieldError(t, c.Do("POST", "/api/me/memberships", body()), cfError(size.ID), "Ce champ est obligatoire.")
	expectFieldError(t, c.Do("POST", "/api/me/memberships", body("customFields", []map[string]any{
		{"fieldId": size.ID, "choiceId": size.Choices[1].ID}, {"fieldId": paid.ID, "checked": true},
	})), cfError(paid.ID), "Ce champ n'existe pas ou n'est pas modifiable.")
	mine := expectMine(t, c.Do("POST", "/api/me/memberships", body("customFields", []map[string]any{
		{"fieldId": size.ID, "choiceId": size.Choices[1].ID},
	})), http.StatusCreated)
	if len(mine.CustomFields) != 2 || hasFieldID(mine.CustomFields, paid.ID) || hasFieldID(mine.CustomFields, certificate.ID) {
		t.Fatalf("champs de l'espace membre : %+v", mine.CustomFields)
	}
	if v := findValue(t, mine.CustomFields, size.ID); v.ChoiceID == nil || *v.ChoiceID != size.Choices[1].ID || !v.Editable {
		t.Fatalf("taille : %+v", v)
	}

	// Administration : tous les champs ; obligatoire réservé exigé à la validation.
	got := expectAdminMembership(t, admin.Do("GET", adminMembershipPath(mine.ID, ""), nil), http.StatusOK)
	if len(got.CustomFields) != 4 {
		t.Fatalf("champs de l'administration : %+v", got.CustomFields)
	}
	expectField(t, admin.Do("PUT", customFieldPath(certificate.ID, ""), updateBody(certificate, "required", true)), http.StatusOK)
	expectFieldError(t, admin.Do("POST", adminMembershipPath(mine.ID, "/validate"), map[string]any{"feeId": f.adult.ID, "kind": "first"}),
		cfError(certificate.ID), "Ce champ est obligatoire.")
	got = expectAdminMembership(t, admin.Do("POST", adminMembershipPath(mine.ID, "/validate"), map[string]any{
		"feeId": f.adult.ID, "kind": "first",
		"customFields": []map[string]any{{"fieldId": certificate.ID, "date": "2026-10-01"}, {"fieldId": paid.ID, "checked": true}},
	}), http.StatusOK)
	if got.Status != "validated" || !*findValue(t, got.CustomFields, paid.ID).Checked {
		t.Fatalf("validation : %s %+v", got.Status, got.CustomFields)
	}

	// Adhésion payée : champs toujours modifiables, état et paiement inchangés.
	setPayment(t, admin, nil)
	expectAdminMembership(t, admin.Do("POST", adminMembershipPath(mine.ID, "/payment"), map[string]any{"method": "cash", "receivedOn": msToday}), http.StatusOK)
	got = expectAdminMembership(t, admin.Do("PUT", adminMembershipPath(mine.ID, "/fields"), map[string]any{
		"customFields": []map[string]any{{"fieldId": paid.ID, "checked": false}, {"fieldId": group.ID, "text": "Groupe A"}},
	}), http.StatusOK)
	if got.Status != "validated" || got.Payment == nil || got.Payment.Status != "paid" || got.AmountCents != 6000 {
		t.Fatalf("état ou paiement modifié : %+v", got.Membership)
	}
	if *findValue(t, got.CustomFields, paid.ID).Checked || *findValue(t, got.CustomFields, group.ID).Text != "Groupe A" {
		t.Fatalf("champs : %+v", got.CustomFields)
	}
	expectFieldError(t, admin.Do("PUT", adminMembershipPath(mine.ID, "/fields"), map[string]any{
		"customFields": []map[string]any{{"fieldId": size.ID, "choiceId": 99999}},
	}), cfError(size.ID), "Choix inconnu.")

	// Espace membre : visible en lecture, rempli par le membre modifiable.
	mine = expectMine(t, c.Do("PUT", myMembershipPath(mine.ID)+"/fields", map[string]any{
		"customFields": []map[string]any{{"fieldId": size.ID, "choiceId": size.Choices[2].ID}},
	}), http.StatusOK)
	if v := findValue(t, mine.CustomFields, group.ID); v.Editable || *v.Text != "Groupe A" {
		t.Fatalf("champ visible : %+v", v)
	}
	expectFieldError(t, c.Do("PUT", myMembershipPath(mine.ID)+"/fields", map[string]any{
		"customFields": []map[string]any{{"fieldId": group.ID, "text": "B"}},
	}), cfError(group.ID), "Ce champ n'existe pas ou n'est pas modifiable.")
	other := f.env.LoggedIn(f.env.Seed("member", true))
	expectProblem(t, other.Do("PUT", myMembershipPath(mine.ID)+"/fields", map[string]any{"customFields": []map[string]any{}}),
		http.StatusNotFound, api.MsgMembershipGone)

	// Création directe par l'administration : obligatoires exigés.
	_, _, lea := f.member(t, "Léa")
	direct := map[string]any{"memberId": lea.ID, "seasonId": f.season.ID, "feeId": f.young.ID, "kind": "first"}
	expectFieldError(t, admin.Do("POST", "/api/admin/memberships", direct), cfError(certificate.ID), "Ce champ est obligatoire.")
	direct["customFields"] = []map[string]any{
		{"fieldId": certificate.ID, "date": "2026-10-02"}, {"fieldId": size.ID, "choiceId": size.Choices[0].ID},
	}
	created := expectAdminMembership(t, admin.Do("POST", "/api/admin/memberships", direct), http.StatusCreated)
	if *findValue(t, created.CustomFields, certificate.ID).Date != "2026-10-02" {
		t.Fatalf("création directe : %+v", created.CustomFields)
	}

	// Adhésion annulée : valeurs en lecture seule, écriture refusée.
	expectAdminMembership(t, admin.Do("POST", adminMembershipPath(created.ID, "/cancel"), map[string]any{"reason": "Départ"}), http.StatusOK)
	got = expectAdminMembership(t, admin.Do("GET", adminMembershipPath(created.ID, ""), nil), http.StatusOK)
	if v := findValue(t, got.CustomFields, certificate.ID); v.Editable || *v.Date != "2026-10-02" {
		t.Fatalf("adhésion annulée : %+v", v)
	}
	expectProblem(t, admin.Do("PUT", adminMembershipPath(created.ID, "/fields"), map[string]any{"customFields": []map[string]any{}}),
		http.StatusConflict, api.MsgMembershipFieldsLocked)

	// Nouvelle saison : les champs repartent vides.
	next := createSeason(t, admin, "2027-09-01", "2028-08-31")
	expectSeason(t, admin.Do("POST", seasonPath(next.ID, "/open"), nil), http.StatusOK)
	fee := createFee(t, admin, next.ID, map[string]string{"label": "Adulte", "amount": "60"})
	renewed := expectAdminMembership(t, admin.Do("POST", "/api/admin/memberships", map[string]any{
		"memberId": m.ID, "seasonId": next.ID, "feeId": fee.ID, "kind": "renewal",
		"customFields": []map[string]any{{"fieldId": certificate.ID, "date": "2027-09-10"}, {"fieldId": size.ID, "choiceId": size.Choices[0].ID}},
	}), http.StatusCreated)
	if v := findValue(t, renewed.CustomFields, paid.ID); *v.Checked {
		t.Error("valeur reprise de l'adhésion précédente")
	}
	if v := findValue(t, renewed.CustomFields, group.ID); v.Text != nil {
		t.Error("texte repris de l'adhésion précédente")
	}
}
