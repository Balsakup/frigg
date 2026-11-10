package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/csvfile"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/database/sqlc"
	"github.com/balsakup/frigg/internal/memberimport"
)

// Importation des membres depuis un fichier CSV (research.md 013 › R1 à
// R11). Aucun état côté serveur : le navigateur renvoie le fichier à chaque
// étape, qui est relu, revalidé puis oublié ; il n'est jamais écrit sur
// disque (FR-021).

const (
	MsgImportNotCSV         = "Ce fichier n'est pas un fichier CSV lisible. Dans votre tableur, choisissez « Enregistrer sous » puis le format CSV."
	MsgImportMalformed      = "Guillemet non refermé à la ligne %d : vérifiez cette ligne dans votre tableur."
	MsgImportNoData         = "Le fichier ne contient aucune ligne de membre sous la ligne d'intitulés."
	MsgImportTooLarge       = "Le fichier dépasse 5 Mo."
	MsgImportTooManyRows    = "Le fichier dépasse 2 000 lignes : découpez-le en plusieurs fichiers."
	MsgImportTooManyColumns = "Le fichier dépasse 100 colonnes."
	MsgImportEmpty          = "Le fichier est vide."
	MsgImportMappingJSON    = "La correspondance des colonnes est illisible."
	MsgImportNothing        = "Aucune ligne à importer."
	MsgImportFieldsChanged  = "Les champs personnalisés ont changé depuis l'aperçu. L'aperçu a été recalculé."
)

const (
	importSamples      = 3
	importSampleLength = 80
)

// ImportColumn est une colonne du fichier.
type ImportColumn struct {
	Index   int      `json:"index"`
	Header  string   `json:"header" doc:"Intitulé tel que lu ; vide s'il est absent"`
	Label   string   `json:"label" doc:"Intitulé, ou « Colonne 3 » s'il est vide ou en double"`
	Samples []string `json:"samples" doc:"Au plus 3 premières valeurs non vides, 80 caractères au plus chacune"`
}

// ImportTargetOption est un champ Frigg auquel associer une colonne.
type ImportTargetOption struct {
	Target            string  `json:"target" doc:"firstName, lastName… ou field:<id>"`
	Label             string  `json:"label"`
	RequiredForImport bool    `json:"requiredForImport" doc:"Vrai : une colonne doit lui être associée"`
	FieldType         *string `json:"fieldType" enum:"short_text,long_text,number,date,bool,choice" doc:"Type du champ personnalisé ; null pour un champ de la fiche"`
}

// ImportFile décrit le fichier lu.
type ImportFile struct {
	Encoding  string `json:"encoding" enum:"utf-8,windows-1252"`
	Separator string `json:"separator" doc:"« ; », « , » ou tabulation"`
	RowCount  int    `json:"rowCount" doc:"Lignes de données non vides"`
}

// ImportCellError est une erreur d'une ligne.
type ImportCellError struct {
	Column      *int    `json:"column" doc:"Index de la colonne ; null : ligne entière"`
	ColumnLabel *string `json:"columnLabel"`
	Message     string  `json:"message"`
}

// ImportMemberRef désigne la fiche existante d'un doublon.
type ImportMemberRef struct {
	ID    int64  `json:"id"`
	Label string `json:"label" doc:"« Dupont Marie (12/03/2012) »"`
}

// ImportRow est une ligne de l'aperçu.
type ImportRow struct {
	Line              int               `json:"line" doc:"Numéro de ligne dans le tableur (intitulés = 1)"`
	Status            string            `json:"status" enum:"ready,error,duplicate"`
	FirstName         string            `json:"firstName" doc:"Valeur brute de la colonne associée"`
	LastName          string            `json:"lastName"`
	BirthDate         string            `json:"birthDate"`
	Errors            []ImportCellError `json:"errors"`
	DuplicateOfMember *ImportMemberRef  `json:"duplicateOfMember"`
	DuplicateOfLine   *int              `json:"duplicateOfLine"`
	MissingRequired   []int64           `json:"missingRequired" doc:"Champs personnalisés obligatoires sans valeur (lignes ready seulement)"`
}

// ImportMissing compte les lignes à importer sans valeur pour un champ
// personnalisé obligatoire.
type ImportMissing struct {
	FieldID int64  `json:"fieldId"`
	Label   string `json:"label"`
	Count   int    `json:"count"`
}

// ImportCounts compte les lignes par état.
type ImportCounts struct {
	Ready      int `json:"ready"`
	Errors     int `json:"errors"`
	Duplicates int `json:"duplicates"`
}

// ImportAnalysis est le résultat de la lecture du fichier.
type ImportAnalysis struct {
	File          ImportFile           `json:"file"`
	Columns       []ImportColumn       `json:"columns"`
	Suggested     []string             `json:"suggested" doc:"Cible proposée par colonne ; vide : ne pas importer"`
	Targets       []ImportTargetOption `json:"targets"`
	FieldsVersion string               `json:"fieldsVersion"`
}

// ImportPreview est l'aperçu complet.
type ImportPreview struct {
	File            ImportFile      `json:"file"`
	Counts          ImportCounts    `json:"counts"`
	MissingRequired []ImportMissing `json:"missingRequired"`
	IncompleteCount int             `json:"incompleteCount" doc:"Lignes à importer auxquelles manque au moins une valeur obligatoire"`
	Rows            []ImportRow     `json:"rows"`
	FieldsVersion   string          `json:"fieldsVersion"`
}

// ImportReport est le rapport d'un import confirmé.
type ImportReport struct {
	Created         int             `json:"created"`
	Errors          int             `json:"errors"`
	Duplicates      int             `json:"duplicates"`
	NewDuplicates   int             `json:"newDuplicates" doc:"Lignes à importer à l'aperçu devenues doublons"`
	MissingRequired []ImportMissing `json:"missingRequired"`
	IncompleteCount int             `json:"incompleteCount"`
}

// ImportFileInput est un fichier déposé.
type ImportFileInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" required:"true"`
	}]
}

// ImportMappingInput est un fichier et sa correspondance.
type ImportMappingInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File    huma.FormFile `form:"file" required:"true"`
		Mapping string        `form:"mapping" required:"true" maxLength:"4096" doc:"Tableau JSON des cibles, une par colonne"`
	}]
}

// ImportRunInput confirme un import.
type ImportRunInput struct {
	RawBody huma.MultipartFormFiles[struct {
		File          huma.FormFile `form:"file" required:"true"`
		Mapping       string        `form:"mapping" required:"true" maxLength:"4096" doc:"Tableau JSON des cibles, une par colonne"`
		FieldsVersion string        `form:"fieldsVersion" required:"true" pattern:"^[0-9a-f]{32}$" patternDescription:"l'empreinte renvoyée par l'aperçu"`
		ExpectedReady int           `form:"expectedReady" required:"false" minimum:"0" maximum:"2000" doc:"Lignes à importer affichées par l'aperçu ; sert à compter les doublons apparus depuis"`
	}]
}

type importAnalysisOutput struct{ Body ImportAnalysis }
type importPreviewOutput struct{ Body ImportPreview }
type importReportOutput struct{ Body ImportReport }

func csvResponse(description string) map[string]*huma.Response {
	return map[string]*huma.Response{"200": {
		Description: description,
		Content:     map[string]*huma.MediaType{"text/csv": {Schema: &huma.Schema{Type: "string", Format: "binary"}}},
	}}
}

func (s *server) registerMemberImport(api huma.API) {
	op := func(id, method, path, summary string, status int) huma.Operation {
		o := huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{tagMembers}, DefaultStatus: status}
		if method == http.MethodPost {
			o.MaxBodyBytes = UploadMaxBody
		}
		return o
	}
	template := op("get-member-import-template", http.MethodGet, "/admin/members/import/template", "Modèle de fichier d'import (CSV)", 0)
	template.Responses = csvResponse("Modèle de fichier (UTF-8, séparateur point-virgule)")
	register(api, auth.Admin, template, s.memberImportTemplate)
	register(api, auth.Admin, op("analyze-member-import", http.MethodPost, "/admin/members/import/analyze", "Lire un fichier d'import", 0), s.analyzeMemberImport)
	register(api, auth.Admin, op("preview-member-import", http.MethodPost, "/admin/members/import/preview", "Aperçu d'un import", 0), s.previewMemberImport)
	rejected := op("export-member-import-rejected", http.MethodPost, "/admin/members/import/rejected", "Lignes rejetées d'un import (CSV)", 0)
	rejected.Responses = csvResponse("Lignes rejetées (UTF-8, séparateur point-virgule)")
	register(api, auth.Admin, rejected, s.exportMemberImportRejected)
	register(api, auth.Admin, op("run-member-import", http.MethodPost, "/admin/members/import", "Importer des membres", http.StatusCreated), s.runMemberImport)
}

// readImportFile lit le fichier déposé en mémoire, sans l'écrire sur disque.
func readImportFile(f huma.FormFile) (csvfile.Table, error) {
	defer f.Close()
	limit := csvfile.DefaultLimits.MaxBytes
	if f.Size > limit {
		return csvfile.Table{}, FieldErrors("form.file", MsgImportTooLarge)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return csvfile.Table{}, FieldErrors("form.file", MsgImportNotCSV)
	}
	t, err := csvfile.Decode(data, csvfile.DefaultLimits)
	if err == nil {
		return t, nil
	}
	msg := MsgImportNotCSV
	var me *csvfile.MalformedError
	switch {
	case errors.As(err, &me):
		msg = fmt.Sprintf(MsgImportMalformed, me.Line)
	case errors.Is(err, csvfile.ErrEmpty):
		msg = MsgImportEmpty
	case errors.Is(err, csvfile.ErrNoData):
		msg = MsgImportNoData
	case errors.Is(err, csvfile.ErrTooLarge):
		msg = MsgImportTooLarge
	case errors.Is(err, csvfile.ErrTooManyRows):
		msg = MsgImportTooManyRows
	case errors.Is(err, csvfile.ErrTooManyColumns):
		msg = MsgImportTooManyColumns
	}
	return csvfile.Table{}, FieldErrors("form.file", msg)
}

// parseMapping lit et vérifie la correspondance envoyée en JSON.
func parseMapping(raw string, labels []string, defs []customfield.Definition) ([]memberimport.Target, error) {
	var m []memberimport.Target
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, FieldErrors("form.mapping", MsgImportMappingJSON)
	}
	errs := memberimport.ValidateMapping(m, labels, defs)
	if len(errs) == 0 {
		return m, nil
	}
	details := make([]*huma.ErrorDetail, len(errs))
	for i, e := range errs {
		loc := "form.mapping"
		if e.Index >= 0 {
			loc = fmt.Sprintf("form.mapping[%d]", e.Index)
		}
		details[i] = &huma.ErrorDetail{Location: loc, Message: e.Message}
	}
	return nil, FieldErrorDetails(details...)
}

// importDefinitions sont les champs personnalisés de membre actifs, quelle
// que soit leur visibilité.
func importDefinitions(ctx context.Context, q *sqlc.Queries) ([]customfield.Definition, error) {
	return loadDefinitions(ctx, q, customfield.ScopeMember, customfield.Admin)
}

// classifyImport lit, valide et classe toutes les lignes avec les fiches
// existantes lues par q.
func (s *server) classifyImport(ctx context.Context, q *sqlc.Queries, t csvfile.Table, m []memberimport.Target, defs []customfield.Definition) ([]memberimport.Result, error) {
	rows, err := q.ListMemberIdentities(ctx)
	if err != nil {
		return nil, err
	}
	existing := make([]memberimport.Identity, len(rows))
	for i, r := range rows {
		existing[i] = memberimport.Identity{ID: r.ID, FirstName: r.FirstName, LastName: r.LastName, BirthDate: r.BirthDate}
	}
	today := s.today()
	parsed := make([]memberimport.Parsed, len(t.Rows))
	for i, r := range t.Rows {
		parsed[i] = memberimport.ParseRow(r, m, defs, today)
	}
	return memberimport.Classify(parsed, existing), nil
}

// importMessage traduit une erreur de ligne ; les erreurs de fiche reprennent
// les messages du formulaire de fiche (memberMessages).
func importMessage(e memberimport.CellError) string {
	if msgs, ok := memberMessages[e.Err]; ok {
		if msg, ok := msgs[e.Field]; ok {
			return msg
		}
		if msg, ok := msgs[""]; ok {
			return msg
		}
	}
	return e.Err.Error()
}

func importFileInfo(t csvfile.Table) ImportFile {
	return ImportFile{Encoding: t.Encoding, Separator: string(t.Separator), RowCount: len(t.Rows)}
}

// importForm est un fichier lu et sa correspondance vérifiée.
type importForm struct {
	table  csvfile.Table
	labels []string
	m      []memberimport.Target
	defs   []customfield.Definition
}

// readImportForm lit le fichier et la correspondance avec les champs
// personnalisés courants. Les erreurs de fichier et de correspondance sont
// des 422.
func (s *server) readImportForm(ctx context.Context, f huma.FormFile, mapping string) (importForm, error) {
	t, err := readImportFile(f)
	if err != nil {
		return importForm{}, err
	}
	defs, err := importDefinitions(ctx, s.Store.Read)
	if err != nil {
		return importForm{}, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	labels := memberimport.ColumnLabels(t.Headers)
	m, err := parseMapping(mapping, labels, defs)
	if err != nil {
		return importForm{}, err
	}
	return importForm{table: t, labels: labels, m: m, defs: defs}, nil
}

func (s *server) analyzeMemberImport(ctx context.Context, in *ImportFileInput) (*importAnalysisOutput, error) {
	t, err := readImportFile(in.RawBody.Data().File)
	if err != nil {
		return nil, err
	}
	defs, err := importDefinitions(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	labels := memberimport.ColumnLabels(t.Headers)
	columns := make([]ImportColumn, len(t.Headers))
	for i, h := range t.Headers {
		samples := []string{}
		for _, r := range t.Rows {
			if len(samples) == importSamples {
				break
			}
			if v := r.Cells[i]; v != "" {
				samples = append(samples, truncateRunes(v, importSampleLength))
			}
		}
		columns[i] = ImportColumn{Index: i, Header: h, Label: labels[i], Samples: samples}
	}
	suggested := memberimport.Suggest(t.Headers, defs)
	out := ImportAnalysis{
		File:          importFileInfo(t),
		Columns:       columns,
		Suggested:     make([]string, len(suggested)),
		Targets:       make([]ImportTargetOption, 0, len(memberimport.StandardTargets)+len(defs)),
		FieldsVersion: memberimport.Fingerprint(defs),
	}
	for i, tg := range suggested {
		out.Suggested[i] = string(tg)
	}
	for _, st := range memberimport.StandardTargets {
		out.Targets = append(out.Targets, ImportTargetOption{Target: string(st.Target), Label: st.Label, RequiredForImport: memberimport.IsRequired(st.Target)})
	}
	for _, d := range defs {
		typ := string(d.Type)
		out.Targets = append(out.Targets, ImportTargetOption{Target: string(memberimport.FieldTarget(d.ID)), Label: d.Label, FieldType: &typ})
	}
	return &importAnalysisOutput{Body: out}, nil
}

// missingSummary compte, parmi les lignes à importer, les valeurs
// obligatoires manquantes par champ, dans l'ordre des champs.
func missingSummary(results []memberimport.Result, defs []customfield.Definition) ([]ImportMissing, int) {
	counts := map[int64]int{}
	incomplete := 0
	for _, r := range results {
		if r.Status != memberimport.StatusReady || len(r.MissingRequired) == 0 {
			continue
		}
		incomplete++
		for _, id := range r.MissingRequired {
			counts[id]++
		}
	}
	out := []ImportMissing{}
	for _, d := range defs {
		if n := counts[d.ID]; n > 0 {
			out = append(out, ImportMissing{FieldID: d.ID, Label: d.Label, Count: n})
		}
	}
	return out, incomplete
}

func importRow(r memberimport.Result, labels []string) ImportRow {
	row := ImportRow{
		Line: r.Line, Status: string(r.Status),
		FirstName: r.Raw.FirstName, LastName: r.Raw.LastName, BirthDate: r.Raw.BirthDate,
		Errors: []ImportCellError{}, MissingRequired: []int64{},
	}
	for _, e := range r.Errors {
		ce := ImportCellError{Message: importMessage(e)}
		if e.Column >= 0 {
			col, label := e.Column, labels[e.Column]
			ce.Column, ce.ColumnLabel = &col, &label
		}
		row.Errors = append(row.Errors, ce)
	}
	if d := r.DuplicateOfMember; d != nil {
		row.DuplicateOfMember = &ImportMemberRef{ID: d.ID, Label: memberimport.MemberLabel(*d)}
	}
	if r.DuplicateOfLine > 0 {
		line := r.DuplicateOfLine
		row.DuplicateOfLine = &line
	}
	if r.Status == memberimport.StatusReady && len(r.MissingRequired) > 0 {
		row.MissingRequired = r.MissingRequired
	}
	return row
}

func (s *server) previewMemberImport(ctx context.Context, in *ImportMappingInput) (*importPreviewOutput, error) {
	form := in.RawBody.Data()
	f, err := s.readImportForm(ctx, form.File, form.Mapping)
	if err != nil {
		return nil, err
	}
	results, err := s.classifyImport(ctx, s.Store.Read, f.table, f.m, f.defs)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches impossible", err)
	}
	ready, errs, dups := memberimport.Counts(results)
	missing, incomplete := missingSummary(results, f.defs)
	out := ImportPreview{
		File:            importFileInfo(f.table),
		Counts:          ImportCounts{Ready: ready, Errors: errs, Duplicates: dups},
		MissingRequired: missing,
		IncompleteCount: incomplete,
		Rows:            make([]ImportRow, len(results)),
		FieldsVersion:   memberimport.Fingerprint(f.defs),
	}
	for i, r := range results {
		out.Rows[i] = importRow(r, f.labels)
	}
	return &importPreviewOutput{Body: out}, nil
}

func (s *server) runMemberImport(ctx context.Context, in *ImportRunInput) (*importReportOutput, error) {
	form := in.RawBody.Data()
	f, err := s.readImportForm(ctx, form.File, form.Mapping)
	if err != nil {
		return nil, err
	}
	var results []memberimport.Result
	var defs []customfield.Definition
	var first, last int64
	err = s.Store.WriteTx(ctx, func(q *sqlc.Queries) error {
		var err error
		if defs, err = importDefinitions(ctx, q); err != nil {
			return err
		}
		if memberimport.Fingerprint(defs) != form.FieldsVersion {
			return huma.Error409Conflict(MsgImportFieldsChanged)
		}
		// Identités relues dans la transaction : l'écrivain unique sérialise
		// les imports et les saisies, aucun doublon n'est créé (FR-015).
		if results, err = s.classifyImport(ctx, q, f.table, f.m, defs); err != nil {
			return err
		}
		now := nowString()
		for _, r := range results {
			if r.Status != memberimport.StatusReady {
				continue
			}
			c := fieldsOf(r.Input)
			id, err := q.CreateMember(ctx, sqlc.CreateMemberParams{
				AccountID: sql.NullInt64{},
				FirstName: c.FirstName, LastName: c.LastName, BirthDate: c.BirthDate,
				Email: c.Email, Phone: c.Phone, AddressLine1: c.AddressLine1, AddressLine2: c.AddressLine2,
				PostalCode: c.Postal, City: c.City, Now: now,
			})
			if err != nil {
				return err
			}
			if first == 0 {
				first = id
			}
			last = id
			for fieldID, v := range r.Fields {
				if err := q.UpsertMemberFieldValue(ctx, fieldValueParams(id, fieldID, v, now)); err != nil {
					return err
				}
			}
		}
		if first == 0 {
			return huma.Error422UnprocessableEntity(MsgImportNothing)
		}
		return nil
	})
	if err != nil {
		return nil, s.txError(ctx, "importation des membres impossible", err)
	}
	ready, errs, dups := memberimport.Counts(results)
	// Sans état côté serveur, l'aperçu renvoie son nombre de lignes à
	// importer : celles qui manquent sont devenues des doublons entre-temps
	// (une erreur nouvelle supposerait des champs changés, donc un 409).
	newDups := max(form.ExpectedReady-ready, 0)
	missing, incomplete := missingSummary(results, defs)
	s.Logger.InfoContext(ctx, "importation de membres", "action", "member.import", "actor_id", CurrentAccount(ctx).ID,
		"created", ready, "errors", errs, "duplicates", dups, "new_duplicates", newDups,
		"missing_required", incomplete, "first_member_id", first, "last_member_id", last)
	return &importReportOutput{Body: ImportReport{
		Created: ready, Errors: errs, Duplicates: dups, NewDuplicates: newDups,
		MissingRequired: missing, IncompleteCount: incomplete,
	}}, nil
}

// csvDownload envoie un CSV en pièce jointe.
func csvDownload(data []byte, filename string) *huma.StreamResponse {
	return &huma.StreamResponse{Body: func(hctx huma.Context) {
		hctx.SetHeader("Content-Type", "text/csv; charset=utf-8")
		hctx.SetHeader("Content-Length", strconv.Itoa(len(data)))
		hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		hctx.SetHeader("X-Content-Type-Options", "nosniff")
		hctx.SetHeader("Cache-Control", "private, no-store")
		hctx.SetStatus(http.StatusOK)
		_, _ = hctx.BodyWriter().Write(data)
	}}
}

func (s *server) exportMemberImportRejected(ctx context.Context, in *ImportMappingInput) (*huma.StreamResponse, error) {
	form := in.RawBody.Data()
	f, err := s.readImportForm(ctx, form.File, form.Mapping)
	if err != nil {
		return nil, err
	}
	results, err := s.classifyImport(ctx, s.Store.Read, f.table, f.m, f.defs)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des fiches impossible", err)
	}
	var buf bytes.Buffer
	w := csvfile.NewWriter(&buf)
	_ = w.Write(memberimport.RejectedHeaders(f.table))
	for i, r := range results {
		if r.Status != memberimport.StatusReady {
			_ = w.Write(memberimport.RejectedRecord(r, f.table.Rows[i], f.labels, importMessage))
		}
	}
	w.Flush()
	return csvDownload(buf.Bytes(), "lignes-rejetees.csv"), nil
}

func (s *server) memberImportTemplate(ctx context.Context, _ *struct{}) (*huma.StreamResponse, error) {
	defs, err := importDefinitions(ctx, s.Store.Read)
	if err != nil {
		return nil, s.internalError(ctx, "lecture des champs personnalisés impossible", err)
	}
	var buf bytes.Buffer
	w := csvfile.NewWriter(&buf)
	_ = w.Write(memberimport.TemplateHeaders(defs))
	w.Flush()
	return csvDownload(buf.Bytes(), "modele-import-membres.csv"), nil
}
