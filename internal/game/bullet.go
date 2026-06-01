package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type bulletKind int

const (
	bkPellet bulletKind = iota // small round shot
	bkLaser                    // elongated player shot
	bkOrb                      // glowing enemy orb
)

// Bullet is a projectile fired by the player or an enemy. Player and enemy
// bullets live in separate slices, but share this type.
type Bullet struct {
	pos, vel Vec2
	radius   float64
	damage   float64
	friendly bool
	life     float64
	col      color.RGBA
	kind     bulletKind
	spin     float64
	dead     bool
}

func (b *Bullet) update(p *Play) {
	b.pos = b.pos.Add(b.vel.Mul(dt))
	b.life -= dt
	b.spin += dt * 6
	// Cull when off the playfield (with margin) or expired.
	const m = 24
	if b.pos.X < -m || b.pos.X > ScreenW+m || b.pos.Y < -m || b.pos.Y > ScreenH+m || b.life <= 0 {
		b.dead = true
	}
}

func (b *Bullet) draw(dst *ebiten.Image) {
	switch b.kind {
	case bkLaser:
		// streak along travel direction
		dir := b.vel.Norm()
		tail := b.pos.Sub(dir.Mul(b.radius * 4))
		drawGlow(dst, b.pos.X, b.pos.Y, b.radius*3.4, b.col, 0.85)
		vector.StrokeLine(dst, float32(tail.X), float32(tail.Y), float32(b.pos.X), float32(b.pos.Y), float32(b.radius*1.7), b.col, true)
		vector.StrokeLine(dst, float32(tail.X), float32(tail.Y), float32(b.pos.X), float32(b.pos.Y), float32(b.radius*0.7), colPlayerHot, true)
	case bkOrb:
		drawGlow(dst, b.pos.X, b.pos.Y, b.radius*3.2, b.col, 0.8)
		vector.DrawFilledCircle(dst, float32(b.pos.X), float32(b.pos.Y), float32(b.radius), b.col, true)
		vector.DrawFilledCircle(dst, float32(b.pos.X), float32(b.pos.Y), float32(b.radius*0.5), color.RGBA{255, 255, 255, 230}, true)
	default: // pellet
		drawGlow(dst, b.pos.X, b.pos.Y, b.radius*2.6, b.col, 0.7)
		vector.DrawFilledCircle(dst, float32(b.pos.X), float32(b.pos.Y), float32(b.radius), b.col, true)
		vector.DrawFilledCircle(dst, float32(b.pos.X), float32(b.pos.Y), float32(b.radius*0.45), color.RGBA{255, 255, 255, 220}, true)
	}
}

// filterBullets removes dead bullets in place, preserving capacity.
func filterBullets(in []*Bullet) []*Bullet {
	out := in[:0]
	for _, b := range in {
		if !b.dead {
			out = append(out, b)
		}
	}
	return out
}
