package season

import (
	"errors"
	"strconv"
	"time"
)

// MaxMonths est la durée maximale d'une saison, celle d'un exercice comptable
// (FR-004).
const MaxMonths = 24

// Season est une saison telle que stockée. Les fonctions du package attendent
// la suite complète triée par date de début.
type Season struct {
	ID       int64
	StartsOn Date
	EndsOn   Date
	OpenedAt *time.Time
}

// State est l'état exposé d'une saison.
type State string

const (
	Draft  State = "draft"
	Open   State = "open"
	Closed State = "closed"
)

// Setup résume la mise en place des saisons pour le tableau de bord (FR-023).
type Setup string

const (
	SetupMissing      Setup = "missing"
	SetupNoCurrent    Setup = "no-current"
	SetupCurrentDraft Setup = "current-draft"
	SetupReady        Setup = "ready"
)

// Erreurs métier. Les messages destinés aux personnes sont rédigés par l'API.
var (
	ErrEndBeforeStart = errors.New("fin avant le début")
	ErrEndInPast      = errors.New("fin passée")
	ErrTooLong        = errors.New("saison trop longue")
	ErrNextInvalid    = errors.New("saison suivante invalide")
	ErrSeasonsChanged = errors.New("liste des saisons modifiée")
	ErrClosed         = errors.New("saison close")
	ErrStartLocked    = errors.New("date de début non modifiable")
	ErrNextOpen       = errors.New("saison suivante ouverte")
	ErrPreviousDraft  = errors.New("saison précédente en brouillon")
	ErrDeleteOpened   = errors.New("saison déjà ouverte")
	ErrDeleteLastOnly = errors.New("suppression réservée à la dernière saison")
)

// FieldError rattache une erreur à un champ saisi (« startsOn », « endsOn »).
// Label nomme la saison concernée quand le message en a besoin.
type FieldError struct {
	Field string
	Err   error
	Label string
}

func (e *FieldError) Error() string { return e.Field + " : " + e.Err.Error() }
func (e *FieldError) Unwrap() error { return e.Err }

// LabelError est une erreur métier qui nomme une autre saison.
type LabelError struct {
	Err   error
	Label string
}

func (e *LabelError) Error() string { return e.Err.Error() + " : " + e.Label }
func (e *LabelError) Unwrap() error { return e.Err }

// StateOf calcule l'état : close dès le lendemain de la fin, quel que soit
// l'état enregistré (FR-012).
func StateOf(s Season, today Date) State {
	switch {
	case s.EndsOn.Before(today):
		return Closed
	case s.OpenedAt != nil:
		return Open
	default:
		return Draft
	}
}

// IsCurrent indique si la saison contient la date du jour, bornes incluses.
func IsCurrent(s Season, today Date) bool {
	return !today.Before(s.StartsOn) && !today.After(s.EndsOn)
}

var monthAbbr = [...]string{"", "janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."}

func baseLabel(s Season) string {
	if s.StartsOn.Year() == s.EndsOn.Year() {
		return strconv.Itoa(s.StartsOn.Year())
	}
	return strconv.Itoa(s.StartsOn.Year()) + "-" + strconv.Itoa(s.EndsOn.Year())
}

// Labels calcule le libellé de chaque saison (research.md 003 › R6). Seules des
// saisons tenant dans une même année civile peuvent partager un libellé : les
// mois les distinguent alors.
func Labels(seasons []Season) map[int64]string {
	count := map[string]int{}
	for _, s := range seasons {
		count[baseLabel(s)]++
	}
	labels := make(map[int64]string, len(seasons))
	for _, s := range seasons {
		l := baseLabel(s)
		if count[l] > 1 {
			l += " (" + monthAbbr[s.StartsOn.Month()] + "–" + monthAbbr[s.EndsOn.Month()] + ")"
		}
		labels[s.ID] = l
	}
	return labels
}

// NextStart renvoie le début imposé de la prochaine saison : le lendemain de la
// fin de la dernière (FR-006).
func NextStart(seasons []Season) (Date, bool) {
	if len(seasons) == 0 {
		return Date{}, false
	}
	return seasons[len(seasons)-1].EndsOn.AddDays(1), true
}

// maxEnd est la dernière date de fin possible pour une saison commençant le
// jour start.
func maxEnd(start Date) Date { return start.AddMonths(MaxMonths).AddDays(-1) }

// checkEnd applique les règles communes à toute date de fin saisie. pastAllowed
// autorise une fin passée : saison créée pour combler un trou après une
// interruption.
func checkEnd(start, end, today Date, pastAllowed bool) error {
	switch {
	case !end.After(start):
		return &FieldError{Field: "endsOn", Err: ErrEndBeforeStart}
	case !pastAllowed && end.Before(today):
		return &FieldError{Field: "endsOn", Err: ErrEndInPast}
	case end.After(maxEnd(start)):
		return &FieldError{Field: "endsOn", Err: ErrTooLong}
	}
	return nil
}

// Actions liste ce qui est permis sur une saison (data-model.md › Actions
// autorisées). OpenBlockedBy nomme la saison précédente encore en brouillon.
type Actions struct {
	EditStartsOn  bool
	EditEndsOn    bool
	Open          bool
	Duplicate     bool
	Delete        bool
	OpenBlockedBy string
}

func neighbours(seasons []Season, i int) (prev, next *Season) {
	if i > 0 {
		prev = &seasons[i-1]
	}
	if i < len(seasons)-1 {
		next = &seasons[i+1]
	}
	return prev, next
}

// ActionsFor calcule les actions autorisées sur la saison d'indice i.
func ActionsFor(seasons []Season, i int, today Date, labels map[int64]string) Actions {
	s := seasons[i]
	state := StateOf(s, today)
	prev, next := neighbours(seasons, i)
	last := next == nil
	var a Actions

	a.EditStartsOn = state == Draft && i == 0
	if state != Closed && (next == nil || StateOf(*next, today) == Draft) {
		lo, hi := EndsOnBounds(seasons, i, today)
		a.EditEndsOn = !lo.After(hi)
	}
	if state == Draft {
		if prev != nil && StateOf(*prev, today) == Draft {
			a.OpenBlockedBy = labels[prev.ID]
		} else {
			a.Open = true
		}
	}
	if last {
		a.Duplicate = true
		a.Delete = s.OpenedAt == nil
	}
	return a
}

// EndsOnBounds renvoie les bornes de la date de fin de la saison d'indice i,
// décalage de la suivante en brouillon compris (data-model.md › Bornes de
// saisie).
func EndsOnBounds(seasons []Season, i int, today Date) (lo, hi Date) {
	s := seasons[i]
	lo = latest(today, s.StartsOn.AddDays(1))
	hi = maxEnd(s.StartsOn)
	if _, next := neighbours(seasons, i); next != nil && StateOf(*next, today) == Draft {
		lo = latest(lo, next.EndsOn.AddMonths(-MaxMonths))
		hi = earliest(hi, next.EndsOn.AddDays(-2))
	}
	return lo, hi
}

// ShiftsLabel renvoie le libellé de la saison suivante dont le début suivra une
// modification de la fin de la saison d'indice i, ou une chaîne vide.
func ShiftsLabel(seasons []Season, i int, today Date, labels map[int64]string) string {
	_, next := neighbours(seasons, i)
	if next == nil || StateOf(*next, today) != Draft || !ActionsFor(seasons, i, today, labels).EditEndsOn {
		return ""
	}
	return labels[next.ID]
}

// NewSeasonEndsOnBounds renvoie le début imposé et les bornes de la fin d'une
// nouvelle saison ; ok est faux s'il n'existe aucune saison (dates libres). La
// fin peut être passée : après une interruption, les saisons manquantes sont
// recréées une à une pour combler le trou.
func NewSeasonEndsOnBounds(seasons []Season) (start, lo, hi Date, ok bool) {
	start, ok = NextStart(seasons)
	if !ok {
		return Date{}, Date{}, Date{}, false
	}
	return start, start.AddDays(1), maxEnd(start), true
}

// SetupOf calcule l'état de mise en place et l'identifiant de la saison
// courante (0 s'il n'y en a pas).
func SetupOf(seasons []Season, today Date) (Setup, int64) {
	if len(seasons) == 0 {
		return SetupMissing, 0
	}
	for _, s := range seasons {
		if IsCurrent(s, today) {
			if StateOf(s, today) == Draft {
				return SetupCurrentDraft, s.ID
			}
			return SetupReady, s.ID
		}
	}
	return SetupNoCurrent, 0
}

// ValidateCreate vérifie une nouvelle saison. Quand des saisons existent, le
// début doit être celui qu'a vu l'administrateur : sinon la liste a changé
// entre-temps (research.md 003 › R5). Seule la première saison doit finir
// aujourd'hui ou plus tard ; les suivantes peuvent être passées pour combler
// un trou.
func ValidateCreate(seasons []Season, startsOn, endsOn, today Date) error {
	next, ok := NextStart(seasons)
	if ok && !startsOn.Equal(next) {
		return ErrSeasonsChanged
	}
	return checkEnd(startsOn, endsOn, today, ok)
}

// DuplicateDates renvoie les dates de la saison qui suit src : même nombre de
// mois si src couvre des mois entiers, même nombre de jours sinon
// (research.md 003 › R7).
func DuplicateDates(src Season) (start, end Date) {
	start = src.EndsOn.AddDays(1)
	if n, ok := MonthsCovered(src.StartsOn, src.EndsOn); ok {
		return start, start.AddMonths(n).AddDays(-1)
	}
	return start, start.AddDays(src.StartsOn.daysUntil(src.EndsOn))
}

// ValidateDuplicate vérifie que la saison d'indice i peut être dupliquée et
// renvoie les dates de la copie, éventuellement déjà passée (trou à combler).
func ValidateDuplicate(seasons []Season, i int) (start, end Date, err error) {
	if i != len(seasons)-1 {
		return Date{}, Date{}, ErrSeasonsChanged
	}
	start, end = DuplicateDates(seasons[i])
	return start, end, nil
}

// CanOpen vérifie que la saison d'indice i peut être ouverte (FR-011). Une
// saison déjà ouverte est acceptée : l'ouverture est idempotente.
func CanOpen(seasons []Season, i int, today Date) error {
	s := seasons[i]
	switch StateOf(s, today) {
	case Closed:
		return ErrClosed
	case Open:
		return nil
	}
	if prev, _ := neighbours(seasons, i); prev != nil && StateOf(*prev, today) == Draft {
		return &LabelError{Err: ErrPreviousDraft, Label: Labels(seasons)[prev.ID]}
	}
	return nil
}

// Plan décrit les écritures d'une modification de dates : la saison elle-même
// et, si NextID n'est pas nul, le nouveau début de la suivante (FR-017).
type Plan struct {
	StartsOn     Date
	EndsOn       Date
	NextID       int64
	NextStartsOn Date
}

// ValidateUpdate vérifie une modification de dates (FR-015 à FR-018).
// startsOn nul ou égal au début actuel laisse le début inchangé.
func ValidateUpdate(seasons []Season, i int, startsOn *Date, endsOn, today Date) (Plan, error) {
	s := seasons[i]
	state := StateOf(s, today)
	if state == Closed {
		return Plan{}, ErrClosed
	}
	plan := Plan{StartsOn: s.StartsOn, EndsOn: endsOn}
	if startsOn != nil && !startsOn.Equal(s.StartsOn) {
		if state != Draft || i != 0 {
			return Plan{}, ErrStartLocked
		}
		plan.StartsOn = *startsOn
	}
	_, next := neighbours(seasons, i)
	if next != nil && !endsOn.Equal(s.EndsOn) && StateOf(*next, today) != Draft {
		return Plan{}, ErrNextOpen
	}
	if err := checkEnd(plan.StartsOn, endsOn, today, false); err != nil {
		return Plan{}, err
	}
	if next != nil && !endsOn.Equal(s.EndsOn) {
		nextStart := endsOn.AddDays(1)
		if !nextStart.Before(next.EndsOn) || next.EndsOn.After(maxEnd(nextStart)) {
			return Plan{}, &FieldError{Field: "endsOn", Err: ErrNextInvalid, Label: Labels(seasons)[next.ID]}
		}
		plan.NextID, plan.NextStartsOn = next.ID, nextStart
	}
	return plan, nil
}

// CanDelete vérifie que la saison d'indice i peut être supprimée (FR-019) :
// la dernière, si elle n'a jamais été ouverte. Une saison close jamais ouverte
// reste supprimable : c'est le cas d'une saison de comblement saisie avec une
// erreur (elle est close dès sa création).
func CanDelete(seasons []Season, i int) error {
	if seasons[i].OpenedAt != nil {
		return ErrDeleteOpened
	}
	if i != len(seasons)-1 {
		return ErrDeleteLastOnly
	}
	return nil
}
