package game

import (
	"bytes"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 44100

type sfxID int

const (
	sfxShoot sfxID = iota
	sfxEnemyShoot
	sfxHurt
	sfxPlayerDie
	sfxExplode
	sfxPowerup
	sfxExtend
	sfxCoin
	sfxDive
	sfxBossPhase
	sfxBossWarn
	sfxBomb
	sfxMenuMove
	sfxMenuSelect
	sfxStart
	sfxCount
)

type voicePool struct {
	players []*audio.Player
	idx     int
	vol     float64
}

// Audio owns the audio context, a pool of voices per sound (so identical
// sounds can overlap), and the looping music track. All methods are safe to
// call on a nil/disabled Audio, which makes headless tests and audio-less
// machines a no-op rather than a crash.
type Audio struct {
	ctx     *audio.Context
	pools   map[sfxID]*voicePool
	music   *audio.Player
	muted   bool
	enabled bool
}

// synthRng is a fixed-seed source so generated sounds are identical every run.
var synthRng = rand.New(rand.NewSource(1))

func newAudio(enabled bool) *Audio {
	if !enabled {
		return &Audio{enabled: false}
	}
	a := &Audio{
		ctx:     audio.NewContext(sampleRate),
		pools:   map[sfxID]*voicePool{},
		enabled: true,
	}
	// (id, pcm, pool size, volume)
	defs := []struct {
		id  sfxID
		pcm []byte
		n   int
		vol float64
	}{
		{sfxShoot, synthShoot(), 10, 0.16},
		{sfxEnemyShoot, synthEnemyShoot(), 8, 0.18},
		{sfxHurt, synthHurt(), 4, 0.5},
		{sfxPlayerDie, synthPlayerDie(), 2, 0.6},
		{sfxExplode, synthExplode(), 8, 0.42},
		{sfxPowerup, synthPowerup(), 3, 0.4},
		{sfxExtend, synthExtend(), 2, 0.45},
		{sfxCoin, synthCoin(), 4, 0.4},
		{sfxDive, synthDive(), 4, 0.35},
		{sfxBossPhase, synthBossPhase(), 2, 0.6},
		{sfxBossWarn, synthBossWarn(), 2, 0.5},
		{sfxBomb, synthBomb(), 2, 0.6},
		{sfxMenuMove, synthMenuMove(), 3, 0.3},
		{sfxMenuSelect, synthMenuSelect(), 2, 0.4},
		{sfxStart, synthStart(), 2, 0.45},
	}
	for _, d := range defs {
		pool := &voicePool{vol: d.vol}
		for i := 0; i < d.n; i++ {
			pool.players = append(pool.players, a.ctx.NewPlayerFromBytes(d.pcm))
		}
		a.pools[d.id] = pool
	}

	// Music loop.
	musicPCM := synthMusic()
	loop := audio.NewInfiniteLoop(bytes.NewReader(musicPCM), int64(len(musicPCM)))
	if mp, err := a.ctx.NewPlayer(loop); err == nil {
		mp.SetVolume(0.32)
		a.music = mp
	}
	return a
}

func (a *Audio) play(id sfxID) {
	if a == nil || !a.enabled || a.muted {
		return
	}
	pool := a.pools[id]
	if pool == nil || len(pool.players) == 0 {
		return
	}
	p := pool.players[pool.idx]
	pool.idx = (pool.idx + 1) % len(pool.players)
	_ = p.Rewind()
	p.SetVolume(pool.vol)
	p.Play()
}

func (a *Audio) startMusic() {
	if a == nil || !a.enabled || a.muted || a.music == nil {
		return
	}
	if !a.music.IsPlaying() {
		a.music.Play()
	}
}

func (a *Audio) toggleMute() bool {
	if a == nil || !a.enabled {
		return true
	}
	a.muted = !a.muted
	if a.music != nil {
		if a.muted {
			a.music.Pause()
		} else {
			a.music.Play()
		}
	}
	return a.muted
}

// --- synthesis ---

func nsamp(dur float64) int { return int(dur * sampleRate) }

type wave int

const (
	wSine wave = iota
	wSquare
	wTri
	wSaw
	wNoise
)

func osc(w wave, phase float64) float64 {
	switch w {
	case wSquare:
		if math.Sin(phase) >= 0 {
			return 1
		}
		return -1
	case wTri:
		return (2 / math.Pi) * math.Asin(math.Sin(phase))
	case wSaw:
		p := math.Mod(phase/(2*math.Pi), 1)
		return 2*p - 1
	case wNoise:
		return synthRng.Float64()*2 - 1
	default:
		return math.Sin(phase)
	}
}

// tone renders a single voice with frequency glide f0->f1 and an envelope
// function env(t) over normalized time t in [0,1].
func tone(f0, f1, dur float64, w wave, env func(t float64) float64) []float64 {
	n := nsamp(dur)
	out := make([]float64, n)
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		f := lerp(f0, f1, t)
		phase += 2 * math.Pi * f / sampleRate
		out[i] = osc(w, phase) * env(t)
	}
	return out
}

func decay(k float64) func(float64) float64 {
	return func(t float64) float64 { return math.Exp(-k * t) }
}

func ar(attack float64) func(float64) float64 {
	return func(t float64) float64 {
		if t < attack {
			return t / attack
		}
		return 1 - (t-attack)/(1-attack)
	}
}

func mix(layers ...[]float64) []float64 {
	n := 0
	for _, l := range layers {
		if len(l) > n {
			n = len(l)
		}
	}
	out := make([]float64, n)
	for _, l := range layers {
		for i, s := range l {
			out[i] += s
		}
	}
	return out
}

func concat(parts ...[]float64) []float64 {
	var out []float64
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func gain(s []float64, g float64) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = v * g
	}
	return out
}

func silence(dur float64) []float64 { return make([]float64, nsamp(dur)) }

// encode converts mono float samples to 16-bit LE stereo PCM, soft-clipping.
func encode(mono []float64) []byte {
	buf := make([]byte, len(mono)*4)
	for i, s := range mono {
		s = math.Tanh(s) // gentle limiter to avoid harsh clipping
		v := int16(clampf(s, -1, 1) * 32000)
		lo := byte(v)
		hi := byte(uint16(v) >> 8)
		j := i * 4
		buf[j], buf[j+1], buf[j+2], buf[j+3] = lo, hi, lo, hi
	}
	return buf
}

func midiFreq(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }

// --- individual SFX ---

func synthShoot() []byte {
	t := tone(1000, 560, 0.07, wSquare, decay(7))
	click := gain(tone(2200, 1800, 0.012, wNoise, decay(8)), 0.4)
	return encode(gain(mix(t, click), 0.7))
}

func synthEnemyShoot() []byte {
	return encode(gain(tone(380, 240, 0.11, wSaw, decay(5)), 0.7))
}

func synthHurt() []byte {
	n := gain(tone(400, 400, 0.16, wNoise, decay(6)), 0.6)
	t := tone(320, 110, 0.18, wSquare, decay(4))
	return encode(gain(mix(n, t), 0.9))
}

func synthPlayerDie() []byte {
	t := tone(520, 50, 0.55, wSaw, decay(2.5))
	n := gain(tone(0, 0, 0.5, wNoise, decay(3)), 0.7)
	return encode(gain(mix(t, n), 0.95))
}

func synthExplode() []byte {
	n := gain(tone(0, 0, 0.32, wNoise, decay(5)), 0.9)
	low := tone(190, 45, 0.32, wTri, decay(4))
	return encode(gain(mix(n, low), 0.9))
}

func synthPowerup() []byte {
	a := tone(660, 680, 0.06, wSquare, ar(0.1))
	b := tone(880, 900, 0.06, wSquare, ar(0.1))
	c := tone(1320, 1340, 0.10, wSquare, decay(3))
	return encode(gain(concat(a, b, c), 0.7))
}

func synthExtend() []byte {
	notes := []float64{523, 659, 784, 1046}
	var parts []float64
	for _, f := range notes {
		parts = append(parts, tone(f, f, 0.09, wSquare, ar(0.08))...)
	}
	return encode(gain(parts, 0.7))
}

func synthCoin() []byte {
	a := tone(988, 988, 0.05, wSquare, ar(0.1))
	b := tone(1318, 1318, 0.10, wSquare, decay(3))
	return encode(gain(concat(a, b), 0.7))
}

func synthDive() []byte {
	t := tone(1300, 280, 0.26, wSaw, decay(2))
	n := gain(tone(0, 0, 0.26, wNoise, decay(4)), 0.25)
	return encode(gain(mix(t, n), 0.8))
}

func synthBossPhase() []byte {
	boom := gain(tone(0, 0, 0.22, wNoise, decay(6)), 0.7)
	rise := tone(110, 320, 0.45, wSaw, ar(0.2))
	return encode(gain(mix(boom, rise), 0.95))
}

func synthBossWarn() []byte {
	beep := func(f float64) []float64 { return tone(f, f, 0.14, wSquare, ar(0.05)) }
	return encode(gain(concat(beep(880), silence(0.06), beep(660), silence(0.06)), 0.7))
}

func synthBomb() []byte {
	sweep := tone(1600, 80, 0.5, wSaw, decay(2))
	n := gain(tone(0, 0, 0.45, wNoise, decay(3)), 0.6)
	return encode(gain(mix(sweep, n), 1.0))
}

func synthMenuMove() []byte {
	return encode(gain(tone(620, 620, 0.04, wSquare, ar(0.2)), 0.6))
}

func synthMenuSelect() []byte {
	return encode(gain(mix(tone(660, 660, 0.13, wSquare, decay(3)), tone(990, 990, 0.13, wSquare, decay(3))), 0.6))
}

func synthStart() []byte {
	notes := []float64{392, 523, 659, 880}
	var parts []float64
	for _, f := range notes {
		parts = append(parts, tone(f, f, 0.08, wSquare, ar(0.1))...)
	}
	return encode(gain(parts, 0.7))
}

// synthMusic builds a short looping chiptune: bass, arpeggio, sparse lead, and
// a simple kick/hat groove over an Am-F-C-G progression.
func synthMusic() []byte {
	const bpm = 132.0
	step := 60.0 / bpm / 4 // sixteenth-note duration
	const bars = 4
	const stepsPerBar = 16
	total := bars * stepsPerBar
	buf := make([]float64, nsamp(step*float64(total))+8)

	put := func(startStep int, s []float64) {
		off := nsamp(step * float64(startStep))
		for i, v := range s {
			if off+i < len(buf) {
				buf[off+i] += v
			}
		}
	}

	// chord tones (MIDI) per bar: Am, F, C, G
	chords := [][]float64{
		{57, 60, 64}, // A C E
		{53, 57, 60}, // F A C
		{48, 52, 55}, // C E G
		{55, 59, 62}, // G B D
	}
	bassNote := []float64{45, 41, 48, 43} // A2 F2 C3 G2

	for bar := 0; bar < bars; bar++ {
		ch := chords[bar]
		// bass: root on step 0 and 8
		for _, s := range []int{0, 8} {
			f := midiFreq(bassNote[bar])
			put(bar*stepsPerBar+s, gain(tone(f, f, step*4, wTri, decay(1.6)), 0.5))
		}
		// arpeggio: 8th notes cycling chord tones, rising octave
		for s := 0; s < stepsPerBar; s += 2 {
			f := midiFreq(ch[(s/2)%len(ch)] + 12)
			put(bar*stepsPerBar+s, gain(tone(f, f, step*1.8, wTri, decay(2.2)), 0.28))
		}
		// kick on quarters, hat on offbeats
		for s := 0; s < stepsPerBar; s += 4 {
			put(bar*stepsPerBar+s, gain(tone(120, 50, step*1.5, wTri, decay(8)), 0.55))
		}
		for s := 2; s < stepsPerBar; s += 4 {
			put(bar*stepsPerBar+s, gain(tone(0, 0, step*0.4, wNoise, decay(20)), 0.12))
		}
	}

	// sparse lead melody (square) over the loop
	lead := []struct {
		step int
		midi float64
		dur  float64
	}{
		{0, 69, 2}, {4, 72, 2}, {8, 76, 2}, {12, 72, 1}, {14, 69, 2},
		{16, 65, 2}, {20, 69, 2}, {24, 72, 4},
		{32, 64, 2}, {36, 67, 2}, {40, 72, 2}, {44, 71, 2},
		{48, 67, 2}, {52, 71, 2}, {56, 74, 2}, {60, 79, 4},
	}
	for _, ln := range lead {
		f := midiFreq(ln.midi)
		put(ln.step, gain(tone(f, f, step*ln.dur, wSquare, decay(2.0)), 0.16))
	}

	return encode(gain(buf, 0.7))
}
