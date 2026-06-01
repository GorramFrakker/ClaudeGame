package game

import "image/color"

// Internal render resolution. The window scales this up with letterboxing,
// so the game looks identical at any window size. Portrait orientation suits
// a vertical shoot-'em-up.
const (
	ScreenW = 480
	ScreenH = 720
)

// dt is the fixed simulation step. Ebitengine ticks Update at 60 TPS, so each
// update advances the world by exactly this much. Using a constant keeps the
// simulation deterministic and easy to reason about.
const dt = 1.0 / 60.0

// Mode is the top-level game state.
type Mode int

const (
	ModeTitle Mode = iota
	ModeHowTo
	ModePlaying
	ModePaused
	ModeGameOver
	ModeVictory
)

// Palette. A cohesive neon-on-deep-space look.
var (
	colBG        = color.RGBA{8, 9, 22, 255}
	colBGTop     = color.RGBA{18, 14, 42, 255}
	colPlayer    = color.RGBA{120, 235, 255, 255}
	colPlayerHot = color.RGBA{235, 255, 255, 255}
	colThruster  = color.RGBA{120, 180, 255, 255}
	colPBullet   = color.RGBA{150, 245, 255, 255}

	colGrunt  = color.RGBA{255, 90, 110, 255}
	colWeaver = color.RGBA{255, 120, 220, 255}
	colDiver  = color.RGBA{255, 170, 70, 255}
	colTurret = color.RGBA{180, 130, 255, 255}
	colBomber = color.RGBA{120, 255, 160, 255}
	colMine   = color.RGBA{255, 210, 90, 255}
	colBoss   = color.RGBA{255, 80, 150, 255}

	colEBullet  = color.RGBA{255, 180, 90, 255}
	colEBullet2 = color.RGBA{255, 110, 170, 255}
	colEBullet3 = color.RGBA{170, 120, 255, 255}

	colShieldPU = color.RGBA{120, 255, 200, 255}
	colPowerPU  = color.RGBA{255, 230, 120, 255}
	colBombPU   = color.RGBA{255, 140, 230, 255}

	colUIText = color.RGBA{222, 232, 248, 255}
	colUIDim  = color.RGBA{120, 132, 160, 255}
	colGold   = color.RGBA{255, 214, 102, 255}
	colDanger = color.RGBA{255, 90, 110, 255}
	colHealth = color.RGBA{120, 235, 255, 255}
)

// withAlpha returns a copy of c with alpha a (0..255).
func withAlpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

// scaleRGB returns c scaled toward black by f (0..1), keeping alpha.
func scaleRGB(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * f),
		G: uint8(float64(c.G) * f),
		B: uint8(float64(c.B) * f),
		A: c.A,
	}
}

// mixRGB linearly blends two colors by t (0 -> a, 1 -> b).
func mixRGB(a, b color.RGBA, t float64) color.RGBA {
	t = clampf(t, 0, 1)
	return color.RGBA{
		R: uint8(lerp(float64(a.R), float64(b.R), t)),
		G: uint8(lerp(float64(a.G), float64(b.G), t)),
		B: uint8(lerp(float64(a.B), float64(b.B), t)),
		A: uint8(lerp(float64(a.A), float64(b.A), t)),
	}
}
