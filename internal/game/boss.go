package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type bossAttack int

const (
	atkRest bossAttack = iota
	atkAimed
	atkSpiral
	atkWall
	atkRings
	atkMinions
)

// Boss is a large multi-phase enemy. The same type powers the mid-campaign
// mini-boss and the final boss, scaled by HP and aggression.
type Boss struct {
	name   string
	pos    Vec2
	hp     float64
	maxHP  float64
	radius float64
	mini   bool

	age      float64
	spin     float64
	hitFlash float64

	intro    bool
	targetY  float64
	sweepAmp float64

	phase   int
	seq     []bossAttack
	seqIdx  int
	attack  bossAttack
	patT    float64 // time remaining in current attack
	emitT   float64 // sub-emit accumulator
	emitCnt int
	spiralA float64

	dying float64
	dead  bool
}

func newBoss(mini bool, diff float64) *Boss {
	b := &Boss{
		mini:     mini,
		pos:      V(ScreenW/2, -80),
		intro:    true,
		phase:    1,
		attack:   atkRest,
		patT:     1.0,
		spiralA:  0,
		sweepAmp: 120,
	}
	if mini {
		b.name = "WARDEN"
		b.maxHP = 2400 * diff
		b.radius = 38
		b.targetY = 120
		b.sweepAmp = 110
	} else {
		b.name = "THE ANNIHILATOR"
		b.maxHP = 5000 * diff
		b.radius = 50
		b.targetY = 132
		b.sweepAmp = 130
	}
	b.hp = b.maxHP
	return b
}

func (b *Boss) takeDamage(d float64) {
	if b.intro || b.dying > 0 {
		return
	}
	b.hp -= d
	b.hitFlash = 0.08
}

func (b *Boss) alive() bool { return !b.dead }

func (b *Boss) hpFrac() float64 {
	if b.maxHP <= 0 {
		return 0
	}
	return clampf(b.hp/b.maxHP, 0, 1)
}

func (b *Boss) update(p *Play) {
	b.age += dt
	b.spin += dt
	if b.hitFlash > 0 {
		b.hitFlash -= dt
	}

	if b.dying > 0 {
		b.updateDying(p)
		return
	}
	if b.hp <= 0 {
		b.dying = 2.6
		p.onBossDefeated(b)
		return
	}

	if b.intro {
		b.pos.Y = approach(b.pos.Y, b.targetY, 70*dt)
		b.pos.X = ScreenW / 2
		if b.pos.Y >= b.targetY-0.5 {
			b.intro = false
			b.setPhase(p, 1)
		}
		return
	}

	// Phase transitions by remaining HP.
	switch {
	case b.hpFrac() <= 0.34 && b.phase < 3:
		b.setPhase(p, 3)
	case b.hpFrac() <= 0.67 && b.phase < 2:
		b.setPhase(p, 2)
	}

	// Movement: horizontal sweep + gentle bob, faster in later phases.
	spd := 0.5 + float64(b.phase)*0.18
	b.pos.X = ScreenW/2 + b.sweepAmp*math.Sin(b.age*spd)
	b.pos.Y = b.targetY + 10*math.Sin(b.age*0.9)

	b.runAttacks(p)
}

func (b *Boss) setPhase(p *Play, ph int) {
	prev := b.phase
	b.phase = ph
	switch ph {
	case 1:
		b.seq = []bossAttack{atkAimed, atkRest, atkWall, atkRest}
	case 2:
		b.seq = []bossAttack{atkSpiral, atkRest, atkAimed, atkWall, atkRest}
	default:
		b.seq = []bossAttack{atkRings, atkSpiral, atkWall, atkAimed, atkMinions}
	}
	b.seqIdx = 0
	b.startAttack(b.seq[0])
	if ph > prev && p != nil {
		p.addShake(16)
		p.particles.ring(b.pos, b.col(), 120, 4)
		p.setFlash(withAlpha(b.col(), 50))
		p.sfx(sfxBossPhase)
	}
}

func (b *Boss) aggression() float64 {
	// Lower means faster emission; phases ramp aggression.
	return 1.0 - 0.12*float64(b.phase-1)
}

func (b *Boss) startAttack(a bossAttack) {
	b.attack = a
	b.emitT = 0
	b.emitCnt = 0
	switch a {
	case atkRest:
		b.patT = 0.9 * b.aggression()
	case atkAimed:
		b.patT = 2.4
	case atkSpiral:
		b.patT = 3.0
	case atkWall:
		b.patT = 1.7
	case atkRings:
		b.patT = 2.6
	case atkMinions:
		b.patT = 0.6
	}
}

func (b *Boss) runAttacks(p *Play) {
	b.patT -= dt
	if b.emitT > 0 {
		b.emitT -= dt
	}
	col := colEBullet2
	if b.phase >= 3 {
		col = colEBullet3
	}

	switch b.attack {
	case atkAimed:
		if b.emitT <= 0 {
			b.emitT = 0.46 * b.aggression()
			base := angleTo(b.pos, p.player.pos)
			n := 5 + b.phase
			p.enemyShootSpread(b.pos, base, 205, enemyBulletDmg, 0.5, n, col)
			p.sfx(sfxEnemyShoot)
		}
	case atkSpiral:
		if b.emitT <= 0 {
			b.emitT = 0.055
			b.spiralA += 0.34
			arms := 2 + (b.phase - 1)
			for k := 0; k < arms; k++ {
				ang := b.spiralA + float64(k)*2*math.Pi/float64(arms)
				p.enemyShootDir(b.pos, ang, 158, enemyBulletDmg, col)
			}
		}
	case atkWall:
		if b.emitCnt == 0 {
			b.emitCnt = 1
			gap := 60 + p.rng.Float64()*(ScreenW-120)
			p.enemyShootWall(b.pos.Y, 150, enemyBulletDmg, gap, 78, col)
			p.sfx(sfxEnemyShoot)
		}
	case atkRings:
		if b.emitT <= 0 {
			b.emitT = 0.7 * b.aggression()
			n := 16 + b.phase*2
			p.enemyShootRadial(b.pos, 138, enemyBulletDmg, n, b.age*1.3, col)
			p.sfx(sfxEnemyShoot)
		}
	case atkMinions:
		if b.emitCnt == 0 {
			b.emitCnt = 1
			kinds := []enemyKind{ekGrunt, ekDiver, ekGrunt}
			for i, k := range kinds {
				x := ScreenW*0.25 + float64(i)*ScreenW*0.25
				p.spawnEnemy(newEnemy(k, V(x, -20-float64(i)*30), p.diff))
			}
		}
	}

	if b.patT <= 0 {
		b.seqIdx = (b.seqIdx + 1) % len(b.seq)
		b.startAttack(b.seq[b.seqIdx])
	}
}

func (b *Boss) updateDying(p *Play) {
	b.dying -= dt
	b.pos.X = ScreenW/2 + b.sweepAmp*0.3*math.Sin(b.age*2)
	p.shake = maxf(p.shake, 6)
	// frequent explosions across the body
	if p.rng.Float64() < 0.6 {
		off := V((p.rng.Float64()-0.5)*b.radius*1.8, (p.rng.Float64()-0.5)*b.radius*1.4)
		p.particles.explosion(b.pos.Add(off), b.col(), 1.8)
		p.sfx(sfxExplode)
	}
	if b.dying <= 0 {
		b.dead = true
		p.particles.explosion(b.pos, color.RGBA{255, 255, 255, 255}, 5)
		p.particles.ring(b.pos, colBoss, 360, 6)
		p.addShake(34)
	}
}

func (b *Boss) col() color.RGBA {
	if b.mini {
		return colTurret
	}
	return colBoss
}

func (b *Boss) draw(dst *ebiten.Image) {
	c := b.col()
	r := b.radius
	dyingFx := 1.0
	if b.dying > 0 && int(b.age*20)%2 == 0 {
		dyingFx = 1.6
	}
	fill := scaleRGB(c, 0.4)
	outline := c
	if b.hitFlash > 0 {
		fill = mixRGB(fill, color.RGBA{255, 255, 255, 255}, 0.7)
		outline = color.RGBA{255, 255, 255, 255}
	}

	drawGlow(dst, b.pos.X, b.pos.Y, r*2.6, c, 0.5*dyingFx)
	// outer hull (hexagon)
	hull := regularPoly(6, r, math.Pi/6+b.spin*0.15)
	drawPoly(dst, hull, b.pos, 0, 1, fill, outline, 0, c, 0)
	// wings
	wing := []Vec2{{-r * 1.5, 0}, {-r * 0.7, -r * 0.4}, {-r * 0.7, r * 0.4}}
	drawPoly(dst, wing, b.pos, 0, 1, fill, outline, 0, c, 0)
	wing2 := []Vec2{{r * 1.5, 0}, {r * 0.7, -r * 0.4}, {r * 0.7, r * 0.4}}
	drawPoly(dst, wing2, b.pos, 0, 1, fill, outline, 0, c, 0)
	// rotating inner ring
	ring := regularPoly(3, r*0.62, -b.spin*0.6)
	drawPoly(dst, ring, b.pos, 0, 1, color.RGBA{}, withAlpha(colPlayerHot, 200), 0, c, 0)
	// core (weak point), pulses red when low
	coreCol := mixRGB(color.RGBA{255, 255, 255, 255}, colDanger, 1-b.hpFrac())
	cr := r*0.28 + 2*math.Sin(b.age*6)
	drawGlow(dst, b.pos.X, b.pos.Y, cr*2.2, coreCol, 0.9)
	pcircle(dst, b.pos, cr, coreCol)
}
