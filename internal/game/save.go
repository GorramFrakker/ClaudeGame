package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Save holds persistent data stored between runs (high score and settings).
type Save struct {
	HighScore int  `json:"high_score"`
	Muted     bool `json:"muted"`
}

// saveDir returns the per-user config directory for the game, creating nothing.
// On Windows this resolves under %AppData%; on other platforms under the
// standard user config dir.
func saveDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Starfall"), nil
}

func savePath() (string, error) {
	dir, err := saveDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "save.json"), nil
}

// LoadSave reads persistent data, returning a zero-value Save on any error so
// the game always starts cleanly even with no prior save.
func LoadSave() Save {
	var s Save
	p, err := savePath()
	if err != nil {
		return s
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s) // tolerate corrupt files
	if s.HighScore < 0 {
		s.HighScore = 0
	}
	return s
}

// Store writes persistent data best-effort; failures are ignored so the game
// never crashes due to a read-only or sandboxed filesystem.
func (s Save) Store() {
	dir, err := saveDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, "save.json"), b, 0o644)
}
