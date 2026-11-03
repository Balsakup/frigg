// Package logo vérifie et réencode le logo déposé (specs/010-general-settings
// › R4). Le fichier reçu n'est jamais conservé tel quel : il est décodé puis
// réencodé en deux variantes, ce qui retire les métadonnées (EXIF, GPS) et
// neutralise les fichiers polyglottes.
package logo

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // décodeur WebP enregistré auprès de image
)

// Bornes du logo.
const (
	MaxBytes      = 1 << 20 // « 1 Mo » de la spec
	MaxPixels     = 16_000_000
	MaxSide       = 8000
	WebMaxSide    = 512
	EmailMaxH     = 96
	EmailMaxW     = 400
	EmailMaxBytes = 100 * 1024
	jpegQuality   = 88
)

var (
	// ErrUnsupported : le contenu n'est pas une image PNG, JPEG ou WebP lisible.
	ErrUnsupported = errors.New("format d'image non accepté")
	// ErrTooBig : le fichier dépasse MaxBytes.
	ErrTooBig = errors.New("fichier trop volumineux")
	// ErrTooLarge : l'image dépasse les bornes en pixels.
	ErrTooLarge = errors.New("image trop grande")
)

// Encoded est une variante encodée.
type Encoded struct {
	Data          []byte
	Type          string
	Width, Height int
}

// Variants est le résultat du traitement d'un logo.
type Variants struct {
	SourceType string
	SourceSize int64
	Web, Email Encoded
}

var sourceTypes = map[string]string{"png": "image/png", "jpeg": "image/jpeg", "webp": "image/webp"}

// Process lit au plus MaxBytes+1 octets de r, vérifie le format et les
// dimensions avant tout décodage complet, puis produit les deux variantes.
func Process(r io.Reader) (Variants, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Variants{}, err
	}
	if len(data) > MaxBytes {
		return Variants{}, ErrTooBig
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Variants{}, ErrUnsupported
	}
	srcType, ok := sourceTypes[format]
	if !ok {
		return Variants{}, ErrUnsupported
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return Variants{}, ErrUnsupported
	}
	if cfg.Width > MaxSide || cfg.Height > MaxSide || cfg.Width*cfg.Height > MaxPixels {
		return Variants{}, ErrTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Variants{}, ErrUnsupported
	}

	asJPEG := format == "jpeg"
	w, h := fit(cfg.Width, cfg.Height, WebMaxSide, WebMaxSide)
	web, err := encode(scale(img, w, h), asJPEG)
	if err != nil {
		return Variants{}, err
	}
	w, h = fit(cfg.Width, cfg.Height, EmailMaxW, EmailMaxH)
	var email Encoded
	for {
		email, err = encode(scale(img, w, h), asJPEG)
		if err != nil {
			return Variants{}, err
		}
		if len(email.Data) <= EmailMaxBytes || w <= 16 || h <= 16 {
			break
		}
		w, h = max(1, w*3/4), max(1, h*3/4)
	}
	if len(email.Data) > EmailMaxBytes {
		return Variants{}, ErrTooLarge
	}
	return Variants{SourceType: srcType, SourceSize: int64(len(data)), Web: web, Email: email}, nil
}

// fit réduit (sans jamais agrandir) en conservant les proportions.
func fit(w, h, maxW, maxH int) (int, int) {
	if w <= maxW && h <= maxH {
		return w, h
	}
	ratio := min(float64(maxW)/float64(w), float64(maxH)/float64(h))
	return max(1, int(float64(w)*ratio+0.5)), max(1, int(float64(h)*ratio+0.5))
}

func scale(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

func encode(img *image.NRGBA, asJPEG bool) (Encoded, error) {
	var buf bytes.Buffer
	b := img.Bounds()
	if asJPEG {
		// Une source JPEG n'a pas de transparence ; fond blanc par sécurité.
		flat := image.NewRGBA(b)
		draw.Draw(flat, b, image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(flat, b, img, b.Min, draw.Over)
		if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return Encoded{}, err
		}
		return Encoded{Data: buf.Bytes(), Type: "image/jpeg", Width: b.Dx(), Height: b.Dy()}, nil
	}
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return Encoded{}, err
	}
	return Encoded{Data: buf.Bytes(), Type: "image/png", Width: b.Dx(), Height: b.Dy()}, nil
}
