package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	playerSpeed      = 275.0
	playerFocusSpeed = 135.0
	playerMaxShield  = 100.0
	playerStartLives = 3
	playerMaxLives   = 5
	playerStartBombs = 3
	playerMaxBombs   = 5
	maxWeapon        = 6
	playerHitR       = 4.0  // tiny hitbox (shmup style)
	playerVisR       = 13.0 // visual size
)

// Player is the hero ship.
type Player struct {
	pos       Vec2
	lives     int
	shield    float64
	weapon    int
	bombs     int
	fireCD    float64
	invuln    float64 // i-frames remaining (seconds)
	respawn   float64 // >0: respawning, ship hidden
	dead      bool    // true: out of lives -> game over
	thrust    float64 // 0..1 engine animation
	focus     bool
	pulse     float64
	trailT    float64
	noHitTime float64 // time since last damage (drives combo grace)
}

func newPlayer() *Player {
	return &Player{
		pos:    V(ScreenW/2, ScreenH-90),
		lives:  playerStartLives,
		shield: playerMaxShield,
		weapon: 1,
		bombs:  playerStartBombs,
	}
}

func (pl *Player) alive() bool { return !pl.dead && pl.respawn <= 0 }

func (pl *Player) update(p *Play, in Input) {
	pl.pulse += dt
	if pl.dead {
		return
	}
	if pl.respawn > 0 {
		pl.respawn -= dt
		if pl.respawn <= 0 {
			pl.pos = V(ScreenW/2, ScreenH-90)
			pl.invuln = 2.4
		}
		return
	}

	if pl.invuln > 0 {
		pl.invuln -= dt
	}
	pl.noHitTime += dt

	// Movement.
	pl.focus = in.Focus
	speed := playerSpeed
	if pl.focus {
		speed = playerFocusSpeed
	}
	pl.pos = pl.pos.Add(in.Move.Mul(speed * dt))
	pl.pos.X = clampf(pl.pos.X, playerVisR, ScreenW-playerVisR)
	pl.pos.Y = clampf(pl.pos.Y, playerVisR+4, ScreenH-playerVisR-4)

	// Engine animation + trail.
	target := 0.35 + 0.65*clampf(in.Move.Len(), 0, 1)
	pl.thrust = approach(pl.thrust, target, 4*dt)
	pl.trailT -= dt
	if pl.trailT <= 0 {
		pl.trailT = 0.02
		ex := pl.pos.Add(V(0, playerVisR*0.7))
		col := mixRGB(colThruster, colPlayerHot, 0.2+0.5*pl.thrust)
		p.particles.trail(ex.Add(V((p.rng.Float64()-0.5)*4, 0)), V(0, 120+pl.thrust*120), col, 4+pl.thrust*3, 0.22)
	}

	// Firing.
	if pl.fireCD > 0 {
		pl.fireCD -= dt
	}
	if in.Fire && pl.fireCD <= 0 {
		pl.fire(p)
	}

	// Bomb.
	if in.BombPressed {
		pl.useBomb(p)
	}
}

func (pl *Player) fire(p *Play) {
	nose := pl.pos.Add(V(0, -playerVisR))
	const up = -math.Pi / 2
	spd := 680.0
	dmg := 7.5
	cd := 0.115

	shoot := func(offx, degTilt, dmgMul, spdMul float64) {
		ang := up + degTilt*math.Pi/180
		b := &Bullet{
			pos:      nose.Add(V(offx, 0)),
			vel:      FromAngle(ang, spd*spdMul),
			radius:   3.0,
			damage:   dmg * dmgMul,
			friendly: true,
			life:     2,
			col:      colPBullet,
			kind:     bkLaser,
		}
		p.spawnPBullet(b)
	}

	switch pl.weapon {
	case 1:
		shoot(0, 0, 1.25, 1)
	case 2:
		shoot(-7, 0, 1, 1)
		shoot(7, 0, 1, 1)
	case 3:
		shoot(0, 0, 1.1, 1)
		shoot(-9, -8, 0.85, 1)
		shoot(9, 8, 0.85, 1)
	case 4:
		shoot(-5, 0, 1, 1)
		shoot(5, 0, 1, 1)
		shoot(-12, -12, 0.8, 0.95)
		shoot(12, 12, 0.8, 0.95)
	case 5:
		shoot(0, 0, 1.15, 1)
		shoot(-7, -7, 0.9, 1)
		shoot(7, 7, 0.9, 1)
		shoot(-13, -16, 0.7, 0.95)
		shoot(13, 16, 0.7, 0.95)
	default: // 6+
		shoot(0, 0, 1.25, 1.05)
		shoot(-6, -6, 1, 1)
		shoot(6, 6, 1, 1)
		shoot(-12, -15, 0.85, 0.97)
		shoot(12, 15, 0.85, 0.97)
		shoot(-16, -45, 0.6, 0.9)
		shoot(16, 45, 0.6, 0.9)
		cd = 0.10
	}
	pl.fireCD = cd
	p.sfx(sfxShoot)
}

func (pl *Player) useBomb(p *Play) {
	if pl.bombs <= 0 || pl.respawn > 0 || pl.dead {
		return
	}
	pl.bombs--
	pl.invuln = maxf(pl.invuln, 1.3)
	p.detonateBomb(pl.pos)
}

// hit applies damage. Returns true if it landed (was not blocked by i-frames).
func (pl *Player) hit(p *Play, dmg float64) bool {
	if pl.dead || pl.respawn > 0 || pl.invuln > 0 {
		return false
	}
	pl.noHitTime = 0
	p.breakCombo()
	pl.shield -= dmg
	p.addShake(7)
	p.particles.sparks(pl.pos, V(0, -1), colPlayer, 14)
	if pl.shield > 0 {
		pl.invuln = 0.55
		p.sfx(sfxHurt)
		return true
	}
	// Lost a life.
	pl.lives--
	pl.shield = playerMaxShield
	p.particles.explosion(pl.pos, colPlayer, 2.6)
	p.particles.debris(pl.pos, colPlayer, 10)
	p.addShake(20)
	p.setFlash(withAlpha(colPlayer, 70))
	p.sfx(sfxPlayerDie)
	p.clearEnemyBullets(true) // mercy clear on death
	if pl.lives < 0 {
		pl.lives = 0
		pl.dead = true
		p.onPlayerDead()
		return true
	}
	pl.respawn = 1.1
	return true
}

func (pl *Player) draw(dst *ebiten.Image) {
	if pl.dead {
		return
	}
	if pl.respawn > 0 {
		return
	}
	// Blink while invulnerable.
	if pl.invuln > 0 && int(pl.invuln*16)%2 == 0 {
		return
	}

	p := pl.pos
	// Engine glow.
	eg := 0.5 + 0.5*math.Sin(pl.pulse*30)
	drawGlow(dst, p.X, p.Y+playerVisR*0.7, 10+6*pl.thrust+eg*3, colThruster, 0.7)
	// Body glow.
	drawGlow(dst, p.X, p.Y, playerVisR*2.2, colPlayer, 0.5)

	// Ship body (arrowhead pointing up).
	body := []Vec2{
		{0, -playerVisR},                       // nose
		{playerVisR * 0.82, playerVisR * 0.7},  // right wing
		{playerVisR * 0.32, playerVisR * 0.45}, // right inner
		{0, playerVisR * 0.7},                  // tail notch
		{-playerVisR * 0.32, playerVisR * 0.45},
		{-playerVisR * 0.82, playerVisR * 0.7},
	}
	drawPoly(dst, body, p, 0, 1, colPlayer, colPlayerHot, 0, color.RGBA{}, 0)
	// Cockpit.
	vector.DrawFilledCircle(dst, float32(p.X), float32(p.Y-playerVisR*0.1), 3, colPlayerHot, true)

	// Focus mode: reveal the true hitbox.
	if pl.focus {
		drawGlow(dst, p.X, p.Y, 10, colDanger, 0.8)
		vector.DrawFilledCircle(dst, float32(p.X), float32(p.Y), float32(playerHitR), color.RGBA{255, 255, 255, 255}, true)
	}
}
