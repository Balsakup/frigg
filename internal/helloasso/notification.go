package helloasso

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// NotificationKey dérive du secret la clé de l'adresse de notification
// (research.md 008 › R6). Elle ne révèle pas le secret et change avec lui.
func NotificationKey(secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("frigg-helloasso-notifications"))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// ValidNotificationKey compare une clé reçue en temps constant.
func ValidNotificationKey(secret, key string) bool {
	return secret != "" && hmac.Equal([]byte(NotificationKey(secret)), []byte(key))
}

// Notification est ce que Frigg lit d'une notification : jamais de quoi
// décider, seulement de quoi retrouver la tentative à relire chez HelloAsso.
type Notification struct {
	EventType string
	// Payment est la tentative Frigg (métadonnées du checkout), si présente.
	Payment int64
	// HelloAssoPaymentID et OrderID identifient le paiement HelloAsso : une
	// notification de remboursement ne porte pas les métadonnées de Frigg.
	HelloAssoPaymentID int64
	OrderID            int64
}

// ParseNotification décode une notification de façon tolérante. ok est faux
// si rien ne permet de retrouver une tentative.
func ParseNotification(body []byte) (Notification, bool) {
	var raw struct {
		EventType string           `json:"eventType"`
		Metadata  metadataEnvelope `json:"metadata"`
		Data      struct {
			ID       int64            `json:"id"`
			Metadata metadataEnvelope `json:"metadata"`
			Order    *struct {
				ID int64 `json:"id"`
			} `json:"order"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Notification{}, false
	}
	n := Notification{EventType: raw.EventType}
	if meta := raw.Metadata.Frigg; meta != nil && meta.Payment > 0 {
		n.Payment = meta.Payment
	} else if meta := raw.Data.Metadata.Frigg; meta != nil && meta.Payment > 0 {
		n.Payment = meta.Payment
	}
	switch raw.EventType {
	case "Payment":
		n.HelloAssoPaymentID = raw.Data.ID
		if raw.Data.Order != nil {
			n.OrderID = raw.Data.Order.ID
		}
	case "Order":
		n.OrderID = raw.Data.ID
	}
	return n, n.Payment > 0 || n.HelloAssoPaymentID > 0 || n.OrderID > 0
}

// NotificationKey renvoie la clé de l'adresse de notification de ce client :
// le secret ne quitte pas le paquet.
func (c *Client) NotificationKey() string { return NotificationKey(c.cfg.ClientSecret) }

// ValidNotificationKey compare une clé reçue à celle de ce client, en temps
// constant.
func (c *Client) ValidNotificationKey(key string) bool {
	return ValidNotificationKey(c.cfg.ClientSecret, key)
}
