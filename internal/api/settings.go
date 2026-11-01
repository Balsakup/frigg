package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
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
	Cash      CashSettings     `json:"cash"`
	Cheque    ChequeSettings   `json:"cheque"`
	Transfer  TransferSettings `json:"transfer"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

// PaymentSettingsBody remplace les moyens de paiement.
type PaymentSettingsBody struct {
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
	}
}

// paymentSettings lit les paramètres des moyens de paiement.
func (s *server) paymentSettings(ctx context.Context, q *sqlc.Queries) (membership.Settings, error) {
	r, err := q.GetPaymentSettings(ctx)
	if err != nil {
		return membership.Settings{}, err
	}
	return settingsFromRow(r), nil
}

func toPaymentSettings(r sqlc.PaymentSetting) PaymentSettings {
	st := settingsFromRow(r)
	return PaymentSettings{
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
	return &PaymentSettingsOutput{Body: toPaymentSettings(r)}, nil
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
		return q.UpdatePaymentSettings(ctx, sqlc.UpdatePaymentSettingsParams{
			CashEnabled: boolInt64(st.CashEnabled), CashInstructions: ns(st.CashInstructions),
			ChequeEnabled: boolInt64(st.ChequeEnabled), ChequePayee: ns(st.ChequePayee),
			ChequeAddress: ns(st.ChequeAddress), ChequeInstructions: ns(st.ChequeInstructions),
			TransferEnabled: boolInt64(st.TransferEnabled), TransferHolder: ns(st.TransferHolder),
			TransferIban: ns(st.TransferIBAN), TransferBic: ns(st.TransferBIC),
			TransferInstructions: ns(st.TransferInstructions), Now: s.timestamp(),
		})
	})
	if err != nil {
		return nil, s.internalError(ctx, "écriture des paramètres impossible", err)
	}
	enabled := make([]string, 0, 3)
	for _, m := range st.EnabledMethods() {
		enabled = append(enabled, string(m))
	}
	s.Logger.InfoContext(ctx, "paramètres de paiement modifiés",
		"action", "settings.payment_update", "actor_id", CurrentAccount(ctx).ID, "enabled", enabled)
	return s.getPaymentSettings(ctx, nil)
}
