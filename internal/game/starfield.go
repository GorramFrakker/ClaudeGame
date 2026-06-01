package game

import (
	"image"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type star struct {
	pos    Vec2
	speed  float64 // px/sec downward
	size   float64
	bright float64
	layer  int
}

// Starfield draws a scrolling, multi-layer parallax background with a subtle
// vertical gradient. It is shared by menus and gameplay for visual continuity.
type Starfield struct {
	stars []star
	rng   *rand.Rand
	bg    *ebiten.Image
}

func newStarfield(rng *rand.Rand) *Starfield {
	sf := &Starfield{rng: rng, bg: makeBackground()}
	const n = 150
	for i := 0; i < n; i++ {
		sf.stars = append(sf.stars, sf.newStar(rng.Float64()*ScreenH))
	}
	return sf
}

func (s *Starfield) newStar(y float64) star {
	layer := s.rng.Intn(3)
	var speed, size, bright float64
	switch layer {
	case 0: // far
		speed = 18 + s.rng.Float64()*14
		size = 1
		bright = 0.30 + s.rng.Float64()*0.20
	case 1: // mid
		speed = 40 + s.rng.Float64()*26
		size = 1.6
		bright = 0.5 + s.rng.Float64()*0.25
	default: // near
		speed = 80 + s.rng.Float64()*60
		size = 2.4
		bright = 0.78 + s.rng.Float64()*0.22
	}
	return star{
		pos:    V(s.rng.Float64()*ScreenW, y),
		speed:  speed,
		size:   size,
		bright: bright,
		layer:  layer,
	}
}

// Update scrolls stars. driftX nudges stars horizontally (parallax response to
// player movement); speedMul lets gameplay accelerate the field for intensity.
func (s *Starfield) Update(driftX, speedMul float64) {
	for i := range s.stars {
		st := &s.stars[i]
		st.pos.Y += st.speed * speedMul * dt
		st.pos.X -= driftX * (0.2 + float64(st.layer)*0.5) * dt
		if st.pos.Y > ScreenH+2 {
			*st = s.newStar(-2)
		}
		if st.pos.X < -4 {
			st.pos.X += ScreenW + 8
		} else if st.pos.X > ScreenW+4 {
			st.pos.X -= ScreenW + 8
		}
	}
}

func (s *Starfield) Draw(dst *ebiten.Image) {
	dst.DrawImage(s.bg, nil)
	for i := range s.stars {
		st := &s.stars[i]
		a := uint8(clampf(st.bright, 0, 1) * 255)
		c := color.RGBA{uint8(200 * st.bright), uint8(215 * st.bright), 255, a}
		vector.DrawFilledRect(dst, float32(st.pos.X), float32(st.pos.Y), float32(st.size), float32(st.size), c, false)
		if st.layer == 2 {
			drawGlow(dst, st.pos.X+st.size/2, st.pos.Y+st.size/2, st.size*2.2, color.RGBA{120, 150, 255, 255}, 0.35*st.bright)
		}
	}
}

// makeBackground builds the static vertical gradient backdrop once.
func makeBackground() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, ScreenW, ScreenH))
	for y := 0; y < ScreenH; y++ {
		t := float64(y) / float64(ScreenH)
		// darker toward the bottom, faint purple glow near the top
		c := mixRGB(colBGTop, colBG, t)
		// add a soft vignette horizontally
		for x := 0; x < ScreenW; x++ {
			dx := (float64(x)/float64(ScreenW) - 0.5) * 2
			vig := 1 - 0.18*dx*dx
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(c.R) * vig),
				G: uint8(float64(c.G) * vig),
				B: uint8(float64(c.B) * vig),
				A: 255,
			})
		}
	}
	return ebiten.NewImageFromImage(img)
}
