// Package memberimport porte les règles de l'importation des membres, sans
// base ni HTTP (research.md 013 › R4 à R8) : correspondance des colonnes,
// lecture et validation des lignes, doublons, modèle et lignes rejetées.
package memberimport

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/textfold"
)

// Target est le champ Frigg auquel une colonne est associée ; None : la
// colonne n'est pas importée.
type Target string

const (
	None         Target = ""
	FirstName    Target = "firstName"
	LastName     Target = "lastName"
	BirthDate    Target = "birthDate"
	Email        Target = "email"
	Phone        Target = "phone"
	AddressLine1 Target = "addressLine1"
	AddressLine2 Target = "addressLine2"
	PostalCode   Target = "postalCode"
	City         Target = "city"
)

const fieldPrefix = "field:"

// FieldTarget est la cible d'un champ personnalisé.
func FieldTarget(id int64) Target { return Target(fieldPrefix + strconv.FormatInt(id, 10)) }

// FieldID renvoie l'identifiant du champ personnalisé visé.
func (t Target) FieldID() (int64, bool) {
	s, ok := strings.CutPrefix(string(t), fieldPrefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseInt(s, 10, 64)
	return id, err == nil && id > 0
}

// StandardTarget est un champ de la fiche importable.
type StandardTarget struct {
	Target Target
	Label  string
}

// StandardTargets sont les champs de la fiche, dans l'ordre de la fiche.
var StandardTargets = []StandardTarget{
	{FirstName, "Prénom"},
	{LastName, "Nom"},
	{BirthDate, "Date de naissance"},
	{Email, "Adresse électronique"},
	{Phone, "Téléphone"},
	{AddressLine1, "Adresse"},
	{AddressLine2, "Complément d'adresse"},
	{PostalCode, "Code postal"},
	{City, "Commune"},
}

// RequiredTargets sont les champs sans lesquels aucune fiche n'est créée.
var RequiredTargets = []Target{FirstName, LastName, BirthDate}

// IsRequired indique un champ exigé par l'import.
func IsRequired(t Target) bool {
	return t == FirstName || t == LastName || t == BirthDate
}

// StandardLabel renvoie le libellé d'un champ de la fiche.
func StandardLabel(t Target) (string, bool) {
	for _, s := range StandardTargets {
		if s.Target == t {
			return s.Label, true
		}
	}
	return "", false
}

var synonyms = map[string]Target{}

func init() {
	for t, words := range map[Target][]string{
		FirstName:    {"prenom", "prenoms", "firstname", "prenomdumembre", "prenomadherent"},
		LastName:     {"nom", "noms", "nomdefamille", "lastname", "name", "nomdumembre", "nomadherent", "nomdusage"},
		BirthDate:    {"datedenaissance", "naissance", "datenaissance", "ddn", "nele", "neele", "birthdate", "dateofbirth"},
		Email:        {"email", "mail", "courriel", "adresseemail", "adressemail", "adresseelectronique", "emailadherent"},
		Phone:        {"telephone", "tel", "portable", "mobile", "telportable", "telephoneportable", "phone", "numerodetelephone"},
		AddressLine1: {"adresse", "adressepostale", "rue", "adresseligne1", "ligne1", "address"},
		AddressLine2: {"complement", "complementdadresse", "adresseligne2", "ligne2", "adresse2"},
		PostalCode:   {"codepostal", "cp", "zip", "postalcode"},
		City:         {"commune", "ville", "localite", "city"},
	} {
		for _, w := range words {
			synonyms[w] = t
		}
	}
}

var finalParen = regexp.MustCompile(`\s*\([^()]*\)\s*$`)

func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, textfold.Fold(strings.ReplaceAll(s, "*", "")))
}

// normalizeHeader replie un intitulé et n'en garde que lettres et chiffres ;
// short retire en plus une parenthèse finale (« (JJ/MM/AAAA) »).
func normalizeHeader(h string) (full, short string) {
	return squeeze(h), squeeze(finalParen.ReplaceAllString(strings.ReplaceAll(h, "*", ""), ""))
}

// Suggest propose une cible par colonne : synonyme d'un champ de la fiche,
// puis libellé d'un champ personnalisé ; une cible n'est proposée qu'une fois.
func Suggest(headers []string, defs []customfield.Definition) []Target {
	type labelled struct {
		full, short string
		target      Target
	}
	fields := make([]labelled, len(defs))
	for i, d := range defs {
		f, s := normalizeHeader(d.Label)
		fields[i] = labelled{f, s, FieldTarget(d.ID)}
	}
	taken := map[Target]bool{}
	out := make([]Target, len(headers))
	for i, h := range headers {
		full, short := normalizeHeader(h)
		candidates := []Target{synonyms[full], synonyms[short]}
		for _, key := range []string{full, short} {
			for _, f := range fields {
				if key != "" && (f.full == key || f.short == key) {
					candidates = append(candidates, f.target)
				}
			}
		}
		for _, c := range candidates {
			if c != None && !taken[c] {
				out[i], taken[c] = c, true
				break
			}
		}
	}
	return out
}

// ColumnLabels nomme les colonnes : leur intitulé, ou « Colonne N » s'il est
// vide ou porté aussi par une colonne précédente.
func ColumnLabels(headers []string) []string {
	seen := map[string]bool{}
	out := make([]string, len(headers))
	for i, h := range headers {
		key := textfold.Fold(h)
		if key == "" || seen[key] {
			out[i] = fmt.Sprintf("Colonne %d", i+1)
		} else {
			out[i] = h
		}
		seen[key] = true
	}
	return out
}

// MappingError est une erreur de correspondance ; Index vaut -1 pour une
// erreur qui ne porte pas sur une colonne.
type MappingError struct {
	Index   int
	Message string
}

const (
	MsgMappingLength  = "La correspondance ne couvre pas toutes les colonnes."
	MsgMappingUnknown = "Ce champ n'existe plus ou n'est pas importable."
)

// ValidateMapping vérifie une correspondance : une cible par colonne au plus,
// des cibles connues, et les trois champs exigés associés (FR-006, FR-007).
func ValidateMapping(m []Target, labels []string, defs []customfield.Definition) []MappingError {
	if len(m) != len(labels) {
		return []MappingError{{Index: -1, Message: MsgMappingLength}}
	}
	known := map[int64]bool{}
	for _, d := range defs {
		known[d.ID] = true
	}
	var errs []MappingError
	firstColumn := map[Target]int{}
	for i, t := range m {
		if t == None {
			continue
		}
		_, standard := StandardLabel(t)
		id, isField := t.FieldID()
		if !standard && !(isField && known[id]) {
			errs = append(errs, MappingError{Index: i, Message: MsgMappingUnknown})
			continue
		}
		if j, ok := firstColumn[t]; ok {
			errs = append(errs, MappingError{Index: i, Message: fmt.Sprintf("Ce champ est déjà associé à la colonne « %s ».", labels[j])})
			continue
		}
		firstColumn[t] = i
	}
	for _, t := range RequiredTargets {
		if _, ok := firstColumn[t]; !ok {
			label, _ := StandardLabel(t)
			errs = append(errs, MappingError{Index: -1, Message: fmt.Sprintf("Associez une colonne au champ « %s ».", label)})
		}
	}
	return errs
}
