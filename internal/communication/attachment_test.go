package communication

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

type zipEntry struct {
	name   string
	data   string
	stored bool
}

func makeZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, e := range entries {
		method := zip.Deflate
		if e.stored {
			method = zip.Store
		}
		f, err := w.CreateHeader(&zip.FileHeader{Name: e.name, Method: method})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(e.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func contentTypes(main string) string {
	return `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Override PartName="/main.xml" ContentType="` + main + `"/></Types>`
}

// Fichiers minimaux d'exemple pour la détection de type.
func sampleFiles(t *testing.T) map[string][]byte {
	t.Helper()
	return map[string][]byte{
		"pdf":  []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n"),
		"png":  append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), make([]byte, 32)...),
		"jpeg": append([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), make([]byte, 32)...),
		"docx": makeZip(t, zipEntry{name: "[Content_Types].xml", data: contentTypes(docxMain)}, zipEntry{name: "word/document.xml", data: "<w/>"}),
		"xlsx": makeZip(t, zipEntry{name: "[Content_Types].xml", data: contentTypes(xlsxMain)}, zipEntry{name: "xl/workbook.xml", data: "<x/>"}),
		"odt":  makeZip(t, zipEntry{name: "mimetype", data: TypeODT, stored: true}, zipEntry{name: "content.xml", data: "<o/>"}),
		"ods":  makeZip(t, zipEntry{name: "mimetype", data: TypeODS, stored: true}, zipEntry{name: "content.xml", data: "<o/>"}),
	}
}

func TestDetectTypeAccepted(t *testing.T) {
	want := map[string]string{"pdf": TypePDF, "png": TypePNG, "jpeg": TypeJPEG, "docx": TypeDOCX, "xlsx": TypeXLSX, "odt": TypeODT, "ods": TypeODS}
	for name, data := range sampleFiles(t) {
		got, err := DetectType(bytes.NewReader(data), int64(len(data)))
		if err != nil || got != want[name] {
			t.Errorf("%s : %q, %v ; attendu %q", name, got, err, want[name])
		}
	}
}

func TestDetectTypeRefused(t *testing.T) {
	cases := map[string][]byte{
		"exécutable renommé .pdf": append([]byte("MZ\x90\x00\x03\x00\x00\x00"), make([]byte, 64)...),
		"ancien .doc (OLE)":       append([]byte("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"), make([]byte, 64)...),
		"texte":                   []byte("Bonjour, ceci est un texte."),
		"zip quelconque":          makeZip(t, zipEntry{name: "a.txt", data: "a"}),
		"docm": makeZip(t, zipEntry{name: "[Content_Types].xml",
			data: contentTypes("application/vnd.ms-word.document.macroEnabled.main+xml")}),
		"docx avec macros": makeZip(t, zipEntry{name: "[Content_Types].xml", data: contentTypes(docxMain)},
			zipEntry{name: "word/vbaProject.bin", data: "x"}),
		"odt compressé":  makeZip(t, zipEntry{name: "mimetype", data: TypeODT}),
		"odt avec Basic": makeZip(t, zipEntry{name: "mimetype", data: TypeODT, stored: true}, zipEntry{name: "Basic/script.xml", data: "x"}),
		"odp":            makeZip(t, zipEntry{name: "mimetype", data: "application/vnd.oasis.opendocument.presentation", stored: true}),
		"[Content_Types].xml trop gros": makeZip(t, zipEntry{name: "[Content_Types].xml",
			data: contentTypes(docxMain) + strings.Repeat(" ", contentTypesMax)}),
	}
	for name, data := range cases {
		if got, err := DetectType(bytes.NewReader(data), int64(len(data))); !errors.Is(err, ErrUnsupportedType) {
			t.Errorf("%s : %q, %v ; attendu un refus", name, got, err)
		}
	}
	if _, err := DetectType(bytes.NewReader(nil), 0); !errors.Is(err, ErrEmptyFile) {
		t.Errorf("fichier vide : %v", err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	long := strings.Repeat("a", 200) + ".pdf"
	for in, want := range map[string]string{
		"../../etc/passwd":       "passwd",
		`C:\dossier\a.pdf`:       "a.pdf",
		"Convocation AG été.pdf": "Convocation AG été.pdf",
		"a\x00b\nc.pdf":          "abc.pdf",
		"   ":                    "piece-jointe",
		"..":                     "piece-jointe",
		"dossier/":               "piece-jointe",
		long:                     strings.Repeat("a", 146) + ".pdf",
		strings.Repeat("é", 151): strings.Repeat("é", 150),
	} {
		got := SanitizeFilename(in)
		if got != want {
			t.Errorf("SanitizeFilename(%q) = %q, attendu %q", in, got, want)
		}
		if utf8.RuneCountInString(got) > OriginalNameMax {
			t.Errorf("%q : trop long", got)
		}
	}
}

func TestStoredName(t *testing.T) {
	n := NewStoredName()
	if !ValidStoredName(n) || n == NewStoredName() {
		t.Fatalf("nom généré : %q", n)
	}
	for _, bad := range []string{"", "../" + n[3:], strings.ToUpper(n), n + "0", n[:31]} {
		if ValidStoredName(bad) {
			t.Errorf("%q accepté", bad)
		}
	}
}

func TestCheckLimits(t *testing.T) {
	for _, tc := range []struct {
		count         int
		existing, add int64
		want          error
	}{
		{0, 0, MaxFileBytes, nil},
		{0, 0, MaxFileBytes + 1, ErrFileTooLarge},
		{4, 4 << 20, 1 << 20, nil},
		{5, 1, 1, ErrTooManyFiles},
		{2, 8 << 20, 3 << 20, ErrTotalTooLarge},
		{1, 5 << 20, 5 << 20, nil},
	} {
		if err := CheckLimits(tc.count, tc.existing, tc.add); !errors.Is(err, tc.want) {
			t.Errorf("CheckLimits(%d, %d, %d) = %v, attendu %v", tc.count, tc.existing, tc.add, err, tc.want)
		}
	}
}
