package logo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
	"os"
	"testing"
)

func pngOf(t *testing.T, w, h int, alpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			a := uint8(255)
			if alpha && x < w/2 {
				a = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{uint8(x), uint8(y), 120, a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// noisyJPEG est une photo très détaillée (bruit), difficile à compresser.
func noisyJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewPCG(1, 2))
	for i := range img.Pix {
		img.Pix[i] = uint8(r.IntN(256))
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withEXIF insère un segment APP1 « Exif » contenant un marqueur après SOI.
func withEXIF(data []byte, marker string) []byte {
	payload := append([]byte("Exif\x00\x00"), marker...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	out := append([]byte{}, data[:2]...)
	out = append(out, seg...)
	out = append(out, payload...)
	return append(out, data[2:]...)
}

func TestProcessAccepted(t *testing.T) {
	webp, err := os.ReadFile("testdata/logo.webp")
	if err != nil {
		t.Fatal(err)
	}
	photo := withEXIF(noisyJPEG(t, 600, 450), "GPS-48.8566-2.3522")
	for _, c := range []struct {
		name, src, out string
		data           []byte
	}{
		{"PNG transparent", "image/png", "image/png", pngOf(t, 800, 400, true)},
		{"JPEG avec EXIF", "image/jpeg", "image/jpeg", photo},
		{"WebP sans perte", "image/webp", "image/png", webp},
	} {
		t.Run(c.name, func(t *testing.T) {
			v, err := Process(bytes.NewReader(c.data))
			if err != nil {
				t.Fatal(err)
			}
			if v.SourceType != c.src || v.Web.Type != c.out || v.Email.Type != c.out || v.SourceSize != int64(len(c.data)) {
				t.Fatalf("types : %s %s %s", v.SourceType, v.Web.Type, v.Email.Type)
			}
			if max(v.Web.Width, v.Web.Height) > WebMaxSide {
				t.Errorf("variante web %dx%d", v.Web.Width, v.Web.Height)
			}
			if v.Email.Height > EmailMaxH || v.Email.Width > EmailMaxW || len(v.Email.Data) > EmailMaxBytes {
				t.Errorf("variante e-mail %dx%d, %d octets", v.Email.Width, v.Email.Height, len(v.Email.Data))
			}
			for _, d := range [][]byte{v.Web.Data, v.Email.Data} {
				if bytes.Contains(d, []byte("Exif")) || bytes.Contains(d, []byte("GPS-48")) {
					t.Error("métadonnées conservées")
				}
				if _, _, err := image.Decode(bytes.NewReader(d)); err != nil {
					t.Errorf("variante illisible : %v", err)
				}
			}
		})
	}
}

func TestProcessKeepsSmallImages(t *testing.T) {
	v, err := Process(bytes.NewReader(pngOf(t, 40, 20, false)))
	if err != nil {
		t.Fatal(err)
	}
	if v.Web.Width != 40 || v.Web.Height != 20 || v.Email.Width != 40 {
		t.Errorf("image agrandie ou réduite : %+v %+v", v.Web.Width, v.Email.Width)
	}
}

func TestProcessRejected(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 4, 4), []color.Color{color.Black}), nil)
	full := pngOf(t, 20, 20, false)
	for _, c := range []struct {
		name string
		data []byte
		want error
	}{
		{"SVG", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ErrUnsupported},
		{"GIF", g.Bytes(), ErrUnsupported},
		{"PDF", []byte("%PDF-1.7\n1 0 obj\n"), ErrUnsupported},
		{"PNG tronqué", full[:len(full)/2], ErrUnsupported},
		{"texte renommé", []byte("bonjour, je ne suis pas une image"), ErrUnsupported},
		{"20 000 × 1 px", pngOf(t, 20000, 1, false), ErrTooLarge},
		{"5 000 × 5 000 px", fakePNGHeader(5000, 5000), ErrTooLarge},
		{"plus de 1 Mo", append(pngOf(t, 10, 10, false), make([]byte, MaxBytes)...), ErrTooBig},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Process(bytes.NewReader(c.data)); !errors.Is(err, c.want) {
				t.Errorf("erreur %v, attendu %v", err, c.want)
			}
		})
	}
}

// fakePNGHeader produit une signature et un en-tête IHDR annonçant w × h
// pixels, sans données : le refus doit venir de DecodeConfig, avant décodage.
func fakePNGHeader(w, h int) []byte {
	full := pngOfSize(1, 1)
	binary.BigEndian.PutUint32(full[16:20], uint32(w))
	binary.BigEndian.PutUint32(full[20:24], uint32(h))
	binary.BigEndian.PutUint32(full[29:33], crc32.ChecksumIEEE(full[12:29]))
	return full
}

func pngOfSize(w, h int) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h)))
	return buf.Bytes()
}
