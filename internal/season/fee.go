package season

// Règles des tarifs de l'adhésion (research.md 004 › R4).

import (
	"cmp"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/balsakup/frigg/internal/textfold"
)

const (
	// MaxFees est le nombre maximal de tarifs d'une saison (FR-004).
	MaxFees = 20
	// MaxAmountCents est le montant maximal d'un tarif : 10 000 € (FR-002).
	MaxAmountCents = 1_000_000
	// MaxFeeLabel et MaxFeeDescription sont les longueurs maximales, en
	// caractères, validées par l'API (FR-001).
	MaxFeeLabel       = 80
	MaxFeeDescription = 500
)

var (
	ErrAmountFormat  = errors.New("montant invalide")
	ErrAmountTooHigh = errors.New("montant trop élevé")
	ErrLabelEmpty    = errors.New("libellé vide")
	ErrLabelTaken    = errors.New("libellé déjà utilisé")
	ErrTooManyFees   = errors.New("trop de tarifs")
	// ErrRenewalRequired : case « Tarif de renouvellement différent » cochée
	// sans montant (research.md 004 › R12).
	ErrRenewalRequired = errors.New("montant de renouvellement manquant")
)

// Fee est un tarif tel que stocké. RenewalAmountCents nil : même montant pour
// toute adhésion.
type Fee struct {
	ID                 int64
	Label              string
	AmountCents        int64
	RenewalAmountCents *int64
	Description        *string
}

// FeeDraft est un tarif tel que saisi. RenewalAmount nil : pas de montant de
// renouvellement distinct.
type FeeDraft struct {
	Label, Amount              string
	RenewalAmount, Description *string
}

// FeeInput est un tarif saisi, nettoyé et validé.
type FeeInput struct {
	Label              string
	AmountCents        int64
	RenewalAmountCents *int64
	Description        *string
}

// FieldErrors regroupe les erreurs de plusieurs champs, renvoyées ensemble
// (FR-017).
type FieldErrors []*FieldError

func (e FieldErrors) Error() string {
	msgs := make([]string, len(e))
	for i, fe := range e {
		msgs[i] = fe.Error()
	}
	return strings.Join(msgs, " ; ")
}

var amountPattern = regexp.MustCompile(`^(\d+)(?:\.(\d{1,2}))?$`)

// maxAmountDigits est le nombre de chiffres de la partie entière de
// MaxAmountCents en euros : au-delà, le montant est trop élevé sans
// conversion, ce qui exclut tout débordement.
const maxAmountDigits = 5

// ParseAmount analyse un montant saisi en euros (« 15,50 », « 1 000 ») et le
// renvoie en centimes (research.md 004 › R3).
func ParseAmount(s string) (int64, error) {
	s = strings.Join(strings.Fields(s), "")
	s = strings.ReplaceAll(s, ",", ".")
	m := amountPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, &FieldError{Field: "amount", Err: ErrAmountFormat}
	}
	whole := strings.TrimLeft(m[1], "0")
	if len(whole) > maxAmountDigits {
		return 0, &FieldError{Field: "amount", Err: ErrAmountTooHigh}
	}
	euros, _ := strconv.ParseInt("0"+whole, 10, 64)
	cents, _ := strconv.ParseInt((m[2] + "00")[:2], 10, 64)
	total := euros*100 + cents
	if total > MaxAmountCents {
		return 0, &FieldError{Field: "amount", Err: ErrAmountTooHigh}
	}
	return total, nil
}

// ValidateFee contrôle un tarif saisi parmi les tarifs existants de la saison
// (id nul : création). À la création, le plafond est vérifié d'abord ; les
// erreurs de champ sont ensuite cumulées dans un FieldErrors.
func ValidateFee(existing []Fee, id int64, d FeeDraft) (FeeInput, error) {
	if id == 0 && len(existing) >= MaxFees {
		return FeeInput{}, ErrTooManyFees
	}
	var errs FieldErrors
	in := FeeInput{Label: strings.TrimSpace(d.Label)}
	if in.Label == "" {
		errs = append(errs, &FieldError{Field: "label", Err: ErrLabelEmpty})
	} else {
		key := textfold.Fold(in.Label)
		if slices.ContainsFunc(existing, func(f Fee) bool { return f.ID != id && textfold.Fold(f.Label) == key }) {
			errs = append(errs, &FieldError{Field: "label", Err: ErrLabelTaken})
		}
	}
	cents, err := ParseAmount(d.Amount)
	var fe *FieldError
	if errors.As(err, &fe) {
		errs = append(errs, fe)
	}
	in.AmountCents = cents
	if d.RenewalAmount != nil {
		if strings.TrimSpace(*d.RenewalAmount) == "" {
			errs = append(errs, &FieldError{Field: "renewalAmount", Err: ErrRenewalRequired})
		} else if renewal, err := ParseAmount(*d.RenewalAmount); errors.As(err, &fe) {
			errs = append(errs, &FieldError{Field: "renewalAmount", Err: fe.Err})
		} else {
			in.RenewalAmountCents = &renewal
		}
	}
	if d.Description != nil {
		if desc := strings.TrimSpace(*d.Description); desc != "" {
			in.Description = &desc
		}
	}
	if len(errs) > 0 {
		return FeeInput{}, errs
	}
	return in, nil
}

// FeeActions liste les actions autorisées sur les tarifs d'une saison.
type FeeActions struct {
	Add, Edit, Delete bool
}

// FeeActionsFor renvoie les actions autorisées selon l'état de la saison et
// son nombre de tarifs : tout est permis tant qu'elle n'est pas close
// (FR-007 à FR-009).
func FeeActionsFor(state State, count int) FeeActions {
	if state == Closed {
		return FeeActions{}
	}
	return FeeActions{Add: count < MaxFees, Edit: true, Delete: true}
}

// SortFees trie les tarifs par montant décroissant, puis par libellé replié,
// puis par identifiant (FR-005).
func SortFees(fees []Fee) {
	slices.SortStableFunc(fees, func(a, b Fee) int {
		return cmp.Or(
			cmp.Compare(b.AmountCents, a.AmountCents),
			strings.Compare(textfold.Fold(a.Label), textfold.Fold(b.Label)),
			cmp.Compare(a.ID, b.ID),
		)
	})
}
