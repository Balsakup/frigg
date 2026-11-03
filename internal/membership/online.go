package membership

import "time"

// OnlineStatus est l'état d'une tentative de paiement par carte
// (data-model.md 008, online_payments).
type OnlineStatus string

const (
	OnlinePending   OnlineStatus = "pending"
	OnlineSucceeded OnlineStatus = "succeeded"
	OnlineFailed    OnlineStatus = "failed"
	OnlineAbandoned OnlineStatus = "abandoned"
	OnlineExpired   OnlineStatus = "expired"
	// OnlineRefunded : paiement remboursé dans HelloAsso, constaté après
	// vérification (le paiement avait réussi).
	OnlineRefunded OnlineStatus = "refunded"
)

// AlertKind est la nature d'un signalement de paiement (R10).
type AlertKind string

const (
	NoAlert             AlertKind = ""
	AlertDuplicate      AlertKind = "duplicate"
	AlertAmountMismatch AlertKind = "amount_mismatch"
	AlertRefundDue      AlertKind = "refund_due"
	AlertUnverified     AlertKind = "unverified"
)

// RefundCancels dit si le remboursement vérifié d'une tentative réussie annule
// l'adhésion : seulement si cette tentative a réglé une adhésion validée.
func RefundCancels(attempt OnlineStatus, m Snapshot, paidByAttempt bool) bool {
	return attempt == OnlineSucceeded && paidByAttempt && m.Status == Validated &&
		m.PaymentStatus == Paid && m.PaidMethod == Card
}

// Indices de retour du navigateur (paramètre « carte » de l'adresse).
const (
	HintNone   = ""
	HintReturn = "return"
	HintError  = "error"
	HintBack   = "back"
)

// CardAvailability dit si la carte est proposée aux adhérents (R7) :
// identifiants configurés, moyen activé et dernier test de connexion réussi
// pour l'environnement en vigueur.
func CardAvailability(configured, enabled bool, testedEnv string, testOK bool, currentEnv string) bool {
	return configured && enabled && testOK && testedEnv != "" && testedEnv == currentEnv
}

// Verified est ce que la relecture de l'intention chez HelloAsso a établi,
// complété de l'indice donné par le navigateur au retour.
type Verified struct {
	Authorized    bool
	ReceivedCents int64
	PaidAt        time.Time
	Hint          string
}

// Settlement est la décision prise sur une tentative.
type Settlement struct {
	// NewStatus est le nouvel état de la tentative ; égal à l'état courant si
	// rien ne change.
	NewStatus OnlineStatus
	// RecordPayment : l'adhésion passe à « Payée » par carte.
	RecordPayment bool
	Alert         AlertKind
}

// SettleOnline décide de l'effet d'une vérification (research.md 008 › R5).
// Un paiement autorisé l'emporte toujours sur un état sans paiement ; un état
// « réussi » est définitif.
func SettleOnline(attempt OnlineStatus, m Snapshot, v Verified) Settlement {
	if attempt == OnlineSucceeded {
		return Settlement{NewStatus: OnlineSucceeded}
	}
	if !v.Authorized {
		if attempt != OnlinePending {
			return Settlement{NewStatus: attempt}
		}
		switch v.Hint {
		case HintError:
			return Settlement{NewStatus: OnlineFailed}
		case HintBack:
			return Settlement{NewStatus: OnlineAbandoned}
		}
		return Settlement{NewStatus: OnlinePending}
	}
	out := Settlement{NewStatus: OnlineSucceeded}
	switch {
	case m.Status == Cancelled:
		out.Alert = AlertRefundDue
	case m.Status == Validated && m.PaymentStatus == Paid:
		out.Alert = AlertDuplicate
	case CanPay(m):
		out.RecordPayment = true
		if v.ReceivedCents != m.AmountCents {
			out.Alert = AlertAmountMismatch
		}
	default:
		// Adhésion sans paiement attendu (gratuite, refusée…) : l'argent doit
		// être rendu.
		out.Alert = AlertRefundDue
	}
	return out
}
