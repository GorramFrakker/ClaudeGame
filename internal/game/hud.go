package game

import (
	"image/color"
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func commaInt(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		s = s[1:]
	}
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += ","
		}
		out += string(c)
	}
	if n < 0 {
		out = "-" + out
	}
	return out
}

func iconShip(dst *ebiten.Image, x, y, s float64, c color.RGBA) {
	body := []Vec2{{0, -s}, {s * 0.8, s * 0.7}, {0, s * 0.35}, {-s * 0.8, s * 0.7}}
	drawPoly(dst, body, V(x, y), 0, 1, scaleRGB(c, 0.7), c, 0, color.RGBA{}, 0)
}

func iconBomb(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	drawGlow(dst, x, y, r*2.2, c, 0.5)
	pcircle(dst, V(x, y), r, scaleRGB(c, 0.7))
	vector.StrokeCircle(dst, float32(x), float32(y), float32(r), 1.4, c, true)
	vector.StrokeLine(dst, float32(x-r*0.5), float32(y), float32(x+r*0.5), float32(y), 1.2, c, true)
	vector.StrokeLine(dst, float32(x), float32(y-r*0.5), float32(x), float32(y+r*0.5), 1.2, c, true)
}

func (p *Play) drawHUD(dst *ebiten.Image) {
	f := p.g.fonts

	// Top gradient strip for legibility.
	vector.DrawFilledRect(dst, 0, 0, ScreenW, 52, color.RGBA{0, 0, 0, 90}, false)

	// Row 1: score + hi-score, wave label.
	f.left(dst, commaInt(int(p.displayScore)), 12, 18, 22, colUIText, true)
	f.left(dst, "HI "+commaInt(maxi(p.g.save.HighScore, p.score)), 13, 38, 11, colUIDim, false)
	f.right(dst, p.director.hudLabel(), ScreenW-12, 18, 14, colGold, true)

	// Combo multiplier.
	if m := p.multiplier(); m > 1 {
		pulse := 1 + 0.08*math.Sin(p.time*16)
		col := mixRGB(colGold, colDanger, clampf(float64(m-1)/7, 0, 1))
		f.center(dst, "x"+strconv.Itoa(m), ScreenW/2, 18, 17*pulse, col, true)
		// combo timer bar
		w := 44.0 * clampf(p.comboTimer/3, 0, 1)
		vector.DrawFilledRect(dst, float32(ScreenW/2-22), 30, float32(w), 3, withAlpha(col, 200), false)
	}

	// Row 2: lives, weapon pips, bombs.
	lifeY := 44.0
	for i := 0; i < p.player.lives; i++ {
		iconShip(dst, 18+float64(i)*16, lifeY, 6, colPlayer)
	}
	// bombs (right side)
	for i := 0; i < p.player.bombs; i++ {
		iconBomb(dst, float64(ScreenW-18-i*16), lifeY, 5, colBombPU)
	}

	// Boss health bar.
	if p.boss != nil {
		p.drawBossBar(dst)
	}

	// Shield bar across the very bottom.
	p.drawShieldBar(dst)

	// Banner (wave / warning).
	p.drawBanner(dst)

	// Brief control hint at the start of the run.
	if p.time < 7 && p.director.idx == 0 {
		a := clampf(minf(p.time, 7-p.time)/1.0, 0, 1)
		f.center(dst, "MOVE: WASD / ARROWS    FIRE: Z / SPACE", ScreenW/2, ScreenH-70, 12, withAlpha(colUIText, uint8(a*220)), false)
		f.center(dst, "BOMB: X      FOCUS: SHIFT", ScreenW/2, ScreenH-52, 12, withAlpha(colUIText, uint8(a*220)), false)
	}
}

func (p *Play) drawShieldBar(dst *ebiten.Image) {
	frac := clampf(p.player.shield/playerMaxShield, 0, 1)
	const h = 6.0
	y := float32(ScreenH - h - 2)
	const pad = 12
	w := float64(ScreenW - 2*pad)
	// backing
	vector.DrawFilledRect(dst, pad, y, float32(w), h, color.RGBA{30, 36, 54, 200}, false)
	col := mixRGB(colDanger, colHealth, frac)
	vector.DrawFilledRect(dst, pad, y, float32(w*frac), h, col, false)
	drawGlow(dst, pad+w*frac, float64(y)+h/2, 10, col, 0.5)
	// weapon level label on the bar
	p.g.fonts.left(dst, "PWR "+strconv.Itoa(p.player.weapon), pad+2, float64(y)-9, 10, colUIDim, true)
}

func (p *Play) drawBossBar(dst *ebiten.Image) {
	b := p.boss
	if b.intro {
		return
	}
	const pad = 30
	w := float64(ScreenW - 2*pad)
	y := float32(58)
	frac := b.hpFrac()
	vector.DrawFilledRect(dst, pad, y, float32(w), 8, color.RGBA{40, 12, 24, 220}, false)
	col := mixRGB(colDanger, b.col(), frac)
	vector.DrawFilledRect(dst, pad, y, float32(w*frac), 8, col, false)
	// phase dividers
	for _, t := range []float64{0.34, 0.67} {
		x := float32(pad + w*t)
		vector.StrokeLine(dst, x, y, x, y+8, 1, color.RGBA{0, 0, 0, 200}, false)
	}
	vector.StrokeRect(dst, pad, y, float32(w), 8, 1, withAlpha(b.col(), 200), false)
	p.g.fonts.center(dst, b.name, ScreenW/2, float64(y)-9, 13, withAlpha(colUIText, 235), true)
}

func (p *Play) drawBanner(dst *ebiten.Image) {
	a := p.director.bannerAlpha()
	if a <= 0 {
		return
	}
	s := p.director.cur()
	if s == nil {
		return
	}
	f := p.g.fonts
	y := ScreenH * 0.40
	if s.title != "" {
		size := 40.0
		if s.kind != skWave {
			size = 34
		}
		f.center(dst, s.title, ScreenW/2, y, size, withAlpha(s.col, uint8(a*255)), true)
	}
	if s.sub != "" {
		f.center(dst, s.sub, ScreenW/2, y+34, 16, withAlpha(colUIText, uint8(a*230)), false)
	}
}
