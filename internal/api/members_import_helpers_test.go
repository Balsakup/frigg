package api_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
)

const importPath = "/api/admin/members/import"

// importForm construit un corps multipart avec le fichier et les champs.
func importForm(t *testing.T, fields map[string]string, name string, data []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	if data != nil {
		part, err := w.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(data)
	}
	_ = w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

func postImport(t *testing.T, c *apitest.Client, suffix string, fields map[string]string, data []byte) apitest.Response {
	t.Helper()
	body, ct := importForm(t, fields, "membres.csv", data)
	return c.Do("POST", importPath+suffix, apitest.Raw(body), "Content-Type", ct)
}

func mappingJSON(t *testing.T, mapping []string) string {
	t.Helper()
	b, err := json.Marshal(mapping)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func analyzeImport(t *testing.T, c *apitest.Client, data []byte) apitest.Response {
	t.Helper()
	return postImport(t, c, "/analyze", nil, data)
}

func previewImport(t *testing.T, c *apitest.Client, data []byte, mapping []string) apitest.Response {
	t.Helper()
	return postImport(t, c, "/preview", map[string]string{"mapping": mappingJSON(t, mapping)}, data)
}

func runImport(t *testing.T, c *apitest.Client, data []byte, mapping []string, version string, extra ...string) apitest.Response {
	t.Helper()
	fields := map[string]string{"mapping": mappingJSON(t, mapping), "fieldsVersion": version}
	for i := 0; i+1 < len(extra); i += 2 {
		fields[extra[i]] = extra[i+1]
	}
	return postImport(t, c, "", fields, data)
}

func rejectedImport(t *testing.T, c *apitest.Client, data []byte, mapping []string) apitest.Response {
	t.Helper()
	return postImport(t, c, "/rejected", map[string]string{"mapping": mappingJSON(t, mapping)}, data)
}

// csvBytes joint des lignes par CRLF.
func csvBytes(lines ...string) []byte { return []byte(strings.Join(lines, "\r\n") + "\r\n") }

// win1252 encode un texte en Windows-1252 (« CSV » d'Excel).
func win1252(t *testing.T, s string) []byte {
	t.Helper()
	b, err := charmap.Windows1252.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// standardHeader et standardMapping : fichier « Prénom;Nom;Date de
// naissance;Mail;Téléphone ».
const standardHeader = "Prénom;Nom;Date de naissance;Mail;Téléphone"

var standardMapping = []string{"firstName", "lastName", "birthDate", "email", "phone"}

// importVersion lit l'empreinte courante par analyze.
func importVersion(t *testing.T, c *apitest.Client) string {
	t.Helper()
	return expectJSON[api.ImportAnalysis](t, analyzeImport(t, c, csvBytes(standardHeader, "A;B;01/01/2000;;")), 200).FieldsVersion
}

func problemOf(t *testing.T, res apitest.Response) apitest.Problem {
	t.Helper()
	var p apitest.Problem
	res.JSON(t, &p)
	return p
}

// hasError indique un détail d'erreur à l'emplacement donné, contenant msg.
func hasError(p apitest.Problem, location, msg string) bool {
	for _, e := range p.Errors {
		if e.Location == location && strings.Contains(e.Message, msg) {
			return true
		}
	}
	return false
}

// importLogs renvoie les lignes de journal member.import.
func importLogs(env *apitest.Env) []string {
	var out []string
	for _, line := range strings.Split(env.Logs.String(), "\n") {
		if strings.Contains(line, `"action":"member.import"`) {
			out = append(out, line)
		}
	}
	return out
}
