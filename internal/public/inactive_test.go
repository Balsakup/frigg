package public

import (
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/season"
)

func TestInactive(t *testing.T) {
	today := season.MustDate("2026-11-17")
	cutoff := InactiveCutoff(today)
	if cutoff.String() != "2025-11-17" {
		t.Fatalf("coupure : %s", cutoff)
	}
	ago := func(months int) time.Time {
		return CutoffInstant(today.AddMonths(-months)).Add(12 * time.Hour)
	}
	var none season.Date
	var never time.Time
	tests := []struct {
		name       string
		createdAt  time.Time
		pending    bool
		seasonEnd  season.Date
		eventEnd   time.Time
		wantActive bool
	}{
		{"créée il y a 13 mois, sans activité", ago(13), false, none, never, false},
		{"créée il y a 11 mois", ago(11), false, none, never, true},
		{"créée le jour de la coupure", CutoffInstant(cutoff), false, none, never, true},
		{"créée la veille de la coupure", CutoffInstant(cutoff).Add(-time.Minute), false, none, never, false},
		{"demande en attente", ago(13), true, none, never, true},
		{"saison terminée il y a 6 mois", ago(13), false, today.AddMonths(-6), never, true},
		{"saison terminée il y a 13 mois", ago(25), false, today.AddMonths(-13), never, false},
		{"saison terminée le jour de la coupure", ago(25), false, cutoff, never, true},
		{"saison terminée la veille de la coupure", ago(25), false, cutoff.AddDays(-1), never, false},
		{"événement terminé il y a 13 mois", ago(25), false, none, ago(13), false},
		{"événement terminé il y a 11 mois", ago(25), false, none, ago(11), true},
		{"événement à venir", ago(25), false, none, ago(-1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Inactive(tt.createdAt, tt.pending, tt.seasonEnd, tt.eventEnd, cutoff); got == tt.wantActive {
				t.Fatalf("Inactive = %v, activité attendue %v", got, tt.wantActive)
			}
		})
	}
}
