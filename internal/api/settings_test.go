package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/helloasso/hatest"
)

// TestPaymentSettings couvre le paramétrage des moyens de paiement (US5).
func TestPaymentSettings(t *testing.T) {
	f := newMsFixture(t)
	get := func() api.PaymentSettings {
		t.Helper()
		var s api.PaymentSettings
		res := f.admin.Do("GET", "/api/admin/settings/payment", nil)
		if res.Status != http.StatusOK {
			t.Fatalf("lecture : %d %s", res.Status, res.Body)
		}
		res.JSON(t, &s)
		return s
	}
	if s := get(); s.Cash.Enabled || s.Cheque.Enabled || s.Transfer.Enabled || s.Transfer.IBAN != nil {
		t.Fatalf("paramètres initiaux : %+v", s)
	}
	if d := dashboard(t, f); !d.PaymentSetupNeeded {
		t.Errorf("encart de paramétrage absent")
	}

	put := func(body map[string]any) apitest.Response { return f.admin.Do("PUT", "/api/admin/settings/payment", body) }
	res := put(map[string]any{
		"cash":     map[string]any{"enabled": true},
		"cheque":   map[string]any{"enabled": true},
		"transfer": map[string]any{"enabled": true, "bic": "BNPAFRP"},
	})
	expectFieldError(t, res, "body.cash.instructions", api.MsgCashInstructionsRequired)
	expectFieldError(t, res, "body.cheque.payee", api.MsgChequePayeeRequired)
	expectFieldError(t, res, "body.transfer.holder", api.MsgTransferHolderRequired)
	expectFieldError(t, res, "body.transfer.iban", api.MsgIBANRequired)
	expectFieldError(t, res, "body.transfer.bic", api.MsgBICInvalid)
	res = put(map[string]any{
		"cash": map[string]any{"enabled": false}, "cheque": map[string]any{"enabled": false},
		"transfer": map[string]any{"enabled": true, "holder": "Frigg", "iban": "FR7630006000011234567890188"},
	})
	expectFieldError(t, res, "body.transfer.iban", api.MsgIBANInvalid)

	setPayment(t, f.admin, map[string]any{
		"cash":     map[string]any{"enabled": true, "instructions": "Au local,\nle samedi matin"},
		"cheque":   map[string]any{"enabled": false, "payee": "Association Frigg"},
		"transfer": map[string]any{"enabled": true, "holder": "Association Frigg", "iban": "fr76 3000 6000 0112 3456 7890 189", "bic": "bnpafrpp"},
	})
	s := get()
	if !s.Cash.Enabled || *s.Cash.Instructions != "Au local,\nle samedi matin" || s.Cheque.Enabled ||
		s.Cheque.Payee == nil || *s.Transfer.IBAN != "FR7630006000011234567890189" || *s.Transfer.BIC != "BNPAFRPP" {
		t.Errorf("paramètres enregistrés : %+v", s)
	}
	if d := dashboard(t, f); d.PaymentSetupNeeded {
		t.Errorf("encart de paramétrage encore affiché")
	}
	setPayment(t, f.admin, map[string]any{"cash": map[string]any{"enabled": false}, "cheque": map[string]any{"enabled": false}, "transfer": map[string]any{"enabled": false}})
	if d := dashboard(t, f); !d.PaymentSetupNeeded {
		t.Errorf("encart de paramétrage absent après désactivation")
	}

	logs := f.env.Logs.String()
	if !strings.Contains(logs, `"action":"settings.payment_update"`) {
		t.Error("journal settings.payment_update absent")
	}
	for _, secret := range []string{"FR76", "3000 6000", "BNPAFRPP", "samedi", "Association Frigg"} {
		if strings.Contains(logs, secret) {
			t.Errorf("%q journalisé", secret)
		}
	}
}

func getSettings(t *testing.T, c *apitest.Client) api.PaymentSettings {
	t.Helper()
	res := c.Do("GET", "/api/admin/settings/payment", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("paramètres : %d %s", res.Status, res.Body)
	}
	var out api.PaymentSettings
	res.JSON(t, &out)
	return out
}

func cardBody(enabled bool) map[string]any {
	return map[string]any{
		"card": map[string]any{"enabled": enabled}, "cash": map[string]any{"enabled": true, "instructions": "Au local"},
		"cheque": map[string]any{"enabled": false}, "transfer": map[string]any{"enabled": false},
	}
}

func TestHelloAssoSettingsNotConfigured(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	got := getSettings(t, admin)
	if got.Card.Configured || got.Card.Available || got.Card.NotificationURL != nil || got.Card.Environment != nil {
		t.Fatalf("carte non configurée : %+v", got.Card)
	}
	res := admin.Do("PUT", "/api/admin/settings/payment", cardBody(true))
	expectFieldError(t, res, "body.card.enabled", api.MsgHelloAssoNotConfigured)
	if res := admin.Do("POST", "/api/admin/settings/payment/helloasso-test", nil); res.Status != http.StatusConflict {
		t.Fatalf("test sans configuration : %d", res.Status)
	}
}

func TestHelloAssoSettingsFlow(t *testing.T) {
	env := apitest.NewWithHelloAsso(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	got := getSettings(t, admin)
	key := helloasso.NotificationKey(hatest.ClientSecret)
	if !got.Card.Configured || got.Card.Enabled || *got.Card.Environment != "sandbox" || *got.Card.Organization != hatest.Organization ||
		got.Card.NotificationURL == nil || !strings.HasSuffix(*got.Card.NotificationURL, "/api/webhooks/helloasso/"+key) || got.Card.LastTest != nil {
		t.Fatalf("carte configurée : %+v", got.Card)
	}
	// Activation avant tout test : refusée.
	expectFieldError(t, admin.Do("PUT", "/api/admin/settings/payment", cardBody(true)), "body.card.enabled", api.MsgHelloAssoTestFirst)

	res := admin.Do("POST", "/api/admin/settings/payment/helloasso-test", nil)
	if res.Status != http.StatusOK {
		t.Fatalf("test : %d %s", res.Status, res.Body)
	}
	res.JSON(t, &got)
	if lt := got.Card.LastTest; lt == nil || !lt.OK || *lt.OrganizationName != hatest.OrgName || lt.Environment != "sandbox" {
		t.Fatalf("dernier test : %+v", got.Card.LastTest)
	}
	res = admin.Do("PUT", "/api/admin/settings/payment", cardBody(true))
	if res.Status != http.StatusOK {
		t.Fatalf("activation : %d %s", res.Status, res.Body)
	}
	got = getSettings(t, admin)
	if !got.Card.Enabled || !got.Card.Available || got.Card.Failing {
		t.Fatalf("carte activée : %+v", got.Card)
	}
	// Enregistrer sans bloc carte conserve l'activation.
	setPayment(t, admin, nil)
	if !getSettings(t, admin).Card.Enabled {
		t.Fatal("activation perdue sans bloc carte")
	}

	// Identifiants refusés : test en échec, carte indisponible et en échec.
	env.HelloAsso.RejectCredentials()
	res = admin.Do("POST", "/api/admin/settings/payment/helloasso-test", nil)
	res.JSON(t, &got)
	if lt := got.Card.LastTest; lt.OK || lt.Error == nil || *lt.Error != "unauthorized" || got.Card.Available || !got.Card.Failing {
		t.Fatalf("test en échec : %+v", got.Card)
	}
	dash := admin.Do("GET", "/api/admin/memberships/dashboard", nil)
	if !strings.Contains(dash.Body, `"helloassoFailing":true`) {
		t.Fatalf("tableau de bord : %s", dash.Body)
	}
	for _, forbidden := range []string{hatest.ClientSecret, "tok-", "ref-"} {
		if strings.Contains(env.Logs.String(), forbidden) || strings.Contains(res.Body, forbidden) {
			t.Fatalf("%q exposé", forbidden)
		}
	}
	member := env.LoggedIn(env.Seed(auth.RoleMember, true))
	if res := member.Do("POST", "/api/admin/settings/payment/helloasso-test", nil); res.Status != http.StatusForbidden {
		t.Fatalf("test par un membre : %d", res.Status)
	}
}

func TestHelloAssoEnvironmentChange(t *testing.T) {
	env := apitest.NewWithHelloAsso(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	env.EnableCard("production") // test réussi pour l'autre environnement
	got := getSettings(t, admin)
	if !got.Card.Enabled || got.Card.Available {
		t.Fatalf("environnement changé : %+v", got.Card)
	}
}
