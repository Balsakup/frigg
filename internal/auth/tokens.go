package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewToken génère un jeton de 256 bits à transmettre par e-mail et l'empreinte
// à stocker (research.md › R8). Le jeton brut n'est jamais conservé.
func NewToken() (raw string, hash []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read ne renvoie jamais d'erreur
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw)
}

// HashToken renvoie l'empreinte SHA-256 d'un jeton brut.
func HashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
