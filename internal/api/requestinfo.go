package api

import "context"

// RequestInfo est partagée entre le middleware de journalisation (qui la crée)
// et le contrôle d'accès (qui y inscrit le compte connecté).
type RequestInfo struct {
	ID        string
	AccountID int64
}

type requestInfoKey struct{}

// WithRequestInfo attache info au contexte de la requête.
func WithRequestInfo(ctx context.Context, info *RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

// RequestInfoFrom renvoie les informations de la requête, ou nil.
func RequestInfoFrom(ctx context.Context) *RequestInfo {
	info, _ := ctx.Value(requestInfoKey{}).(*RequestInfo)
	return info
}

// SetRequestAccount inscrit le compte connecté pour la journalisation.
func SetRequestAccount(ctx context.Context, accountID int64) {
	if info := RequestInfoFrom(ctx); info != nil {
		info.AccountID = accountID
	}
}
