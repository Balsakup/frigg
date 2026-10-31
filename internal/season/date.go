// Package season porte les règles métier des saisons, sans base ni HTTP
// (research.md 003 › R4) : dates civiles, états, libellés, actions autorisées
// et validations.
package season

import (
	"errors"
	"time"
)

const layout = "2006-01-02"

// ErrInvalidDate signale une date qui n'est pas au format AAAA-MM-JJ ou qui
// n'existe pas (30 février).
var ErrInvalidDate = errors.New("date invalide")

// Date est une date civile, sans heure ni fuseau (minuit UTC en interne).
type Date struct{ t time.Time }

// ParseDate lit une date AAAA-MM-JJ.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(layout, s)
	if err != nil || t.Format(layout) != s {
		return Date{}, ErrInvalidDate
	}
	return Date{t}, nil
}

// MustDate est ParseDate pour les constantes et les tests.
func MustDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic("date invalide : " + s)
	}
	return d
}

func (d Date) String() string { return d.t.Format(layout) }

func (d Date) AddDays(n int) Date { return Date{d.t.AddDate(0, 0, n)} }

// AddMonths ajoute n mois. Comme time.AddDate, il déborde sur le mois suivant
// depuis un jour qui n'existe pas dans le mois d'arrivée (31 janvier + 1 mois) :
// il n'est exact que depuis un jour présent dans tous les mois, en pratique un
// 1er du mois.
func (d Date) AddMonths(n int) Date { return Date{d.t.AddDate(0, n, 0)} }

func (d Date) Before(o Date) bool   { return d.t.Before(o.t) }
func (d Date) After(o Date) bool    { return d.t.After(o.t) }
func (d Date) Equal(o Date) bool    { return d.t.Equal(o.t) }
func (d Date) Compare(o Date) int   { return d.t.Compare(o.t) }
func (d Date) Year() int            { return d.t.Year() }
func (d Date) Month() time.Month    { return d.t.Month() }
func (d Date) Day() int             { return d.t.Day() }
func (d Date) IsFirstOfMonth() bool { return d.Day() == 1 }
func (d Date) IsLastOfMonth() bool  { return d.AddDays(1).Day() == 1 }

// daysUntil renvoie le nombre de jours de d à o.
func (d Date) daysUntil(o Date) int { return int(o.t.Sub(d.t).Hours() / 24) }

// MonthsCovered renvoie le nombre de mois d'une période qui commence un 1er et
// finit le dernier jour d'un mois ; ok est faux sinon.
func MonthsCovered(start, end Date) (n int, ok bool) {
	if !start.IsFirstOfMonth() || !end.IsLastOfMonth() {
		return 0, false
	}
	return (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1, true
}

func latest(a Date, others ...Date) Date {
	for _, o := range others {
		if o.After(a) {
			a = o
		}
	}
	return a
}

func earliest(a Date, others ...Date) Date {
	for _, o := range others {
		if o.Before(a) {
			a = o
		}
	}
	return a
}

// Paris est le fuseau de l'association (hypothèse de la spec 003). La base des
// fuseaux est embarquée dans le binaire (time/tzdata, cmd/frigg).
var Paris = mustLoad("Europe/Paris")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// Today renvoie la date du jour à Paris pour l'instant now.
func Today(now time.Time) Date {
	y, m, d := now.In(Paris).Date()
	return Date{time.Date(y, m, d, 0, 0, 0, 0, time.UTC)}
}
