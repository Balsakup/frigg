package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
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
