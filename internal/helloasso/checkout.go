package helloasso

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// MaxItemName est la longueur maximale de l'intitulé accepté par HelloAsso.
const MaxItemName = 250

// ErrRedirectHost : HelloAsso a renvoyé une page de paiement d'un hôte
// inattendu ; elle n'est jamais transmise au navigateur (R3).
var ErrRedirectHost = errors.New("page de paiement HelloAsso d'un hôte inattendu")

// Payer préremplit le payeur ; les champs vides sont omis.
type Payer struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Email     string `json:"email,omitempty"`
}

// Metadata relie une intention à la tentative Frigg qui l'a créée et à sa
// cible : une adhésion ou une inscription à un événement (012 › R4).
// Les intentions créées avant 012 n'ont que Membership.
type Metadata struct {
	Payment      int64  `json:"payment"`
	Membership   int64  `json:"membership,omitempty"`
	Registration int64  `json:"registration,omitempty"`
	Reference    string `json:"reference,omitempty"`
}

type metadataEnvelope struct {
	Frigg *Metadata `json:"frigg,omitempty"`
}

// CheckoutRequest décrit une intention de paiement.
type CheckoutRequest struct {
	AmountCents int64
	ItemName    string
	Payer       *Payer
	ReturnURL   string
	ErrorURL    string
	BackURL     string
	Metadata    Metadata
}

// CheckoutIntent est la réponse à la création.
type CheckoutIntent struct {
	ID          int64
	RedirectURL string
}

type initCheckoutBody struct {
	TotalAmount      int64            `json:"totalAmount"`
	InitialAmount    int64            `json:"initialAmount"`
	ItemName         string           `json:"itemName"`
	BackURL          string           `json:"backUrl"`
	ErrorURL         string           `json:"errorUrl"`
	ReturnURL        string           `json:"returnUrl"`
	ContainsDonation bool             `json:"containsDonation"`
	Payer            *Payer           `json:"payer,omitempty"`
	Metadata         metadataEnvelope `json:"metadata"`
}

// TruncateItemName tronque un intitulé à MaxItemName caractères.
func TruncateItemName(s string) string {
	if utf8.RuneCountInString(s) <= MaxItemName {
		return s
	}
	return string([]rune(s)[:MaxItemName])
}

// CreateCheckout crée une intention de paiement. Jamais de nouvelle tentative
// automatique : une intention en double serait un risque de double paiement.
func (c *Client) CreateCheckout(ctx context.Context, r CheckoutRequest) (CheckoutIntent, error) {
	body := initCheckoutBody{
		TotalAmount: r.AmountCents, InitialAmount: r.AmountCents,
		ItemName: TruncateItemName(r.ItemName),
		BackURL:  r.BackURL, ErrorURL: r.ErrorURL, ReturnURL: r.ReturnURL,
		Payer:    r.Payer,
		Metadata: metadataEnvelope{Frigg: &r.Metadata},
	}
	if body.Payer != nil && *body.Payer == (Payer{}) {
		body.Payer = nil
	}
	var out struct {
		ID          int64  `json:"id"`
		RedirectURL string `json:"redirectUrl"`
	}
	path := "/v5/organizations/" + url.PathEscape(c.cfg.Organization) + "/checkout-intents"
	if err := c.do(ctx, http.MethodPost, path, body, &out, true); err != nil {
		return CheckoutIntent{}, err
	}
	if out.ID == 0 || out.RedirectURL == "" {
		return CheckoutIntent{}, ErrUnexpected
	}
	if !c.trustedRedirect(out.RedirectURL) {
		return CheckoutIntent{}, ErrRedirectHost
	}
	return CheckoutIntent{ID: out.ID, RedirectURL: out.RedirectURL}, nil
}

// trustedRedirect vérifie que la page de paiement appartient à HelloAsso (ou
// au faux HelloAsso des tests).
func (c *Client) trustedRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if c.cfg.APIURL != "" {
		if api, err := url.Parse(c.cfg.APIURL); err == nil && strings.EqualFold(api.Hostname(), host) {
			return true
		}
	}
	if u.Scheme != "https" {
		return false
	}
	for _, d := range []string{"helloasso.com", "helloasso-sandbox.com"} {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// Payment est un paiement d'une commande HelloAsso.
type Payment struct {
	ID          int64
	AmountCents int64
	TipCents    int64
	State       string
	Date        time.Time
}

// ReceivedCents est le montant reçu par l'association, hors contribution
// volontaire.
func (p Payment) ReceivedCents() int64 { return max(p.AmountCents-p.TipCents, 0) }

// Refunded indique un paiement remboursé chez HelloAsso.
func (p Payment) Refunded() bool { return p.State == "Refunded" }

// Order est la commande créée par un paiement autorisé.
type Order struct {
	ID       int64
	Payments []Payment
}

// AuthorizedPayment renvoie le premier paiement autorisé de la commande.
func (o Order) AuthorizedPayment() (Payment, bool) {
	for _, p := range o.Payments {
		if p.State == "Authorized" {
			return p, true
		}
	}
	return Payment{}, false
}

// PaymentByID renvoie un paiement de la commande.
func (o Order) PaymentByID(id int64) (Payment, bool) {
	for _, p := range o.Payments {
		if p.ID == id {
			return p, true
		}
	}
	return Payment{}, false
}

// Checkout est une intention relue chez HelloAsso.
type Checkout struct {
	ID       int64
	Metadata *Metadata
	// Order est nil tant que le paiement n'est pas autorisé.
	Order *Order
}

// amount décode un montant HelloAsso, entier ou décimal (centimes).
type amount int64

func (a *amount) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		*a = 0
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*a = amount(math.Round(f))
	return nil
}

type checkoutResponse struct {
	ID       int64            `json:"id"`
	Metadata metadataEnvelope `json:"metadata"`
	Order    *struct {
		ID       int64 `json:"id"`
		Payments []struct {
			ID        int64     `json:"id"`
			Amount    amount    `json:"amount"`
			AmountTip amount    `json:"amountTip"`
			State     string    `json:"state"`
			Date      time.Time `json:"date"`
		} `json:"payments"`
	} `json:"order"`
}

// GetCheckout relit une intention, avec sa commande si le paiement est
// autorisé. Le décodage ignore les champs inconnus.
func (c *Client) GetCheckout(ctx context.Context, id int64) (Checkout, error) {
	var out checkoutResponse
	path := "/v5/organizations/" + url.PathEscape(c.cfg.Organization) + "/checkout-intents/" + strconv.FormatInt(id, 10)
	if err := c.do(ctx, http.MethodGet, path, nil, &out, true); err != nil {
		return Checkout{}, err
	}
	ch := Checkout{ID: out.ID, Metadata: out.Metadata.Frigg}
	if out.Order != nil && out.Order.ID != 0 {
		o := &Order{ID: out.Order.ID}
		for _, p := range out.Order.Payments {
			o.Payments = append(o.Payments, Payment{
				ID: p.ID, AmountCents: int64(p.Amount), TipCents: int64(p.AmountTip), State: p.State, Date: p.Date,
			})
		}
		ch.Order = o
	}
	return ch, nil
}

// MarshalMetadata sérialise des métadonnées comme dans une intention (tests).
func MarshalMetadata(m Metadata) []byte {
	b, _ := json.Marshal(metadataEnvelope{Frigg: &m})
	return b
}
