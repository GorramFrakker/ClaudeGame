package game

import (
	"math"
	"math/rand"
	"testing"
)

// newTestPlay builds a Play with audio disabled and a deterministic RNG so the
// simulation tests are reproducible and never touch a sound device.
func newTestPlay(seed int64) *Play {
	g := &Game{
		rng:   rand.New(rand.NewSource(seed)),
		audio: newAudio(false),
		save:  Save{},
	}
	return newPlay(g)
}

// TestFullCampaignCompletes drives an invulnerable, always-firing ship through
// the entire campaign and asserts it reaches victory. This is the end-to-end
// guard that the whole game loop (waves -> mini-boss -> final boss -> win) is
// reachable and never panics over a long run.
func TestFullCampaignCompletes(t *testing.T) {
	p := newTestPlay(42)
	p.player.weapon = maxWeapon

	const budget = 60 * 60 * 40 // 40 simulated minutes, hard ceiling
	frame := 0
	for ; frame < budget; frame++ {
		// God mode: keep the ship alive so the run always progresses.
		p.player.invuln = 999
		p.player.shield = playerMaxShield

		ts := float64(frame) / 60
		in := Input{
			Move: V(math.Sin(ts*1.7)*0.85, math.Sin(ts*0.9)*0.35),
			Fire: true,
		}
		p.Update(in)
		if p.result != resRunning {
			break
		}
	}

	if p.result != resWin {
		t.Fatalf("campaign did not reach victory: result=%d frame=%d state=%d stageIdx=%d enemies=%d bossNil=%v",
			p.result, frame, p.director.state, p.director.idx, len(p.enemies), p.boss == nil)
	}
	secs := float64(frame) / 60
	t.Logf("god-mode clear: %d frames = %.1fs (%.1f min), kills=%d, score=%d",
		frame, secs, secs/60, p.enemiesKilled, p.score)
}

// TestBossKillTimes measures how long the bosses take to kill when the player
// sits beneath them firing — the *fastest* possible fight. Real fights are
// longer, so these set a lower bound used to sanity-check boss HP tuning.
func TestBossKillTimes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mini   bool
		weapon int
	}{
		{"mini@lvl4", true, 4},
		{"final@lvl4", false, 4},
		{"final@lvl6", false, 6},
	} {
		p := newTestPlay(7)
		p.player.weapon = tc.weapon
		p.boss = newBoss(tc.mini, 1.8)
		// disable the director so it doesn't spawn waves on top.
		p.director.state = dsDone

		frame := 0
		const budget = 60 * 60 * 8
		for ; frame < budget; frame++ {
			p.player.invuln = 999
			// stay directly under the boss for max accuracy
			p.player.pos.X = clampf(p.boss.pos.X, playerVisR, ScreenW-playerVisR)
			p.Update(Input{Fire: true})
			if p.boss == nil {
				break
			}
		}
		if p.boss != nil {
			t.Errorf("%s: boss not killed within %d frames (hp=%.0f/%.0f)", tc.name, budget, p.boss.hp, p.boss.maxHP)
			continue
		}
		t.Logf("%s: killed in %.1fs", tc.name, float64(frame)/60)
	}
}

// TestEarlyEnemiesDie verifies basic offense: a few grunts in front of a firing
// player should be destroyed promptly.
func TestEarlyEnemiesDie(t *testing.T) {
	p := newTestPlay(1)
	p.player.weapon = 2
	p.director.state = dsDone // no spawns
	for i := 0; i < 5; i++ {
		spawnAt(p, ekGrunt, ScreenW/2, 120)
	}
	for frame := 0; frame < 60*8; frame++ {
		p.player.invuln = 999
		p.player.pos = V(ScreenW/2, ScreenH-120)
		p.Update(Input{Fire: true})
		if len(p.enemies) == 0 {
			t.Logf("cleared 5 stacked grunts in %.2fs, kills=%d", float64(frame)/60, p.enemiesKilled)
			return
		}
	}
	t.Fatalf("grunts not cleared: %d remain", len(p.enemies))
}

// TestPlayerCanDie ensures the lose condition works: with no input and constant
// fire aimed at it, an un-helped ship eventually runs out of lives.
func TestPlayerCanDie(t *testing.T) {
	p := newTestPlay(3)
	p.director.state = dsDone
	// Surround the player with point-blank enemy fire each frame.
	for frame := 0; frame < 60*30; frame++ {
		if frame%6 == 0 {
			p.enemyShootDir(p.player.pos.Add(V(0, -30)), math.Pi/2, 240, enemyBulletDmg, colEBullet)
		}
		p.Update(Input{})
		if p.result == resLose {
			t.Logf("player died after %.2fs as expected", float64(frame)/60)
			return
		}
	}
	t.Fatalf("player never died despite sustained point-blank fire (lives=%d shield=%.0f)", p.player.lives, p.player.shield)
}
