package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/mail"
	"github.com/balsakup/frigg/internal/season"
)

// E-mails des événements et des inscriptions (research.md 012 › R10).
// L'adresse de contact est celle de 009 : compte gestionnaire actif, sinon
// adresse de la fiche ; une même adresse reçoit un seul e-mail par action,
// qui nomme chaque inscrit concerné.

var frenchWeekdays = [...]string{"dimanche", "lundi", "mardi", "mercredi", "jeudi", "vendredi", "samedi"}

func frenchHour(t time.Time) string {
	if t.Minute() == 0 {
		return fmt.Sprintf("%d h", t.Hour())
	}
	return fmt.Sprintf("%d h %02d", t.Hour(), t.Minute())
}

func frenchDay(t time.Time, withYear bool) string {
	day := fmt.Sprint(t.Day())
	if t.Day() == 1 {
		day = "1er"
	}
	out := fmt.Sprintf("%s %s %s", frenchWeekdays[t.Weekday()], day, frenchMonths[t.Month()-1])
	if withYear {
		out += fmt.Sprintf(" %d", t.Year())
	}
	return out
}

// frenchDateTimeRange formate une plage à Paris : « samedi 12 décembre 2026
// de 14 h à 17 h » ou « du lundi 13 avril 2027 à 9 h au vendredi 17 avril
// 2027 à 17 h ».
func frenchDateTimeRange(start, end time.Time) string {
	start, end = start.In(season.Paris), end.In(season.Paris)
	if season.Today(start) == season.Today(end) {
		return fmt.Sprintf("%s de %s à %s", frenchDay(start, true), frenchHour(start), frenchHour(end))
	}
	return fmt.Sprintf("du %s à %s au %s à %s", frenchDay(start, true), frenchHour(start), frenchDay(end, true), frenchHour(end))
}

// noticeRow est une inscription à prévenir.
type noticeRow struct {
	RegistrationID int64
	MemberID       int64
	FirstName      string
	LastName       string
	AccountEmail   string
	MemberEmail    string
	Reference      string
	AmountCents    int64
	PaymentStatus  string
}

func eventTimes(ev sqlc.Event) (time.Time, time.Time) {
	return parseTime(ev.StartsAt), parseTime(ev.EndsAt)
}

// sendEventNotices met en file un e-mail par adresse de contact. extra
// complète les données communes ; perRecipient peut les ajuster (montant,
// lien) à partir des inscriptions du destinataire. Renvoie le nombre
// d'e-mails.
func (s *server) sendEventNotices(ctx context.Context, q *sqlc.Queries, template string, ev sqlc.Event, rows []noticeRow,
	extra mail.Data, perRecipient func(d *mail.Data, rows []noticeRow) string) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	cands := make([]communication.Candidate, len(rows))
	byMember := make(map[int64][]noticeRow, len(rows))
	for i, r := range rows {
		cands[i] = communication.Candidate{
			MemberID: r.MemberID, FirstName: r.FirstName, LastName: r.LastName,
			AccountEmail: r.AccountEmail, MemberEmail: r.MemberEmail,
		}
		byMember[r.MemberID] = append(byMember[r.MemberID], r)
	}
	start, end := eventTimes(ev)
	sent := 0
	for _, rcp := range communication.BuildAudience(cands).Recipients {
		data := extra
		data.Event, data.Location, data.When = ev.Title, ev.Location, frenchDateTimeRange(start, end)
		var mine []noticeRow
		data.Members = nil
		for _, id := range rcp.MemberIDs {
			for _, r := range byMember[id] {
				mine = append(mine, r)
				data.Members = append(data.Members, strings.TrimSpace(r.FirstName+" "+r.LastName))
				if r.PaymentStatus == "paid" {
					data.Paid = true
				}
			}
		}
		path := ""
		if perRecipient != nil {
			path = perRecipient(&data, mine)
		}
		if err := s.enqueueNotice(ctx, q, template, rcp.Email, data, path); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

// notifyEventRegistrants prévient toutes les inscriptions confirmées d'un
// événement (changement, annulation, rétablissement).
func (s *server) notifyEventRegistrants(ctx context.Context, q *sqlc.Queries, template string, ev sqlc.Event, extra mail.Data) (int, error) {
	list, err := q.ListNoticeRecipientsOfEvent(ctx, ev.ID)
	if err != nil {
		return 0, err
	}
	rows := make([]noticeRow, len(list))
	for i, r := range list {
		rows[i] = noticeRow{
			RegistrationID: r.ID, MemberID: r.MemberID, FirstName: r.FirstName, LastName: r.LastName,
			AccountEmail: r.AccountEmail, MemberEmail: r.MemberEmail, Reference: r.PaymentReference.String,
			AmountCents: r.AmountCents, PaymentStatus: r.PaymentStatus.String,
		}
	}
	return s.sendEventNotices(ctx, q, template, ev, rows, extra, nil)
}

// notifyRegistrations prévient les inscriptions données (créées ensemble,
// payée, retirée). Le lien mène à l'inscription, ou à « À régler » quand
// plusieurs inscriptions du destinataire restent à payer.
func (s *server) notifyRegistrations(ctx context.Context, q *sqlc.Queries, template string, ev sqlc.Event, ids []int64, extra mail.Data) error {
	rows := make([]noticeRow, 0, len(ids))
	for _, id := range ids {
		r, err := q.GetRegistrationNotice(ctx, id)
		if err != nil {
			return err
		}
		rows = append(rows, noticeRow{
			RegistrationID: r.ID, MemberID: r.MemberID, FirstName: r.FirstName, LastName: r.LastName,
			AccountEmail: r.AccountEmail, MemberEmail: r.MemberEmail, Reference: r.PaymentReference.String,
			AmountCents: r.AmountCents, PaymentStatus: r.PaymentStatus.String,
		})
	}
	_, err := s.sendEventNotices(ctx, q, template, ev, rows, extra, func(d *mail.Data, mine []noticeRow) string {
		var due int64
		var refs []string
		dueCount := 0
		for _, r := range mine {
			if r.Reference != "" {
				refs = append(refs, r.Reference)
			}
			if r.PaymentStatus == "due" || r.PaymentStatus == "announced" {
				due += r.AmountCents
				dueCount++
			}
		}
		if d.Amount == "" && due > 0 {
			d.Amount = formatEuros(due)
		}
		if d.Reference == "" {
			d.Reference = strings.Join(refs, ", ")
		}
		switch {
		case template != "registration-confirmed" || dueCount == 0:
			return ""
		case dueCount > 1:
			return "/a-regler"
		default:
			for _, r := range mine {
				if r.PaymentStatus == "due" || r.PaymentStatus == "announced" {
					return fmt.Sprintf("/mes-inscriptions/%d", r.RegistrationID)
				}
			}
		}
		return ""
	})
	return err
}
