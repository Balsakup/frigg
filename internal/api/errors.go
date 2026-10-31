package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/validation"
)

// Titres des erreurs en français (constitution, Conventions de langue).
var errorTitles = map[int]string{
	http.StatusBadRequest:            "Requête invalide",
	http.StatusUnauthorized:          "Connexion requise",
	http.StatusForbidden:             "Accès refusé",
	http.StatusNotFound:              "Introuvable",
	http.StatusMethodNotAllowed:      "Méthode non autorisée",
	http.StatusConflict:              "Conflit",
	http.StatusGone:                  "Lien expiré",
	http.StatusRequestEntityTooLarge: "Requête trop volumineuse",
	http.StatusUnsupportedMediaType:  "Format non pris en charge",
	http.StatusUnprocessableEntity:   "Données invalides",
	http.StatusTooManyRequests:       "Trop de tentatives",
	http.StatusInternalServerError:   "Erreur interne",
}

// Messages génériques de huma traduits.
var detailTranslations = map[string]string{
	"validation failed":               "Les données envoyées sont invalides.",
	"request body is required":        "Le corps de la requête est obligatoire.",
	"unexpected error occurred":       "Une erreur inattendue s'est produite.",
	"error while marshaling response": "Une erreur inattendue s'est produite.",
}

// MsgTooManyAttempts est la réponse neutre en cas de dépassement de débit.
const MsgTooManyAttempts = "Trop de tentatives. Réessayez dans quelques minutes."

func init() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		if t, ok := detailTranslations[msg]; ok {
			msg = t
		}
		title, ok := errorTitles[status]
		if !ok {
			title = http.StatusText(status)
		}
		details := make([]*huma.ErrorDetail, 0, len(errs))
		for _, err := range errs {
			if err == nil {
				continue
			}
			var d huma.ErrorDetailer
			if errors.As(err, &d) {
				details = append(details, d.ErrorDetail())
			} else {
				details = append(details, &huma.ErrorDetail{Message: err.Error()})
			}
		}
		return &huma.ErrorModel{Status: status, Title: title, Detail: msg, Errors: details}
	}
	translateValidation()
}

// FieldErrors renvoie une erreur 422 avec une entrée par message, à l'emplacement
// location (par exemple « body.password »).
func FieldErrors(location string, messages ...string) error {
	errs := make([]error, len(messages))
	for i, m := range messages {
		errs[i] = &huma.ErrorDetail{Location: location, Message: m}
	}
	return huma.Error422UnprocessableEntity("Les données envoyées sont invalides.", errs...)
}

// FieldErrorDetails renvoie une erreur 422 avec une entrée par détail, chacun
// à son propre emplacement (erreurs de plusieurs champs renvoyées ensemble).
func FieldErrorDetails(details ...*huma.ErrorDetail) error {
	errs := make([]error, len(details))
	for i, d := range details {
		errs[i] = d
	}
	return huma.Error422UnprocessableEntity("Les données envoyées sont invalides.", errs...)
}

// TooManyAttempts renvoie l'erreur 429 neutre avec l'en-tête Retry-After.
func TooManyAttempts() error {
	h := http.Header{}
	h.Set("Retry-After", strconv.Itoa(60))
	return huma.ErrorWithHeaders(huma.Error429TooManyRequests(MsgTooManyAttempts), h)
}

func translateValidation() {
	validation.MsgUnexpectedProperty = "propriété inattendue"
	validation.MsgExpectedRFC3339DateTime = "date et heure attendues au format RFC 3339"
	validation.MsgExpectedRFC1123DateTime = "date et heure attendues au format RFC 1123"
	validation.MsgExpectedRFC3339Date = "Saisissez une date valide."
	validation.MsgExpectedRFC3339Time = "heure attendue au format RFC 3339"
	validation.MsgExpectedRFC5322Email = "adresse électronique invalide : %v"
	validation.MsgExpectedRFC5322EmailBare = "adresse électronique invalide"
	validation.MsgExpectedRFC5890Hostname = "nom d'hôte invalide"
	validation.MsgExpectedRFC2673IPv4 = "adresse IPv4 invalide"
	validation.MsgExpectedRFC2373IPv6 = "adresse IPv6 invalide"
	validation.MsgExpectedRFCIPAddr = "adresse IP invalide"
	validation.MsgExpectedRFC3986URI = "URI invalide : %v"
	validation.MsgExpectedRFC3986AbsoluteURI = "URI absolue attendue"
	validation.MsgExpectedRFC4122UUID = "UUID invalide : %v"
	validation.MsgExpectedRFC6570URITemplate = "modèle d'URI invalide"
	validation.MsgExpectedRFC6901JSONPointer = "pointeur JSON invalide"
	validation.MsgExpectedRFC6901RelativeJSONPointer = "pointeur JSON relatif invalide"
	validation.MsgExpectedRegexp = "expression régulière invalide : %v"
	validation.MsgExpectedMatchAtLeastOneSchema = "la valeur ne correspond à aucun des formats acceptés"
	validation.MsgExpectedMatchExactlyOneSchema = "la valeur doit correspondre à exactement un format"
	validation.MsgExpectedNotMatchSchema = "la valeur ne doit pas correspondre à ce format"
	validation.MsgExpectedPropertyNameInObject = "nom de propriété attendu dans l'objet"
	validation.MsgExpectedBoolean = "booléen attendu"
	validation.MsgExpectedNull = "valeur nulle attendue"
	validation.MsgExpectedDuration = "durée invalide : %v"
	validation.MsgExpectedNumber = "nombre attendu"
	validation.MsgExpectedInteger = "nombre entier attendu"
	validation.MsgExpectedString = "texte attendu"
	validation.MsgExpectedBase64String = "texte encodé en base64 attendu"
	validation.MsgExpectedArray = "liste attendue"
	validation.MsgExpectedObject = "objet attendu"
	validation.MsgExpectedArrayItemsUnique = "les éléments de la liste doivent être uniques"
	validation.MsgExpectedOneOf = "la valeur doit être l'une de : \"%s\""
	validation.MsgExpectedConst = "la valeur doit être %v"
	validation.MsgExpectedMinimumNumber = "le nombre doit être supérieur ou égal à %v"
	validation.MsgExpectedExclusiveMinimumNumber = "le nombre doit être supérieur à %v"
	validation.MsgExpectedMaximumNumber = "le nombre doit être inférieur ou égal à %v"
	validation.MsgExpectedExclusiveMaximumNumber = "le nombre doit être inférieur à %v"
	validation.MsgExpectedNumberBeMultipleOf = "le nombre doit être un multiple de %v"
	validation.MsgExpectedMinLength = "au moins %d caractères attendus"
	validation.MsgExpectedMaxLength = "au plus %d caractères attendus"
	validation.MsgExpectedBePattern = "le texte doit être %s"
	validation.MsgExpectedMatchPattern = "le texte doit respecter le motif %s"
	validation.MsgExpectedMinItems = "au moins %d éléments attendus"
	validation.MsgExpectedMaxItems = "au plus %d éléments attendus"
	validation.MsgExpectedMinProperties = "au moins %d propriétés attendues"
	validation.MsgExpectedMaxProperties = "au plus %d propriétés attendues"
	validation.MsgExpectedRequiredProperty = "la propriété %s est obligatoire"
	validation.MsgExpectedDependentRequiredProperty = "la propriété %s est obligatoire quand %s est présente"
	validation.MsgExpectedResolvableSchemaRef = "référence de schéma introuvable : %s"
}
