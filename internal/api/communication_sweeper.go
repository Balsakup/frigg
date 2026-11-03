package api

import (
	"context"
	"time"
)

// CommunicationAttachmentSweeper supprime les pièces jointes des préparations
// abandonnées (FR-017g de 009) et les fichiers orphelins (research.md 009 › R6).
type CommunicationAttachmentSweeper struct {
	s        *server
	Interval time.Duration
}

const (
	// draftAttachmentTTL : durée de conservation d'une pièce non envoyée.
	draftAttachmentTTL = 24 * time.Hour
	// orphanFileAge : âge minimal d'un fichier inconnu de la base avant
	// suppression, pour ne pas toucher un dépôt en cours.
	orphanFileAge = time.Hour
)

// NewCommunicationAttachmentSweeper crée le balayage (intervalle de 15 minutes).
func NewCommunicationAttachmentSweeper(deps Deps) *CommunicationAttachmentSweeper {
	return &CommunicationAttachmentSweeper{s: &server{Deps: deps}, Interval: 15 * time.Minute}
}

// Run balaie à intervalle régulier jusqu'à l'annulation de ctx.
func (w *CommunicationAttachmentSweeper) Run(ctx context.Context) {
	t := time.NewTicker(w.Interval)
	defer t.Stop()
	for {
		w.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick supprime les pièces de préparation expirées puis les fichiers orphelins.
func (w *CommunicationAttachmentSweeper) Tick(ctx context.Context) {
	s := w.s
	if s.Files == nil {
		return
	}
	now := s.now()
	expired, err := s.Store.Read.ListExpiredDraftAttachments(ctx, now.Add(-draftAttachmentTTL).UTC().Format(timeFormat))
	if err != nil {
		s.Logger.ErrorContext(ctx, "lecture des pièces jointes abandonnées impossible", "error", err)
		return
	}
	for _, a := range expired {
		if ctx.Err() != nil {
			return
		}
		n, err := s.Store.Write.DeleteDraftAttachment(ctx, a.ID)
		if err != nil {
			s.Logger.ErrorContext(ctx, "suppression d'une pièce jointe abandonnée impossible", "attachment_id", a.ID, "error", err)
			continue
		}
		if n == 0 {
			continue // envoyée entre-temps
		}
		if err := s.Files.Remove(a.StoredName); err != nil {
			s.Logger.WarnContext(ctx, "suppression d'un fichier de pièce jointe impossible", "attachment_id", a.ID, "error", err)
		}
		s.Logger.InfoContext(ctx, "pièce jointe abandonnée supprimée", "attachment_id", a.ID)
	}

	names, err := s.Store.Read.ListAllStoredNames(ctx)
	if err != nil {
		s.Logger.ErrorContext(ctx, "lecture des pièces jointes impossible", "error", err)
		return
	}
	known := make(map[string]bool, len(names))
	for _, n := range names {
		known[n] = true
	}
	// Les fichiers sont datés par le système de fichiers, sur l'horloge réelle.
	removed, err := s.Files.PurgeOrphans(known, time.Now().Add(-orphanFileAge))
	if err != nil {
		s.Logger.ErrorContext(ctx, "purge des fichiers orphelins impossible", "error", err)
	}
	if removed > 0 {
		s.Logger.InfoContext(ctx, "fichiers orphelins supprimés", "count", removed)
	}
}
