// Package event porte les règles pures des événements et des inscriptions,
// sans base de données (research.md 012 › R3). Les messages destinés aux
// utilisateurs sont associés aux erreurs par la couche API.
package event

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/balsakup/frigg/internal/season"
)

// Status est l'état enregistré d'un événement.
type Status string

const (
	Draft     Status = "draft"
	Published Status = "published"
	Cancelled Status = "cancelled"
)

// Phase est l'état présenté : « Terminé » est calculé, jamais stocké (FR-005).
type Phase string

const (
	PhaseDraft     Phase = "draft"
	PhaseUpcoming  Phase = "upcoming"
	PhaseFinished  Phase = "finished"
	PhaseCancelled Phase = "cancelled"
)

// Openness dit qui peut être inscrit (FR-011).
type Openness string

const (
	SeasonMembers Openness = "season_members"
	AllMembers    Openness = "all_members"
)

// Bornes de FR-002.
const (
	MaxTitle         = 120
	MaxLocation      = 200
	MaxDescription   = 5000
	MaxCapacity      = 10000
	MaxPriceCents    = season.MaxAmountCents
	MaxCancelMessage = 1000
	// LocalLayout est le format des dates-heures locales de l'API (R8).
	LocalLayout = "2006-01-02T15:04"
)

var (
	ErrRequired        = errors.New("valeur manquante")
	ErrTooLong         = errors.New("texte trop long")
	ErrDateTime        = errors.New("date et heure invalides")
	ErrNonexistentTime = errors.New("heure inexistante (changement d'heure)")
	ErrEndBeforeStart  = errors.New("fin avant le début")
	ErrNoSeason        = errors.New("aucune saison ne couvre cette date")
	ErrCapacity        = errors.New("capacité hors bornes")
	ErrPrice           = errors.New("prix hors bornes")
	ErrOpenness        = errors.New("ouverture inconnue")
	ErrCloseBeforeOpen = errors.New("clôture avant l'ouverture")
	ErrCloseAfterEnd   = errors.New("clôture après la fin")
)

// ParseLocal lit une date-heure locale « 2026-12-12T14:00 » à Paris. Une heure
// qui n'existe pas (passage à l'heure d'été) est refusée ; une heure ambiguë
// (passage à l'heure d'hiver) prend la première occurrence.
func ParseLocal(s string) (time.Time, error) {
	t, err := time.ParseInLocation(LocalLayout, strings.TrimSpace(s), season.Paris)
	if err != nil {
		return time.Time{}, ErrDateTime
	}
	// time.ParseInLocation normalise une heure inexistante : on la détecte en
	// relisant l'heure obtenue.
	if t.Format(LocalLayout) != strings.TrimSpace(s) {
		return time.Time{}, ErrNonexistentTime
	}
	// Heure ambiguë : Go ne garantit pas l'occurrence retenue ; on prend la
	// première, si l'heure précédente porte la même heure locale.
	if earlier := t.Add(-time.Hour); earlier.Format(LocalLayout) == t.Format(LocalLayout) {
		return earlier, nil
	}
	return t, nil
}

// FormatLocal écrit l'instant t en date-heure locale de Paris.
func FormatLocal(t time.Time) string { return t.In(season.Paris).Format(LocalLayout) }

// Input est un événement saisi.
type Input struct {
	Title                string
	Location             string
	Description          string
	StartsAt             string
	EndsAt               string
	Openness             Openness
	Capacity             *int
	PriceCents           int64
	RegistrationOpensAt  *string
	RegistrationClosesAt *string
}

// SeasonSpan est une saison réduite à ses bornes.
type SeasonSpan struct {
	ID       int64
	StartsOn season.Date
	EndsOn   season.Date
}

// Event est un événement validé, instants en UTC.
type Event struct {
	SeasonID             int64
	Title                string
	Location             string
	Description          string // vide : aucune
	StartsAt             time.Time
	EndsAt               time.Time
	Openness             Openness
	Capacity             *int
	PriceCents           int64
	RegistrationOpensAt  *time.Time
	RegistrationClosesAt *time.Time
}

// SeasonOf renvoie la saison qui contient la date locale de t.
func SeasonOf(t time.Time, seasons []SeasonSpan) (int64, bool) {
	d := season.Today(t)
	for _, s := range seasons {
		if !d.Before(s.StartsOn) && !d.After(s.EndsOn) {
			return s.ID, true
		}
	}
	return 0, false
}

// SeasonBounds renvoie les instants UTC de minuit local du premier jour et du
// lendemain du dernier jour d'une saison (contrôles de R8).
func SeasonBounds(starts, ends season.Date) (from, to time.Time) {
	local := func(d season.Date) time.Time {
		return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, season.Paris).UTC()
	}
	return local(starts), local(ends.AddDays(1))
}

func cleanLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func cleanText(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// ValidateEvent nettoie et contrôle un événement saisi. Les erreurs de champ
// sont cumulées ; Field porte le nom JSON du champ.
func ValidateEvent(in Input, seasons []SeasonSpan) (Event, error) {
	var errs season.FieldErrors
	add := func(field string, err error) { errs = append(errs, &season.FieldError{Field: field, Err: err}) }
	ev := Event{
		Title:       cleanLine(in.Title),
		Location:    cleanLine(in.Location),
		Description: cleanText(in.Description),
		Openness:    in.Openness,
		Capacity:    in.Capacity,
		PriceCents:  in.PriceCents,
	}
	text := func(field, v string, limit int) {
		switch {
		case v == "":
			add(field, ErrRequired)
		case utf8.RuneCountInString(v) > limit:
			add(field, ErrTooLong)
		}
	}
	text("title", ev.Title, MaxTitle)
	text("location", ev.Location, MaxLocation)
	if utf8.RuneCountInString(ev.Description) > MaxDescription {
		add("description", ErrTooLong)
	}

	starts, errStart := ParseLocal(in.StartsAt)
	if errStart != nil {
		add("startsAt", errStart)
	}
	ends, errEnd := ParseLocal(in.EndsAt)
	if errEnd != nil {
		add("endsAt", errEnd)
	}
	if errStart == nil {
		ev.StartsAt = starts.UTC()
		if id, ok := SeasonOf(starts, seasons); ok {
			ev.SeasonID = id
		} else {
			add("startsAt", ErrNoSeason)
		}
	}
	if errEnd == nil {
		ev.EndsAt = ends.UTC()
		if errStart == nil && !ends.After(starts) {
			add("endsAt", ErrEndBeforeStart)
		}
	}

	if in.Openness != SeasonMembers && in.Openness != AllMembers {
		add("openness", ErrOpenness)
	}
	if in.Capacity != nil && (*in.Capacity < 1 || *in.Capacity > MaxCapacity) {
		add("capacity", ErrCapacity)
	}
	if in.PriceCents < 0 || in.PriceCents > MaxPriceCents {
		add("price", ErrPrice)
	}

	optional := func(field string, s *string) *time.Time {
		if s == nil || strings.TrimSpace(*s) == "" {
			return nil
		}
		t, err := ParseLocal(*s)
		if err != nil {
			add(field, err)
			return nil
		}
		u := t.UTC()
		return &u
	}
	ev.RegistrationOpensAt = optional("registrationOpensAt", in.RegistrationOpensAt)
	ev.RegistrationClosesAt = optional("registrationClosesAt", in.RegistrationClosesAt)
	if o, c := ev.RegistrationOpensAt, ev.RegistrationClosesAt; o != nil && c != nil && !c.After(*o) {
		add("registrationClosesAt", ErrCloseBeforeOpen)
	}
	if c := ev.RegistrationClosesAt; c != nil && errEnd == nil && c.After(ev.EndsAt) {
		add("registrationClosesAt", ErrCloseAfterEnd)
	}

	if len(errs) > 0 {
		return Event{}, errs
	}
	return ev, nil
}

// State est l'état d'un événement enregistré utile aux décisions.
type State struct {
	Status               Status
	StartsAt             time.Time
	EndsAt               time.Time
	PublishedAt          *time.Time
	RegistrationOpensAt  *time.Time
	RegistrationClosesAt *time.Time
	Capacity             *int
}

// PhaseOf calcule la phase présentée à l'instant now.
func PhaseOf(s State, now time.Time) Phase {
	switch {
	case s.Status == Draft:
		return PhaseDraft
	case s.Status == Cancelled:
		return PhaseCancelled
	case !now.Before(s.EndsAt):
		return PhaseFinished
	default:
		return PhaseUpcoming
	}
}

// Finished : la fin est passée.
func Finished(s State, now time.Time) bool { return !now.Before(s.EndsAt) }
