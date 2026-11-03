package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/helloasso"
	"github.com/balsakup/frigg/internal/membership"
)

// Adresse de notification de HelloAsso (research.md 008 › R6, constitution
// v1.7.0, principe II). Sans la clé valide : toujours 200 vide, sans
// traitement. Avec la clé : 200 si la notification est traitée ou ignorée,
// 503 sur erreur passagère pour que HelloAsso la renvoie. Corps de réponse
// toujours vide.

// NotificationPathPrefix est le préfixe de l'adresse (le serveur y tronque
// les corps à NotificationMaxBody).
const NotificationPathPrefix = "/api/webhooks/helloasso/"

// NotificationMaxBody est la taille lue d'une notification ; le surplus est
// ignoré, jamais refusé.
const NotificationMaxBody = 1 << 20

// NotificationInput : la clé est une chaîne libre et le corps est lu brut,
// pour qu'aucune validation huma ne produise de réponse dépendant du contenu.
type NotificationInput struct {
	Key     string `path:"key"`
	RawBody []byte
}

// NotificationOutput n'a jamais de corps.
type NotificationOutput struct {
	Status int
}

func (s *server) registerHelloAssoWebhook(api huma.API) {
	register(api, auth.Public, huma.Operation{
		OperationID:  "receive-helloasso-notification",
		Method:       http.MethodPost,
		Path:         "/webhooks/helloasso/{key}",
		Summary:      "Recevoir une notification de HelloAsso",
		Description:  "Adresse à déclarer dans l'espace HelloAsso de l'association. La réponse ne révèle rien à qui ne détient pas la clé.",
		Tags:         []string{tagSettings},
		MaxBodyBytes: NotificationMaxBody + 1,
		Responses: map[string]*huma.Response{
			"200": {Description: "Notification reçue, traitée ou ignorée"},
			"429": {Description: "Trop de requêtes"},
			"503": {Description: "Erreur passagère : HelloAsso renverra la notification"},
		},
	}, s.receiveHelloAssoNotification)
	// huma rend tout RawBody obligatoire : un corps vide recevrait un 400
	// dépendant du contenu. Le corps devient facultatif après enregistrement.
	if item := api.OpenAPI().Paths["/webhooks/helloasso/{key}"]; item != nil && item.Post != nil && item.Post.RequestBody != nil {
		item.Post.RequestBody.Required = false
	}
}

func (s *server) receiveHelloAssoNotification(ctx context.Context, in *NotificationInput) (*NotificationOutput, error) {
	ok := &NotificationOutput{Status: http.StatusOK}
	if !s.Limits.NotificationIP.Allow(clientIP(ctx)) {
		return &NotificationOutput{Status: http.StatusTooManyRequests}, nil
	}
	if s.HelloAsso == nil || !s.HelloAsso.ValidNotificationKey(in.Key) {
		return ok, nil
	}
	n, parsed := helloasso.ParseNotification(in.RawBody)
	if !parsed {
		s.Logger.DebugContext(ctx, "notification ignorée", "reason", "unrelated", "event", n.EventType)
		return ok, nil
	}
	event := n.EventType
	var op sqlc.OnlinePayment
	var err error
	if n.Payment > 0 {
		op, err = s.Store.Read.GetOnlinePayment(ctx, n.Payment)
	} else {
		// Remboursement : la notification porte le paiement HelloAsso, pas
		// les métadonnées de Frigg.
		op, err = s.Store.Read.FindOnlinePaymentByHelloAsso(ctx, sqlc.FindOnlinePaymentByHelloAssoParams{
			Environment:        s.HelloAsso.Environment(),
			HelloassoPaymentID: sql.NullInt64{Int64: n.HelloAssoPaymentID, Valid: n.HelloAssoPaymentID > 0},
			OrderID:            sql.NullInt64{Int64: n.OrderID, Valid: n.OrderID > 0},
		})
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.Logger.DebugContext(ctx, "notification ignorée", "reason", "unknown_payment", "event", event)
			return ok, nil
		}
		s.Logger.ErrorContext(ctx, "lecture du paiement en ligne impossible", "error", err)
		return &NotificationOutput{Status: http.StatusServiceUnavailable}, nil
	}
	if op.Environment != s.HelloAsso.Environment() {
		s.Logger.DebugContext(ctx, "notification ignorée", "reason", "environment", "event", event)
		return ok, nil
	}
	if _, err := s.verifyOnlinePayment(ctx, op.ID, sourceNotification, membership.HintNone); err != nil {
		if !errors.Is(err, errHelloAssoUnavailable) {
			s.Logger.ErrorContext(ctx, "vérification du paiement impossible", "error", err, "online_payment_id", op.ID)
		}
		return &NotificationOutput{Status: http.StatusServiceUnavailable}, nil
	}
	return ok, nil
}
