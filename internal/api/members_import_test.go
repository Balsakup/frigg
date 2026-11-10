package api_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
	"github.com/balsakup/frigg/internal/csvfile"
)

func newImportEnv(t *testing.T) (*apitest.Env, *apitest.Client) {
	t.Helper()
	env := apitest.New(t)
	env.SetToday("2026-11-10")
	return env, env.LoggedIn(env.Seed(auth.RoleAdmin, true))
}

func identityCount(t *testing.T, env *apitest.Env) int {
	t.Helper()
	rows, err := env.Store.Read.ListMemberIdentities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return len(rows)
}

func TestImportAnalyze(t *testing.T) {
	_, admin := newImportEnv(t)
	data := win1252(t, "NOM;Prénom;Date de naissance;Tél.;Remarques\r\n"+
		"Lefèvre;Léa;03/04/2014;06 12 34 56 78;\r\n"+
		"Dupont;Hélène;12/03/2012;;"+strings.Repeat("é", 100)+"\r\n"+
		"Martin;Paul;3/4/2012;;\r\n"+
		"Durand;Hugo;30/06/2011;;\r\n")
	a := expectJSON[api.ImportAnalysis](t, analyzeImport(t, admin, data), http.StatusOK)
	if a.File.Encoding != "windows-1252" || a.File.Separator != ";" || a.File.RowCount != 4 {
		t.Errorf("fichier %+v", a.File)
	}
	if got := a.Columns[1].Samples; !slices.Equal(got, []string{"Léa", "Hélène", "Paul"}) {
		t.Errorf("exemples %q", got)
	}
	if n := len([]rune(a.Columns[4].Samples[0])); n != 80 {
		t.Errorf("exemple de %d caractères, attendu 80", n)
	}
	if !slices.Equal(a.Suggested, []string{"lastName", "firstName", "birthDate", "phone", ""}) {
		t.Errorf("proposition %q", a.Suggested)
	}
	if a.Targets[0].Target != "firstName" || !a.Targets[0].RequiredForImport || !a.Targets[2].RequiredForImport || a.Targets[3].RequiredForImport || len(a.Targets) != 9 {
		t.Errorf("cibles %+v", a.Targets)
	}
	if len(a.FieldsVersion) != 32 {
		t.Errorf("empreinte %q", a.FieldsVersion)
	}
}

func TestImportFileRejected(t *testing.T) {
	_, admin := newImportEnv(t)
	many := func(n, cols int) []byte {
		var b strings.Builder
		b.WriteString(strings.TrimSuffix(strings.Repeat("a;", cols), ";") + "\n")
		for range n {
			b.WriteString("x\n")
		}
		return []byte(b.String())
	}
	cases := []struct {
		name string
		data []byte
		msg  string
	}{
		{"vide", []byte(" "), api.MsgImportEmpty},
		{"en-tête seul", csvBytes(standardHeader), api.MsgImportNoData},
		{"classeur renommé", []byte("PK\x03\x04\x14\x00\x06\x00"), api.MsgImportNotCSV},
		{"guillemet non refermé", csvBytes(standardHeader, "A;B;01/01/2000;;", "\"C;D;01/01/2000;;"), "Guillemet non refermé à la ligne 3"},
		{"2 001 lignes", many(2001, 1), api.MsgImportTooManyRows},
		{"101 colonnes", many(1, 101), api.MsgImportTooManyColumns},
		{"trop gros", bytes.Repeat([]byte("a"), 5<<20+1), api.MsgImportTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := analyzeImport(t, admin, c.data)
			if res.Status != http.StatusUnprocessableEntity {
				t.Fatalf("statut %d : %s", res.Status, res.Body)
			}
			if p := problemOf(t, res); !hasError(p, "form.file", c.msg) {
				t.Errorf("erreurs %+v, attendu %q", p.Errors, c.msg)
			}
		})
	}
}

func TestImportPreview(t *testing.T) {
	env, admin := newImportEnv(t)
	insertOrphan(t, env, "Marie", "Dupont", "2012-03-12")
	before := identityCount(t, env)
	data := csvBytes(standardHeader,
		"Léa;Lefèvre;03/04/2014;lea@example.org;06 12 34 56 78",
		"",
		"Marie;DUPONT;12/03/2012;;",
		"Paul;Martin;3/4/12;paul.example.org;",
	)
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, standardMapping), http.StatusOK)
	if p.Counts != (api.ImportCounts{Ready: 1, Errors: 1, Duplicates: 1}) {
		t.Errorf("compteurs %+v", p.Counts)
	}
	lines := []int{}
	for _, r := range p.Rows {
		lines = append(lines, r.Line)
	}
	if !slices.Equal(lines, []int{2, 4, 5}) {
		t.Errorf("lignes %v", lines)
	}
	if p.Rows[0].Status != "ready" || p.Rows[0].FirstName != "Léa" || len(p.Rows[0].Errors) != 0 || p.Rows[0].MissingRequired == nil {
		t.Errorf("ligne prête %+v", p.Rows[0])
	}
	if identityCount(t, env) != before {
		t.Error("l'aperçu a créé des fiches")
	}
}

func TestImportMappingErrors(t *testing.T) {
	_, admin := newImportEnv(t)
	data := csvBytes(standardHeader, "Léa;Lefèvre;03/04/2014;;")
	cases := []struct {
		name     string
		mapping  string
		location string
		msg      string
	}{
		{"JSON invalide", "[firstName", "form.mapping", api.MsgImportMappingJSON},
		{"cible inconnue", `["firstName","lastName","foo","email","phone"]`, "form.mapping[2]", "n'existe plus"},
		{"même cible", `["firstName","lastName","birthDate","lastName",""]`, "form.mapping[3]", "colonne « Nom »"},
		{"date non associée", `["firstName","lastName","","email","phone"]`, "form.mapping", "Associez une colonne au champ « Date de naissance »."},
		{"longueur", `["firstName"]`, "form.mapping", "toutes les colonnes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := postImport(t, admin, "/preview", map[string]string{"mapping": c.mapping}, data)
			if res.Status != http.StatusUnprocessableEntity {
				t.Fatalf("statut %d : %s", res.Status, res.Body)
			}
			if p := problemOf(t, res); !hasError(p, c.location, c.msg) {
				t.Errorf("erreurs %+v", p.Errors)
			}
		})
	}
}

func TestImportRun(t *testing.T) {
	env, admin := newImportEnv(t)
	data := csvBytes(standardHeader,
		"Léa;Lefèvre;03/04/2014;Lea@Example.org;06 12 34 56 78",
		"Paul;Martin;15/09/2013;;",
		"Hugo;;30/06/2011;;",
	)
	r := expectJSON[api.ImportReport](t, runImport(t, admin, data, standardMapping, importVersion(t, admin)), http.StatusCreated)
	if r.Created != 2 || r.Errors != 1 || r.Duplicates != 0 || r.NewDuplicates != 0 {
		t.Errorf("rapport %+v", r)
	}
	page := adminPage(t, admin, "?account=none")
	if page.Total != 2 {
		t.Fatalf("%d fiches sans compte", page.Total)
	}
	var lea api.AdminMember
	for _, m := range page.Items {
		if m.FirstName == "Léa" {
			lea = m
		}
	}
	if lea.Account != nil || lea.IsSelf || *lea.Email != "lea@example.org" || *lea.Phone != "0612345678" {
		t.Errorf("fiche %+v", lea)
	}
	env.FlushMail()
	if n := len(env.Mail.Messages()); n != 0 {
		t.Errorf("%d e-mails envoyés", n)
	}
	logs := importLogs(env)
	if len(logs) != 1 {
		t.Fatalf("journaux %q", logs)
	}
	for _, want := range []string{`"created":2`, `"errors":1`, `"duplicates":0`, `"first_member_id"`, `"last_member_id"`, `"actor_id"`} {
		if !strings.Contains(logs[0], want) {
			t.Errorf("journal sans %s : %s", want, logs[0])
		}
	}
	for _, forbidden := range []string{"Léa", "Lefèvre", "lea@example", "membres.csv", "Hugo"} {
		if strings.Contains(logs[0], forbidden) {
			t.Errorf("donnée personnelle %q dans le journal", forbidden)
		}
	}
}

// Une ligne déclarée doublon à l'import est aussi refusée comme doublon par
// la création manuelle (même règle, 005).
func TestImportDuplicateRuleMatchesManual(t *testing.T) {
	_, admin := newImportEnv(t)
	data := csvBytes(standardHeader, "  léa ;LEFEVRE;03/04/2014;;")
	expectJSON[api.ImportReport](t, runImport(t, admin, data, standardMapping, importVersion(t, admin)), http.StatusCreated)
	if res := admin.Do("POST", "/api/admin/members", memberBody("Léa", "Lefèvre", "2014-04-03", "accountId", nil)); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("création manuelle : %d", res.Status)
	}
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, standardMapping), http.StatusOK)
	if p.Rows[0].Status != "duplicate" || p.Rows[0].DuplicateOfMember == nil {
		t.Errorf("ligne %+v", p.Rows[0])
	}
}

func TestImportNothing(t *testing.T) {
	env, admin := newImportEnv(t)
	insertOrphan(t, env, "Marie", "Dupont", "2012-03-12")
	res := runImport(t, admin, csvBytes(standardHeader, "Marie;Dupont;12/03/2012;;"), standardMapping, importVersion(t, admin))
	if res.Status != http.StatusUnprocessableEntity || !strings.Contains(res.Body, api.MsgImportNothing) {
		t.Errorf("statut %d : %s", res.Status, res.Body)
	}
	if identityCount(t, env) != 1 {
		t.Error("fiche créée")
	}
}

func TestImportFieldsVersion(t *testing.T) {
	env, admin := newImportEnv(t)
	res := runImport(t, admin, csvBytes(standardHeader, "Léa;Lefèvre;03/04/2014;;"), standardMapping, strings.Repeat("0", 32))
	if res.Status != http.StatusConflict || !strings.Contains(res.Body, api.MsgImportFieldsChanged) {
		t.Errorf("statut %d : %s", res.Status, res.Body)
	}
	if identityCount(t, env) != 0 {
		t.Error("fiche créée")
	}
	if res := runImport(t, admin, csvBytes(standardHeader, "Léa;Lefèvre;03/04/2014;;"), standardMapping, "pas-une-empreinte"); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("empreinte mal formée : %d", res.Status)
	}
}

func TestImportDuplicateSincePreview(t *testing.T) {
	env, admin := newImportEnv(t)
	data := csvBytes(standardHeader, "Léa;Lefèvre;03/04/2014;;", "Paul;Martin;15/09/2013;;")
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, standardMapping), http.StatusOK)
	if p.Counts.Ready != 2 {
		t.Fatalf("aperçu %+v", p.Counts)
	}
	expectMember(t, admin.Do("POST", "/api/admin/members", memberBody("Léa", "Lefèvre", "2014-04-03", "accountId", nil)), http.StatusCreated)
	r := expectJSON[api.ImportReport](t, runImport(t, admin, data, standardMapping, p.FieldsVersion, "expectedReady", "2"), http.StatusCreated)
	if r.Created != 1 || r.Duplicates != 1 || r.NewDuplicates != 1 {
		t.Errorf("rapport %+v", r)
	}
	if identityCount(t, env) != 2 {
		t.Errorf("%d fiches", identityCount(t, env))
	}
}

func TestImportConcurrent(t *testing.T) {
	env, admin := newImportEnv(t)
	lines := []string{standardHeader}
	for i := range 20 {
		lines = append(lines, fmt.Sprintf("P%d;N%d;01/01/2000;;", i, i))
	}
	data := csvBytes(lines...)
	version := importVersion(t, admin)
	var wg sync.WaitGroup
	reports := make([]api.ImportReport, 2)
	statuses := make([]int, 2)
	for i := range 2 {
		wg.Go(func() {
			res := runImport(t, admin, data, standardMapping, version)
			statuses[i] = res.Status
			if res.Status == http.StatusCreated {
				res.JSON(t, &reports[i])
			}
		})
	}
	wg.Wait()
	if total := reports[0].Created + reports[1].Created; total != 20 {
		t.Errorf("créées %d (statuts %v)", total, statuses)
	}
	if n := identityCount(t, env); n != 20 {
		t.Errorf("%d fiches, attendu 20", n)
	}
}

func TestImportAtomic(t *testing.T) {
	env, admin := newImportEnv(t)
	if _, err := env.Store.WriteDB.Exec(`CREATE TRIGGER fail_import BEFORE INSERT ON members WHEN new.last_name = 'Boom' BEGIN SELECT RAISE(ABORT, 'boom'); END`); err != nil {
		t.Fatal(err)
	}
	data := csvBytes(standardHeader, "Léa;Lefèvre;03/04/2014;;", "Bam;Boom;01/01/2000;;", "Paul;Martin;15/09/2013;;")
	if res := runImport(t, admin, data, standardMapping, importVersion(t, admin)); res.Status != http.StatusInternalServerError {
		t.Errorf("statut %d : %s", res.Status, res.Body)
	}
	if n := identityCount(t, env); n != 0 {
		t.Errorf("%d fiches créées", n)
	}
}

// Le fichier n'est jamais écrit sur disque (FR-021).
func TestImportNoFileKept(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	env, admin := newImportEnv(t)
	var seq int
	var name, file string
	if err := env.Store.ReadDB.QueryRow("PRAGMA database_list").Scan(&seq, &name, &file); err != nil {
		t.Fatal(err)
	}
	dbDir := filepath.Dir(file)
	snapshot := func() []string {
		var out []string
		for _, dir := range []string{tmp, dbDir} {
			_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() && !strings.Contains(filepath.Base(p), "frigg.db") {
					out = append(out, p)
				}
				return nil
			})
		}
		return out
	}
	before := snapshot()
	data := csvBytes(standardHeader, "Léa;Lefèvre;03/04/2014;;", "Hugo;;30/06/2011;;")
	analyzeImport(t, admin, data)
	previewImport(t, admin, data, standardMapping)
	rejectedImport(t, admin, data, standardMapping)
	runImport(t, admin, data, standardMapping, importVersion(t, admin))
	if after := snapshot(); !slices.Equal(before, after) {
		t.Errorf("fichiers écrits : avant %q, après %q", before, after)
	}
}

func TestImportPreviewReasons(t *testing.T) {
	env, admin := newImportEnv(t)
	marie := insertOrphan(t, env, "Marie", "Dupont", "2012-03-12")
	data := csvBytes(standardHeader,
		";Martin;3/4/12;paul.example.org;",
		"Marie;Dupont;12/03/2012;;",
		"Paul;Martin;15/09/2013;;",
		"Paul;Martin;15-09-2013;;",
		"Hugo;Durand;30/06/2011;;;surplus",
	)
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, standardMapping), http.StatusOK)
	errs := p.Rows[0].Errors
	if len(errs) != 3 || *errs[0].ColumnLabel != "Prénom" || *errs[1].ColumnLabel != "Date de naissance" || *errs[2].ColumnLabel != "Mail" {
		t.Errorf("erreurs %+v", errs)
	}
	if errs[0].Message != api.MsgFirstNameRequired || errs[2].Message != api.MsgEmailFormat {
		t.Errorf("messages du formulaire de fiche attendus : %+v", errs)
	}
	if d := p.Rows[1].DuplicateOfMember; d == nil || d.ID != marie || d.Label != "Dupont Marie (12/03/2012)" {
		t.Errorf("doublon en base %+v", p.Rows[1])
	}
	if d := p.Rows[3].DuplicateOfLine; d == nil || *d != 4 {
		t.Errorf("doublon dans le fichier %+v", p.Rows[3])
	}
	if e := p.Rows[4].Errors; len(e) != 1 || e[0].Column != nil {
		t.Errorf("ligne longue %+v", e)
	}
}

func TestImportRejectedCSV(t *testing.T) {
	env, admin := newImportEnv(t)
	insertOrphan(t, env, "Marie", "Dupont", "2012-03-12")
	data := csvBytes(standardHeader,
		"Léa;Lefèvre;03/04/2014;;",
		"Hugo;Durand;3/4/12;hugo.example.org;+32 470 12 34 56",
		"Marie;Dupont;12/03/2012;;",
		"Léa;Lefevre;03-04-2014;;",
	)
	res := rejectedImport(t, admin, data, standardMapping)
	if res.Status != http.StatusOK {
		t.Fatalf("statut %d : %s", res.Status, res.Body)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type %q", ct)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "lignes-rejetees.csv") {
		t.Errorf("Content-Disposition %q", cd)
	}
	if !strings.HasPrefix(res.Body, "\xef\xbb\xbf") {
		t.Error("BOM absente")
	}
	lines := strings.Split(strings.TrimSpace(strings.TrimPrefix(res.Body, "\xef\xbb\xbf")), "\n")
	want := []string{
		standardHeader + ";Raison du rejet",
		"Hugo;Durand;3/4/12;hugo.example.org;'+32 470 12 34 56;\"Date de naissance : Indiquez l'année sur 4 chiffres (JJ/MM/AAAA). ; Mail : " + api.MsgEmailFormat + "\"",
		"Marie;Dupont;12/03/2012;;;Doublon de la fiche Dupont Marie (12/03/2012)",
		"Léa;Lefevre;03-04-2014;;;Doublon de la ligne 2",
	}
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	if !slices.Equal(lines, want) {
		t.Errorf("fichier\n%q\nattendu\n%q", lines, want)
	}
}

func TestImportRejectedRoundTrip(t *testing.T) {
	env, admin := newImportEnv(t)
	insertOrphan(t, env, "Marie", "Dupont", "2012-03-12")
	data := csvBytes(standardHeader,
		"Léa;Lefèvre;03/04/2014;;",
		"Hugo;Durand;3/4/12;;+32 470 12 34 56",
		"Marie;Dupont;12/03/2012;;",
	)
	rejected := rejectedImport(t, admin, data, standardMapping).Body
	expectJSON[api.ImportReport](t, runImport(t, admin, data, standardMapping, importVersion(t, admin)), http.StatusCreated)
	fixed := []byte(strings.Replace(rejected, "3/4/12", "03/04/2012", 1))

	a := expectJSON[api.ImportAnalysis](t, analyzeImport(t, admin, fixed), http.StatusOK)
	if a.Suggested[len(a.Suggested)-1] != "" {
		t.Errorf("« Raison du rejet » proposée : %q", a.Suggested)
	}
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, fixed, a.Suggested), http.StatusOK)
	if p.Counts != (api.ImportCounts{Ready: 1, Errors: 0, Duplicates: 1}) {
		t.Errorf("aperçu %+v %+v", p.Counts, p.Rows)
	}
	r := expectJSON[api.ImportReport](t, runImport(t, admin, fixed, a.Suggested, p.FieldsVersion), http.StatusCreated)
	if r.Created != 1 {
		t.Errorf("rapport %+v", r)
	}
	for _, m := range adminPage(t, admin, "?account=none").Items {
		if m.FirstName == "Hugo" && (m.Phone == nil || *m.Phone != "+32470123456") {
			t.Errorf("téléphone %v", m.Phone)
		}
	}
}

func TestImportCustomFields(t *testing.T) {
	env, admin := newImportEnv(t)
	licence := createField(t, admin, fieldBody("member", "Numéro de licence", "short_text", "required", true))
	cat := createField(t, admin, fieldBody("member", "Catégorie", "choice", "choices", []string{"Poussin", "Benjamin", "Minime"}, "visibility", "editable"))
	archived := createField(t, admin, fieldBody("member", "Ancien", "short_text"))
	expectField(t, admin.Do("POST", customFieldPath(archived.ID, "/archive"), nil), http.StatusOK)
	createField(t, admin, fieldBody("membership", "Taille", "short_text"))

	data := csvBytes("Prénom;Nom;Date de naissance;Licence;Catégorie",
		"Léa;Lefèvre;03/04/2014;L-001;poussin",
		"Paul;Martin;15/09/2013;;Minime",
		"Hugo;Durand;30/06/2011;;Cadet",
	)
	a := expectJSON[api.ImportAnalysis](t, analyzeImport(t, admin, data), http.StatusOK)
	targets := []string{}
	for _, tg := range a.Targets[9:] {
		targets = append(targets, tg.Label)
	}
	if !slices.Equal(targets, []string{"Numéro de licence", "Catégorie"}) {
		t.Errorf("champs proposés %q", targets)
	}
	fieldT := func(id int64) string { return fmt.Sprintf("field:%d", id) }
	if a.Suggested[4] != fieldT(cat.ID) || a.Suggested[3] != "" {
		t.Errorf("proposition %q", a.Suggested)
	}
	mapping := []string{"firstName", "lastName", "birthDate", fieldT(licence.ID), fieldT(cat.ID)}
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, mapping), http.StatusOK)
	if p.Counts.Ready != 2 || p.IncompleteCount != 1 || len(p.MissingRequired) != 1 || p.MissingRequired[0] != (api.ImportMissing{FieldID: licence.ID, Label: "Numéro de licence", Count: 1}) {
		t.Errorf("aperçu %+v %+v", p.Counts, p.MissingRequired)
	}
	if !slices.Equal(p.Rows[1].MissingRequired, []int64{licence.ID}) || len(p.Rows[0].MissingRequired) != 0 || len(p.Rows[2].MissingRequired) != 0 {
		t.Errorf("manquants par ligne %v %v %v", p.Rows[0].MissingRequired, p.Rows[1].MissingRequired, p.Rows[2].MissingRequired)
	}
	if e := p.Rows[2].Errors; len(e) != 1 || !strings.Contains(e[0].Message, "Choix possibles : Poussin, Benjamin, Minime") {
		t.Errorf("choix inconnu %+v", e)
	}
	r := expectJSON[api.ImportReport](t, runImport(t, admin, data, mapping, p.FieldsVersion), http.StatusCreated)
	if r.Created != 2 || r.IncompleteCount != 1 {
		t.Errorf("rapport %+v", r)
	}
	var leaID int64
	for _, m := range adminPage(t, admin, "?account=none").Items {
		if m.FirstName == "Léa" {
			leaID = m.ID
		}
	}
	lea := expectMember(t, admin.Do("GET", adminMemberPath(leaID), nil), http.StatusOK)
	if v := findValue(t, lea.CustomFields, licence.ID); v.Text == nil || *v.Text != "L-001" {
		t.Errorf("licence %+v", v)
	}
	if v := findValue(t, lea.CustomFields, cat.ID); v.ChoiceID == nil || *v.ChoiceID != cat.Choices[0].ID {
		t.Errorf("catégorie %+v", v)
	}
	if logs := importLogs(env); len(logs) != 1 || !strings.Contains(logs[0], `"missing_required":1`) {
		t.Errorf("journal %q", logs)
	}
}

func TestImportArchivedTarget(t *testing.T) {
	_, admin := newImportEnv(t)
	cat := createField(t, admin, fieldBody("member", "Catégorie", "short_text"))
	data := csvBytes("Prénom;Nom;Date de naissance;Catégorie", "Léa;Lefèvre;03/04/2014;A")
	mapping := []string{"firstName", "lastName", "birthDate", fmt.Sprintf("field:%d", cat.ID)}
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, mapping), http.StatusOK)

	// Champ modifié entre l'aperçu et l'import : 409.
	expectField(t, admin.Do("PUT", customFieldPath(cat.ID, ""), updateBody(cat, "label", "Catégorie 2026")), http.StatusOK)
	if res := runImport(t, admin, data, mapping, p.FieldsVersion); res.Status != http.StatusConflict {
		t.Errorf("champ modifié : %d %s", res.Status, res.Body)
	}
	// Champ archivé : la correspondance est refusée.
	expectField(t, admin.Do("POST", customFieldPath(cat.ID, "/archive"), nil), http.StatusOK)
	res := previewImport(t, admin, data, mapping)
	if res.Status != http.StatusUnprocessableEntity || !hasError(problemOf(t, res), "form.mapping[3]", "n'existe plus") {
		t.Errorf("champ archivé : %d %s", res.Status, res.Body)
	}
}

func TestImportTemplate(t *testing.T) {
	_, admin := newImportEnv(t)
	createField(t, admin, fieldBody("member", "Catégorie", "choice", "choices", []string{"Poussin", "Benjamin"}))
	createField(t, admin, fieldBody("member", "Bénévole", "bool"))
	res := admin.Do("GET", importPath+"/template", nil)
	if res.Status != http.StatusOK || res.Header.Get("Content-Type") != "text/csv; charset=utf-8" || !strings.Contains(res.Header.Get("Content-Disposition"), "modele-import-membres.csv") {
		t.Fatalf("modèle : %d %v", res.Status, res.Header)
	}
	if !strings.HasPrefix(res.Body, "\xef\xbb\xbf") {
		t.Error("BOM absente")
	}
	body := strings.TrimSpace(strings.TrimPrefix(res.Body, "\xef\xbb\xbf"))
	if strings.Count(body, "\n") != 0 || !strings.HasPrefix(body, "Prénom *;Nom *;Date de naissance * (JJ/MM/AAAA);") || !strings.HasSuffix(body, ";Catégorie (Poussin, Benjamin);Bénévole (oui/non)") {
		t.Errorf("modèle %q", body)
	}
	filled := []byte(res.Body + "Léa;Lefèvre;03/04/2014;;;;;;;Poussin;oui\r\n")
	a := expectJSON[api.ImportAnalysis](t, analyzeImport(t, admin, filled), http.StatusOK)
	for i, s := range a.Suggested {
		if s == "" {
			t.Errorf("colonne %q non reconnue", a.Columns[i].Label)
		}
	}
}

// Objectif de performance : aperçu et import de 2 000 lignes en moins de 2 s.
func TestImportLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("test de performance")
	}
	env, admin := newImportEnv(t)
	lines := []string{standardHeader}
	for i := range csvfile.DefaultLimits.MaxRows {
		lines = append(lines, fmt.Sprintf("Prénom%d;Nom%d;01/01/2000;p%d@example.org;0612345678", i, i, i))
	}
	data := csvBytes(lines...)
	start := time.Now()
	p := expectJSON[api.ImportPreview](t, previewImport(t, admin, data, standardMapping), http.StatusOK)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("aperçu en %v", d)
	}
	start = time.Now()
	r := expectJSON[api.ImportReport](t, runImport(t, admin, data, standardMapping, p.FieldsVersion), http.StatusCreated)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("import en %v", d)
	}
	if r.Created != 2000 || identityCount(t, env) != 2000 {
		t.Errorf("créées %d", r.Created)
	}
}
