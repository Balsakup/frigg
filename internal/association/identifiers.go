package association

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Messages d'erreur des identifiants (contracts/api.md).
const (
	MsgRNAInvalid      = "Le numéro RNA commence par W suivi de 9 chiffres (W123456789)."
	MsgSIRETLength     = "Le numéro SIRET comporte 14 chiffres."
	MsgSIRETInvalid    = "Ce numéro SIRET n'est pas valide. Vérifiez-le sur l'avis de situation Sirene."
	MsgPhoneInvalid    = "Saisissez un numéro de téléphone français (01 23 45 67 89)."
	MsgWebURLInvalid   = "Saisissez une adresse web complète, commençant par https://."
	MsgEmailInvalid    = "Saisissez une adresse électronique valide."
	MsgPostalCode      = "Le code postal comporte 5 chiffres."
	MsgFoundedInFuture = "La date de création ne peut pas être dans le futur."
	MsgDateInvalid     = "Saisissez une date valide."
)

// WebURLMax est la longueur maximale d'une adresse web.
const WebURLMax = 300

var (
	rnaRe    = regexp.MustCompile(`^W[0-9]{9}$`)
	digitsRe = regexp.MustCompile(`^[0-9]+$`)
	// Caractères de présentation retirés des numéros saisis.
	phoneSeparators = strings.NewReplacer(" ", "", ".", "", "-", "", "(", "", ")", "", " ", "")
)

// NormalizeRNA renvoie le numéro RNA en majuscules, ou false s'il n'a pas la
// forme W suivi de 9 chiffres.
func NormalizeRNA(s string) (string, bool) {
	s = strings.ToUpper(s)
	return s, rnaRe.MatchString(s)
}

// NormalizeSIRET retire les espaces et vérifie la longueur et la clé de Luhn.
// Le second résultat est le message d'erreur, vide si le numéro est valide.
func NormalizeSIRET(s string) (string, string) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), " ", "")
	if len(s) != 14 || !digitsRe.MatchString(s) {
		return s, MsgSIRETLength
	}
	if !siretChecksum(s) {
		return s, MsgSIRETInvalid
	}
	return s, ""
}

// siretChecksum applique la clé de Luhn ; les établissements de La Poste
// (SIREN 356000000) suivent une règle propre : somme des chiffres multiple de 5.
func siretChecksum(s string) bool {
	if strings.HasPrefix(s, "356000000") {
		sum := 0
		for _, c := range s {
			sum += int(c - '0')
		}
		return sum%5 == 0
	}
	sum := 0
	for i := range len(s) {
		n := int(s[len(s)-1-i] - '0')
		if i%2 == 1 {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
	}
	return sum%10 == 0
}

// Indicatifs des départements d'outre-mer à numérotation à 10 chiffres.
var overseasCodes = []string{"262", "269", "590", "594", "596"}

// NormalizePhone renvoie un numéro français sous sa forme nationale à 10
// chiffres (0123456789), ou false s'il n'est pas reconnu.
func NormalizePhone(s string) (string, bool) {
	s = phoneSeparators.Replace(s)
	switch {
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	case strings.HasPrefix(s, "00"):
		s = s[2:]
	default:
		if len(s) == 10 && digitsRe.MatchString(s) && s[0] == '0' && s[1] != '0' {
			return s, true
		}
		return "", false
	}
	if !digitsRe.MatchString(s) {
		return "", false
	}
	for _, code := range append([]string{"33"}, overseasCodes...) {
		if rest, ok := strings.CutPrefix(s, code); ok && len(rest) == 9 && rest[0] != '0' {
			return "0" + rest, true
		}
	}
	return "", false
}

// ValidWebURL indique si s est une adresse web http(s) complète.
func ValidWebURL(s string) bool {
	if utf8.RuneCountInString(s) > WebURLMax || strings.ContainsAny(s, " \t\r\n") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
