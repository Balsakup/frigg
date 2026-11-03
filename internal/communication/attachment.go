package communication

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Types de pièces jointes acceptés (FR-017b), détectés sur le contenu.
const (
	TypePDF  = "application/pdf"
	TypeJPEG = "image/jpeg"
	TypePNG  = "image/png"
	TypeDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	TypeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	TypeODT  = "application/vnd.oasis.opendocument.text"
	TypeODS  = "application/vnd.oasis.opendocument.spreadsheet"
)

// AcceptedTypes liste les types acceptés.
var AcceptedTypes = []string{TypePDF, TypeJPEG, TypePNG, TypeDOCX, TypeXLSX, TypeODT, TypeODS}

// Erreurs des pièces jointes.
var (
	ErrUnsupportedType = errors.New("type de fichier non accepté")
	ErrEmptyFile       = errors.New("fichier vide")
	ErrFileTooLarge    = errors.New("fichier trop volumineux")
	ErrTooManyFiles    = errors.New("trop de pièces jointes")
	ErrTotalTooLarge   = errors.New("pièces jointes trop volumineuses au total")
)

// contentTypesMax borne la lecture de [Content_Types].xml : le reste de
// l'archive n'est jamais décompressé (pas d'exposition aux bombes ZIP).
const contentTypesMax = 64 << 10

const (
	docxMain = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"
	xlsxMain = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"
)

// DetectType renvoie le type d'un fichier d'après son contenu, jamais d'après
// son nom ni le type annoncé par le navigateur (research.md › R4).
func DetectType(r io.ReaderAt, size int64) (string, error) {
	if size <= 0 {
		return "", ErrEmptyFile
	}
	head := make([]byte, min(size, 512))
	if _, err := r.ReadAt(head, 0); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	switch sniffed := http.DetectContentType(head); sniffed {
	case TypePDF, TypeJPEG, TypePNG:
		return sniffed, nil
	}
	if !bytes.HasPrefix(head, []byte("PK\x03\x04")) {
		return "", ErrUnsupportedType
	}
	zr, err := zip.NewReader(r, size)
	if err != nil || len(zr.File) == 0 {
		return "", ErrUnsupportedType
	}
	for _, f := range zr.File {
		// Macros VBA (Office) ou Basic/scripts (OpenDocument) : refusées.
		if strings.HasSuffix(strings.ToLower(f.Name), "vbaproject.bin") ||
			strings.HasPrefix(f.Name, "Basic/") || strings.HasPrefix(f.Name, "Scripts/") {
			return "", ErrUnsupportedType
		}
	}
	if t, ok := openDocumentType(zr.File[0]); ok {
		return t, nil
	}
	for _, f := range zr.File {
		if f.Name != "[Content_Types].xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", ErrUnsupportedType
		}
		data, err := io.ReadAll(io.LimitReader(rc, contentTypesMax+1))
		rc.Close()
		if err != nil || len(data) > contentTypesMax || bytes.Contains(bytes.ToLower(data), []byte("macroenabled")) {
			return "", ErrUnsupportedType
		}
		switch {
		case bytes.Contains(data, []byte(docxMain)):
			return TypeDOCX, nil
		case bytes.Contains(data, []byte(xlsxMain)):
			return TypeXLSX, nil
		}
		return "", ErrUnsupportedType
	}
	return "", ErrUnsupportedType
}

// openDocumentType lit l'entrée « mimetype » qu'un document OpenDocument
// place en tête d'archive, stockée sans compression.
func openDocumentType(f *zip.File) (string, bool) {
	if f.Name != "mimetype" || f.Method != zip.Store || f.UncompressedSize64 > 100 {
		return "", false
	}
	rc, err := f.Open()
	if err != nil {
		return "", false
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, 100))
	if err != nil {
		return "", false
	}
	switch t := string(data); t {
	case TypeODT, TypeODS:
		return t, true
	}
	return "", false
}

// OriginalNameMax est la longueur maximale d'un nom de fichier conservé.
const OriginalNameMax = 150

// SanitizeFilename réduit un nom fourni par le navigateur à son dernier
// élément, sans caractère de contrôle, tronqué à 150 caractères en conservant
// l'extension (research.md › R5).
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "piece-jointe"
	}
	if utf8.RuneCountInString(name) <= OriginalNameMax {
		return name
	}
	ext := filepath.Ext(name)
	if utf8.RuneCountInString(ext) > 10 {
		ext = ""
	}
	base := []rune(strings.TrimSuffix(name, ext))
	return string(base[:OriginalNameMax-utf8.RuneCountInString(ext)]) + ext
}

var storedNameRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// NewStoredName génère le nom d'un fichier sur disque : 16 octets aléatoires
// en hexadécimal.
func NewStoredName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ValidStoredName indique si name a la forme d'un nom généré : seul un tel
// nom peut servir à construire un chemin.
func ValidStoredName(name string) bool { return storedNameRe.MatchString(name) }

// CheckLimits vérifie qu'un fichier de newBytes octets peut s'ajouter à
// existingCount pièces totalisant existingBytes octets.
func CheckLimits(existingCount int, existingBytes, newBytes int64) error {
	switch {
	case newBytes > MaxFileBytes:
		return ErrFileTooLarge
	case existingCount >= MaxFiles:
		return ErrTooManyFiles
	case existingBytes+newBytes > MaxTotalBytes:
		return ErrTotalTooLarge
	}
	return nil
}
