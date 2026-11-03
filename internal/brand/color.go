// Package brand calcule les couleurs de l'association et leurs teintes
// dérivées, avec une garantie de contraste (specs/010-general-settings › R3).
// Il sert à l'interface (variables CSS) comme aux e-mails.
package brand

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGB est une couleur sRGB sur 8 bits par canal.
type RGB struct{ R, G, B uint8 }

// ErrInvalidHex signale un code couleur qui n'est pas de la forme #rrggbb.
var ErrInvalidHex = errors.New("code couleur invalide")

// ParseHex lit un code « #rrggbb » (majuscules acceptées).
func ParseHex(s string) (RGB, error) {
	if len(s) != 7 || s[0] != '#' {
		return RGB{}, ErrInvalidHex
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return RGB{}, ErrInvalidHex
	}
	return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, nil
}

// MustHex est ParseHex pour les constantes du paquet.
func MustHex(s string) RGB {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Hex renvoie le code « #rrggbb » en minuscules.
func (c RGB) Hex() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

// Triplet renvoie « r g b », format des variables CSS de app.css.
func (c RGB) Triplet() string { return fmt.Sprintf("%d %d %d", c.R, c.G, c.B) }

// NormalizeHex renvoie le code en minuscules, ou une erreur.
func NormalizeHex(s string) (string, error) {
	c, err := ParseHex(strings.TrimSpace(s))
	if err != nil {
		return "", err
	}
	return c.Hex(), nil
}

func toLinear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func fromLinear(c float64) uint8 {
	if c <= 0.0031308 {
		c *= 12.92
	} else {
		c = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return uint8(math.Round(math.Max(0, math.Min(1, c)) * 255))
}

// Luminance est la luminance relative du WCAG 2.x.
func (c RGB) Luminance() float64 {
	return 0.2126*toLinear(c.R) + 0.7152*toLinear(c.G) + 0.0722*toLinear(c.B)
}

// Contrast renvoie le rapport de contraste WCAG entre deux couleurs (1 à 21).
func Contrast(a, b RGB) float64 {
	la, lb := a.Luminance(), b.Luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Mix renvoie c posée avec l'opacité alpha sur bg (mélange sRGB, comme les
// couleurs à opacité de CSS, par exemple bg-brand/10).
func Mix(c, bg RGB, alpha float64) RGB {
	m := func(a, b uint8) uint8 { return uint8(math.Round(float64(a)*alpha + float64(b)*(1-alpha))) }
	return RGB{m(c.R, bg.R), m(c.G, bg.G), m(c.B, bg.B)}
}

// oklch est une couleur dans l'espace OKLCH (luminosité, chroma, teinte).
type oklch struct{ L, C, H float64 }

func (c RGB) oklch() oklch {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	L := 0.2104542553*l + 0.7936177850*m - 0.0040720468*s
	A := 1.9779984951*l - 2.4285922050*m + 0.4505937099*s
	B := 0.0259040371*l + 0.7827717662*m - 0.8086757660*s
	return oklch{L, math.Hypot(A, B), math.Atan2(B, A)}
}

func (o oklch) rgb() RGB {
	A, B := o.C*math.Cos(o.H), o.C*math.Sin(o.H)
	l := o.L + 0.3963377774*A + 0.2158037573*B
	m := o.L - 0.1055613458*A - 0.0638541728*B
	s := o.L - 0.0894841775*A - 1.2914855480*B
	l, m, s = l*l*l, m*m*m, s*s*s
	return RGB{
		fromLinear(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		fromLinear(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		fromLinear(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
	}
}
