package membership

import (
	"errors"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/season"
)

const testIBAN = "FR7630006000011234567890189"

func TestIBAN(t *testing.T) {
	if got := NormalizeIBAN(" fr76 3000 6000 0112 3456 7890 189 "); got != testIBAN {
		t.Fatalf("normalisation : %q", got)
	}
	valid := []string{testIBAN, "DE89370400440532013000", "BE68539007547034"}
	for _, v := range valid {
		if !ValidIBAN(v) {
			t.Errorf("%s devrait être valide", v)
		}
	}
	invalid := []string{
		"FR7630006000011234567890188", // clé fausse
		"FR76",                        // trop court
		"FR76-3000-6000",              // caractères interdits
		"7630006000011234567890189FR", // pays absent
		"FR76" + strings.Repeat("1", 31),
	}
	for _, v := range invalid {
		if ValidIBAN(v) {
			t.Errorf("%s devrait être invalide", v)
		}
	}
	if got := FormatIBAN(testIBAN); got != "FR76 3000 6000 0112 3456 7890 189" {
		t.Errorf("format : %q", got)
	}
}

func TestBIC(t *testing.T) {
	for _, v := range []string{"BNPAFRPP", "BNPAFRPPXXX", NormalizeBIC(" bnpa frpp ")} {
		if !ValidBIC(v) {
			t.Errorf("%s devrait être valide", v)
		}
	}
	for _, v := range []string{"BNPAFRP", "BNPA1RPP", "BNPAFRPPXX"} {
		if ValidBIC(v) {
			t.Errorf("%s devrait être invalide", v)
		}
	}
}

func fieldErr(t *testing.T, err error, field string, want error) {
	t.Helper()
	var fes season.FieldErrors
	if !errors.As(err, &fes) {
		t.Fatalf("erreurs de champ attendues, obtenu %v", err)
	}
	for _, fe := range fes {
		if fe.Field == field {
			if !errors.Is(fe, want) {
				t.Errorf("%s : %v, attendu %v", field, fe.Err, want)
			}
			return
		}
	}
	t.Errorf("aucune erreur sur %s dans %v", field, err)
}

func TestValidatePaymentSettings(t *testing.T) {
	// Tout désactivé et vide : valide.
	if _, err := ValidatePaymentSettings(Settings{}); err != nil {
		t.Fatalf("paramètres vides : %v", err)
	}

	// Moyens activés sans leurs informations : erreurs cumulées.
	_, err := ValidatePaymentSettings(Settings{CashEnabled: true, ChequeEnabled: true, TransferEnabled: true})
	fieldErr(t, err, "cash.instructions", ErrInstructionsRequired)
	fieldErr(t, err, "cheque.payee", ErrPayeeRequired)
	fieldErr(t, err, "transfer.holder", ErrHolderRequired)
	fieldErr(t, err, "transfer.iban", ErrIBANRequired)

	// IBAN et BIC invalides refusés même si le moyen est désactivé.
	_, err = ValidatePaymentSettings(Settings{TransferIBAN: "FR7630006000011234567890188", TransferBIC: "BNPAFRP"})
	fieldErr(t, err, "transfer.iban", ErrIBANInvalid)
	fieldErr(t, err, "transfer.bic", ErrBICInvalid)

	// Longueurs.
	_, err = ValidatePaymentSettings(Settings{ChequeAddress: strings.Repeat("a", 301)})
	fieldErr(t, err, "cheque.address", ErrTextTooLong)

	// Paramètres valides : nettoyés, retours à la ligne conservés.
	got, err := ValidatePaymentSettings(Settings{
		CashEnabled:      true,
		CashInstructions: "  Au local,\n  le samedi   matin  \n",
		ChequeEnabled:    true,
		ChequePayee:      "  Association   Frigg ",
		TransferEnabled:  true,
		TransferHolder:   "Association Frigg",
		TransferIBAN:     "fr76 3000 6000 0112 3456 7890 189",
		TransferBIC:      "bnpafrpp",
	})
	if err != nil {
		t.Fatalf("paramètres valides : %v", err)
	}
	if got.CashInstructions != "Au local,\nle samedi matin" || got.ChequePayee != "Association Frigg" ||
		got.TransferIBAN != testIBAN || got.TransferBIC != "BNPAFRPP" {
		t.Errorf("nettoyage : %+v", got)
	}
	if m := got.EnabledMethods(); len(m) != 3 || m[0] != Cash || m[2] != Transfer {
		t.Errorf("moyens activés : %v", m)
	}

	// Moyen désactivé : informations conservées.
	got, err = ValidatePaymentSettings(Settings{ChequePayee: "Frigg"})
	if err != nil || got.ChequePayee != "Frigg" || got.Enabled(Cheque) {
		t.Errorf("moyen désactivé : %+v, %v", got, err)
	}
}
