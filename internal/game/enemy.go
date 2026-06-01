package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type enemyKind int

const (
	ekGrunt enemyKind = iota
	ekWeaver
	ekDiver
	ekTurret
	ekBomber
	ekMine
)

// Enemy is a non-boss hostile. Behavior is driven by its kind in update().
type Enemy struct {
	kind   enemyKind
	pos    Vec2
	vel    Vec2
	hp     float64
	maxHP  float64
	radius float64
	col    color.RGBA
	value  int
	drop   float64 // 0..1 chance to drop a power-up

	age      float64
	fireT    float64
	fireInt  float64
	seed     float64
	spin     float64
	hitFlash float64

	// movement state
	baseX     float64
	amp, freq float64
	targetY   float64
	entered   bool
	leaving   bool
	diveState int
	diveT     float64
	triggered bool // mine has been set off

	dead bool
}

// filterEnemies removes dead enemies in place.
func filterEnemies(in []*Enemy) []*Enemy {
	out := in[:0]
	for _, e := range in {
		if !e.dead {
			out = append(out, e)
		}
	}
	return out
}

func newEnemy(kind enemyKind, pos Vec2, diff float64) *Enemy {
	e := &Enemy{kind: kind, pos: pos, seed: 0}
	hpMul := diff
	switch kind {
	case ekGrunt:
		e.hp, e.radius, e.value, e.drop = 18, 12, 100, 0.10
		e.col = colGrunt
		e.vel = V(0, 95)
		e.fireInt = 1.7
	case ekWeaver:
		e.hp, e.radius, e.value, e.drop = 24, 13, 150, 0.12
		e.col = colWeaver
		e.baseX = pos.X
		e.amp = 70
		e.freq = 2.2
		e.vel = V(0, 72)
		e.fireInt = 1.5
	case ekDiver:
		e.hp, e.radius, e.value, e.drop = 22, 12, 170, 0.12
		e.col = colDiver
		e.vel = V(0, 70)
		e.targetY = ScreenH*0.26 + 40
		e.fireInt = 99
	case ekTurret:
		e.hp, e.radius, e.value, e.drop = 130, 21, 420, 0.7
		e.col = colTurret
		e.vel = V(0, 58)
		e.targetY = 70 + 70
		e.fireInt = 1.7
	case ekBomber:
		e.hp, e.radius, e.value, e.drop = 95, 18, 360, 0.6
		e.col = colBomber
		e.vel = V(0, 52)
		e.targetY = 150
		e.fireInt = 2.3
	case ekMine:
		e.hp, e.radius, e.value, e.drop = 10, 11, 120, 0.08
		e.col = colMine
		e.vel = V(0, 48)
		e.fireInt = 99
	}
	e.hp *= hpMul
	e.maxHP = e.hp
	e.fireInt /= (0.7 + 0.3*diff)
	e.fireT = 0.4 + 0.9*float64(int(pos.X+pos.Y)%7)/7 // desync initial fire
	return e
}

func (e *Enemy) takeDamage(d float64) {
	e.hp -= d
	e.hitFlash = 0.12
}

func (e *Enemy) update(p *Play) {
	e.age += dt
	e.spin += dt
	if e.hitFlash > 0 {
		e.hitFlash -= dt
	}
	if e.fireT > 0 {
		e.fireT -= dt
	}

	switch e.kind {
	case ekGrunt:
		e.updateGrunt(p)
	case ekWeaver:
		e.updateWeaver(p)
	case ekDiver:
		e.updateDiver(p)
	case ekTurret:
		e.updateTurret(p)
	case ekBomber:
		e.updateBomber(p)
	case ekMine:
		e.updateMine(p)
	}

	e.pos = e.pos.Add(e.vel.Mul(dt))
	e.cull()
}

func (e *Enemy) onScreen() bool {
	return e.pos.Y > 6 && e.pos.Y < ScreenH-40
}

func (e *Enemy) cull() {
	const m = 50
	if e.pos.Y > ScreenH+m || e.pos.X < -m || e.pos.X > ScreenW+m {
		e.dead = true
	}
}

func (e *Enemy) updateGrunt(p *Play) {
	// gentle horizontal drift, bounce off the sides
	if e.vel.X == 0 {
		if e.seed == 0 {
			e.seed = 1
			e.vel.X = float64(int(e.pos.X)%2)*40 - 20
		}
	}
	if e.pos.X < e.radius+8 || e.pos.X > ScreenW-e.radius-8 {
		e.vel.X = -e.vel.X
	}
	if e.onScreen() && e.fireT <= 0 {
		e.fireT = e.fireInt
		p.enemyShootAimed(e.pos, 165, enemyBulletDmg, e.col)
	}
}

func (e *Enemy) updateWeaver(p *Play) {
	e.pos.X = e.baseX + e.amp*math.Sin(e.age*e.freq+e.seed)
	if e.onScreen() && e.fireT <= 0 {
		e.fireT = e.fireInt
		p.enemyShootAimed(e.pos, 185, enemyBulletDmg, e.col)
	}
}

func (e *Enemy) updateDiver(p *Play) {
	switch e.diveState {
	case 0:
		if e.pos.Y >= e.targetY {
			e.diveState = 1
			e.diveT = 0.55
			e.vel = V(0, 6)
		}
	case 1:
		e.diveT -= dt
		e.hitFlash = 0.1 // telegraph shimmer
		if e.diveT <= 0 {
			dir := V(0, 1)
			if p.player.alive() {
				dir = p.player.pos.Sub(e.pos).Norm()
			}
			e.vel = dir.Mul(360)
			e.diveState = 2
			p.enemyShootSpread(e.pos, e.vel.Angle(), 200, enemyBulletDmg, 0.5, 3, e.col)
			p.sfx(sfxDive)
		}
	}
}

func (e *Enemy) updateTurret(p *Play) {
	if !e.entered {
		if e.pos.Y >= e.targetY {
			e.entered = true
			e.vel = V(float64(int(e.pos.Y)%2)*70-35, 0)
		}
		return
	}
	if e.pos.X < e.radius+12 || e.pos.X > ScreenW-e.radius-12 {
		e.vel.X = -e.vel.X
	}
	if e.age > 13 && !e.leaving {
		e.leaving = true
		e.vel = V(0, 90)
	}
	if !e.leaving && e.fireT <= 0 {
		e.fireT = e.fireInt
		base := angleTo(e.pos, p.player.pos)
		p.enemyShootSpread(e.pos, base, 175, enemyBulletDmg, 0.42, 4, e.col)
		p.sfx(sfxEnemyShoot)
	}
}

func (e *Enemy) updateBomber(p *Play) {
	if !e.entered {
		if e.pos.Y >= e.targetY {
			e.entered = true
			e.vel = V(0, 0)
		}
		return
	}
	// slow sideways sway
	e.vel.X = 36 * math.Sin(e.age*0.8)
	if e.age > 12 && !e.leaving {
		e.leaving = true
		e.vel = V(0, 80)
	}
	if !e.leaving && e.fireT <= 0 {
		e.fireT = e.fireInt
		p.enemyShootRadial(e.pos, 140, enemyBulletDmg, 12, e.age, e.col)
		p.sfx(sfxEnemyShoot)
	}
}

func (e *Enemy) updateMine(p *Play) {
	// drift down with slight homing toward the player's x
	if p.player.alive() {
		dx := p.player.pos.X - e.pos.X
		e.vel.X = approach(e.vel.X, clampf(dx, -40, 40), 60*dt)
		if dist(e.pos, p.player.pos) < e.radius+playerVisR+14 {
			e.hp = 0
			e.triggered = true
		}
	}
	e.vel.Y = 48 + 14*math.Sin(e.age*3+e.seed)
}

// onDeath spawns kind-specific death effects (e.g., mines bursting).
func (e *Enemy) onDeath(p *Play) {
	switch e.kind {
	case ekMine:
		p.enemyShootRadial(e.pos, 175, enemyBulletDmg, 9, e.age, e.col)
	case ekBomber:
		p.enemyShootRadial(e.pos, 120, enemyBulletDmg, 6, e.age+1, e.col)
	}
}

func (e *Enemy) draw(dst *ebiten.Image) {
	r := e.radius
	fill := scaleRGB(e.col, 0.45)
	outline := e.col
	if e.hitFlash > 0 {
		fill = mixRGB(fill, color.RGBA{255, 255, 255, 255}, 0.8)
		outline = color.RGBA{255, 255, 255, 255}
	}
	glowI := 0.45
	// low-HP flicker for tanky foes
	if e.maxHP > 60 && e.hp/e.maxHP < 0.35 && int(e.age*12)%2 == 0 {
		glowI = 0.8
	}

	switch e.kind {
	case ekGrunt:
		shape := []Vec2{{0, r}, {-0.85 * r, -0.5 * r}, {-0.3 * r, -0.18 * r}, {0, -0.6 * r}, {0.3 * r, -0.18 * r}, {0.85 * r, -0.5 * r}}
		drawPoly(dst, shape, e.pos, 0, 1, fill, outline, r*2.2, e.col, glowI)
	case ekWeaver:
		shape := []Vec2{{0, r}, {0.72 * r, 0}, {0, -r}, {-0.72 * r, 0}}
		drawPoly(dst, shape, e.pos, e.age*1.5, 1, fill, outline, r*2.2, e.col, glowI)
	case ekDiver:
		rot := 0.0
		if e.diveState == 2 {
			rot = e.vel.Angle() - math.Pi/2
		}
		shape := []Vec2{{0, 1.25 * r}, {0.55 * r, -0.4 * r}, {0, -0.15 * r}, {-0.55 * r, -0.4 * r}}
		gi := glowI
		if e.diveState == 1 {
			gi = 0.5 + 0.5*math.Sin(e.age*40)
		}
		drawPoly(dst, shape, e.pos, rot, 1, fill, outline, r*2.4, e.col, gi)
	case ekTurret:
		shape := regularPoly(6, r, e.spin*0.4)
		drawPoly(dst, shape, e.pos, 0, 1, fill, outline, r*2.0, e.col, glowI)
		inner := regularPoly(6, r*0.5, -e.spin*0.8)
		drawPoly(dst, inner, e.pos, 0, 1, color.RGBA{}, outline, 0, e.col, 0)
	case ekBomber:
		shape := regularPoly(8, r, e.spin*0.2)
		drawPoly(dst, shape, e.pos, 0, 1, fill, outline, r*2.1, e.col, glowI)
		drawGlow(dst, e.pos.X, e.pos.Y, r*0.7, color.RGBA{255, 255, 255, 255}, 0.4+0.3*math.Sin(e.age*5))
	case ekMine:
		shape := starPoly(7, r, r*0.5, e.spin*1.6)
		gi := 0.4 + 0.4*math.Sin(e.age*6+e.seed)
		drawPoly(dst, shape, e.pos, 0, 1, fill, outline, r*2.2, e.col, gi)
	}
}
