package brand

// Couleurs fixes de l'artefact de référence (web/app/app.css).
var (
	White  = RGB{255, 255, 255}
	Black  = RGB{0, 0, 0}
	Ink    = MustHex("#181f29")
	Canvas = MustHex("#f5f6f8")
	// Couleurs par défaut de Frigg.
	DefaultPrimary = MustHex("#1f5278")
	DefaultAccent  = MustHex("#c44822")
)

// MinContrast est le contraste minimal du texte (RGAA, principe IV).
const MinContrast = 4.5

// tintAlpha est l'opacité des fonds teintés (bg-brand/10).
const tintAlpha = 0.10

// Palette regroupe les couleurs de l'association et leurs teintes dérivées :
// Brand et Accent servent de fond plein, On* au texte posé dessus, *Ink au
// texte et aux repères dans cette teinte sur les fonds clairs.
type Palette struct {
	Brand, BrandInk, OnBrand    RGB
	Accent, AccentInk, OnAccent RGB
}

// Default est la palette de Frigg.
var Default = Derive(nil, nil)

// Derive calcule la palette ; nil = couleur par défaut.
func Derive(primary, accent *RGB) Palette {
	p, a := DefaultPrimary, DefaultAccent
	if primary != nil {
		p = *primary
	}
	if accent != nil {
		a = *accent
	}
	return Palette{
		Brand: p, BrandInk: ink(p), OnBrand: on(p),
		Accent: a, AccentInk: ink(a), OnAccent: on(a),
	}
}

// on renvoie la couleur du texte posé sur c.
func on(c RGB) RGB {
	for _, t := range []RGB{White, Ink} {
		if Contrast(t, c) >= MinContrast {
			return t
		}
	}
	return Black
}

// ink assombrit c (luminosité OKLCH, teinte conservée) jusqu'à un contraste
// suffisant avec le plus foncé des fonds clairs : canvas et le fond teinté.
func ink(c RGB) RGB {
	bg := Canvas
	if tint := Mix(c, Canvas, tintAlpha); tint.Luminance() < bg.Luminance() {
		bg = tint
	}
	o := c.oklch()
	cur := c
	for Contrast(cur, bg) < MinContrast {
		o.L -= 0.02
		if o.L <= 0 {
			return Ink
		}
		cur = o.rgb()
	}
	return cur
}
