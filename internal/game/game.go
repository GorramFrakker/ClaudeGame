package game

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var titleMenu = []string{"LAUNCH", "HOW TO PLAY", "QUIT"}

// Game is the root object implementing ebiten.Game. It owns shared resources
// and drives the top-level state machine (menus + gameplay).
type Game struct {
	fonts *Fonts
	audio *Audio
	rng   *rand.Rand
	stars *Starfield
	save  Save

	mode Mode
	play *Play

	menuSel        int
	titleT         float64
	fade           float64
	endInputDelay  float64
	quit           bool
	frame          int64
	mutedFlash     float64

	worldImg *ebiten.Image

	// test/automation hooks
	demo        bool
	god         bool
	startStage  int
	shotTargets []int64
	shotIdx     int
	shotDir     string
	exitAt      int64
}

// NewGame constructs the game, loading save data and initializing audio/fonts.
func NewGame() (*Game, error) {
	fonts, err := newFonts()
	if err != nil {
		return nil, err
	}
	seed := time.Now().UnixNano()
	g := &Game{
		fonts:    fonts,
		rng:      rand.New(rand.NewSource(seed)),
		save:     LoadSave(),
		mode:       ModeTitle,
		startStage: -1,
		worldImg:   ebiten.NewImage(ScreenW, ScreenH),
	}
	g.stars = newStarfield(g.rng)

	audioOn := os.Getenv("STARFALL_NOAUDIO") == "" && audioDeviceLikely()
	g.audio = newAudio(audioOn)
	if g.save.Muted {
		g.audio.muted = true
	}
	g.audio.startMusic()

	g.configureTestHooks()
	return g, nil
}

func (g *Game) configureTestHooks() {
	g.demo = os.Getenv("STARFALL_DEMO") != ""
	g.god = os.Getenv("STARFALL_GOD") != ""
	if v := os.Getenv("STARFALL_START_STAGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			g.startStage = n
		}
	}
	if v := os.Getenv("STARFALL_SHOT_AT"); v != "" {
		for _, part := range strings.Split(v, ",") {
			if n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
				g.shotTargets = append(g.shotTargets, n)
			}
		}
		sort.Slice(g.shotTargets, func(i, j int) bool { return g.shotTargets[i] < g.shotTargets[j] })
		g.shotDir = os.Getenv("STARFALL_SHOT_DIR")
		if g.shotDir == "" {
			g.shotDir = "shots"
		}
		_ = os.MkdirAll(g.shotDir, 0o755)
	}
	if v := os.Getenv("STARFALL_EXIT_AT"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			g.exitAt = n
		}
	}
}

// audioDeviceLikely reports whether a usable audio output device is probably
// present. Ebitengine can only run once per process, so a mid-run audio-device
// failure can't be recovered from — we must decide up front. Windows and macOS
// effectively always expose a default device; on Linux/BSD we require a sound
// device node so headless machines simply run silently instead of crashing.
func audioDeviceLikely() bool {
	switch runtime.GOOS {
	case "windows", "darwin", "js":
		return true
	default:
		for _, p := range []string{"/dev/snd", "/dev/dsp"} {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
		return false
	}
}

func (g *Game) Layout(outW, outH int) (int, int) { return ScreenW, ScreenH }

func (g *Game) Update() error {
	g.frame++
	g.titleT += dt
	if g.fade > 0 {
		g.fade = approach(g.fade, 0, 3.2*dt)
	}
	if g.mutedFlash > 0 {
		g.mutedFlash -= dt
	}
	if g.endInputDelay > 0 {
		g.endInputDelay -= dt
	}

	in := readInput()
	if g.demo && g.mode == ModeTitle && g.frame > 20 {
		in.ConfirmPressed = true
	}
	if g.demo && g.mode == ModePlaying {
		in = demoInput(g.frame)
	}

	if in.MutePressed {
		muted := g.audio.toggleMute()
		g.save.Muted = muted
		g.save.Store()
		g.mutedFlash = 1.2
	}
	if in.FSPressed {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	switch g.mode {
	case ModeTitle:
		g.updateTitle(in)
	case ModeHowTo:
		if in.ConfirmPressed || in.BackPressed {
			g.audio.play(sfxMenuSelect)
			g.gotoMode(ModeTitle)
		}
	case ModePlaying:
		g.updatePlaying(in)
	case ModePaused:
		g.updatePaused(in)
	case ModeGameOver, ModeVictory:
		g.updateEnd(in)
	}

	// Parallax: stars react to player movement and threat level.
	drift, speed := 0.0, 1.0
	if g.play != nil {
		speed = g.play.starSpeed
		if g.mode == ModePlaying {
			drift = in.Move.X * 40
		}
	}
	if g.mode == ModePaused {
		speed = 0
	}
	g.stars.Update(drift, speed)

	if g.quit {
		return ebiten.Termination
	}
	if g.exitAt > 0 && g.frame >= g.exitAt {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) gotoMode(m Mode) {
	g.mode = m
	g.fade = 1
}

func (g *Game) startNewGame() {
	g.play = newPlay(g)
	if g.startStage >= 0 {
		g.play.director.idx = clampi(g.startStage, 0, len(g.play.director.stages)-1)
		g.play.director.startBanner()
	}
	g.mode = ModePlaying
	g.fade = 1
	g.audio.startMusic()
	g.audio.play(sfxStart)
}

func (g *Game) updateTitle(in Input) {
	if keyJust(ebiten.KeyUp, ebiten.KeyW) {
		g.menuSel = (g.menuSel + len(titleMenu) - 1) % len(titleMenu)
		g.audio.play(sfxMenuMove)
	}
	if keyJust(ebiten.KeyDown, ebiten.KeyS) {
		g.menuSel = (g.menuSel + 1) % len(titleMenu)
		g.audio.play(sfxMenuMove)
	}
	if in.ConfirmPressed {
		switch g.menuSel {
		case 0:
			g.audio.play(sfxMenuSelect)
			g.startNewGame()
		case 1:
			g.audio.play(sfxMenuSelect)
			g.gotoMode(ModeHowTo)
		case 2:
			g.quit = true
		}
	}
}

func (g *Game) updatePlaying(in Input) {
	if in.PausePressed {
		g.audio.play(sfxMenuSelect)
		g.mode = ModePaused
		return
	}
	if g.play == nil {
		g.gotoMode(ModeTitle)
		return
	}
	if g.god {
		g.play.player.invuln = 999
		g.play.player.shield = playerMaxShield
	}
	g.play.Update(in)
	if g.play.result != resRunning && g.play.endTimer <= 0 {
		if g.play.result == resWin {
			g.endGame(ModeVictory)
		} else {
			g.endGame(ModeGameOver)
		}
	}
}

func (g *Game) updatePaused(in Input) {
	if in.PausePressed || in.ConfirmPressed {
		g.audio.play(sfxMenuSelect)
		g.mode = ModePlaying
		return
	}
	if keyJust(ebiten.KeyQ) {
		g.audio.play(sfxMenuSelect)
		g.play = nil
		g.menuSel = 0
		g.gotoMode(ModeTitle)
	}
}

func (g *Game) endGame(m Mode) {
	if g.play != nil && g.play.score > g.save.HighScore {
		g.save.HighScore = g.play.score
		g.save.Store()
	}
	g.mode = m
	g.fade = 0.8
	g.endInputDelay = 0.6
	if m == ModeVictory {
		g.audio.play(sfxExtend)
	} else {
		g.audio.play(sfxPlayerDie)
	}
}

func (g *Game) updateEnd(in Input) {
	if g.endInputDelay > 0 {
		return
	}
	if in.ConfirmPressed {
		g.audio.play(sfxMenuSelect)
		g.startNewGame()
	}
	if in.BackPressed {
		g.audio.play(sfxMenuSelect)
		g.play = nil
		g.menuSel = 0
		g.gotoMode(ModeTitle)
	}
}

// demoInput drives an automated ship for attract-mode screenshots/tests.
func demoInput(frame int64) Input {
	t := float64(frame) / 60
	return Input{
		Move: V(math.Sin(t*1.3)*0.9, math.Sin(t*0.55)*0.5),
		Fire: true,
	}
}

// --- drawing ---

func (g *Game) Draw(screen *ebiten.Image) {
	g.worldImg.Clear()
	g.stars.Draw(g.worldImg)
	if g.play != nil && g.mode != ModeTitle && g.mode != ModeHowTo {
		g.play.DrawWorld(g.worldImg)
	}

	var op ebiten.DrawImageOptions
	if g.play != nil && g.play.shake > 0.3 && g.mode == ModePlaying {
		sh := g.play.shake
		t := g.play.time
		ox := (math.Sin(t*43) + math.Sin(t*97)) * 0.5 * sh * 0.45
		oy := (math.Cos(t*51) + math.Sin(t*89)) * 0.5 * sh * 0.45
		op.GeoM.Translate(ox, oy)
	}
	screen.DrawImage(g.worldImg, &op)

	switch g.mode {
	case ModeTitle:
		g.drawTitle(screen)
	case ModeHowTo:
		g.drawHowTo(screen)
	case ModePlaying:
		g.play.drawHUD(screen)
	case ModePaused:
		g.play.drawHUD(screen)
		g.drawPaused(screen)
	case ModeGameOver:
		g.play.drawHUD(screen)
		g.drawEnd(screen, false)
	case ModeVictory:
		g.play.drawHUD(screen)
		g.drawEnd(screen, true)
	}

	// Full-screen flashes from gameplay.
	if g.play != nil {
		if g.play.flash > 0 {
			a := uint8(clampf(g.play.flash, 0, 1) * float64(g.play.flashCol.A))
			vector_fill(screen, withAlpha(g.play.flashCol, a))
		}
		if g.play.bombFlash > 0 {
			a := uint8(clampf(g.play.bombFlash, 0, 1) * 150)
			vector_fill(screen, color.RGBA{220, 240, 255, a})
		}
	}

	// Mute indicator.
	if g.mutedFlash > 0 && g.audio != nil {
		msg := "MUSIC ON"
		if g.audio.muted {
			msg = "MUTED"
		}
		g.fonts.center(screen, msg, ScreenW/2, ScreenH/2, 18, withAlpha(colUIText, uint8(clampf(g.mutedFlash, 0, 1)*255)), true)
	}

	// Fade overlay between modes.
	if g.fade > 0 {
		vector_fill(screen, color.RGBA{0, 0, 0, uint8(clampf(g.fade, 0, 1) * 255)})
	}

	g.maybeScreenshot(screen)
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	f := g.fonts
	// dim the field a touch for menu legibility
	vector_fill(screen, color.RGBA{4, 6, 16, 110})

	glow := 0.5 + 0.5*math.Sin(g.titleT*2)
	cx := float64(ScreenW / 2)
	drawGlow(screen, cx, 150, 150, colPlayer, 0.25+0.15*glow)
	f.center(screen, "STARFALL", cx, 150, 64, colPlayerHot, true)
	f.center(screen, "STARFALL", cx, 150, 64, withAlpha(colPlayer, 120), true)
	f.center(screen, "— HOLD THE LINE —", cx, 196, 16, colUIDim, false)

	for i, item := range titleMenu {
		y := 320 + float64(i)*42
		col := colUIDim
		label := item
		if i == g.menuSel {
			col = colGold
			label = ">  " + item + "  <"
			drawGlow(screen, cx, y, 90, colGold, 0.18)
		}
		f.center(screen, label, cx, y, 22, col, i == g.menuSel)
	}

	f.center(screen, "HIGH SCORE   "+commaInt(g.save.HighScore), cx, 520, 15, colUIText, false)
	f.center(screen, "Arrows/WASD to choose · Enter/Z to select", cx, 640, 12, colUIDim, false)
	f.center(screen, "M: mute   F11: fullscreen", cx, 662, 12, colUIDim, false)
	f.center(screen, "v1.0", cx, ScreenH-14, 11, withAlpha(colUIDim, 160), false)
}

func (g *Game) drawHowTo(screen *ebiten.Image) {
	f := g.fonts
	vector_fill(screen, color.RGBA{4, 6, 16, 180})
	cx := float64(ScreenW / 2)
	f.center(screen, "HOW TO PLAY", cx, 70, 34, colPlayerHot, true)

	lines := []struct {
		k, v string
	}{
		{"MOVE", "Arrow keys or WASD"},
		{"FIRE", "Z / Space (hold)"},
		{"BOMB", "X  — clears bullets, big damage"},
		{"FOCUS", "Shift — slow, precise, shows hitbox"},
		{"PAUSE", "Esc / P"},
	}
	y := 130.0
	for _, ln := range lines {
		f.right(screen, ln.k, cx-12, y, 16, colGold, true)
		f.left(screen, ln.v, cx+12, y, 16, colUIText, false)
		y += 30
	}

	y += 6
	f.center(screen, "COLLECT", cx, y, 16, colUIDim, true)
	y += 28
	legend := []struct {
		c color.RGBA
		s string
	}{
		{colPowerPU, "P  Power — upgrade your weapon"},
		{colShieldPU, "S  Shield — restore your shield"},
		{colBombPU, "B  Bomb — extra bomb"},
		{colGold, "$  Bonus — points"},
	}
	for _, lg := range legend {
		iconShip(screen, cx-118, y, 6, lg.c)
		f.left(screen, lg.s, cx-104, y, 14, colUIText, false)
		y += 26
	}

	y += 14
	f.center(screen, "Survive 10 waves, destroy the Annihilator.", cx, y, 14, colUIDim, false)
	f.center(screen, "Chain kills for a higher score multiplier!", cx, y+22, 14, colUIDim, false)

	f.center(screen, "Press Z / Esc to return", cx, ScreenH-40, 14, colGold, true)
}

func (g *Game) drawPaused(screen *ebiten.Image) {
	vector_fill(screen, color.RGBA{0, 0, 0, 150})
	cx := float64(ScreenW / 2)
	g.fonts.center(screen, "PAUSED", cx, ScreenH/2-30, 44, colPlayerHot, true)
	g.fonts.center(screen, "Esc / Enter — resume", cx, ScreenH/2+24, 16, colUIText, false)
	g.fonts.center(screen, "Q — quit to title", cx, ScreenH/2+50, 16, colUIDim, false)
}

func (g *Game) drawEnd(screen *ebiten.Image, win bool) {
	vector_fill(screen, color.RGBA{0, 0, 0, 170})
	cx := float64(ScreenW / 2)
	f := g.fonts

	title, col := "GAME OVER", colDanger
	if win {
		title, col = "VICTORY!", colGold
		drawGlow(screen, cx, 150, 160, colGold, 0.2+0.1*math.Sin(g.titleT*3))
	}
	f.center(screen, title, cx, 150, 56, col, true)
	if win {
		f.center(screen, "The Annihilator is destroyed. You held the line.", cx, 196, 13, colUIText, false)
	}

	p := g.play
	newRecord := p != nil && p.score >= g.save.HighScore && p.score > 0
	y := 280.0
	stat := func(k, v string) {
		f.right(screen, k, cx-10, y, 16, colUIDim, false)
		f.left(screen, v, cx+10, y, 16, colUIText, true)
		y += 30
	}
	if p != nil {
		stat("SCORE", commaInt(p.score))
		stat("WAVE REACHED", p.director.hudLabel())
		stat("ENEMIES DOWN", strconv.Itoa(p.enemiesKilled))
		stat("BEST CHAIN", "x"+strconv.Itoa(clampi(1+p.maxCombo/4, 1, 8))+"  ("+strconv.Itoa(p.maxCombo)+")")
	}
	y += 6
	if newRecord {
		pulse := 0.5 + 0.5*math.Sin(g.titleT*8)
		f.center(screen, ">> NEW HIGH SCORE <<", cx, y, 20, withAlpha(colGold, uint8(150+105*pulse)), true)
	} else {
		f.center(screen, "HIGH SCORE   "+commaInt(g.save.HighScore), cx, y, 15, colUIDim, false)
	}

	f.center(screen, "Z — play again", cx, ScreenH-70, 16, colGold, true)
	f.center(screen, "Esc — title", cx, ScreenH-46, 14, colUIDim, false)
}

// vector_fill draws a translucent full-screen overlay that alpha-blends with
// whatever is already on the destination (unlike Image.Fill, which replaces).
func vector_fill(dst *ebiten.Image, c color.RGBA) {
	if c.A == 0 {
		return
	}
	vector.DrawFilledRect(dst, 0, 0, ScreenW, ScreenH, c, false)
}

func (g *Game) maybeScreenshot(screen *ebiten.Image) {
	// Capture on the first Draw at or past each target frame. Ebiten may skip
	// Draw calls when the renderer is slow, so we must not require an exact match.
	if g.shotIdx >= len(g.shotTargets) || g.frame < g.shotTargets[g.shotIdx] {
		return
	}
	g.shotIdx++
	buf := make([]byte, 4*ScreenW*ScreenH)
	screen.ReadPixels(buf)
	img := &image.RGBA{Pix: buf, Stride: 4 * ScreenW, Rect: image.Rect(0, 0, ScreenW, ScreenH)}
	path := filepath.Join(g.shotDir, "shot_"+strconv.FormatInt(g.frame, 10)+".png")
	frw, err := os.Create(path)
	if err != nil {
		return
	}
	defer frw.Close()
	_ = png.Encode(frw, img)
}
