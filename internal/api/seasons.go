package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/season"
)

const (
	tagSeasons = "Saisons"

	MsgSeasonGone        = "Cette saison n'existe plus."
	MsgSeasonsChanged    = "La liste des saisons a changé. Rechargez la page."
	MsgSeasonClosed      = "Une saison close ne peut plus être modifiée."
	MsgSeasonStartLocked = "La date de début de cette saison ne peut pas être modifiée."
	MsgNextSeasonOpen    = "La saison suivante est déjà ouverte : la date de fin ne peut plus changer."
	MsgSeasonClosedOpen  = "Une saison close ne peut plus être ouverte."
	MsgPreviousDraft     = "Ouvrez d'abord la saison %s."
	MsgDeleteOpened      = "Une saison qui a été ouverte ne peut pas être supprimée."
	MsgDeleteLastOnly    = "Seule la dernière saison peut être supprimée."

	MsgInvalidDate       = "Saisissez une date valide."
	MsgEndBeforeStart    = "La date de fin doit être postérieure à la date de début."
	MsgEndInPast         = "La date de fin ne peut pas être passée."
	MsgSeasonTooLong     = "Une saison ne peut pas durer plus de 24 mois."
	MsgNextSeasonInvalid = "Cette date rendrait la saison %s invalide."
)

// SeasonActions liste les actions autorisées sur une saison (research.md 003 ›
// R8) : le frontend n'affiche que celles-ci.
type SeasonActions struct {
	EditStartsOn      bool    `json:"editStartsOn"`
	EditEndsOn        bool    `json:"editEndsOn"`
	Open              bool    `json:"open"`
	Duplicate         bool    `json:"duplicate"`
	Delete            bool    `json:"delete"`
	OpenBlockedReason *string `json:"openBlockedReason" doc:"Raison pour laquelle l'ouverture n'est pas proposée"`
}

// Season est la représentation d'une saison pour les administrateurs.
type Season struct {
	ID                int64         `json:"id"`
	Label             string        `json:"label"`
	StartsOn          string        `json:"startsOn" format:"date"`
	EndsOn            string        `json:"endsOn" format:"date"`
	Status            string        `json:"status" enum:"draft,open,closed"`
	IsCurrent         bool          `json:"isCurrent" doc:"La saison contient la date du jour"`
	OpenedAt          *time.Time    `json:"openedAt"`
	CreatedAt         time.Time     `json:"createdAt"`
	EndsOnMin         string        `json:"endsOnMin" format:"date" doc:"Date de fin la plus proche autorisée"`
	EndsOnMax         string        `json:"endsOnMax" format:"date" doc:"Date de fin la plus lointaine autorisée"`
	ShiftsSeasonLabel *string       `json:"shiftsSeasonLabel" doc:"Saison suivante dont le début suivra une modification de la fin"`
	Actions           SeasonActions `json:"actions"`
}

// SeasonList est la liste des saisons et l'état de leur mise en place.
type SeasonList struct {
	Items         []Season `json:"items" doc:"De la plus récente à la plus ancienne"`
	Today         string   `json:"today" format:"date" doc:"Date du jour à Paris"`
	CurrentID     *int64   `json:"currentId"`
	NextStartsOn  *string  `json:"nextStartsOn" format:"date" doc:"Début imposé de la prochaine saison ; null s'il n'en existe aucune"`
	NextEndsOnMin *string  `json:"nextEndsOnMin" format:"date"`
	NextEndsOnMax *string  `json:"nextEndsOnMax" format:"date"`
	Setup         string   `json:"setup" enum:"missing,no-current,current-draft,ready"`
}

func toDomain(rows []sqlc.Season) ([]season.Season, error) {
	out := make([]season.Season, len(rows))
	for i, r := range rows {
		start, err := season.ParseDate(r.StartsOn)
		if err != nil {
			return nil, fmt.Errorf("saison %d : début %q : %w", r.ID, r.StartsOn, err)
		}
		end, err := season.ParseDate(r.EndsOn)
		if err != nil {
			return nil, fmt.Errorf("saison %d : fin %q : %w", r.ID, r.EndsOn, err)
		}
		out[i] = season.Season{ID: r.ID, StartsOn: start, EndsOn: end, OpenedAt: optionalTime(r.OpenedAt)}
	}
	return out, nil
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// buildSeasons calcule libellés, états et actions pour toute la suite, dans
// l'ordre chronologique.
func buildSeasons(rows []sqlc.Season, today season.Date) ([]Season, error) {
	seasons, err := toDomain(rows)
	if err != nil {
		return nil, err
	}
	labels := season.Labels(seasons)
	out := make([]Season, len(seasons))
	for i, s := range seasons {
		a := season.ActionsFor(seasons, i, today, labels)
		lo, hi := season.EndsOnBounds(seasons, i, today)
		created, _ := time.Parse(timeFormat, rows[i].CreatedAt)
		var blocked *string
		if a.OpenBlockedBy != "" {
			blocked = optionalString(fmt.Sprintf(MsgPreviousDraft, a.OpenBlockedBy))
		}
		out[i] = Season{
			ID:                s.ID,
			Label:             labels[s.ID],
			StartsOn:          s.StartsOn.String(),
			EndsOn:            s.EndsOn.String(),
			Status:            string(season.StateOf(s, today)),
			IsCurrent:         season.IsCurrent(s, today),
			OpenedAt:          s.OpenedAt,
			CreatedAt:         created,
			EndsOnMin:         lo.String(),
			EndsOnMax:         hi.String(),
			ShiftsSeasonLabel: optionalString(season.ShiftsLabel(seasons, i, today, labels)),
			Actions: SeasonActions{
				EditStartsOn:      a.EditStartsOn,
				EditEndsOn:        a.EditEndsOn,
				Open:              a.Open,
				Duplicate:         a.Duplicate,
				Delete:            a.Delete,
				OpenBlockedReason: blocked,
			},
		}
	}
	return out, nil
}

func buildSeasonList(rows []sqlc.Season, today season.Date) (SeasonList, error) {
	items, err := buildSeasons(rows, today)
	if err != nil {
		return SeasonList{}, err
	}
	seasons, _ := toDomain(rows)
	slices.Reverse(items)
	list := SeasonList{Items: items, Today: today.String()}
	setup, current := season.SetupOf(seasons, today)
	list.Setup = string(setup)
	if current != 0 {
		list.CurrentID = &current
	}
	if start, lo, hi, ok := season.NewSeasonEndsOnBounds(seasons); ok {
		list.NextStartsOn = optionalString(start.String())
		list.NextEndsOnMin = optionalString(lo.String())
		list.NextEndsOnMax = optionalString(hi.String())
	}
	return list, nil
}

// indexOf renvoie la position de la saison id dans la suite, ou -1.
func indexOf(seasons []season.Season, id int64) int {
	return slices.IndexFunc(seasons, func(s season.Season) bool { return s.ID == id })
}

// seasonError traduit une erreur du domaine en réponse de l'API.
func seasonError(err error) error {
	var fe *season.FieldError
	if errors.As(err, &fe) {
		msg := map[error]string{
			season.ErrInvalidDate:    MsgInvalidDate,
			season.ErrEndBeforeStart: MsgEndBeforeStart,
			season.ErrEndInPast:      MsgEndInPast,
			season.ErrTooLong:        MsgSeasonTooLong,
		}[fe.Err]
		if errors.Is(fe.Err, season.ErrNextInvalid) {
			msg = fmt.Sprintf(MsgNextSeasonInvalid, fe.Label)
		}
		return FieldErrors("body."+fe.Field, msg)
	}
	var le *season.LabelError
	if errors.As(err, &le) && errors.Is(err, season.ErrPreviousDraft) {
		return huma.Error409Conflict(fmt.Sprintf(MsgPreviousDraft, le.Label))
	}
	for code, msg := range map[error]string{
		season.ErrSeasonsChanged: MsgSeasonsChanged,
		season.ErrStartLocked:    MsgSeasonStartLocked,
		season.ErrNextOpen:       MsgNextSeasonOpen,
		season.ErrDeleteOpened:   MsgDeleteOpened,
		season.ErrDeleteLastOnly: MsgDeleteLastOnly,
	} {
		if errors.Is(err, code) {
			return huma.Error409Conflict(msg)
		}
	}
	return err
}

// logSeasonAction trace une écriture réussie sur une saison (research.md 003 ›
// R13).
func (s *server) logSeasonAction(ctx context.Context, action string, seasonID int64, attrs ...any) {
	attrs = append([]any{"action", action, "actor_id", CurrentAccount(ctx).ID, "season_id", seasonID}, attrs...)
	s.Logger.InfoContext(ctx, "action d'administration sur une saison", attrs...)
}

func (s *server) registerSeasons(api huma.API) {
	register(api, auth.Admin, huma.Operation{
		OperationID: "list-seasons",
		Method:      http.MethodGet,
		Path:        "/admin/seasons",
		Summary:     "Saisons de l'association",
		Tags:        []string{tagSeasons},
	}, s.listSeasons)

	register(api, auth.Admin, huma.Operation{
		OperationID: "get-season",
		Method:      http.MethodGet,
		Path:        "/admin/seasons/{id}",
		Summary:     "Fiche d'une saison",
		Tags:        []string{tagSeasons},
	}, s.getSeason)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "create-season",
		Method:        http.MethodPost,
		Path:          "/admin/seasons",
		Summary:       "Créer une saison",
		Tags:          []string{tagSeasons},
		DefaultStatus: http.StatusCreated,
	}, s.createSeason)

	register(api, auth.Admin, huma.Operation{
		OperationID: "update-season",
		Method:      http.MethodPatch,
		Path:        "/admin/seasons/{id}",
		Summary:     "Modifier les dates d'une saison",
		Tags:        []string{tagSeasons},
	}, s.updateSeason)

	register(api, auth.Admin, huma.Operation{
		OperationID: "open-season",
		Method:      http.MethodPost,
		Path:        "/admin/seasons/{id}/open",
		Summary:     "Ouvrir une saison",
		Tags:        []string{tagSeasons},
	}, s.openSeason)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "duplicate-season",
		Method:        http.MethodPost,
		Path:          "/admin/seasons/{id}/duplicate",
		Summary:       "Créer la saison suivante à partir d'une saison",
		Tags:          []string{tagSeasons},
		DefaultStatus: http.StatusCreated,
	}, s.duplicateSeason)

	register(api, auth.Admin, huma.Operation{
		OperationID:   "delete-season",
		Method:        http.MethodDelete,
		Path:          "/admin/seasons/{id}",
		Summary:       "Supprimer une saison en brouillon",
		Tags:          []string{tagSeasons},
		DefaultStatus: http.StatusNoContent,
	}, s.deleteSeason)
}

type SeasonListOutput struct {
	Body SeasonList
}

func (s *server) listSeasons(ctx context.Context, _ *struct{}) (*SeasonListOutput, error) {
	rows, err := s.Store.Read.ListSeasons(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	list, err := buildSeasonList(rows, s.today())
	if err != nil {
		return nil, s.internalError(ctx, "saisons illisibles", err)
	}
	return &SeasonListOutput{Body: list}, nil
}

type SeasonIDInput struct {
	ID int64 `path:"id" doc:"Identifiant de la saison"`
}

type SeasonOutput struct {
	Body Season
}

func (s *server) getSeason(ctx context.Context, in *SeasonIDInput) (*SeasonOutput, error) {
	return s.seasonOutput(ctx, in.ID)
}

// seasonOutput relit la suite et renvoie la saison id, avec des actions
// calculées sur l'état validé.
func (s *server) seasonOutput(ctx context.Context, id int64) (*SeasonOutput, error) {
	rows, err := s.Store.Read.ListSeasons(ctx)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des saisons impossible", err)
	}
	items, err := buildSeasons(rows, s.today())
	if err != nil {
		return nil, s.internalError(ctx, "saisons illisibles", err)
	}
	i := slices.IndexFunc(items, func(it Season) bool { return it.ID == id })
	if i < 0 {
		return nil, huma.Error404NotFound(MsgSeasonGone)
	}
	return &SeasonOutput{Body: items[i]}, nil
}

// seasonWrite exécute fn dans une transaction d'écriture sur la suite relue
// (research.md 003 › R5). id nul : aucune saison désignée (création), i vaut -1.
func (s *server) seasonWrite(ctx context.Context, id int64, fn func(q *sqlc.Queries, seasons []season.Season, i int, today season.Date) error) error {
	today := s.today()
	err := s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		rows, err := q.ListSeasons(ctx)
		if err != nil {
			return err
		}
		seasons, err := toDomain(rows)
		if err != nil {
			return err
		}
		i := -1
		if id != 0 {
			if i = indexOf(seasons, id); i < 0 {
				return huma.Error404NotFound(MsgSeasonGone)
			}
		}
		return seasonError(fn(q, seasons, i, today))
	})
	if err != nil {
		return s.txError(ctx, "écriture de la saison impossible", err)
	}
	return nil
}

type SeasonDatesBody struct {
	StartsOn string `json:"startsOn" format:"date" doc:"Premier jour, inclus"`
	EndsOn   string `json:"endsOn" format:"date" doc:"Dernier jour, inclus"`
}

type CreateSeasonInput struct {
	Body SeasonDatesBody
}

func parseField(field, value string) (season.Date, error) {
	d, err := season.ParseDate(value)
	if err != nil {
		return season.Date{}, FieldErrors("body."+field, MsgInvalidDate)
	}
	return d, nil
}

func (s *server) createSeason(ctx context.Context, in *CreateSeasonInput) (*SeasonOutput, error) {
	start, err := parseField("startsOn", in.Body.StartsOn)
	if err != nil {
		return nil, err
	}
	end, err := parseField("endsOn", in.Body.EndsOn)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.seasonWrite(ctx, 0, func(q *sqlc.Queries, seasons []season.Season, _ int, today season.Date) error {
		if err := season.ValidateCreate(seasons, start, end, today); err != nil {
			return err
		}
		row, err := q.CreateSeason(ctx, sqlc.CreateSeasonParams{StartsOn: start.String(), EndsOn: end.String(), CreatedAt: nowString()})
		id = row.ID
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logSeasonAction(ctx, "season.create", id, "starts_on", start.String(), "ends_on", end.String())
	return s.seasonOutput(ctx, id)
}

type UpdateSeasonInput struct {
	ID   int64 `path:"id" doc:"Identifiant de la saison"`
	Body struct {
		StartsOn string `json:"startsOn,omitempty" required:"false" format:"date" doc:"Nouveau premier jour (première saison en brouillon uniquement) ; absent = inchangé"`
		EndsOn   string `json:"endsOn" format:"date" doc:"Nouveau dernier jour, inclus"`
	}
}

func (s *server) updateSeason(ctx context.Context, in *UpdateSeasonInput) (*SeasonOutput, error) {
	var start *season.Date
	if in.Body.StartsOn != "" {
		d, err := parseField("startsOn", in.Body.StartsOn)
		if err != nil {
			return nil, err
		}
		start = &d
	}
	end, err := parseField("endsOn", in.Body.EndsOn)
	if err != nil {
		return nil, err
	}
	var plan season.Plan
	err = s.seasonWrite(ctx, in.ID, func(q *sqlc.Queries, seasons []season.Season, i int, today season.Date) error {
		if season.StateOf(seasons[i], today) == season.Closed {
			return huma.Error409Conflict(MsgSeasonClosed)
		}
		if plan, err = season.ValidateUpdate(seasons, i, start, end, today); err != nil {
			return err
		}
		if err := q.UpdateSeasonDates(ctx, sqlc.UpdateSeasonDatesParams{ID: in.ID, StartsOn: plan.StartsOn.String(), EndsOn: plan.EndsOn.String()}); err != nil {
			return err
		}
		if plan.NextID == 0 {
			return nil
		}
		next := seasons[i+1]
		return q.UpdateSeasonDates(ctx, sqlc.UpdateSeasonDatesParams{ID: next.ID, StartsOn: plan.NextStartsOn.String(), EndsOn: next.EndsOn.String()})
	})
	if err != nil {
		return nil, err
	}
	attrs := []any{"starts_on", plan.StartsOn.String(), "ends_on", plan.EndsOn.String()}
	if plan.NextID != 0 {
		attrs = append(attrs, "shifted_season_id", plan.NextID, "shifted_starts_on", plan.NextStartsOn.String())
	}
	s.logSeasonAction(ctx, "season.update", in.ID, attrs...)
	return s.seasonOutput(ctx, in.ID)
}

func (s *server) openSeason(ctx context.Context, in *SeasonIDInput) (*SeasonOutput, error) {
	changed := false
	err := s.seasonWrite(ctx, in.ID, func(q *sqlc.Queries, seasons []season.Season, i int, today season.Date) error {
		if err := season.CanOpen(seasons, i, today); err != nil {
			if errors.Is(err, season.ErrClosed) {
				return huma.Error409Conflict(MsgSeasonClosedOpen)
			}
			return err
		}
		n, err := q.OpenSeason(ctx, sqlc.OpenSeasonParams{ID: in.ID, Now: sql.NullString{String: nowString(), Valid: true}})
		changed = n > 0
		return err
	})
	if err != nil {
		return nil, err
	}
	if changed {
		s.logSeasonAction(ctx, "season.open", in.ID)
	}
	return s.seasonOutput(ctx, in.ID)
}

func (s *server) duplicateSeason(ctx context.Context, in *SeasonIDInput) (*SeasonOutput, error) {
	var id int64
	var start, end season.Date
	err := s.seasonWrite(ctx, in.ID, func(q *sqlc.Queries, seasons []season.Season, i int, _ season.Date) error {
		var err error
		if start, end, err = season.ValidateDuplicate(seasons, i); err != nil {
			return err
		}
		row, err := q.CreateSeason(ctx, sqlc.CreateSeasonParams{StartsOn: start.String(), EndsOn: end.String(), CreatedAt: nowString()})
		id = row.ID
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logSeasonAction(ctx, "season.duplicate", id, "source_id", in.ID, "starts_on", start.String(), "ends_on", end.String())
	return s.seasonOutput(ctx, id)
}

func (s *server) deleteSeason(ctx context.Context, in *SeasonIDInput) (*struct{}, error) {
	err := s.seasonWrite(ctx, in.ID, func(q *sqlc.Queries, seasons []season.Season, i int, _ season.Date) error {
		if err := season.CanDelete(seasons, i); err != nil {
			return err
		}
		_, err := q.DeleteSeason(ctx, in.ID)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.logSeasonAction(ctx, "season.delete", in.ID)
	return nil, nil
}
