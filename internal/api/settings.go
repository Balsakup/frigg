package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/config"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/membership"
	"github.com/balsakup/frigg/internal/season"
)

// Paramètres de l'association : moyens de paiement (research.md 006 › R7).

const (
	MsgCashInstructionsRequired = "Indiquez où et quand régler en espèces."
	MsgChequePayeeRequired      = "Indiquez l'ordre des chèques."
	MsgTransferHolderRequired   = "Indiquez le titulaire du compte."
	MsgIBANRequired             = "Saisissez l'IBAN."
	MsgIBANInvalid              = "Cet IBAN n'est pas valide. Vérifiez-le sur votre relevé d'identité bancaire."
	MsgBICInvalid               = "Saisissez un BIC de 8 ou 11 caractères."
	MsgTextTooLong              = "Ce texte est trop long."
	MsgHelloAssoNotConfigured   = "Les identifiants HelloAsso ne sont pas renseignés dans la configuration de l'instance."
	MsgHelloAssoTestFirst       = "Testez d'abord la connexion à HelloAsso."
	MsgHelloAssoConfigure       = "Renseignez les identifiants HelloAsso dans la configuration de l'instance."
)

func (s *server) registerSettings(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "get-payment-settings",
		Method:      http.MethodGet,
		Path:        "/admin/settings/payment",
		Summary:     "Moyens de paiement proposés aux adhérents",
		Tags:        []string{tagSettings},
	}, s.getPaymentSettings)

	register(api, auth.Admin, huma.Operation{
		OperationID: "update-payment-settings",
		Method:      http.MethodPut,
		Path:        "/admin/settings/payment",
		Summary:     "Modifier les moyens de paiement",
		Tags:        []string{tagSettings},
	}, s.updatePaymentSettings)

	register(api, auth.Admin, huma.Operation{
		OperationID: "test-helloasso-connection",
		Method:      http.MethodPost,
		Path:        "/admin/settings/payment/helloasso-test",
		Summary:     "Tester la connexion à HelloAsso",
		Tags:        []string{tagSettings},
	}, s.testHelloAssoConnection)
}

// HelloAssoTest est le résultat du dernier test de connexion.
type HelloAssoTest struct {
	At               time.Time `json:"at"`
	Environment      string    `json:"environment" enum:"sandbox,production"`
	OK               bool      `json:"ok"`
	OrganizationName *string   `json:"organizationName"`
	Error            *string   `json:"error" enum:"unauthorized,forbidden,not_found,unavailable,unexpected"`
}

// CardSettings décrit le paiement par carte via HelloAsso (research.md 008 › R7).
type CardSettings struct {
	Enabled         bool           `json:"enabled" doc:"Choix de l'administrateur"`
	Available       bool           `json:"available" doc:"Carte proposée aux adhérents"`
	Configured      bool           `json:"configured" doc:"Identifiants présents dans la configuration de l'instance"`
	Environment     *string        `json:"environment" enum:"sandbox,production"`
	Organization    *string        `json:"organization" doc:"Identifiant de l'association sur HelloAsso"`
	NotificationURL *string        `json:"notificationUrl" doc:"Adresse à déclarer dans l'espace HelloAsso"`
	LastTest        *HelloAssoTest `json:"lastTest"`
	Failing         bool           `json:"failing" doc:"Carte activée et connexion à HelloAsso en échec"`
}

// CashSettings sont les informations du paiement en espèces.
type CashSettings struct {
	Enabled      bool    `json:"enabled"`
	Instructions *string `json:"instructions"`
}

// ChequeSettings sont les informations du paiement par chèque.
type ChequeSettings struct {
	Enabled      bool    `json:"enabled"`
	Payee        *string `json:"payee"`
	Address      *string `json:"address"`
	Instructions *string `json:"instructions"`
}

// TransferSettings sont les informations du paiement par virement.
type TransferSettings struct {
	Enabled      bool    `json:"enabled"`
	Holder       *string `json:"holder"`
	IBAN         *string `json:"iban"`
	BIC          *string `json:"bic"`
	Instructions *string `json:"instructions"`
}

// PaymentSettings sont les moyens de paiement de l'association.
type PaymentSettings struct {
	Card      CardSettings     `json:"card"`
	Cash      CashSettings     `json:"cash"`
	Cheque    ChequeSettings   `json:"cheque"`
	Transfer  TransferSettings `json:"transfer"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

// PaymentSettingsBody remplace les moyens de paiement.
type PaymentSettingsBody struct {
	// Card est facultatif : absent, l'activation de la carte est conservée.
	Card *struct {
		Enabled bool `json:"enabled"`
	} `json:"card,omitempty" required:"false"`
	Cash struct {
		Enabled      bool   `json:"enabled"`
		Instructions string `json:"instructions,omitempty" required:"false" maxLength:"1000"`
	} `json:"cash"`
	Cheque struct {
		Enabled      bool   `json:"enabled"`
		Payee        string `json:"payee,omitempty" required:"false" maxLength:"120"`
		Address      string `json:"address,omitempty" required:"false" maxLength:"300"`
		Instructions string `json:"instructions,omitempty" required:"false" maxLength:"1000"`
	} `json:"cheque"`
	Transfer struct {
		Enabled      bool   `json:"enabled"`
		Holder       string `json:"holder,omitempty" required:"false" maxLength:"120"`
		IBAN         string `json:"iban,omitempty" required:"false" maxLength:"50"`
		BIC          string `json:"bic,omitempty" required:"false" maxLength:"20"`
		Instructions string `json:"instructions,omitempty" required:"false" maxLength:"1000"`
	} `json:"transfer"`
}

type PaymentSettingsOutput struct{ Body PaymentSettings }

type UpdatePaymentSettingsInput struct{ Body PaymentSettingsBody }

func settingsFromRow(r sqlc.PaymentSetting) membership.Settings {
	return membership.Settings{
		CashEnabled:          r.CashEnabled == 1,
		CashInstructions:     r.CashInstructions.String,
		ChequeEnabled:        r.ChequeEnabled == 1,
		ChequePayee:          r.ChequePayee.String,
		ChequeAddress:        r.ChequeAddress.String,
		ChequeInstructions:   r.ChequeInstructions.String,
		TransferEnabled:      r.TransferEnabled == 1,
		TransferHolder:       r.TransferHolder.String,
		TransferIBAN:         r.TransferIban.String,
		TransferBIC:          r.TransferBic.String,
		TransferInstructions: r.TransferInstructions.String,
		CardEnabled:          r.CardEnabled == 1,
	}
}

// paymentSettings lit les paramètres des moyens de paiement, avec la
// disponibilité de la carte (research.md 008 › R7).
func (s *server) paymentSettings(ctx context.Context, q *sqlc.Queries) (membership.Settings, error) {
	r, err := q.GetPaymentSettings(ctx)
	if err != nil {
		return membership.Settings{}, err
	}
	st := settingsFromRow(r)
	st.CardAvailable = s.cardAvailable(r)
	st.CardTest = s.helloAssoEnvironment() == config.HelloAssoSandbox
	return st, nil
}

// helloAssoEnvironment renvoie l'environnement HelloAsso configuré, ou "".
func (s *server) helloAssoEnvironment() string {
	if s.HelloAsso == nil {
		return ""
	}
	return s.HelloAsso.Environment()
}

// cardAvailable applique la règle de disponibilité de la carte.
func (s *server) cardAvailable(r sqlc.PaymentSetting) bool {
	return membership.CardAvailability(s.HelloAsso != nil, r.CardEnabled == 1,
		r.HelloassoTestEnvironment.String, r.HelloassoTestOk.Int64 == 1, s.helloAssoEnvironment())
}

func (s *server) toPaymentSettings(r sqlc.PaymentSetting) PaymentSettings {
	st := settingsFromRow(r)
	return PaymentSettings{
		Card: s.cardSettings(r),
		Cash: CashSettings{Enabled: st.CashEnabled, Instructions: optionalString(st.CashInstructions)},
		Cheque: ChequeSettings{
			Enabled: st.ChequeEnabled, Payee: optionalString(st.ChequePayee),
			Address: optionalString(st.ChequeAddress), Instructions: optionalString(st.ChequeInstructions),
		},
		Transfer: TransferSettings{
			Enabled: st.TransferEnabled, Holder: optionalString(st.TransferHolder), IBAN: optionalString(st.TransferIBAN),
			BIC: optionalString(st.TransferBIC), Instructions: optionalString(st.TransferInstructions),
		},
		UpdatedAt: parseTime(r.UpdatedAt),
	}
}

func (s *server) getPaymentSettings(ctx context.Context, _ *struct{}) (*PaymentSettingsOutput, error) {
	r, err := s.Store.Read.GetPaymentSettings(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des paramètres impossible", err)
	}
	return &PaymentSettingsOutput{Body: s.toPaymentSettings(r)}, nil
}

var settingsMessages = map[error]string{
	membership.ErrInstructionsRequired: MsgCashInstructionsRequired,
	membership.ErrPayeeRequired:        MsgChequePayeeRequired,
	membership.ErrHolderRequired:       MsgTransferHolderRequired,
	membership.ErrIBANRequired:         MsgIBANRequired,
	membership.ErrIBANInvalid:          MsgIBANInvalid,
	membership.ErrBICInvalid:           MsgBICInvalid,
	membership.ErrTextTooLong:          MsgTextTooLong,
}

func boolInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func (s *server) updatePaymentSettings(ctx context.Context, in *UpdatePaymentSettingsInput) (*PaymentSettingsOutput, error) {
	b := in.Body
	st, err := membership.ValidatePaymentSettings(membership.Settings{
		CashEnabled: b.Cash.Enabled, CashInstructions: b.Cash.Instructions,
		ChequeEnabled: b.Cheque.Enabled, ChequePayee: b.Cheque.Payee, ChequeAddress: b.Cheque.Address,
		ChequeInstructions: b.Cheque.Instructions,
		TransferEnabled:    b.Transfer.Enabled, TransferHolder: b.Transfer.Holder, TransferIBAN: b.Transfer.IBAN,
		TransferBIC: b.Transfer.BIC, TransferInstructions: b.Transfer.Instructions,
	})
	var fes season.FieldErrors
	if errors.As(err, &fes) {
		details := make([]*huma.ErrorDetail, len(fes))
		for i, fe := range fes {
			details[i] = &huma.ErrorDetail{Location: "body." + fe.Field, Message: settingsMessages[fe.Err]}
		}
		return nil, FieldErrorDetails(details...)
	}
	if err != nil {
		return nil, s.internalError(ctx, "paramètres invalides", err)
	}
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		current, err := q.GetPaymentSettings(ctx)
		if err != nil {
			return err
		}
		card := current.CardEnabled == 1
		if b.Card != nil {
			if b.Card.Enabled && !card {
				switch {
				case s.HelloAsso == nil:
					return FieldErrors("body.card.enabled", MsgHelloAssoNotConfigured)
				case current.HelloassoTestOk.Int64 != 1 || current.HelloassoTestEnvironment.String != s.helloAssoEnvironment():
					return FieldErrors("body.card.enabled", MsgHelloAssoTestFirst)
				}
			}
			card = b.Card.Enabled
		}
		return q.UpdatePaymentSettings(ctx, sqlc.UpdatePaymentSettingsParams{
			CardEnabled: boolInt64(card),
			CashEnabled: boolInt64(st.CashEnabled), CashInstructions: ns(st.CashInstructions),
			ChequeEnabled: boolInt64(st.ChequeEnabled), ChequePayee: ns(st.ChequePayee),
			ChequeAddress: ns(st.ChequeAddress), ChequeInstructions: ns(st.ChequeInstructions),
			TransferEnabled: boolInt64(st.TransferEnabled), TransferHolder: ns(st.TransferHolder),
			TransferIban: ns(st.TransferIBAN), TransferBic: ns(st.TransferBIC),
			TransferInstructions: ns(st.TransferInstructions), Now: s.timestamp(),
		})
	})
	if err != nil {
		return nil, s.txError(ctx, "écriture des paramètres impossible", err)
	}
	enabled := make([]string, 0, 4)
	if b.Card != nil && b.Card.Enabled {
		enabled = append(enabled, string(membership.Card))
	}
	for _, m := range st.EnabledMethods() {
		enabled = append(enabled, string(m))
	}
	s.Logger.InfoContext(ctx, "paramètres de paiement modifiés",
		"action", "settings.payment_update", "actor_id", CurrentAccount(ctx).ID, "enabled", enabled)
	return s.getPaymentSettings(ctx, nil)
}

// cardSettings décrit l'état du paiement par carte pour l'administration. Le
// secret n'en sort jamais : seule la clé dérivée figure dans l'adresse.
func (s *server) cardSettings(r sqlc.PaymentSetting) CardSettings {
	c := CardSettings{Enabled: r.CardEnabled == 1, Configured: s.HelloAsso != nil, Available: s.cardAvailable(r)}
	if s.HelloAsso != nil {
		env, slug := s.HelloAsso.Environment(), s.HelloAsso.OrganizationSlug()
		c.Environment, c.Organization = &env, &slug
		u := s.Config.BaseURL + NotificationPathPrefix + s.HelloAsso.NotificationKey()
		c.NotificationURL = &u
	}
	if r.HelloassoTestAt.Valid {
		c.LastTest = &HelloAssoTest{
			At: parseTime(r.HelloassoTestAt.String), Environment: r.HelloassoTestEnvironment.String,
			OK: r.HelloassoTestOk.Int64 == 1, OrganizationName: optionalValue(r.HelloassoOrganizationName),
			Error: optionalValue(r.HelloassoTestError),
		}
	}
	c.Failing = s.helloAssoFailing(r)
	return c
}

// helloAssoFailing : carte activée et dernier test en échec, ou incident
// d'appel plus récent que le dernier succès (research.md 008 › R9).
func (s *server) helloAssoFailing(r sqlc.PaymentSetting) bool {
	if r.CardEnabled != 1 || s.HelloAsso == nil {
		return false
	}
	return (r.HelloassoTestAt.Valid && r.HelloassoTestOk.Int64 != 1) || r.HelloassoErrorAt.Valid
}

func (s *server) testHelloAssoConnection(ctx context.Context, _ *struct{}) (*PaymentSettingsOutput, error) {
	if s.HelloAsso == nil {
		return nil, huma.Error409Conflict(MsgHelloAssoConfigure)
	}
	name, err := s.HelloAsso.Organization(ctx)
	code := helloasso.ErrorCode(err)
	params := sqlc.RecordHelloAssoTestParams{
		Now: ns(s.timestamp()), Environment: ns(s.HelloAsso.Environment()), Ok: nid(boolInt64(err == nil)),
		ErrorCode: ns(code), OrganizationName: ns(name),
	}
	if e := s.Store.Write.RecordHelloAssoTest(ctx, params); e != nil {
		return nil, s.internalError(ctx, "enregistrement du test impossible", e)
	}
	s.Logger.InfoContext(ctx, "test de connexion HelloAsso", "action", "settings.helloasso_test",
		"actor_id", CurrentAccount(ctx).ID, "ok", err == nil, "error_code", code, "environment", s.HelloAsso.Environment())
	return s.getPaymentSettings(ctx, nil)
}
