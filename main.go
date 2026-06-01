// Command starfall is a neon vertical-scrolling arcade shoot-'em-up.
//
// Beat all ten waves, the Warden mini-boss, and the Annihilator to win — a
// complete run takes roughly twenty minutes.
package main

import (
	"log"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"starfall/internal/game"
)

func main() {
	g, err := game.NewGame()
	if err != nil {
		log.Fatalf("starfall: failed to start: %v", err)
	}

	ebiten.SetWindowSize(600, 900)
	ebiten.SetWindowTitle("STARFALL")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(400, 600, -1, -1)

	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		// An audio-device failure should never take the whole game down with a
		// fatal stack trace; exit cleanly instead. (Audio is pre-checked at
		// startup, so this is a last resort.)
		if isAudioError(err) {
			log.Printf("starfall: audio error, exiting: %v", err)
			return
		}
		log.Fatalf("starfall: %v", err)
	}
}

func isAudioError(err error) bool {
	s := strings.ToLower(err.Error())
	for _, k := range []string{"audio", "oto", "alsa", "snd_pcm"} {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}
