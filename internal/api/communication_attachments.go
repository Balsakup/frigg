package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/communication"
	"github.com/balsakup/frigg/internal/database/sqlc"
)

// Pièces jointes des communications (research.md 009 › R3 à R5, R10).

const (
	MsgAttachmentType  = "Ce type de fichier n'est pas accepté. Formats acceptés : PDF, JPEG, PNG, Word (.docx), Excel (.xlsx), OpenDocument (.odt, .ods)."
	MsgAttachmentSize  = "Le fichier dépasse 5 Mo."
	MsgAttachmentEmpty = "Le fichier est vide."
	MsgAttachmentCount = "Une communication accepte au plus 5 pièces jointes."
	MsgAttachmentGone  = "Pièce jointe introuvable."
)

// UploadMaxBody est la taille maximale d'une requête de dépôt : un fichier de
// 5 Mio et les en-têtes multipart.
const UploadMaxBody = communication.MaxFileBytes + 64<<10

// UploadAttachmentInput est le dépôt d'un fichier pour une préparation.
type UploadAttachmentInput struct {
	RawBody huma.MultipartFormFiles[struct {
		DraftKey string        `form:"draftKey" required:"true" pattern:"^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$" patternDescription:"une clé de préparation (UUID v4)"`
		File     huma.FormFile `form:"file" required:"true"`
	}]
}

// AttachmentOutput est une pièce jointe déposée.
type AttachmentOutput struct{ Body Attachment }

// AttachmentIDInput désigne une pièce de préparation.
type AttachmentIDInput struct {
	ID int64 `path:"id"`
}

// DownloadAttachmentInput désigne une pièce d'une communication envoyée.
type DownloadAttachmentInput struct {
	ID           int64 `path:"id"`
	AttachmentID int64 `path:"attachmentId"`
}

func (s *server) registerCommunicationAttachments(api huma.API, op func(id, method, path, summary string, status int) huma.Operation) {
	upload := op("upload-communication-attachment", http.MethodPost, "/admin/communications/attachments", "Déposer une pièce jointe", http.StatusCreated)
	upload.MaxBodyBytes = UploadMaxBody
	register(api, auth.Admin, upload, s.uploadAttachment)
	register(api, auth.Admin, op("delete-communication-attachment", http.MethodDelete, "/admin/communications/attachments/{id}", "Retirer une pièce jointe", http.StatusNoContent), s.deleteAttachment)
	download := op("download-communication-attachment", http.MethodGet, "/admin/communications/{id}/attachments/{attachmentId}", "Télécharger une pièce jointe", 0)
	download.Responses = map[string]*huma.Response{"200": {
		Description: "Le fichier",
		Content:     map[string]*huma.MediaType{"application/octet-stream": {Schema: &huma.Schema{Type: "string", Format: "binary"}}},
	}}
	register(api, auth.Admin, download, s.downloadAttachment)
}

// formatMegabytes formate une taille comme le frontend : « 7,2 Mo ».
func formatMegabytes(n int64) string {
	v := strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64)
	return strings.Replace(v, ".", ",", 1) + " Mo"
}

// attachmentLimitMessage traduit une erreur de limite ; existing est le total
// déjà joint.
func attachmentLimitMessage(err error, existing int64) string {
	switch {
	case errors.Is(err, communication.ErrFileTooLarge):
		return MsgAttachmentSize
	case errors.Is(err, communication.ErrTooManyFiles):
		return MsgAttachmentCount
	case errors.Is(err, communication.ErrTotalTooLarge):
		return fmt.Sprintf("Les pièces jointes dépassent 10 Mo au total (%s déjà joints).", formatMegabytes(existing))
	case errors.Is(err, communication.ErrEmptyFile):
		return MsgAttachmentEmpty
	}
	return MsgAttachmentType
}

func (s *server) uploadAttachment(ctx context.Context, in *UploadAttachmentInput) (*AttachmentOutput, error) {
	form := in.RawBody.Data()
	file := form.File
	defer file.Close()
	if s.Files == nil {
		return nil, s.internalError(ctx, "répertoire des pièces jointes non configuré", errors.New("Deps.Files nil"))
	}
	author := CurrentAccount(ctx).ID
	if _, err := s.Store.Read.GetCommunicationIDByDraftKey(ctx, form.DraftKey); err == nil {
		return nil, FieldErrors("form.draftKey", MsgCommunicationSent)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, s.internalError(ctx, "lecture d'une communication impossible", err)
	}
	if file.Size > communication.MaxFileBytes {
		return nil, FieldErrors("form.file", MsgAttachmentSize)
	}

	stored, size, sum, err := s.Files.Save(file, communication.MaxFileBytes)
	if errors.Is(err, communication.ErrFileTooLarge) || errors.Is(err, communication.ErrEmptyFile) {
		return nil, FieldErrors("form.file", attachmentLimitMessage(err, 0))
	}
	if err != nil {
		return nil, s.internalError(ctx, "enregistrement d'une pièce jointe impossible", err)
	}
	keep := false
	defer func() {
		if !keep {
			_ = s.Files.Remove(stored)
		}
	}()
	contentType, err := s.detectStored(stored, size)
	if err != nil {
		return nil, FieldErrors("form.file", MsgAttachmentType)
	}

	var row sqlc.CommunicationAttachment
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		drafts, err := q.ListDraftAttachments(ctx, sqlc.ListDraftAttachmentsParams{DraftKey: form.DraftKey, UploadedBy: nid(author)})
		if err != nil {
			return err
		}
		var total int64
		for _, d := range drafts {
			total += d.SizeBytes
		}
		if err := communication.CheckLimits(len(drafts), total, size); err != nil {
			return FieldErrors("form.file", attachmentLimitMessage(err, total))
		}
		row, err = q.CreateCommunicationAttachment(ctx, sqlc.CreateCommunicationAttachmentParams{
			DraftKey: form.DraftKey, UploadedBy: nid(author), StoredName: stored,
			OriginalName: communication.SanitizeFilename(file.Filename), ContentType: contentType,
			SizeBytes: size, Sha256: sum, Now: s.timestamp(),
		})
		return err
	})
	if err != nil {
		return nil, s.apiError(ctx, "enregistrement d'une pièce jointe impossible", err)
	}
	keep = true
	s.Logger.InfoContext(ctx, "pièce jointe déposée", "attachment_id", row.ID, "author_id", author, "size", size, "type", contentType)
	return &AttachmentOutput{Body: toAttachments([]sqlc.CommunicationAttachment{row})[0]}, nil
}

// detectStored détecte le type d'un fichier enregistré d'après son contenu.
func (s *server) detectStored(stored string, size int64) (string, error) {
	f, err := s.Files.Open(stored)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return communication.DetectType(f, size)
}

func (s *server) deleteAttachment(ctx context.Context, in *AttachmentIDInput) (*struct{}, error) {
	author := CurrentAccount(ctx).ID
	var stored string
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		a, err := q.GetDraftAttachment(ctx, sqlc.GetDraftAttachmentParams{ID: in.ID, UploadedBy: nid(author)})
		if errors.Is(err, sql.ErrNoRows) {
			return huma.Error404NotFound(MsgAttachmentGone)
		}
		if err != nil {
			return err
		}
		stored = a.StoredName
		_, err = q.DeleteDraftAttachment(ctx, a.ID)
		return err
	})
	if err != nil {
		return nil, s.apiError(ctx, "suppression d'une pièce jointe impossible", err)
	}
	// Après validation : un échec est rattrapé par le balayage des orphelins.
	if s.Files != nil {
		if err := s.Files.Remove(stored); err != nil {
			s.Logger.WarnContext(ctx, "suppression d'un fichier de pièce jointe impossible", "attachment_id", in.ID, "error", err)
		}
	}
	s.Logger.InfoContext(ctx, "pièce jointe supprimée", "attachment_id", in.ID, "author_id", author)
	return nil, nil
}

// contentDisposition construit l'en-tête de téléchargement (RFC 6266) : un
// repli ASCII et le nom d'origine encodé en UTF-8.
func contentDisposition(name string) string {
	fallback := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			return '_'
		}
		return r
	}, name)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, url.PathEscape(name))
}

func (s *server) downloadAttachment(ctx context.Context, in *DownloadAttachmentInput) (*huma.StreamResponse, error) {
	a, err := s.Store.Read.GetCommunicationAttachment(ctx, sqlc.GetCommunicationAttachmentParams{ID: in.AttachmentID, CommunicationID: nid(in.ID)})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, huma.Error404NotFound(MsgAttachmentGone)
	}
	if err != nil {
		return nil, s.internalError(ctx, "lecture d'une pièce jointe impossible", err)
	}
	if s.Files == nil {
		return nil, s.internalError(ctx, "répertoire des pièces jointes non configuré", errors.New("Deps.Files nil"))
	}
	f, err := s.Files.Open(a.StoredName)
	if err != nil {
		return nil, s.internalError(ctx, "fichier de pièce jointe introuvable", err)
	}
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		defer f.Close()
		hctx.SetHeader("Content-Type", a.ContentType)
		hctx.SetHeader("Content-Length", strconv.FormatInt(a.SizeBytes, 10))
		hctx.SetHeader("Content-Disposition", contentDisposition(a.OriginalName))
		hctx.SetHeader("X-Content-Type-Options", "nosniff")
		hctx.SetHeader("Cache-Control", "private, no-store")
		hctx.SetStatus(http.StatusOK)
		_, _ = io.Copy(hctx.BodyWriter(), f)
	}}, nil
}
