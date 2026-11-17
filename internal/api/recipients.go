package api

import (
	"context"
	"database/sql"

	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Destinataire des e-mails d'adhésion (006 FR-023) et d'inscription
// (012 FR-030), unifié par 016 (research.md › R8, FR-035, FR-040).

// noticeContact rassemble ce qu'il faut savoir d'une fiche pour la prévenir :
// son compte gestionnaire éventuel et son adresse.
type noticeContact struct {
	AccountID       sql.NullInt64
	AccountEmail    string
	AccountDisabled bool
	MemberEmail     string
}

// recipientOf renvoie l'adresse à laquelle prévenir une fiche :
//   - l'adresse du compte qui la gère, s'il est actif ;
//   - sinon, l'adresse de la fiche si elle est sans compte, avec tracked vrai :
//     l'e-mail porte alors un lien personnel neuf ;
//   - sinon, si le compte est désactivé et que disabledFallback est vrai,
//     l'adresse de la fiche, sans lien personnel ;
//   - sinon "" : aucun e-mail.
//
// Les adhésions passent disabledFallback à faux (006 FR-023 : rien pour un
// compte désactivé), les inscriptions à vrai (012, comportement inchangé).
func recipientOf(c noticeContact, disabledFallback bool) (to string, tracked bool) {
	switch {
	case c.AccountID.Valid && !c.AccountDisabled && c.AccountEmail != "":
		return c.AccountEmail, false
	case !c.AccountID.Valid && c.MemberEmail != "":
		return c.MemberEmail, true
	case c.AccountID.Valid && c.AccountDisabled && disabledFallback && c.MemberEmail != "":
		return c.MemberEmail, false
	}
	return "", false
}

// noticePath renvoie le chemin du lien d'un e-mail. Pour un destinataire
// suivi par lien, il émet un lien personnel neuf pour son adresse et renvoie
// trackingPath muni du jeton dans le fragment ; sinon memberPath (éventuellement
// vide : pas de lien). À appeler dans la transaction qui met l'e-mail en file.
func (s *server) noticePath(ctx context.Context, q *sqlc.Queries, to string, tracked bool, memberPath, trackingPath string) (string, error) {
	if !tracked {
		return memberPath, nil
	}
	raw, err := s.issueTrackingLink(ctx, q, to)
	if err != nil {
		return "", err
	}
	return trackingPath + "#jeton=" + raw, nil
}
