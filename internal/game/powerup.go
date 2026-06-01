package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type powerupKind int

const (
	puPower powerupKind = iota
	puShield
	puBomb
	puLife
	puScore
)

// Powerup is a drifting collectible. When the player nears it, it is magnetized
// toward the ship.
type Powerup struct {
	pos    Vec2
	vel    Vec2
	kind   powerupKind
	radius float64
	age    float64
	dead   bool
}

func newPowerup(pos Vec2, kind powerupKind) *Powerup {
	return &Powerup{pos: pos, kind: kind, radius: 11, vel: V(0, 20)}
}

// filterPowerups removes collected/expired power-ups in place.
func filterPowerups(in []*Powerup) []*Powerup {
	out := in[:0]
	for _, pu := range in {
		if !pu.dead {
			out = append(out, pu)
		}
	}
	return out
}

func puColor(k powerupKind) color.RGBA {
	switch k {
	case puPower:
		return colPowerPU
	case puShield:
		return colShieldPU
	case puBomb:
		return colBombPU
	case puLife:
		return colHealth
	default:
		return colGold
	}
}

func puLetter(k powerupKind) string {
	switch k {
	case puPower:
		return "P"
	case puShield:
		return "S"
	case puBomb:
		return "B"
	case puLife:
		return "1"
	default:
		return "$"
	}
}

func (pu *Powerup) update(p *Play) {
	pu.age += dt
	pu.vel.Y = approach(pu.vel.Y, 58, 90*dt)
	sway := 26 * math.Cos(pu.age*2.4)

	if p.player.alive() {
		d := dist(pu.pos, p.player.pos)
		const magnet = 120.0
		if d < magnet {
			dir := p.player.pos.Sub(pu.pos).Norm()
			pull := lerp(300, 30, d/magnet)
			pu.pos = pu.pos.Add(dir.Mul(pull * dt))
		}
		if d < playerVisR+pu.radius {
			pu.collect(p)
			pu.dead = true
			return
		}
	}
	pu.pos.Y += pu.vel.Y * dt
	pu.pos.X += sway * dt
	if pu.pos.Y > ScreenH+30 {
		pu.dead = true
	}
}

func (pu *Powerup) collect(p *Play) {
	pl := p.player
	switch pu.kind {
	case puPower:
		if pl.weapon < maxWeapon {
			pl.weapon++
			p.popText(pl.pos, "WEAPON UP", colPowerPU)
		} else {
			p.addScore(500)
			p.popText(pl.pos, "+500", colPowerPU)
		}
		p.sfx(sfxPowerup)
	case puShield:
		pl.shield = minf(playerMaxShield, pl.shield+55)
		p.popText(pl.pos, "SHIELD", colShieldPU)
		p.sfx(sfxPowerup)
	case puBomb:
		if pl.bombs < playerMaxBombs {
			pl.bombs++
			p.popText(pl.pos, "+BOMB", colBombPU)
		} else {
			p.addScore(400)
			p.popText(pl.pos, "+400", colBombPU)
		}
		p.sfx(sfxPowerup)
	case puLife:
		if pl.lives < playerMaxLives {
			pl.lives++
			p.popText(pl.pos, "1UP", colHealth)
		} else {
			p.addScore(2000)
			p.popText(pl.pos, "+2000", colHealth)
		}
		p.sfx(sfxExtend)
	case puScore:
		p.addScore(750)
		p.popText(pl.pos, "+750", colGold)
		p.sfx(sfxCoin)
	}
	p.particles.sparks(pu.pos, V(0, -1), puColor(pu.kind), 14)
	p.particles.ring(pu.pos, puColor(pu.kind), 22, 2)
}

func (pu *Powerup) draw(dst *ebiten.Image, f *Fonts) {
	c := puColor(pu.kind)
	r := pu.radius
	bob := math.Sin(pu.age*3) * 1.5
	pos := V(pu.pos.X, pu.pos.Y+bob)
	pulse := 0.55 + 0.45*math.Sin(pu.age*4)
	// gem
	gem := []Vec2{{0, -r}, {r, 0}, {0, r}, {-r, 0}}
	drawPoly(dst, gem, pos, 0, 1, scaleRGB(c, 0.4), c, r*2.6, c, 0.4+0.3*pulse)
	f.center(dst, puLetter(pu.kind), pos.X, pos.Y, 13, color.RGBA{255, 255, 255, 255}, true)
}
