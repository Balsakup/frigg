package brand

import "testing"

func TestParseHex(t *testing.T) {
	c, err := ParseHex("#1F5278")
	if err != nil || c.Hex() != "#1f5278" || c.Triplet() != "31 82 120" {
		t.Fatalf("ParseHex : %v %v", c, err)
	}
	for _, bad := range []string{"#fff", "1f5278", "#12345g", "", "#1f52789"} {
		if _, err := ParseHex(bad); err == nil {
			t.Errorf("%q accepté", bad)
		}
	}
}

func TestContrast(t *testing.T) {
	if c := Contrast(White, Black); c < 20.99 || c > 21.01 {
		t.Errorf("blanc/noir = %v", c)
	}
	if c := Contrast(White, White); c != 1 {
		t.Errorf("blanc/blanc = %v", c)
	}
}

func check(t *testing.T, primary, accent RGB) {
	t.Helper()
	p := Derive(&primary, &accent)
	if p.Brand != primary || p.Accent != accent {
		t.Fatalf("couleurs choisies modifiées : %v", p)
	}
	for _, c := range []struct {
		name string
		fg   RGB
		bgs  []RGB
	}{
		{"onBrand", p.OnBrand, []RGB{p.Brand}},
		{"onAccent", p.OnAccent, []RGB{p.Accent}},
		{"brandInk", p.BrandInk, []RGB{Canvas, White, Mix(p.Brand, Canvas, 0.10), Mix(p.Brand, White, 0.10), Mix(p.Brand, White, 0.05)}},
		{"accentInk", p.AccentInk, []RGB{Canvas, White, Mix(p.Accent, Canvas, 0.10)}},
	} {
		for _, bg := range c.bgs {
			if r := Contrast(c.fg, bg); r < MinContrast {
				t.Errorf("%s %s sur %s (couleur %s) : %.2f", c.name, c.fg.Hex(), bg.Hex(), primary.Hex(), r)
			}
		}
	}
}

func TestDeriveContrastSweep(t *testing.T) {
	samples := []RGB{
		MustHex("#fff3b0"), MustHex("#ffff00"), MustHex("#777777"), Black, White,
		MustHex("#00ff00"), MustHex("#00ffff"), MustHex("#ff00ff"), MustHex("#1f5278"),
	}
	for r := 0; r <= 255; r += 51 {
		for g := 0; g <= 255; g += 51 {
			for b := 0; b <= 255; b += 51 {
				samples = append(samples, RGB{uint8(r), uint8(g), uint8(b)})
			}
		}
	}
	if len(samples) < 200 {
		t.Fatalf("échantillon trop petit : %d", len(samples))
	}
	for _, c := range samples {
		check(t, c, c)
	}
}

func TestDefault(t *testing.T) {
	if Default.Brand.Hex() != "#1f5278" || Default.Accent.Hex() != "#c44822" {
		t.Errorf("palette par défaut : %v", Default)
	}
	if Default.BrandInk != Default.Brand || Default.OnBrand != White {
		t.Errorf("la couleur par défaut est déjà lisible : %s %s", Default.BrandInk.Hex(), Default.OnBrand.Hex())
	}
}
