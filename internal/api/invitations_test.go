package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/mail/mailtest"
)

func invite(t *testing.T, env *apitest.Env, admin *apitest.Client, email, role string) (api.Invitation, string) {
	t.Helper()
	body := map[string]string{"email": email}
	if role != "" {
		body["role"] = role
	}
	res := admin.Do("POST", "/api/admin/invitations", body)
	if res.Status != http.StatusCreated {
		t.Fatalf("invitation de %s : %d %s", email, res.Status, res.Body)
	}
	var inv api.Invitation
	res.JSON(t, &inv)
	env.FlushMail()
	sent, ok := env.Mail.Last(inv.Email)
	if !ok {
		t.Fatalf("aucun e-mail d'invitation pour %s", inv.Email)
	}
	return inv, mailtest.Token(sent.Msg)
}

func TestCreateAndListInvitations(t *testing.T) {
	env := apitest.New(t)
	adminEmail := env.Seed(auth.RoleAdmin, true)
	admin := env.LoggedIn(adminEmail)

	inv, token := invite(t, env, admin, " Thomas@Exemple.fr ", "")
	if inv.Email != "thomas@exemple.fr" || inv.Role != auth.RoleMember || inv.Status != "pending" || inv.InvitedBy == nil || *inv.InvitedBy != adminEmail {
		t.Fatalf("invitation inattendue : %+v", inv)
	}
	if d := time.Until(inv.ExpiresAt); d < 6*24*time.Hour || d > 7*24*time.Hour {
		t.Fatalf("échéance à %v, attendue à 7 jours", d)
	}
	sent, _ := env.Mail.Last("thomas@exemple.fr")
	if !strings.Contains(sent.Msg.Subject, "Invitation à rejoindre") || !strings.Contains(sent.Msg.Text, "http://frigg.test/invitation#"+token) {
		t.Fatalf("e-mail inattendu : %s\n%s", sent.Msg.Subject, sent.Msg.Text)
	}

	invite(t, env, admin, "lea@exemple.fr", auth.RoleAdmin)
	var list struct{ Items []api.Invitation }
	admin.Do("GET", "/api/admin/invitations", nil).JSON(t, &list)
	if len(list.Items) != 2 || list.Items[0].Email != "lea@exemple.fr" || list.Items[0].Role != auth.RoleAdmin {
		t.Fatalf("liste inattendue : %+v", list.Items)
	}
}

func TestInvitationConflicts(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	existing := env.Seed(auth.RoleMember, true)
	invite(t, env, admin, "thomas@exemple.fr", "")

	for email, want := range map[string]string{existing: api.MsgAccountExists, "THOMAS@exemple.fr": api.MsgInvitationPending} {
		res := admin.Do("POST", "/api/admin/invitations", map[string]string{"email": email})
		var p apitest.Problem
		res.JSON(t, &p)
		if res.Status != http.StatusConflict || p.Detail != want {
			t.Errorf("%s : %d %q, attendu 409 %q", email, res.Status, p.Detail, want)
		}
	}
	res := admin.Do("POST", "/api/admin/invitations", map[string]string{"email": "pas-une-adresse"})
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, "body.email") {
		t.Errorf("adresse invalide : %d %s", res.Status, res.Body)
	}
}

func TestAcceptInvitation(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	_, token := invite(t, env, admin, "lea@exemple.fr", auth.RoleAdmin)
	guest := env.Client()

	var preview struct{ Email, Role string }
	res := guest.Do("POST", "/api/auth/invitation/lookup", map[string]string{"token": token})
	res.JSON(t, &preview)
	if res.Status != http.StatusOK || preview.Email != "lea@exemple.fr" || preview.Role != auth.RoleAdmin {
		t.Fatalf("consultation : %d %+v", res.Status, preview)
	}

	res = guest.Do("POST", "/api/auth/invitation/accept", map[string]string{"token": token, "password": "faible"})
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity || len(p.Errors) != 3 || p.Errors[0].Location != "body.password" {
		t.Fatalf("mot de passe faible : %d %+v", res.Status, p)
	}

	res = guest.Do("POST", "/api/auth/invitation/accept", map[string]string{"token": token, "password": "Motdepasse1"})
	var s api.Session
	res.JSON(t, &s)
	if res.Status != http.StatusOK || s.Account.Role != auth.RoleAdmin || !s.Account.EmailVerified {
		t.Fatalf("acceptation : %d %s", res.Status, res.Body)
	}
	if res := guest.Do("GET", "/api/session", nil); res.Status != http.StatusOK {
		t.Fatalf("session après acceptation : %d", res.Status)
	}

	for path, body := range map[string]map[string]string{
		"/api/auth/invitation/lookup": {"token": token},
		"/api/auth/invitation/accept": {"token": token, "password": "Motdepasse1"},
	} {
		res := env.Client().Do("POST", path, body)
		res.JSON(t, &p)
		if res.Status != http.StatusGone || p.Detail != api.MsgInvitationGone {
			t.Errorf("%s jeton réutilisé : %d %q", path, res.Status, p.Detail)
		}
	}
	var list struct{ Items []api.Invitation }
	admin.Do("GET", "/api/admin/invitations", nil).JSON(t, &list)
	if len(list.Items) != 0 {
		t.Fatalf("invitation acceptée encore listée : %+v", list.Items)
	}
}

func TestResendAndCancelInvitation(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	inv, oldToken := invite(t, env, admin, "thomas@exemple.fr", "")

	if res := admin.Do("POST", fmt.Sprintf("/api/admin/invitations/%d/resend", inv.ID), nil); res.Status != http.StatusOK {
		t.Fatalf("renvoi : %d %s", res.Status, res.Body)
	}
	env.FlushMail()
	sent, _ := env.Mail.Last("thomas@exemple.fr")
	newToken := mailtest.Token(sent.Msg)
	if newToken == "" || newToken == oldToken {
		t.Fatal("aucun nouveau jeton envoyé")
	}
	if res := env.Client().Do("POST", "/api/auth/invitation/lookup", map[string]string{"token": oldToken}); res.Status != http.StatusGone {
		t.Fatalf("ancien jeton : %d, attendu 410", res.Status)
	}

	if res := admin.Do("DELETE", fmt.Sprintf("/api/admin/invitations/%d", inv.ID), nil); res.Status != http.StatusNoContent {
		t.Fatalf("annulation : %d %s", res.Status, res.Body)
	}
	if res := env.Client().Do("POST", "/api/auth/invitation/lookup", map[string]string{"token": newToken}); res.Status != http.StatusGone {
		t.Fatalf("invitation annulée : %d, attendu 410", res.Status)
	}
	if res := admin.Do("DELETE", fmt.Sprintf("/api/admin/invitations/%d", inv.ID), nil); res.Status != http.StatusNotFound {
		t.Fatalf("seconde annulation : %d, attendu 404", res.Status)
	}
}

func TestExpiredInvitation(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	inv, token := invite(t, env, admin, "thomas@exemple.fr", "")
	if _, err := env.Store.WriteDB.Exec("UPDATE invitations SET expires_at = '2000-01-01T00:00:00Z' WHERE id = ?", inv.ID); err != nil {
		t.Fatal(err)
	}
	if res := env.Client().Do("POST", "/api/auth/invitation/accept", map[string]string{"token": token, "password": "Motdepasse1"}); res.Status != http.StatusGone {
		t.Fatalf("invitation expirée : %d, attendu 410", res.Status)
	}
	var list struct{ Items []api.Invitation }
	admin.Do("GET", "/api/admin/invitations", nil).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Status != "expired" {
		t.Fatalf("statut attendu expired : %+v", list.Items)
	}
}

// Le jeton est vérifié avant le mot de passe : un faux jeton ne déclenche ni la
// règle de mot de passe ni le hachage argon2id.
func TestAcceptInvitationChecksTokenFirst(t *testing.T) {
	env := apitest.New(t)
	res := env.Client().Do("POST", "/api/auth/invitation/accept", map[string]string{"token": "inconnu", "password": "faible"})
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusGone || p.Detail != api.MsgInvitationGone {
		t.Fatalf("faux jeton : %d %+v", res.Status, p)
	}
}
