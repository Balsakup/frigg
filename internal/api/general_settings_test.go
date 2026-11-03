package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/association"
	"github.com/balsakup/frigg/internal/auth"
)

func getGeneral(t *testing.T, c *apitest.Client) api.GeneralSettings {
	t.Helper()
	res := c.Do("GET", "/api/admin/settings/general", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("lecture : %d %s", res.Status, res.Body)
	}
	var g api.GeneralSettings
	res.JSON(t, &g)
	return g
}

// expectFieldLocation vérifie qu'une réponse 422 signale le champ location.
func expectFieldLocation(t *testing.T, res apitest.Response, location string) {
	t.Helper()
	var p apitest.Problem
	res.JSON(t, &p)
	if res.Status != http.StatusUnprocessableEntity {
		t.Fatalf("statut %d, attendu 422 : %s", res.Status, res.Body)
	}
	for _, e := range p.Errors {
		if e.Location == location {
			return
		}
	}
	t.Fatalf("aucune erreur sur %s : %s", location, res.Body)
}

func putGeneral(c *apitest.Client, body map[string]any) apitest.Response {
	return c.Do("PUT", "/api/admin/settings/general", body)
}

// US1 : nom et tagline.
func TestGeneralSettingsNameAndTagline(t *testing.T) {
	env := apitest.New(t)
	env.SetToday("2026-11-03")
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))

	g := getGeneral(t, admin)
	if g.Name != "" || g.Tagline != nil || g.SIRET != nil || g.Social.Instagram != nil {
		t.Fatalf("paramètres initiaux : %+v", g)
	}

	res := putGeneral(admin, map[string]any{"name": "  Les Sentiers du Val ", "tagline": "Randonnées et sorties nature depuis 1987"})
	if res.Status != http.StatusOK {
		t.Fatalf("enregistrement : %d %s", res.Status, res.Body)
	}
	g = getGeneral(t, admin)
	if g.Name != "Les Sentiers du Val" || g.Tagline == nil || *g.Tagline != "Randonnées et sorties nature depuis 1987" {
		t.Fatalf("relu : %+v", g)
	}

	var sess api.Session
	admin.Do("GET", "/api/session", nil).JSON(t, &sess)
	if sess.Association.Name != "Les Sentiers du Val" || sess.Association.Tagline == nil {
		t.Errorf("session : %+v", sess.Association)
	}

	expectFieldError(t, putGeneral(admin, map[string]any{"name": "   ", "tagline": "Autre"}), "body.name", association.MsgNameRequired)
	expectFieldLocation(t, putGeneral(admin, map[string]any{"name": "Club", "tagline": strings.Repeat("a", 151)}), "body.tagline")
	if g := getGeneral(t, admin); g.Name != "Les Sentiers du Val" || *g.Tagline != "Randonnées et sorties nature depuis 1987" {
		t.Errorf("valeurs modifiées malgré l'erreur : %+v", g)
	}

	if res := putGeneral(admin, map[string]any{"name": "Les Sentiers du Val", "tagline": "  "}); res.Status != http.StatusOK {
		t.Fatalf("tagline vide : %d %s", res.Status, res.Body)
	}
	if g := getGeneral(t, admin); g.Tagline != nil {
		t.Errorf("tagline vide conservée : %v", *g.Tagline)
	}

	// Le nouveau nom figure dans les e-mails mis en file ensuite.
	inv := admin.Do("POST", "/api/admin/invitations", map[string]string{"email": "thomas@exemple.fr"})
	if inv.Status != http.StatusCreated {
		t.Fatalf("invitation : %d %s", inv.Status, inv.Body)
	}
	env.FlushMail()
	sent, ok := env.Mail.Last("thomas@exemple.fr")
	if !ok || !strings.Contains(sent.Msg.Subject, "Les Sentiers du Val") || !strings.Contains(sent.Msg.HTML, "Les Sentiers du Val") {
		t.Errorf("e-mail d'invitation : %+v", sent.Msg.Subject)
	}

	logs := env.Logs.String()
	if !strings.Contains(logs, "paramètres généraux modifiés") || !strings.Contains(logs, `"tagline"`) {
		t.Errorf("journal absent : %s", logs)
	}
	if strings.Contains(logs, "Randonnées et sorties") {
		t.Error("valeur de la tagline dans les journaux")
	}
}

// US2 : identité légale et coordonnées ; US4 : réseaux sociaux.
func TestGeneralSettingsLegalAndContact(t *testing.T) {
	env := apitest.New(t)
	env.SetToday("2026-11-03")
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))

	full := map[string]any{
		"name": "Les Sentiers du Val", "rna": "w751234567", "siret": "732 829 320 00074", "foundedOn": "1987-05-12",
		"address":      map[string]any{"line1": "1 rue du Moulin", "line2": "", "postalCode": "95000", "city": "Cergy"},
		"contactEmail": "bureau@sentiers.fr", "phone": "+33 1 23 45 67 89", "website": "https://sentiers.fr",
		"social": map[string]any{"instagram": "https://instagram.com/sentiers", "facebook": ""},
	}
	if res := putGeneral(admin, full); res.Status != http.StatusOK {
		t.Fatalf("enregistrement : %d %s", res.Status, res.Body)
	}
	g := getGeneral(t, admin)
	if *g.RNA != "W751234567" || *g.SIRET != "73282932000074" || *g.FoundedOn != "1987-05-12" ||
		*g.Phone != "0123456789" || *g.Address.PostalCode != "95000" || g.Address.Line2 != nil ||
		*g.Social.Instagram != "https://instagram.com/sentiers" || g.Social.Facebook != nil {
		t.Fatalf("relu : %+v", g)
	}

	res := putGeneral(admin, map[string]any{
		"name": "Autre nom", "siret": "73282932000075", "foundedOn": "2026-11-04", "website": "www.exemple.fr",
		"social": map[string]any{"instagram": "instagram.com/x"},
	})
	expectFieldError(t, res, "body.siret", association.MsgSIRETInvalid)
	expectFieldError(t, res, "body.foundedOn", association.MsgFoundedInFuture)
	expectFieldError(t, res, "body.website", association.MsgWebURLInvalid)
	expectFieldError(t, res, "body.social.instagram", association.MsgWebURLInvalid)
	if g := getGeneral(t, admin); g.Name != "Les Sentiers du Val" || *g.SIRET != "73282932000074" {
		t.Errorf("valeurs modifiées malgré l'erreur : %+v", g)
	}
	expectFieldError(t, putGeneral(admin, map[string]any{"name": "Club", "contactEmail": "bureau@"}), "body.contactEmail", association.MsgEmailInvalid)
	expectFieldError(t, putGeneral(admin, map[string]any{"name": "Club", "address": map[string]any{"postalCode": "7500"}}), "body.address.postalCode", association.MsgPostalCode)

	logs := env.Logs.String()
	for _, secret := range []string{"bureau@sentiers.fr", "0123456789", "73282932000074", "rue du Moulin", "instagram.com"} {
		if strings.Contains(logs, secret) {
			t.Errorf("%q figure dans les journaux", secret)
		}
	}
}
