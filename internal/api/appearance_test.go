package api_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/balsakup/frigg/internal/api"
	"github.com/balsakup/frigg/internal/api/apitest"
	"github.com/balsakup/frigg/internal/auth"
)

func testPNG(t *testing.T, w, h int, padding int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 200, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return append(buf.Bytes(), make([]byte, padding)...)
}

func uploadLogo(t *testing.T, c *apitest.Client, name string, data []byte) apitest.Response {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	return c.Do("PUT", "/api/admin/settings/appearance/logo", apitest.Raw(buf.Bytes()), "Content-Type", w.FormDataContentType())
}

func expectAppearance(t *testing.T, res apitest.Response) api.Appearance {
	t.Helper()
	if res.Status != http.StatusOK {
		t.Fatalf("statut %d : %s", res.Status, res.Body)
	}
	var a api.Appearance
	res.JSON(t, &a)
	return a
}

func logoDir(t *testing.T, env *apitest.Env) []string {
	t.Helper()
	entries, err := os.ReadDir(env.LogoFiles.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestAppearanceColors(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))

	a := expectAppearance(t, admin.Do("GET", "/api/admin/settings/appearance", nil))
	if a.PrimaryColor != nil || a.Palette.Brand != "#1f5278" || a.Logo != nil {
		t.Fatalf("état initial : %+v", a)
	}
	a = expectAppearance(t, admin.Do("PUT", "/api/admin/settings/appearance", map[string]any{"primaryColor": "#FFF3B0", "accentColor": nil}))
	if *a.PrimaryColor != "#fff3b0" || a.Palette.Brand != "#fff3b0" || a.Palette.OnBrand != "#181f29" || a.Palette.BrandInk == "#fff3b0" || a.AccentColor != nil {
		t.Errorf("couleurs : %+v", a)
	}
	expectFieldError(t, admin.Do("PUT", "/api/admin/settings/appearance", map[string]any{"primaryColor": "#fff", "accentColor": nil}),
		"body.primaryColor", api.MsgColorInvalid)

	res := admin.Do("GET", "/api/admin/settings/appearance/palette?primary=%23000000", nil)
	var p api.Palette
	res.JSON(t, &p)
	if res.Status != http.StatusOK || p.Brand != "#000000" || p.OnBrand != "#ffffff" {
		t.Errorf("aperçu : %d %+v", res.Status, p)
	}
	if a := expectAppearance(t, admin.Do("GET", "/api/admin/settings/appearance", nil)); *a.PrimaryColor != "#fff3b0" {
		t.Errorf("l'aperçu a modifié les couleurs : %+v", a)
	}
	a = expectAppearance(t, admin.Do("PUT", "/api/admin/settings/appearance", map[string]any{"primaryColor": nil, "accentColor": nil}))
	if a.PrimaryColor != nil || a.Palette.Brand != "#1f5278" {
		t.Errorf("rétablissement : %+v", a)
	}
	if !strings.Contains(env.Logs.String(), "couleurs modifiées") {
		t.Error("journal des couleurs absent")
	}
}

func TestAppearanceLogo(t *testing.T) {
	env := apitest.New(t)
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))

	a := expectAppearance(t, uploadLogo(t, admin, "../Logo du club.png", testPNG(t, 800, 400, 0)))
	if a.Logo == nil || a.Logo.Width != 512 || a.Logo.Height != 256 || strings.Contains(a.Logo.OriginalName, "/") {
		t.Fatalf("logo : %+v", a.Logo)
	}
	first := a.Logo.URL
	if n := len(logoDir(t, env)); n != 2 {
		t.Errorf("%d fichier(s) dans le répertoire du logo, attendu 2", n)
	}

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	expectFieldError(t, uploadLogo(t, admin, "logo.svg", svg), "form.file", api.MsgLogoFormat)
	expectFieldError(t, uploadLogo(t, admin, "gros.png", testPNG(t, 10, 10, 1<<20)), "form.file", api.MsgLogoFormat)
	if res := uploadLogo(t, admin, "enorme.png", make([]byte, 2<<20)); res.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("corps de 2 Mio : %d", res.Status)
	}
	if a := expectAppearance(t, admin.Do("GET", "/api/admin/settings/appearance", nil)); a.Logo == nil || a.Logo.URL != first {
		t.Errorf("logo modifié par un dépôt refusé : %+v", a.Logo)
	}

	a = expectAppearance(t, uploadLogo(t, admin, "nouveau.png", testPNG(t, 100, 100, 0)))
	if a.Logo.URL == first || len(logoDir(t, env)) != 2 {
		t.Errorf("remplacement : %s, fichiers %v", a.Logo.URL, logoDir(t, env))
	}
	anon := env.Client()
	if res := anon.Do("GET", first, nil); res.Status != http.StatusNotFound {
		t.Errorf("ancien logo : %d", res.Status)
	}

	if res := admin.Do("DELETE", "/api/admin/settings/appearance/logo", nil); res.Status != http.StatusNoContent {
		t.Fatalf("retrait : %d %s", res.Status, res.Body)
	}
	if names := logoDir(t, env); len(names) != 0 {
		t.Errorf("fichiers restants : %v", names)
	}
	if res := admin.Do("DELETE", "/api/admin/settings/appearance/logo", nil); res.Status != http.StatusNoContent {
		t.Errorf("retrait sans logo : %d", res.Status)
	}
	logs := env.Logs.String()
	for _, want := range []string{"logo déposé", "logo retiré"} {
		if !strings.Contains(logs, want) {
			t.Errorf("journal %q absent", want)
		}
	}
}

// Identité publique : exactement nom, logo et palette (principe II v1.8.0).
func TestIdentityPublic(t *testing.T) {
	env := apitest.New(t)
	env.SetToday("2026-11-03")
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	anon := env.Client()

	res := anon.Do("GET", "/api/identity", nil)
	var id api.Identity
	res.JSON(t, &id)
	if res.Status != http.StatusOK || id.Name != "Mon association" || id.LogoURL != nil || id.Palette.Brand != "#1f5278" {
		t.Fatalf("identité par défaut : %d %+v", res.Status, id)
	}

	if r := putGeneral(admin, map[string]any{
		"name": "Les Sentiers du Val", "tagline": "Secret tagline", "siret": "73282932000074",
		"contactEmail": "bureau@sentiers.fr", "phone": "0123456789",
	}); r.Status != http.StatusOK {
		t.Fatalf("paramètres : %d %s", r.Status, r.Body)
	}
	a := expectAppearance(t, uploadLogo(t, admin, "logo.png", testPNG(t, 300, 100, 0)))

	res = anon.Do("GET", "/api/identity", nil)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(res.Body), &raw); err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for k := range raw {
		keys = append(keys, k)
	}
	if len(raw) != 3 || raw["name"] == nil || raw["logoUrl"] == nil || raw["palette"] == nil {
		t.Errorf("clés de l'identité publique : %v", keys)
	}
	for _, secret := range []string{"Secret tagline", "73282932000074", "bureau@", "0123456789", "logo.png"} {
		if strings.Contains(res.Body, secret) {
			t.Errorf("%q exposé sans session", secret)
		}
	}
	if res.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("Cache-Control : %q", res.Header.Get("Cache-Control"))
	}

	img := anon.Do("GET", a.Logo.URL, nil)
	if img.Status != http.StatusOK || img.Header.Get("Content-Type") != "image/png" ||
		img.Header.Get("X-Content-Type-Options") != "nosniff" || img.Header.Get("Content-Disposition") != "inline" ||
		img.Header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("logo public : %d %v", img.Status, img.Header)
	}
	if _, err := png.Decode(strings.NewReader(img.Body)); err != nil {
		t.Errorf("logo illisible : %v", err)
	}
	var emailName string
	if err := env.Store.ReadDB.QueryRow(`SELECT email_name FROM association_logo`).Scan(&emailName); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"/api/identity/logo/" + emailName, "/api/identity/logo/..%2Fx", "/api/identity/logo/" + strings.Repeat("0", 32)} {
		if r := anon.Do("GET", bad, nil); r.Status != http.StatusNotFound {
			t.Errorf("%s : %d", bad, r.Status)
		}
	}

	var sess api.Session
	admin.Do("GET", "/api/session", nil).JSON(t, &sess)
	if sess.Association.LogoURL == nil || *sess.Association.LogoURL != a.Logo.URL {
		t.Errorf("logo de la session : %v", sess.Association.LogoURL)
	}
	for _, path := range []string{"/api/admin/settings/general", "/api/association/contact", "/api/admin/settings/appearance"} {
		if r := anon.Do("GET", path, nil); r.Status != http.StatusUnauthorized {
			t.Errorf("%s sans session : %d", path, r.Status)
		}
	}

	// E-mail mis en file après la fonctionnalité : habillé à l'envoi, avec le logo.
	if r := admin.Do("POST", "/api/admin/invitations", map[string]string{"email": "thomas@exemple.fr"}); r.Status != http.StatusCreated {
		t.Fatalf("invitation : %d", r.Status)
	}
	var wrap int
	if err := env.Store.ReadDB.QueryRow(`SELECT wrap FROM outbound_emails WHERE recipient = 'thomas@exemple.fr'`).Scan(&wrap); err != nil || wrap != 1 {
		t.Errorf("wrap = %d (%v)", wrap, err)
	}
	env.FlushMail()
	sent, _ := env.Mail.Last("thomas@exemple.fr")
	if sent.Msg.FromName != "Les Sentiers du Val" || len(sent.Msg.Inline) != 1 || !strings.Contains(sent.Msg.HTML, `src="cid:logo"`) {
		t.Errorf("e-mail : expéditeur %q, %d image(s)", sent.Msg.FromName, len(sent.Msg.Inline))
	}
}

// US4 : coordonnées présentées aux membres, sans identité légale.
func TestAssociationContact(t *testing.T) {
	env := apitest.New(t)
	env.SetToday("2026-11-03")
	admin := env.LoggedIn(env.Seed(auth.RoleAdmin, true))
	member := env.LoggedIn(env.Seed(auth.RoleMember, true))
	if r := putGeneral(admin, map[string]any{
		"name": "Club", "rna": "W751234567", "siret": "73282932000074", "purpose": "Objet", "shortName": "CL",
		"contactEmail": "bureau@club.fr", "phone": "0123456789", "social": map[string]any{"instagram": "https://instagram.com/club"},
	}); r.Status != http.StatusOK {
		t.Fatalf("paramètres : %d %s", r.Status, r.Body)
	}
	res := member.Do("GET", "/api/association/contact", nil)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(res.Body), &raw); err != nil || res.Status != http.StatusOK {
		t.Fatalf("%d %s", res.Status, res.Body)
	}
	for _, k := range []string{"rna", "siret", "foundedOn", "purpose", "shortName", "name", "tagline"} {
		if _, ok := raw[k]; ok {
			t.Errorf("%s exposé aux membres", k)
		}
	}
	var c api.AssociationContact
	res.JSON(t, &c)
	if *c.ContactEmail != "bureau@club.fr" || *c.Phone != "0123456789" || *c.Social.Instagram != "https://instagram.com/club" {
		t.Errorf("contact : %+v", c)
	}
}
