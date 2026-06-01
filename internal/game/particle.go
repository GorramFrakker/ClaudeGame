package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type particle struct {
	pos, vel Vec2
	life     float64
	maxLife  float64
	size     float64
	drag     float64
	grav     float64
	col      color.RGBA
	glow     bool
}

type shockwave struct {
	pos     Vec2
	r       float64
	maxR    float64
	life    float64
	maxLife float64
	width   float64
	col     color.RGBA
}

// Particles is a lightweight pooled particle + shockwave system. Everything is
// drawn additively for a glowing, energetic feel.
type Particles struct {
	list  []particle
	waves []shockwave
	rng   *rand.Rand
}

func newParticles(rng *rand.Rand) *Particles {
	return &Particles{rng: rng}
}

const maxParticles = 1800

func (ps *Particles) add(p particle) {
	if len(ps.list) >= maxParticles {
		// drop the oldest to bound work
		copy(ps.list, ps.list[1:])
		ps.list[len(ps.list)-1] = p
		return
	}
	ps.list = append(ps.list, p)
}

func (ps *Particles) Update() {
	dst := ps.list[:0]
	for _, p := range ps.list {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vel.Y += p.grav * dt
		damp := 1 - p.drag*dt
		if damp < 0 {
			damp = 0
		}
		p.vel = p.vel.Mul(damp)
		p.pos = p.pos.Add(p.vel.Mul(dt))
		dst = append(dst, p)
	}
	ps.list = dst

	wdst := ps.waves[:0]
	for _, w := range ps.waves {
		w.life -= dt
		if w.life <= 0 {
			continue
		}
		t := 1 - w.life/w.maxLife
		w.r = w.maxR * easeOut(t)
		wdst = append(wdst, w)
	}
	ps.waves = wdst
}

func (ps *Particles) Draw(dst *ebiten.Image) {
	for i := range ps.list {
		p := &ps.list[i]
		a := clampf(p.life/p.maxLife, 0, 1)
		if p.glow {
			drawGlow(dst, p.pos.X, p.pos.Y, p.size*(0.6+a), p.col, a)
		} else {
			c := withAlpha(p.col, uint8(a*255))
			s := float32(maxf(p.size*a, 0.6))
			vector.DrawFilledRect(dst, float32(p.pos.X)-s/2, float32(p.pos.Y)-s/2, s, s, c, false)
		}
	}
	for i := range ps.waves {
		w := &ps.waves[i]
		a := clampf(w.life/w.maxLife, 0, 1)
		col := withAlpha(w.col, uint8(a*220))
		if w.r > 0.5 {
			vector.StrokeCircle(dst, float32(w.pos.X), float32(w.pos.Y), float32(w.r), float32(w.width*(0.4+a)), col, true)
		}
		drawGlow(dst, w.pos.X, w.pos.Y, w.r*0.7, w.col, a*0.25)
	}
}

func easeOut(t float64) float64 { return 1 - (1-t)*(1-t) }

// --- spawners ---

// explosion produces a burst of glowing shards, a bright flash, and a ring.
// power scales count, speed, and size (1 = small enemy, 3+ = boss).
func (ps *Particles) explosion(pos Vec2, base color.RGBA, power float64) {
	n := int(14 * power)
	if n > 90 {
		n = 90
	}
	for i := 0; i < n; i++ {
		ang := ps.rng.Float64() * 2 * math.Pi
		spd := (40 + ps.rng.Float64()*180) * (0.6 + power*0.4)
		life := 0.35 + ps.rng.Float64()*0.5*power
		c := base
		if ps.rng.Float64() < 0.4 {
			c = mixRGB(base, color.RGBA{255, 255, 255, 255}, 0.6)
		}
		ps.add(particle{
			pos:     pos,
			vel:     FromAngle(ang, spd),
			life:    life,
			maxLife: life,
			size:    (2 + ps.rng.Float64()*3) * power,
			drag:    2.0 + ps.rng.Float64()*2,
			col:     c,
			glow:    true,
		})
	}
	// central flash
	ps.add(particle{pos: pos, life: 0.16, maxLife: 0.16, size: 16 * power, col: color.RGBA{255, 255, 255, 255}, glow: true, drag: 1})
	ps.ring(pos, base, 26*power, 2.2*power)
}

// sparks emits a small directional shower (used for bullet impacts).
func (ps *Particles) sparks(pos, dir Vec2, base color.RGBA, count int) {
	baseAng := dir.Angle()
	for i := 0; i < count; i++ {
		ang := baseAng + (ps.rng.Float64()-0.5)*1.7
		spd := 60 + ps.rng.Float64()*160
		life := 0.18 + ps.rng.Float64()*0.22
		ps.add(particle{
			pos:     pos,
			vel:     FromAngle(ang, spd),
			life:    life,
			maxLife: life,
			size:    1.4 + ps.rng.Float64()*1.6,
			drag:    3,
			col:     base,
			glow:    true,
		})
	}
}

// trail adds a single soft fading dot, used for engine/bomb trails.
func (ps *Particles) trail(pos, vel Vec2, base color.RGBA, size, life float64) {
	ps.add(particle{
		pos:     pos,
		vel:     vel,
		life:    life,
		maxLife: life,
		size:    size,
		drag:    2,
		col:     base,
		glow:    true,
	})
}

// ring adds an expanding shockwave.
func (ps *Particles) ring(pos Vec2, base color.RGBA, maxR, width float64) {
	life := 0.4 + maxR/600
	ps.waves = append(ps.waves, shockwave{
		pos:     pos,
		maxR:    maxR,
		life:    life,
		maxLife: life,
		width:   width,
		col:     base,
	})
}

// debris scatters a few solid (non-glow) chunks for weight on big kills.
func (ps *Particles) debris(pos Vec2, base color.RGBA, count int) {
	for i := 0; i < count; i++ {
		ang := ps.rng.Float64() * 2 * math.Pi
		spd := 50 + ps.rng.Float64()*140
		life := 0.6 + ps.rng.Float64()*0.6
		ps.add(particle{
			pos:     pos,
			vel:     FromAngle(ang, spd),
			life:    life,
			maxLife: life,
			size:    2 + ps.rng.Float64()*2.5,
			drag:    1.2,
			grav:    120,
			col:     scaleRGB(base, 0.8),
			glow:    false,
		})
	}
}

func (ps *Particles) clear() {
	ps.list = ps.list[:0]
	ps.waves = ps.waves[:0]
}

func (ps *Particles) count() int { return len(ps.list) }
