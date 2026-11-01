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

func TestMyMembers(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	claire := env.LoggedIn(env.Seed(auth.RoleMember, true))

	l := myMembers(t, claire)
	if len(l.Items) != 0 || !l.Actions.Add || !l.Actions.MarkSelf {
		t.Fatalf("liste vide : %+v", l)
	}

	leo := createMine(t, claire, memberBody("Léo", "Martin", "2015-04-12"))
	if leo.Age != 11 || leo.IsSelf || leo.Address != nil || leo.Email != nil {
		t.Fatalf("fiche créée : %+v", leo)
	}
	me := createMine(t, claire, memberBody("Zoé", "Martin", "1985-06-01", "isSelf", true, "email", "Claire@Exemple.fr"))
	if !me.IsSelf || me.Email == nil || *me.Email != "claire@exemple.fr" {
		t.Fatalf("fiche « Vous » : %+v", me)
	}
	if l := myMembers(t, claire); l.Actions.MarkSelf {
		t.Fatal("markSelf devrait être faux")
	}
	expectFieldError(t, claire.Do("POST", "/api/me/members", memberBody("Autre", "Moi", "1980-01-01", "isSelf", true)),
		"body.isSelf", api.MsgSelfTaken)
	elodie := createMine(t, claire, memberBody("Élodie", "Martin", "2018-01-01"))

	l = myMembers(t, claire)
	got := []int64{}
	for _, m := range l.Items {
		got = append(got, m.ID)
	}
	if fmt.Sprint(got) != fmt.Sprint([]int64{me.ID, elodie.ID, leo.ID}) {
		t.Fatalf("ordre : %v", got)
	}

	// Modification : téléphone normalisé, adresse, retrait du « Vous ».
	res := claire.Do("PUT", myMemberPath(leo.ID), memberBody("Léo", "Martin", "2015-04-12",
		"phone", "06.12.34.56.78", "addressLine1", "1 rue de la Paix", "postalCode", "75001", "city", "Paris"))
	upd := expectMember(t, res, http.StatusOK)
	if upd.Phone == nil || *upd.Phone != "0612345678" || upd.Address == nil || upd.Address.City != "Paris" {
		t.Fatalf("modification : %+v", upd)
	}
	expectMember(t, claire.Do("PUT", myMemberPath(me.ID), memberBody("Zoé", "Martin", "1985-06-01")), http.StatusOK)
	if l := myMembers(t, claire); !l.Actions.MarkSelf {
		t.Fatal("markSelf devrait redevenir vrai")
	}

	// Suppression.
	if res := claire.Do("DELETE", myMemberPath(elodie.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("suppression : %d %s", res.Status, res.Body)
	}
	expectProblem(t, claire.Do("GET", myMemberPath(elodie.ID), nil), http.StatusNotFound, api.MsgMemberGone)
	expectProblem(t, claire.Do("DELETE", myMemberPath(elodie.ID), nil), http.StatusNotFound, api.MsgMemberGone)

	// Erreurs cumulées.
	res = claire.Do("POST", "/api/me/members", memberBody(" ", "Martin", "2026-11-02",
		"phone", "0612", "addressLine1", "1 rue de la Paix"))
	for loc, msg := range map[string]string{
		"body.firstName":  api.MsgFirstNameRequired,
		"body.birthDate":  api.MsgBirthFuture,
		"body.phone":      api.MsgPhoneFormat,
		"body.postalCode": api.MsgAddressIncomplete,
		"body.city":       api.MsgAddressIncomplete,
	} {
		expectFieldError(t, res, loc, msg)
	}
	expectFieldError(t, claire.Do("POST", "/api/me/members", memberBody("A", "B", "1900-01-01")), "body.birthDate", api.MsgBirthTooOld)
	expectFieldError(t, claire.Do("POST", "/api/me/members", memberBody("A", "B", "2000-01-01", "email", "claire")), "body.email", api.MsgEmailFormat)
	expectFieldError(t, claire.Do("POST", "/api/me/members", memberBody("A", "B", "2000-01-01",
		"addressLine1", "x", "postalCode", "7500", "city", "Paris")), "body.postalCode", api.MsgPostalCode)
	expectFieldError(t, claire.Do("POST", "/api/me/members", memberBody("A", "B", "2000-01-01", "addressLine2", "Bât. B")),
		"body.addressLine2", api.MsgAddressLine2Alone)
	expectFieldError(t, claire.Do("POST", "/api/me/members", memberBody("A", "B", "2000-13-01")), "body.birthDate", "Saisissez une date valide.")

	if n := memberLogs(t, env, "member.create", "me", "Martin", "Léo", "claire@"); n != 3 {
		t.Fatalf("%d journaux member.create, attendu 3", n)
	}
	if n := memberLogs(t, env, "member.update", "me", "Martin", "0612345678"); n != 2 {
		t.Fatalf("%d journaux member.update, attendu 2", n)
	}
	if n := memberLogs(t, env, "member.delete", "me", "Martin"); n != 1 {
		t.Fatalf("%d journaux member.delete, attendu 1", n)
	}
}

func TestMyMembersDuplicate(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	otherEmail := env.Seed(auth.RoleMember, true)
	other := env.LoggedIn(otherEmail)
	claire := env.LoggedIn(env.Seed(auth.RoleMember, true))

	createMine(t, other, memberBody("Léo", "Martin", "2015-04-12"))
	res := claire.Do("POST", "/api/me/members", memberBody("léo ", " MARTIN", "2015-04-12"))
	expectFieldError(t, res, "body.confirmDuplicate", api.MsgDuplicate)
	if strings.Contains(res.Body, otherEmail) {
		t.Fatal("le message de doublon révèle le compte de la fiche existante")
	}
	dup := createMine(t, claire, memberBody("léo", "MARTIN", "2015-04-12", "confirmDuplicate", true))

	// Modification sans changement d'identité : pas de nouveau contrôle.
	expectMember(t, claire.Do("PUT", myMemberPath(dup.ID), memberBody("léo", "MARTIN", "2015-04-12", "phone", "0612345678")), http.StatusOK)
	// Changement d'identité vers un doublon : contrôle.
	other2 := createMine(t, claire, memberBody("Jeanne", "Petit", "2010-01-01"))
	expectFieldError(t, claire.Do("PUT", myMemberPath(other2.ID), memberBody("Leo", "Martin", "2015-04-12")),
		"body.confirmDuplicate", api.MsgDuplicate)
}

func TestMyMembersCapacity(t *testing.T) {
	env := apitest.New(t)
	env.SetToday(membersToday)
	claire := env.LoggedIn(env.Seed(auth.RoleMember, true))
	for i := range 20 {
		createMine(t, claire, memberBody(fmt.Sprintf("Enfant%02d", i), "Martin", "2015-04-12"))
	}
	if l := myMembers(t, claire); l.Actions.Add {
		t.Fatal("add devrait être faux à 20 fiches")
	}
	expectProblem(t, claire.Do("POST", "/api/me/members", memberBody("Un", "DeTrop", "2015-04-12")),
		http.StatusConflict, api.MsgMyMembersFull)
}
