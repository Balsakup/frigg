package api_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
)

func firstNames(ms []api.AudienceMember) []string {
	out := []string{}
	for _, m := range ms {
		out = append(out, m.FirstName)
	}
	return out
}

func TestCommunicationAudience(t *testing.T) {
	f := newCmFixture(t)

	a := f.audience(t, unpaid)
	if a.MembersCount != 4 || a.RecipientsCount != 2 || a.UnreachableCount != 1 {
		t.Fatalf("en impayé : %d ciblés, %d destinataires, %d non joignables", a.MembersCount, a.RecipientsCount, a.UnreachableCount)
	}
	if got := firstNames(a.Members); !slices.Equal(got, []string{"Paul", "Inès", "Léo"}) {
		t.Errorf("joignables : %v", got)
	}
	if got := firstNames(a.Unreachable); !slices.Equal(got, []string{"Hugo"}) || a.Unreachable[0].ContactEmail != nil {
		t.Errorf("non joignables : %v", got)
	}
	// L'adresse du compte prime sur celle de la fiche (Inès).
	if *a.Members[1].ContactEmail != f.claire {
		t.Errorf("adresse de contact d'Inès : %s", *a.Members[1].ContactEmail)
	}
	if a.Preset != "unpaid" || a.TargetLabel != "2026-2027 · En impayé" {
		t.Errorf("cible : %s %q", a.Preset, a.TargetLabel)
	}

	all := f.audience(t, []string{"up_to_date", "announced", "due"})
	if all.MembersCount != 6 || all.RecipientsCount != 4 || all.UnreachableCount != 1 || all.Preset != "all" {
		t.Errorf("tous : %d ciblés, %d destinataires, %d non joignables, %s", all.MembersCount, all.RecipientsCount, all.UnreachableCount, all.Preset)
	}
	if slices.Contains(firstNames(all.Members), "Marc") {
		t.Error("une demande en attente est ciblée")
	}
	upToDate := f.audience(t, []string{"up_to_date"})
	if got := firstNames(upToDate.Members); !slices.Equal(got, []string{"Zoé", "Anne"}) {
		t.Errorf("à jour (payée et gratuite) : %v", got)
	}
}

func TestCommunicationAudienceStates(t *testing.T) {
	f := newCmFixture(t)
	db := f.env.Store.WriteDB
	// Refusée et annulée : exclues ; carte non aboutie : à payer.
	refused := f.insertMembership(t, "Rémi", "Vert", nil, "remi@exemple.fr", f.adult.ID, "pending", "", 0)
	if _, err := db.Exec(`UPDATE memberships SET status = 'refused', reason = 'Doublon' WHERE member_id = ?`, refused); err != nil {
		t.Fatal(err)
	}
	cancelled := f.insertMembership(t, "Lou", "Gris", nil, "lou@exemple.fr", f.adult.ID, "validated", "due", 6000)
	if _, err := db.Exec(`UPDATE memberships SET status = 'cancelled' WHERE member_id = ?`, cancelled); err != nil {
		t.Fatal(err)
	}
	card := f.insertMembership(t, "Tom", "Bleu", nil, "tom@exemple.fr", f.adult.ID, "validated", "due", 6000)
	if _, err := db.Exec(`INSERT INTO online_payments (membership_id, environment, amount_cents, status, created_at, updated_at, finished_at)
		SELECT id, 'sandbox', 6000, 'failed', '2026-10-01T10:00:00Z', '2026-10-01T10:00:00Z', '2026-10-01T10:00:00Z' FROM memberships WHERE member_id = ?`, card); err != nil {
		t.Fatal(err)
	}
	// Compte désactivé : son adresse reste utilisée.
	if _, err := db.Exec(`UPDATE accounts SET disabled_at = '2026-10-01T10:00:00Z' WHERE email = ?`, f.paul); err != nil {
		t.Fatal(err)
	}
	a := f.audience(t, []string{"due", "announced", "up_to_date"})
	got := firstNames(append(a.Members, a.Unreachable...))
	for _, no := range []string{"Rémi", "Lou", "Marc"} {
		if slices.Contains(got, no) {
			t.Errorf("%s ne devrait pas être ciblé", no)
		}
	}
	due := f.audience(t, []string{"due"})
	if !slices.Contains(firstNames(due.Members), "Tom") {
		t.Error("carte non aboutie : attendue à payer")
	}
	if !slices.Contains(firstNames(f.audience(t, []string{"announced"}).Members), "Paul") {
		t.Error("compte désactivé exclu")
	}
}

func TestCommunicationAudienceErrors(t *testing.T) {
	f := newCmFixture(t)
	draft := createSeason(t, f.admin, "2027-09-01", "2028-08-31")
	for name, tc := range map[string]struct {
		path, location string
	}{
		"saison en brouillon":  {audiencePath(draft.ID, unpaid), "query.seasonId"},
		"saison inexistante":   {audiencePath(9999, unpaid), "query.seasonId"},
		"situation inconnue":   {audiencePath(f.season.ID, []string{"paid"}), ""},
		"tarif d'autre saison": {audiencePath(f.season.ID, unpaid, 9999), "query.feeIds"},
		"sans situation":       {fmt.Sprintf("/api/admin/communications/audience?seasonId=%d", f.season.ID), ""},
	} {
		res := f.admin.Do("GET", tc.path, nil)
		if res.Status != http.StatusUnprocessableEntity {
			t.Errorf("%s : statut %d %s", name, res.Status, res.Body)
			continue
		}
		if tc.location != "" {
			expectCmFieldError(t, res, http.StatusUnprocessableEntity, tc.location, "")
		}
	}
	// Cible vide : 200, compteurs à 0.
	empty := f.audience(t, []string{"up_to_date"}, f.young.ID, f.adult.ID)
	if empty.MembersCount != 2 {
		t.Errorf("à jour, deux tarifs : %d", empty.MembersCount)
	}
	none := f.audience(t, []string{"due"}, f.adult.ID)
	if none.MembersCount != 0 || none.RecipientsCount != 0 || none.Members == nil {
		t.Errorf("cible vide : %+v", none)
	}
}

// US3 : restriction par tarif et cible personnalisée.
func TestCommunicationAudienceFees(t *testing.T) {
	f := newCmFixture(t)
	young := f.audience(t, unpaid, f.young.ID)
	if got := firstNames(young.Members); !slices.Equal(got, []string{"Inès", "Léo"}) || young.UnreachableCount != 1 {
		t.Errorf("en impayé, Jeune : %v", got)
	}
	if young.Preset != "unpaid" || young.TargetLabel != "2026-2027 · En impayé · Tarifs : Jeune" {
		t.Errorf("libellé : %q", young.TargetLabel)
	}
	custom := f.audience(t, []string{"due", "due"}, f.adult.ID, f.young.ID, f.young.ID)
	if custom.Preset != "custom" || !slices.Equal(custom.Target.Situations, []string{"due"}) ||
		!slices.Equal(custom.Target.FeeIDs, []int64{f.young.ID, f.adult.ID}) {
		t.Errorf("cible normalisée : %+v", custom.Target)
	}
	if custom.TargetLabel != "2026-2027 · Personnalisée : À payer · Tarifs : Jeune, Adulte" {
		t.Errorf("libellé : %q", custom.TargetLabel)
	}
	path := audiencePath(f.season.ID, unpaid)
	for i := range 51 {
		path += fmt.Sprintf("&feeIds=%d", i+1)
	}
	if res := f.admin.Do("GET", path, nil); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("51 tarifs : %d", res.Status)
	}
}

// SC-002 : décompte d'une saison de 1 000 adhésions en moins de 2 s.
func TestCommunicationAudienceVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("volume")
	}
	f := newCmFixture(t)
	tx, err := f.env.Store.WriteDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
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
	a := f.audience(t, unpaid)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("décompte en %v", d)
	}
	if a.RecipientsCount != 1002 {
		t.Errorf("destinataires : %d", a.RecipientsCount)
	}
}
