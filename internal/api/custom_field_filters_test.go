package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

func adminMemberPage(t *testing.T, c *apitest.Client, query string) api.AdminMemberPage {
	t.Helper()
	res := c.Do("GET", "/api/admin/members"+query, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("liste des membres %s : %d %s", query, res.Status, res.Body)
	}
	var p api.AdminMemberPage
	res.JSON(t, &p)
	return p
}

func memberNames(p api.AdminMemberPage) []string {
	out := make([]string, len(p.Items))
	for i, m := range p.Items {
		out[i] = m.FirstName
	}
	return out
}

// TestCustomFieldFilters couvre les filtres de liste et la recherche par
// champ personnalisé (US4, FR-023 à FR-025).
func TestCustomFieldFilters(t *testing.T) {
	f := newMsFixture(t)
	admin := f.admin
	licence := createField(t, admin, fieldBody("member", "Numéro de licence fédérale", "short_text"))
	volunteer := createField(t, admin, fieldBody("member", "Bénévole", "bool"))
	level := createField(t, admin, fieldBody("member", "Niveau", "choice", "choices", []string{"Débutant", "Confirmé"}))
	paid := createField(t, admin, fieldBody("membership", "A payé sa licence fédérale", "bool"))
	note := createField(t, admin, fieldBody("member", "Remarque", "long_text"))

	c, _, lucas := f.member(t, "Lucas")
	cLea, _, lea := f.member(t, "Léa")
	f.member(t, "Noé")
	put := func(m api.Member, inputs ...map[string]any) {
		t.Helper()
		current := expectMember(t, admin.Do("GET", adminMemberPath(m.ID), nil), http.StatusOK)
		var account *int64
		if current.Account != nil {
			account = &current.Account.ID
		}
		expectMember(t, admin.Do("PUT", adminMemberPath(m.ID), adminMemberUpdate(m, account, "confirmDuplicate", true, "customFields", inputs)), http.StatusOK)
	}
	put(lucas, cfText(licence.ID, "1234567"), map[string]any{"fieldId": volunteer.ID, "checked": true},
		map[string]any{"fieldId": level.ID, "choiceId": level.Choices[0].ID}, cfText(note.ID, "Ébène 3456"))
	put(lea, cfText(licence.ID, "Ébène-77"), map[string]any{"fieldId": level.ID, "choiceId": level.Choices[1].ID})

	field := func(v string) string { return "?field=" + url.QueryEscape(v) }
	check := func(query string, want ...string) {
		t.Helper()
		p := adminMemberPage(t, admin, query)
		if fmt.Sprint(memberNames(p)) != fmt.Sprint(want) || p.Total != int64(len(want)) {
			t.Errorf("%s : %v (total %d), attendu %v", query, memberNames(p), p.Total, want)
		}
	}
	check(field(fmt.Sprintf("%d:true", volunteer.ID)), "Lucas")
	check(field(fmt.Sprintf("%d:false", volunteer.ID)), "Léa", "Noé")
	check(field(fmt.Sprintf("%d:c%d", level.ID, level.Choices[1].ID)), "Léa")
	check(field(fmt.Sprintf("%d:empty", level.ID)), "Noé")
	check(field(fmt.Sprintf("%d:false", volunteer.ID))+"&field="+url.QueryEscape(fmt.Sprintf("%d:c%d", level.ID, level.Choices[1].ID)), "Léa")
	check(field(fmt.Sprintf("%d:false", volunteer.ID))+"&account=none")

	// Recherche par fragment du texte court, sans casse ni accents ; le texte
	// long n'est pas indexé.
	check("?q=3456", "Lucas")
	check("?q=EBENE", "Léa")
	for _, bad := range []string{"abc", fmt.Sprintf("%d:true", licence.ID), fmt.Sprintf("%d:empty", volunteer.ID), fmt.Sprintf("%d:c999", level.ID), fmt.Sprintf("%d:true", paid.ID)} {
		expectFieldError(t, admin.Do("GET", "/api/admin/members"+field(bad), nil), "query.field", "Filtre inconnu ou périmé.")
	}

	// Champ archivé : plus trouvé ni filtrable ; rétabli : de nouveau trouvé.
	expectField(t, admin.Do("POST", customFieldPath(licence.ID, "/archive"), nil), http.StatusOK)
	check("?q=3456")
	expectField(t, admin.Do("POST", customFieldPath(volunteer.ID, "/archive"), nil), http.StatusOK)
	expectFieldError(t, admin.Do("GET", "/api/admin/members"+field(fmt.Sprintf("%d:true", volunteer.ID)), nil), "query.field", "Filtre inconnu ou périmé.")
	expectField(t, admin.Do("POST", customFieldPath(licence.ID, "/restore"), nil), http.StatusOK)
	check("?q=3456", "Lucas")
	put(lucas, cfText(licence.ID, ""))
	check("?q=3456")

	// Adhésions : « A payé sa licence fédérale : Non » sur la saison.
	r1 := request(t, c, lucas.ID, f.season.ID, f.adult.ID, "first")
	r2 := request(t, cLea, lea.ID, f.season.ID, f.adult.ID, "first")
	validate(t, admin, r1.ID, f.adult.ID, "first")
	expectAdminMembership(t, admin.Do("PUT", adminMembershipPath(r1.ID, "/fields"), map[string]any{
		"customFields": []map[string]any{{"fieldId": paid.ID, "checked": true}},
	}), http.StatusOK)
	page := adminMemberships(t, admin, fmt.Sprintf("?season=%d", f.season.ID)+"&field="+url.QueryEscape(fmt.Sprintf("%d:false", paid.ID)))
	if page.Total != 1 || page.Items[0].ID != r2.ID {
		t.Fatalf("adhésions non payées : %+v", page.Items)
	}
	page = adminMemberships(t, admin, fmt.Sprintf("?season=%d&status=validated", f.season.ID)+"&field="+url.QueryEscape(fmt.Sprintf("%d:true", paid.ID)))
	if page.Total != 1 || page.Items[0].ID != r1.ID {
		t.Fatalf("adhésions payées et validées : %+v", page.Items)
	}
	expectFieldError(t, admin.Do("GET", "/api/admin/memberships"+field(fmt.Sprintf("%d:true", level.ID)), nil), "query.field", "Filtre inconnu ou périmé.")
}

// TestMemberSearchByFieldPerformance : sur 2 000 fiches, un membre est
// retrouvé par un fragment de son numéro de licence en moins de 2 s (SC-004).
func TestMemberSearchByFieldPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("test de performance")
	}
	f := newMsFixture(t)
	licence := createField(t, f.admin, fieldBody("member", "Numéro de licence fédérale", "short_text"))
	volunteer := createField(t, f.admin, fieldBody("member", "Bénévole", "bool"))
	ctx := context.Background()
	const n = 2000
	err := f.env.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		now := "2026-11-01T10:00:00Z"
		for i := range n {
			id, err := q.CreateMember(ctx, sqlc.CreateMemberParams{
				FirstName: fmt.Sprintf("Prénom%04d", i), LastName: fmt.Sprintf("Nom%04d", i), BirthDate: "2000-01-01", Now: now,
			})
			if err != nil {
				return err
			}
			p := sqlc.UpsertMemberFieldValueParams{OwnerID: id, FieldID: licence.ID, Now: now}
			p.ValueText.String, p.ValueText.Valid = fmt.Sprintf("L%07d", 1000000+i), true
			if err := q.UpsertMemberFieldValue(ctx, p); err != nil {
				return err
			}
			if i%2 == 0 {
				b := sqlc.UpsertMemberFieldValueParams{OwnerID: id, FieldID: volunteer.ID, Now: now}
				b.ValueBool.Int64, b.ValueBool.Valid = 1, true
				if err := q.UpsertMemberFieldValue(ctx, b); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"?q=1001234", "?q=1001234&field=" + url.QueryEscape(fmt.Sprintf("%d:true", volunteer.ID))} {
		start := time.Now()
		p := adminMemberPage(t, f.admin, query)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%s : %v, attendu moins de 2 s", query, d)
		}
		if p.Total != 1 || p.Items[0].FirstName != "Prénom1234" {
			t.Errorf("%s : %v", query, memberNames(p))
		}
	}
}
