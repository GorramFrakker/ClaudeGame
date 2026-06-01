package game

import (
	"image/color"
	"math"
	"math/rand"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	enemyBulletDmg = 22.0
	bodyDmg        = 40.0
	eBulletR       = 5.0
	maxEBullets    = 1400
)

// Result is the outcome of a play session.
type Result int

const (
	resRunning Result = iota
	resWin
	resLose
)

type floatText struct {
	pos     Vec2
	vel     Vec2
	text    string
	col     color.RGBA
	life    float64
	maxLife float64
}

// Play is one gameplay session: the player, all hostiles, projectiles, the
// effects systems, and the campaign director that scripts the action.
type Play struct {
	g         *Game
	rng       *rand.Rand
	particles *Particles

	player   *Player
	enemies  []*Enemy
	pBullets []*Bullet
	eBullets []*Bullet
	powerups []*Powerup
	boss     *Boss

	director *Director

	score        int
	displayScore float64
	combo        int
	comboTimer   float64
	maxCombo     int

	diff float64

	shake     float64
	flash     float64
	flashCol  color.RGBA
	bombFlash float64

	time     float64
	result   Result
	endTimer float64

	floats []floatText

	enemiesKilled int
	starSpeed     float64
}

func newPlay(g *Game) *Play {
	p := &Play{
		g:         g,
		rng:       g.rng,
		particles: newParticles(g.rng),
		player:    newPlayer(),
		diff:      1.0,
		starSpeed: 1,
	}
	p.director = newDirector()
	return p
}

func (p *Play) sfx(id sfxID) { p.g.audio.play(id) }

func (p *Play) addScore(n int) {
	p.score += n
	if p.score > 99999999 {
		p.score = 99999999
	}
}

func (p *Play) multiplier() int { return clampi(1+p.combo/4, 1, 8) }

func (p *Play) addCombo() {
	p.combo++
	p.comboTimer = 3.0
	if p.combo > p.maxCombo {
		p.maxCombo = p.combo
	}
}

func (p *Play) breakCombo() {
	p.combo = 0
	p.comboTimer = 0
}

func (p *Play) addShake(s float64)         { p.shake = maxf(p.shake, s) }
func (p *Play) setFlash(c color.RGBA)      { p.flash = 1; p.flashCol = c }
func (p *Play) popText(pos Vec2, s string, c color.RGBA) {
	p.floats = append(p.floats, floatText{
		pos: pos.Add(V(0, -12)), vel: V(0, -42), text: s, col: c, life: 1, maxLife: 1,
	})
}

// --- projectile spawning ---

func (p *Play) spawnPBullet(b *Bullet) { p.pBullets = append(p.pBullets, b) }

func (p *Play) spawnEBullet(pos, vel Vec2, dmg, radius float64, col color.RGBA) {
	if len(p.eBullets) >= maxEBullets {
		return
	}
	p.eBullets = append(p.eBullets, &Bullet{
		pos: pos, vel: vel, radius: radius, damage: dmg, col: col, kind: bkOrb, life: 9,
	})
}

func (p *Play) enemyShootDir(from Vec2, ang, speed, dmg float64, col color.RGBA) {
	p.spawnEBullet(from, FromAngle(ang, speed), dmg, eBulletR, col)
}

func (p *Play) enemyShootAimed(from Vec2, speed, dmg float64, col color.RGBA) {
	p.enemyShootDir(from, angleTo(from, p.player.pos), speed, dmg, col)
}

func (p *Play) enemyShootSpread(from Vec2, base, speed, dmg, spread float64, count int, col color.RGBA) {
	if count < 1 {
		count = 1
	}
	start := base - spread/2
	step := 0.0
	if count > 1 {
		step = spread / float64(count-1)
	}
	for i := 0; i < count; i++ {
		p.enemyShootDir(from, start+step*float64(i), speed, dmg, col)
	}
}

func (p *Play) enemyShootRadial(from Vec2, speed, dmg float64, count int, phase float64, col color.RGBA) {
	if count < 1 {
		return
	}
	for i := 0; i < count; i++ {
		ang := phase + float64(i)/float64(count)*2*math.Pi
		p.enemyShootDir(from, ang, speed, dmg, col)
	}
}

func (p *Play) enemyShootWall(y, speed, dmg, gapX, gapW float64, col color.RGBA) {
	const spacing = 27.0
	for x := spacing / 2; x < ScreenW; x += spacing {
		if absf(x-gapX) < gapW/2 {
			continue
		}
		p.spawnEBullet(V(x, y), V(0, speed), dmg, eBulletR, col)
	}
}

func (p *Play) spawnEnemy(e *Enemy)     { p.enemies = append(p.enemies, e) }
func (p *Play) spawnPowerup(pos Vec2, k powerupKind) {
	p.powerups = append(p.powerups, newPowerup(pos, k))
}

// dropFor possibly drops a power-up at pos, weighted toward what the player
// currently needs.
func (p *Play) dropFor(pos Vec2, chance float64) {
	if p.rng.Float64() > chance {
		return
	}
	pl := p.player
	type wk struct {
		k powerupKind
		w float64
	}
	ws := []wk{{puBomb, 12}, {puScore, 22}, {puLife, 3}}
	if pl.weapon < maxWeapon {
		ws = append(ws, wk{puPower, 36})
	} else {
		ws = append(ws, wk{puPower, 6})
	}
	if pl.shield < playerMaxShield*0.6 {
		ws = append(ws, wk{puShield, 30})
	} else {
		ws = append(ws, wk{puShield, 10})
	}
	total := 0.0
	for _, w := range ws {
		total += w.w
	}
	r := p.rng.Float64() * total
	for _, w := range ws {
		r -= w.w
		if r <= 0 {
			p.spawnPowerup(pos, w.k)
			return
		}
	}
}

// --- event hooks ---

func (p *Play) killEnemy(e *Enemy) {
	if e.dead {
		return
	}
	e.dead = true
	e.onDeath(p)
	power := clampf(e.radius/12, 0.8, 2.4)
	p.particles.explosion(e.pos, e.col, power)
	if e.radius > 16 {
		p.particles.debris(e.pos, e.col, 8)
	}
	p.addScore(e.value * p.multiplier())
	p.addCombo()
	p.enemiesKilled++
	p.dropFor(e.pos, e.drop)
	p.sfx(sfxExplode)
}

func (p *Play) detonateBomb(pos Vec2) {
	p.bombFlash = 0.5
	p.setFlash(color.RGBA{200, 230, 255, 90})
	p.addShake(26)
	p.sfx(sfxBomb)
	p.particles.ring(pos, colPlayer, 380, 6)
	p.particles.explosion(pos, colPlayerHot, 3)
	p.clearEnemyBullets(true)
	for _, e := range p.enemies {
		e.takeDamage(150)
		if e.hp <= 0 {
			p.killEnemy(e)
		}
	}
	if p.boss != nil {
		p.boss.takeDamage(650)
	}
}

func (p *Play) clearEnemyBullets(score bool) {
	pts := 0
	for _, b := range p.eBullets {
		if b.dead {
			continue
		}
		b.dead = true
		p.particles.trail(b.pos, V(0, 0), colGold, 4, 0.25)
		pts += 10
	}
	if score && pts > 0 {
		if pts > 2000 {
			pts = 2000
		}
		p.addScore(pts)
	}
}

func (p *Play) onPlayerDead() {
	if p.result == resRunning {
		p.result = resLose
		p.endTimer = 2.2
		p.addShake(30)
	}
}

func (p *Play) onBossDefeated(b *Boss) {
	bonus := 6000
	if !b.mini {
		bonus = 30000
	}
	p.addScore(bonus)
	p.popText(b.pos, "+"+strconv.Itoa(bonus), colGold)
	p.addShake(22)
	p.setFlash(withAlpha(b.col(), 70))
	p.sfx(sfxExtend)
}

func (p *Play) win() {
	if p.result == resRunning {
		p.result = resWin
		p.endTimer = 3.2
	}
}

// --- main update ---

func (p *Play) Update(in Input) {
	p.time += dt

	p.director.update(p)

	p.player.update(p, in)
	for _, e := range p.enemies {
		e.update(p)
	}
	if p.boss != nil {
		p.boss.update(p)
	}
	for _, b := range p.pBullets {
		b.update(p)
	}
	for _, b := range p.eBullets {
		b.update(p)
	}
	for _, pu := range p.powerups {
		pu.update(p)
	}
	p.particles.Update()

	p.collide()

	if p.boss != nil && p.boss.dead {
		p.boss = nil
	}

	p.enemies = filterEnemies(p.enemies)
	p.pBullets = filterBullets(p.pBullets)
	p.eBullets = filterBullets(p.eBullets)
	p.powerups = filterPowerups(p.powerups)

	if p.combo > 0 {
		p.comboTimer -= dt
		if p.comboTimer <= 0 {
			p.breakCombo()
		}
	}

	// animated score readout
	d := float64(p.score) - p.displayScore
	p.displayScore += maxf(absf(d)*0.18, 1) * sign(d)
	if absf(float64(p.score)-p.displayScore) < 1 {
		p.displayScore = float64(p.score)
	}

	p.shake = approach(p.shake, 0, 55*dt)
	if p.flash > 0 {
		p.flash -= dt * 2.6
	}
	if p.bombFlash > 0 {
		p.bombFlash -= dt * 2
	}

	// floating texts
	fd := p.floats[:0]
	for _, ft := range p.floats {
		ft.life -= dt
		ft.pos = ft.pos.Add(ft.vel.Mul(dt))
		ft.vel = ft.vel.Mul(0.92)
		if ft.life > 0 {
			fd = append(fd, ft)
		}
	}
	p.floats = fd

	if p.result != resRunning {
		p.endTimer -= dt
	}

	// background intensity rises with on-screen threat
	threat := 1.0
	if p.boss != nil {
		threat = 1.7
	}
	p.starSpeed = approach(p.starSpeed, threat, dt)
}

func sign(x float64) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// collide resolves all collisions for the frame.
func (p *Play) collide() {
	pl := p.player

	// Player bullets vs enemies / boss.
	for _, b := range p.pBullets {
		if b.dead {
			continue
		}
		hit := false
		for _, e := range p.enemies {
			if e.dead {
				continue
			}
			if circlesOverlap(b.pos, b.radius, e.pos, e.radius) {
				e.takeDamage(b.damage)
				p.particles.sparks(b.pos, b.vel.Mul(-1), e.col, 4)
				if e.hp <= 0 {
					p.killEnemy(e)
				}
				hit = true
				break
			}
		}
		if hit {
			b.dead = true
			continue
		}
		if p.boss != nil && !p.boss.intro && p.boss.dying <= 0 &&
			circlesOverlap(b.pos, b.radius, p.boss.pos, p.boss.radius) {
			p.boss.takeDamage(b.damage)
			p.particles.sparks(b.pos, b.vel.Mul(-1), p.boss.col(), 4)
			b.dead = true
		}
	}

	if !pl.alive() {
		return
	}

	// Enemy bullets vs player.
	for _, b := range p.eBullets {
		if b.dead {
			continue
		}
		if circlesOverlap(b.pos, b.radius, pl.pos, playerHitR) {
			if pl.hit(p, b.damage) {
				b.dead = true
			}
			if !pl.alive() {
				return
			}
		}
	}

	// Enemy bodies vs player (ramming).
	for _, e := range p.enemies {
		if e.dead {
			continue
		}
		if circlesOverlap(e.pos, e.radius*0.8, pl.pos, playerHitR) {
			pl.hit(p, bodyDmg)
			e.takeDamage(40)
			if e.hp <= 0 {
				p.killEnemy(e)
			}
			if !pl.alive() {
				return
			}
		}
	}

	// Boss body vs player.
	if p.boss != nil && !p.boss.intro && p.boss.dying <= 0 &&
		circlesOverlap(p.boss.pos, p.boss.radius*0.7, pl.pos, playerHitR) {
		pl.hit(p, bodyDmg)
	}
}

// --- drawing ---

// DrawWorld renders the playfield (everything affected by screen shake).
func (p *Play) DrawWorld(dst *ebiten.Image) {
	for _, pu := range p.powerups {
		pu.draw(dst, p.g.fonts)
	}
	for _, e := range p.enemies {
		e.draw(dst)
	}
	if p.boss != nil {
		p.boss.draw(dst)
	}
	p.player.draw(dst)
	for _, b := range p.eBullets {
		b.draw(dst)
	}
	for _, b := range p.pBullets {
		b.draw(dst)
	}
	p.particles.Draw(dst)
	p.drawFloats(dst)
}

func (p *Play) drawFloats(dst *ebiten.Image) {
	for i := range p.floats {
		ft := &p.floats[i]
		a := clampf(ft.life/ft.maxLife, 0, 1)
		c := withAlpha(ft.col, uint8(a*255))
		p.g.fonts.center(dst, ft.text, ft.pos.X, ft.pos.Y, 14, c, true)
	}
}
