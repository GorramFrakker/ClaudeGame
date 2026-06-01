package game

import (
	"bytes"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// Fonts caches text faces. GoTextFaceSource caches rasterized glyphs per size
// internally, so we keep one source per weight and build lightweight faces on
// demand, memoizing them by (size, weight).
type Fonts struct {
	reg   *text.GoTextFaceSource
	bold  *text.GoTextFaceSource
	cache map[fontKey]*text.GoTextFace
}

type fontKey struct {
	size int
	bold bool
}

func newFonts() (*Fonts, error) {
	reg, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, err
	}
	bold, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		return nil, err
	}
	return &Fonts{reg: reg, bold: bold, cache: map[fontKey]*text.GoTextFace{}}, nil
}

func (f *Fonts) face(size float64, bold bool) *text.GoTextFace {
	k := fontKey{size: int(math.Round(size * 2)), bold: bold}
	if fc, ok := f.cache[k]; ok {
		return fc
	}
	src := f.reg
	if bold {
		src = f.bold
	}
	fc := &text.GoTextFace{Source: src, Size: size}
	f.cache[k] = fc
	return fc
}

// text alignment shorthands
const (
	alStart  = text.AlignStart
	alCenter = text.AlignCenter
	alEnd    = text.AlignEnd
)

// drawText draws text with the given horizontal/vertical anchor alignment.
// (x,y) is the anchor point; alignment controls how the text sits around it.
func (f *Fonts) drawText(dst *ebiten.Image, s string, x, y, size float64, clr color.Color, bold bool, ah, av text.Align) {
	face := f.face(size, bold)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	op.LineSpacing = size * 1.25
	op.PrimaryAlign = ah
	op.SecondaryAlign = av
	text.Draw(dst, s, face, op)
}

// center draws horizontally and vertically centered text at (x,y).
func (f *Fonts) center(dst *ebiten.Image, s string, x, y, size float64, clr color.Color, bold bool) {
	f.drawText(dst, s, x, y, size, clr, bold, alCenter, alCenter)
}

// left draws left-aligned, vertically centered text with its left edge at x.
func (f *Fonts) left(dst *ebiten.Image, s string, x, y, size float64, clr color.Color, bold bool) {
	f.drawText(dst, s, x, y, size, clr, bold, alStart, alCenter)
}

// right draws right-aligned, vertically centered text with its right edge at x.
func (f *Fonts) right(dst *ebiten.Image, s string, x, y, size float64, clr color.Color, bold bool) {
	f.drawText(dst, s, x, y, size, clr, bold, alEnd, alCenter)
}

func (f *Fonts) measure(s string, size float64, bold bool) (w, h float64) {
	face := f.face(size, bold)
	return text.Measure(s, face, size*1.25)
}

// makeGlow builds a radial-gradient sprite used for additive glows, soft
// particles, and bullet halos. White with a smooth alpha falloff so it can be
// tinted to any color at draw time.
func makeGlow(size int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := float64(size-1) / 2
	rad := c
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - c
			dy := float64(y) - c
			d := math.Hypot(dx, dy) / rad
			a := 0.0
			if d < 1 {
				// smooth falloff, brighter core
				f := 1 - d
				a = f * f
			}
			// Store premultiplied-alpha white (Go's image.RGBA is premultiplied).
			// This matters for additive blending, which uses the premultiplied
			// RGB directly: writing 255 here would make even transparent pixels
			// add full white and render the sprite as a solid square.
			v := uint8(a * 255)
			img.SetRGBA(x, y, color.RGBA{v, v, v, v})
		}
	}
	return ebiten.NewImageFromImage(img)
}

// glow is a shared additive glow sprite (centered, visible radius ~ size/2).
var glow = makeGlow(128)

const glowVisibleR = 63.5 // visible radius of the glow sprite in source pixels

// drawGlow renders an additive colored glow of the given world radius at (x,y).
func drawGlow(dst *ebiten.Image, x, y, radius float64, clr color.RGBA, intensity float64) {
	if radius <= 0 || intensity <= 0 {
		return
	}
	s := radius / glowVisibleR
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-glowVisibleR, -glowVisibleR)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(x, y)
	// Apply intensity to RGB (not alpha): additive blending uses the
	// premultiplied RGB, so scaling alpha alone would not dim the glow.
	ci := color.RGBA{
		R: uint8(clampf(float64(clr.R)*intensity, 0, 255)),
		G: uint8(clampf(float64(clr.G)*intensity, 0, 255)),
		B: uint8(clampf(float64(clr.B)*intensity, 0, 255)),
		A: 255,
	}
	op.ColorScale.ScaleWithColor(ci)
	op.Blend = ebiten.BlendLighter
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(glow, op)
}
