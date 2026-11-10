package memberimport

import (
	"github.com/balsakup/frigg/internal/customfield"
	"github.com/balsakup/frigg/internal/season"
)

var today = season.MustDate("2026-11-10")

func defsFixture() []customfield.Definition {
	return []customfield.Definition{
		{ID: 3, Label: "Numéro de licence", Type: customfield.TypeShortText, Required: true},
		{ID: 5, Label: "Catégorie", Type: customfield.TypeChoice, Choices: []customfield.Choice{{ID: 51, Label: "Poussin"}, {ID: 52, Label: "Benjamin"}, {ID: 53, Label: "Minime"}}},
		{ID: 7, Label: "Bénévole", Type: customfield.TypeBool},
		{ID: 8, Label: "Arrivée", Type: customfield.TypeDate},
		{ID: 9, Label: "Taille (cm)", Type: customfield.TypeNumber},
	}
}

func plainMessage(e CellError) string { return e.Err.Error() }
