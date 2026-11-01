package membership

import (
	"errors"
	"math/big"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/balsakup/frigg/internal/season"
)

// Longueurs maximales des informations des moyens de paiement (data-model).
const (
	MaxInstructions = 1000
	MaxPayee        = 120
	MaxHolder       = 120
	MaxAddress      = 300
)

var (
	ErrInstructionsRequired = errors.New("instructions manquantes")
	ErrPayeeRequired        = errors.New("ordre manquant")
	ErrHolderRequired       = errors.New("titulaire manquant")
	ErrIBANRequired         = errors.New("IBAN manquant")
	ErrIBANInvalid          = errors.New("IBAN invalide")
	ErrBICInvalid           = errors.New("BIC invalide")
	ErrTextTooLong          = errors.New("texte trop long")
)

// Settings sont les paramètres des moyens de paiement. Les textes vides sont
// des informations absentes.
type Settings struct {
	CashEnabled      bool
	CashInstructions string

	ChequeEnabled      bool
	ChequePayee        string
	ChequeAddress      string
	ChequeInstructions string

	TransferEnabled      bool
	TransferHolder       string
	TransferIBAN         string
	TransferBIC          string
	TransferInstructions string
}

// EnabledMethods liste les moyens activés, dans l'ordre d'affichage.
func (s Settings) EnabledMethods() []Method {
	var out []Method
	if s.CashEnabled {
		out = append(out, Cash)
	}
	if s.ChequeEnabled {
		out = append(out, Cheque)
	}
	if s.TransferEnabled {
		out = append(out, Transfer)
	}
	return out
}

// Enabled indique si un moyen est activé.
func (s Settings) Enabled(m Method) bool {
	return slices.Contains(s.EnabledMethods(), m)
}

// NormalizeIBAN retire les espaces et passe en majuscules.
func NormalizeIBAN(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), ""))
}

var ibanPattern = regexp.MustCompile(`^[A-Z]{2}[0-9]{2}[A-Z0-9]{11,30}$`)

// ValidIBAN vérifie la forme et la clé de contrôle (ISO 13616, modulo 97) d'un
// IBAN normalisé.
func ValidIBAN(iban string) bool {
	if !ibanPattern.MatchString(iban) {
		return false
	}
	rearranged := iban[4:] + iban[:4]
	var digits strings.Builder
	for _, r := range rearranged {
		if r >= 'A' && r <= 'Z' {
			digits.WriteString(big.NewInt(int64(r-'A') + 10).String())
		} else {
			digits.WriteRune(r)
		}
	}
	n, ok := new(big.Int).SetString(digits.String(), 10)
	return ok && new(big.Int).Mod(n, big.NewInt(97)).Int64() == 1
}

// FormatIBAN groupe un IBAN normalisé par quatre caractères.
func FormatIBAN(iban string) string {
	var b strings.Builder
	for i, r := range iban {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

var bicPattern = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)

// NormalizeBIC retire les espaces et passe en majuscules.
func NormalizeBIC(s string) string { return NormalizeIBAN(s) }

// ValidBIC vérifie la forme d'un BIC normalisé (8 ou 11 caractères).
func ValidBIC(bic string) bool { return bicPattern.MatchString(bic) }

// ValidatePaymentSettings nettoie et contrôle des paramètres saisis. Les
// informations obligatoires ne le sont que pour un moyen activé ; un IBAN ou un
// BIC saisi est toujours contrôlé. Les erreurs sont cumulées.
func ValidatePaymentSettings(in Settings) (Settings, error) {
	s := in
	s.CashInstructions = cleanText(in.CashInstructions)
	s.ChequePayee = strings.Join(strings.Fields(in.ChequePayee), " ")
	s.ChequeAddress = cleanText(in.ChequeAddress)
	s.ChequeInstructions = cleanText(in.ChequeInstructions)
	s.TransferHolder = strings.Join(strings.Fields(in.TransferHolder), " ")
	s.TransferIBAN = NormalizeIBAN(in.TransferIBAN)
	s.TransferBIC = NormalizeBIC(in.TransferBIC)
	s.TransferInstructions = cleanText(in.TransferInstructions)

	var errs season.FieldErrors
	add := func(field string, err error) { errs = append(errs, &season.FieldError{Field: field, Err: err}) }
	long := func(field, v string, limit int) bool {
		if utf8.RuneCountInString(v) > limit {
			add(field, ErrTextTooLong)
			return true
		}
		return false
	}

	if !long("cash.instructions", s.CashInstructions, MaxInstructions) && s.CashEnabled && s.CashInstructions == "" {
		add("cash.instructions", ErrInstructionsRequired)
	}
	if !long("cheque.payee", s.ChequePayee, MaxPayee) && s.ChequeEnabled && s.ChequePayee == "" {
		add("cheque.payee", ErrPayeeRequired)
	}
	long("cheque.address", s.ChequeAddress, MaxAddress)
	long("cheque.instructions", s.ChequeInstructions, MaxInstructions)
	if !long("transfer.holder", s.TransferHolder, MaxHolder) && s.TransferEnabled && s.TransferHolder == "" {
		add("transfer.holder", ErrHolderRequired)
	}
	switch {
	case s.TransferIBAN == "" && s.TransferEnabled:
		add("transfer.iban", ErrIBANRequired)
	case s.TransferIBAN != "" && !ValidIBAN(s.TransferIBAN):
		add("transfer.iban", ErrIBANInvalid)
	}
	if s.TransferBIC != "" && !ValidBIC(s.TransferBIC) {
		add("transfer.bic", ErrBICInvalid)
	}
	long("transfer.instructions", s.TransferInstructions, MaxInstructions)

	if len(errs) > 0 {
		return Settings{}, errs
	}
	return s, nil
}
