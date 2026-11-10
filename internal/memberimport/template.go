package memberimport

import (
	"fmt"
	"strings"

	"github.com/balsakup/frigg/internal/csvfile"
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/season"
)

// RejectedReasonHeader est l'intitulé de la colonne ajoutée au fichier des
// lignes rejetées ; il n'a aucun synonyme, donc n'est pas importé.
const RejectedReasonHeader = "Raison du rejet"

// TemplateHeaders sont les intitulés du modèle de fichier, tous reconnus par
// Suggest (research.md › R7). Seuls les champs exigés par l'import portent
// « * ».
func TemplateHeaders(defs []customfield.Definition) []string {
	out := make([]string, 0, len(StandardTargets)+len(defs))
	for _, s := range StandardTargets {
		h := s.Label
		if IsRequired(s.Target) {
			h += " *"
		}
		if s.Target == BirthDate {
			h += " (JJ/MM/AAAA)"
		}
		out = append(out, h)
	}
	for _, d := range defs {
		out = append(out, d.Label+typeHint(d))
	}
	return out
}

func typeHint(d customfield.Definition) string {
	switch d.Type {
	case customfield.TypeBool:
		return " (oui/non)"
	case customfield.TypeDate:
		return " (JJ/MM/AAAA)"
	case customfield.TypeNumber:
		return " (nombre)"
	case customfield.TypeChoice:
		labels := make([]string, 0, maxListedChoices+1)
		for i, c := range d.Choices {
			if i == maxListedChoices {
				labels = append(labels, "…")
				break
			}
			labels = append(labels, c.Label)
		}
		return " (" + strings.Join(labels, ", ") + ")"
	}
	return ""
}

// RejectedHeaders sont les intitulés d'origine, neutralisés, suivis de la
// raison du rejet.
func RejectedHeaders(t csvfile.Table) []string {
	out := make([]string, 0, len(t.Headers)+1)
	for _, h := range t.Headers {
		out = append(out, csvfile.SafeCell(h))
	}
	return append(out, RejectedReasonHeader)
}

// FormatDate présente une date ISO en JJ/MM/AAAA.
func FormatDate(iso string) string {
	if d, err := season.ParseDate(iso); err == nil {
		return fmt.Sprintf("%02d/%02d/%d", d.Day(), d.Month(), d.Year())
	}
	return iso
}

// MemberLabel présente une fiche existante : « Dupont Marie (12/03/2012) ».
func MemberLabel(i Identity) string {
	return fmt.Sprintf("%s %s (%s)", i.LastName, i.FirstName, FormatDate(i.BirthDate))
}

// RejectedRecord est la ligne du fichier des lignes rejetées : cellules
// d'origine neutralisées, puis la raison. message traduit une erreur en
// texte affiché ; labels sont les noms des colonnes (ColumnLabels).
func RejectedRecord(r Result, row csvfile.Row, labels []string, message func(CellError) string) []string {
	out := make([]string, 0, len(row.Cells)+1)
	for _, c := range row.Cells {
		out = append(out, csvfile.SafeCell(c))
	}
	return append(out, csvfile.SafeCell(Reason(r, labels, message)))
}

// Reason explique le rejet d'une ligne.
func Reason(r Result, labels []string, message func(CellError) string) string {
	switch {
	case r.DuplicateOfMember != nil:
		return "Doublon de la fiche " + MemberLabel(*r.DuplicateOfMember)
	case r.DuplicateOfLine > 0:
		return fmt.Sprintf("Doublon de la ligne %d", r.DuplicateOfLine)
	}
	parts := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msg := message(e)
		if e.Column >= 0 && e.Column < len(labels) {
			msg = labels[e.Column] + " : " + msg
		}
		parts[i] = msg
	}
	return strings.Join(parts, " ; ")
}
