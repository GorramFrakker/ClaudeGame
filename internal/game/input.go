package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input is a snapshot of the player's controls for one frame, abstracting over
// keyboard and gamepad so the rest of the game never touches raw devices.
type Input struct {
	Move           Vec2 // desired move direction, each axis in [-1,1]
	Fire           bool // held: fire main weapon
	Focus          bool // held: precision/slow movement
	BombPressed    bool // edge: detonate a bomb
	PausePressed   bool // edge: toggle pause
	ConfirmPressed bool // edge: menu confirm / start
	BackPressed    bool // edge: menu back / quit-to-title
	MutePressed    bool // edge: toggle audio
	FSPressed      bool // edge: toggle fullscreen
	AnyPressed     bool // edge: any control pressed (for "press any key")
}

func keyDown(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return true
		}
	}
	return false
}

func keyJust(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

// readInput gathers the current frame's controls.
func readInput() Input {
	var in Input

	// Keyboard movement.
	if keyDown(ebiten.KeyLeft, ebiten.KeyA) {
		in.Move.X -= 1
	}
	if keyDown(ebiten.KeyRight, ebiten.KeyD) {
		in.Move.X += 1
	}
	if keyDown(ebiten.KeyUp, ebiten.KeyW) {
		in.Move.Y -= 1
	}
	if keyDown(ebiten.KeyDown, ebiten.KeyS) {
		in.Move.Y += 1
	}

	in.Fire = keyDown(ebiten.KeySpace, ebiten.KeyZ, ebiten.KeyJ)
	in.Focus = keyDown(ebiten.KeyShiftLeft, ebiten.KeyShiftRight)
	in.BombPressed = keyJust(ebiten.KeyX, ebiten.KeyK, ebiten.KeySlash)
	in.PausePressed = keyJust(ebiten.KeyEscape, ebiten.KeyP)
	in.ConfirmPressed = keyJust(ebiten.KeyEnter, ebiten.KeyNumpadEnter, ebiten.KeySpace, ebiten.KeyZ)
	in.BackPressed = keyJust(ebiten.KeyEscape)
	in.MutePressed = keyJust(ebiten.KeyM)
	in.FSPressed = keyJust(ebiten.KeyF11)

	// Gamepad support (first connected pad). Adds to keyboard rather than
	// replacing it, so either works at any time.
	pads := ebiten.AppendGamepadIDs(nil)
	if len(pads) > 0 {
		id := pads[0]
		readGamepad(id, &in)
	}

	in.AnyPressed = in.ConfirmPressed || in.BombPressed || in.PausePressed ||
		len(inpututil.AppendJustPressedKeys(nil)) > 0

	// Normalize diagonal movement so it isn't faster than cardinal movement.
	if l := in.Move.Len(); l > 1 {
		in.Move = in.Move.Mul(1 / l)
	}
	return in
}

func readGamepad(id ebiten.GamepadID, in *Input) {
	const dead = 0.28
	if ebiten.IsStandardGamepadLayoutAvailable(id) {
		ax := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		ay := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if absf(ax) > dead {
			in.Move.X += ax
		}
		if absf(ay) > dead {
			in.Move.Y += ay
		}
		// D-pad.
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			in.Move.X -= 1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			in.Move.X += 1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			in.Move.Y -= 1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			in.Move.Y += 1
		}
		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom) ||
			ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomRight) {
			in.Fire = true
		}
		in.Focus = in.Focus || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomLeft)
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) ||
			inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightLeft) {
			in.BombPressed = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			in.PausePressed = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) ||
			inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight) {
			in.ConfirmPressed = true
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightRight) {
			in.BackPressed = true
		}
	}
}
