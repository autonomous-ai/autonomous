package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State persists deployed component versions.
type State struct {
	Components map[string]string `json:"components"`
}

// Load reads state from file, or returns empty state if file does not exist.
func Load(path string) (*State, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &State{Components: map[string]string{}}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read state %s: %w", path, err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state %s: %w", path, err)
	}
	if s.Components == nil {
		s.Components = map[string]string{}
	}
	return &s, nil
}

// Save writes state to file.
func Save(path string, s *State) error {
	if s.Components == nil {
		s.Components = map[string]string{}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
