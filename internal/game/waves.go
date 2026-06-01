package game

import (
	"image/color"
	"strconv"
)

type stageKind int

const (
	skWave stageKind = iota
	skMini
	skBoss
)

type spawnEvent struct {
	at float64
	do func(p *Play)
}

type stage struct {
	kind    stageKind
	title   string
	sub     string
	col     color.RGBA
	label   string // HUD label, e.g. "WAVE 3/10"
	events  []spawnEvent
	minTime float64
	diff    float64
}

type dirState int

const (
	dsBanner dirState = iota
	dsRun
	dsBossWait
	dsDone
)

// Director scripts the campaign: it shows banners, spawns waves and bosses,
// and decides when to advance based on the battlefield state.
type Director struct {
	stages      []stage
	idx         int
	state       dirState
	stageT      float64
	bannerT     float64
	bannerMax   float64
	nextEvent   int
	bossSpawned bool
}

func newDirector() *Director {
	d := &Director{stages: buildCampaign()}
	d.startBanner()
	return d
}

func (d *Director) cur() *stage {
	if d.idx < 0 || d.idx >= len(d.stages) {
		return nil
	}
	return &d.stages[d.idx]
}

func (d *Director) startBanner() {
	s := d.cur()
	if s == nil {
		d.state = dsDone
		return
	}
	d.state = dsBanner
	d.bannerMax = 2.3
	if s.kind != skWave {
		d.bannerMax = 2.9
	}
	d.bannerT = d.bannerMax
	d.stageT = 0
	d.nextEvent = 0
	d.bossSpawned = false
}

func (d *Director) update(p *Play) {
	s := d.cur()
	if s == nil {
		d.state = dsDone
		return
	}
	p.diff = s.diff

	switch d.state {
	case dsBanner:
		d.bannerT -= dt
		// trigger warning sound once for boss banners
		if s.kind != skWave && d.bannerT <= d.bannerMax-0.05 && d.bannerT > d.bannerMax-0.1 {
			p.sfx(sfxBossWarn)
		}
		if d.bannerT <= 0 {
			if s.kind == skWave {
				d.state = dsRun
			} else {
				p.boss = newBoss(s.kind == skMini, s.diff)
				d.bossSpawned = true
				d.state = dsBossWait
			}
		}
	case dsRun:
		d.stageT += dt
		for d.nextEvent < len(s.events) && s.events[d.nextEvent].at <= d.stageT {
			s.events[d.nextEvent].do(p)
			d.nextEvent++
		}
		done := d.nextEvent >= len(s.events)
		if done && len(p.enemies) == 0 && d.stageT >= s.minTime {
			d.advance(p)
		}
	case dsBossWait:
		if d.bossSpawned && p.boss == nil {
			// boss fully cleared
			if s.kind == skBoss {
				p.win()
				d.state = dsDone
			} else {
				d.advance(p)
			}
		}
	case dsDone:
	}
}

func (d *Director) advance(p *Play) {
	d.idx++
	if d.idx >= len(d.stages) {
		d.state = dsDone
		return
	}
	d.startBanner()
}

// bannerAlpha returns 0..1 for fade in/out of the current banner.
func (d *Director) bannerAlpha() float64 {
	if d.state != dsBanner || d.bannerMax <= 0 {
		return 0
	}
	t := d.bannerMax - d.bannerT // elapsed
	const fin, fout = 0.35, 0.6
	if t < fin {
		return t / fin
	}
	if d.bannerT < fout {
		return d.bannerT / fout
	}
	return 1
}

func (d *Director) hudLabel() string {
	s := d.cur()
	if s == nil {
		return ""
	}
	return s.label
}

// --- formation helpers ---

func spawnRow(p *Play, kind enemyKind, n int, y, margin float64) {
	if n <= 0 {
		return
	}
	for i := 0; i < n; i++ {
		x := margin + (float64(i)+0.5)/float64(n)*(ScreenW-2*margin)
		p.spawnEnemy(newEnemy(kind, V(x, y), p.diff))
	}
}

func spawnVee(p *Play, kind enemyKind, n int, cx, y, dx, dy float64) {
	for i := 0; i < n; i++ {
		side := 1.0
		idx := i / 2
		if i%2 == 1 {
			side = -1
		}
		x := cx + side*float64(idx+1)*dx
		yy := y - float64(idx)*dy
		p.spawnEnemy(newEnemy(kind, V(x, yy), p.diff))
	}
}

func spawnAt(p *Play, kind enemyKind, x, y float64) {
	p.spawnEnemy(newEnemy(kind, V(x, y), p.diff))
}

// ev is a small constructor for a timed spawn event.
func ev(at float64, do func(p *Play)) spawnEvent { return spawnEvent{at: at, do: do} }

// buildCampaign returns the full ordered list of stages. Difficulty climbs
// from 1.0 to roughly 1.9 across ~10 waves, with a mini-boss in the middle and
// the final boss at the end.
func buildCampaign() []stage {
	waveTotal := 10
	num := 0
	wave := func(title, sub string, diff float64, minTime float64, events []spawnEvent) stage {
		num++
		return stage{
			kind:    skWave,
			title:   "WAVE " + strconv.Itoa(num),
			sub:     sub,
			col:     colUIText,
			label:   "WAVE " + strconv.Itoa(num) + "/" + strconv.Itoa(waveTotal),
			events:  events,
			minTime: minTime,
			diff:    diff,
		}
	}

	stages := []stage{
		// 1: gentle intro — straight grunts
		wave("", "INCOMING", 1.00, 14, []spawnEvent{
			ev(0.0, func(p *Play) { spawnRow(p, ekGrunt, 5, -20, 30) }),
			ev(2.5, func(p *Play) { spawnRow(p, ekGrunt, 5, -20, 50) }),
			ev(5.5, func(p *Play) { spawnVee(p, ekGrunt, 5, ScreenW/2, -20, 40, 26) }),
			ev(9.0, func(p *Play) { spawnRow(p, ekGrunt, 6, -20, 24) }),
		}),
		// 2: grunts + first weavers
		wave("", "WEAVERS DETECTED", 1.08, 16, []spawnEvent{
			ev(0.0, func(p *Play) { spawnRow(p, ekWeaver, 3, -20, 60) }),
			ev(3.0, func(p *Play) { spawnRow(p, ekGrunt, 6, -20, 24) }),
			ev(6.0, func(p *Play) { spawnRow(p, ekWeaver, 4, -20, 50) }),
			ev(9.5, func(p *Play) { spawnVee(p, ekGrunt, 7, ScreenW/2, -20, 36, 24) }),
			ev(12.5, func(p *Play) { spawnRow(p, ekWeaver, 3, -20, 60) }),
		}),
		// 3: divers appear
		wave("", "DIVE BOMBERS", 1.16, 16, []spawnEvent{
			ev(0.0, func(p *Play) { spawnRow(p, ekWeaver, 4, -20, 50) }),
			ev(2.5, func(p *Play) { spawnAt(p, ekDiver, ScreenW*0.3, -20); spawnAt(p, ekDiver, ScreenW*0.7, -20) }),
			ev(5.5, func(p *Play) { spawnRow(p, ekGrunt, 6, -20, 24) }),
			ev(8.0, func(p *Play) {
				spawnAt(p, ekDiver, ScreenW*0.2, -20)
				spawnAt(p, ekDiver, ScreenW*0.5, -40)
				spawnAt(p, ekDiver, ScreenW*0.8, -20)
			}),
			ev(12.0, func(p *Play) { spawnRow(p, ekWeaver, 5, -20, 44) }),
		}),
		// 4: first turret (mini-tank) with escorts
		wave("", "ARMORED CRUISER", 1.24, 14, []spawnEvent{
			ev(0.0, func(p *Play) { spawnAt(p, ekTurret, ScreenW/2, -30) }),
			ev(1.5, func(p *Play) { spawnRow(p, ekGrunt, 5, -20, 40) }),
			ev(6.0, func(p *Play) { spawnRow(p, ekWeaver, 4, -20, 50) }),
			ev(9.0, func(p *Play) { spawnAt(p, ekTurret, ScreenW*0.3, -30); spawnAt(p, ekTurret, ScreenW*0.7, -30) }),
		}),
		// 5: bombers + mines
		wave("", "MINEFIELD", 1.32, 16, []spawnEvent{
			ev(0.0, func(p *Play) { spawnAt(p, ekBomber, ScreenW*0.35, -30); spawnAt(p, ekBomber, ScreenW*0.65, -30) }),
			ev(3.0, func(p *Play) { spawnRow(p, ekMine, 4, -20, 50) }),
			ev(6.5, func(p *Play) { spawnRow(p, ekGrunt, 6, -20, 24) }),
			ev(9.0, func(p *Play) { spawnRow(p, ekMine, 5, -20, 40) }),
			ev(12.0, func(p *Play) { spawnAt(p, ekBomber, ScreenW*0.5, -30) }),
		}),
		// 6: MINI-BOSS
		{
			kind: skMini, title: "WARNING", sub: "WARDEN APPROACHING",
			col: colTurret, label: "MINI-BOSS", diff: 1.34,
		},
		// 7: heavier mixed assault
		wave("", "RELENTLESS", 1.42, 16, []spawnEvent{
			ev(0.0, func(p *Play) { spawnRow(p, ekWeaver, 5, -20, 44) }),
			ev(2.5, func(p *Play) { spawnVee(p, ekDiver, 5, ScreenW/2, -20, 44, 24) }),
			ev(5.5, func(p *Play) { spawnAt(p, ekTurret, ScreenW*0.25, -30); spawnAt(p, ekTurret, ScreenW*0.75, -30) }),
			ev(9.0, func(p *Play) { spawnRow(p, ekGrunt, 7, -20, 22) }),
			ev(12.0, func(p *Play) { spawnRow(p, ekWeaver, 6, -20, 40) }),
		}),
		// 8: diver swarm + bombers
		wave("", "SWARM", 1.52, 17, []spawnEvent{
			ev(0.0, func(p *Play) { spawnAt(p, ekBomber, ScreenW*0.3, -30); spawnAt(p, ekBomber, ScreenW*0.7, -30) }),
			ev(2.0, func(p *Play) {
				spawnAt(p, ekDiver, ScreenW*0.15, -20)
				spawnAt(p, ekDiver, ScreenW*0.4, -30)
				spawnAt(p, ekDiver, ScreenW*0.6, -30)
				spawnAt(p, ekDiver, ScreenW*0.85, -20)
			}),
			ev(5.5, func(p *Play) { spawnRow(p, ekMine, 5, -20, 40) }),
			ev(8.0, func(p *Play) { spawnVee(p, ekDiver, 7, ScreenW/2, -20, 40, 22) }),
			ev(11.5, func(p *Play) { spawnRow(p, ekWeaver, 6, -20, 40) }),
		}),
		// 9: fortress line
		wave("", "FORTRESS LINE", 1.62, 16, []spawnEvent{
			ev(0.0, func(p *Play) { spawnRow(p, ekTurret, 3, -30, 60) }),
			ev(3.0, func(p *Play) { spawnRow(p, ekGrunt, 8, -20, 20) }),
			ev(6.0, func(p *Play) { spawnAt(p, ekBomber, ScreenW*0.5, -30); spawnRow(p, ekMine, 4, -20, 50) }),
			ev(9.5, func(p *Play) { spawnVee(p, ekDiver, 6, ScreenW/2, -20, 44, 22) }),
			ev(12.5, func(p *Play) { spawnRow(p, ekWeaver, 6, -20, 40) }),
		}),
		// 10: the gauntlet before the final boss
		wave("", "THE GAUNTLET", 1.72, 18, []spawnEvent{
			ev(0.0, func(p *Play) { spawnRow(p, ekWeaver, 7, -20, 34) }),
			ev(2.5, func(p *Play) { spawnAt(p, ekTurret, ScreenW*0.2, -30); spawnAt(p, ekTurret, ScreenW*0.8, -30) }),
			ev(5.0, func(p *Play) { spawnVee(p, ekDiver, 7, ScreenW/2, -20, 40, 20) }),
			ev(8.0, func(p *Play) { spawnAt(p, ekBomber, ScreenW*0.3, -30); spawnAt(p, ekBomber, ScreenW*0.7, -30) }),
			ev(10.5, func(p *Play) { spawnRow(p, ekMine, 6, -20, 34) }),
			ev(13.0, func(p *Play) { spawnRow(p, ekGrunt, 9, -20, 18); spawnRow(p, ekWeaver, 4, -40, 60) }),
		}),
		// 11: FINAL BOSS
		{
			kind: skBoss, title: "FINAL BATTLE", sub: "THE ANNIHILATOR",
			col: colBoss, label: "FINAL BOSS", diff: 1.8,
		},
	}
	return stages
}
