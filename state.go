package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

const stateVersion = 1

type PerpetualSnapshot struct {
	Symbol               string    `json:"symbol"`
	ObservedAt           time.Time `json:"observed_at"`
	NotionalOpenInterest float64   `json:"notional_open_interest"`
}

type LocalState struct {
	Version    int                          `json:"version"`
	Perpetuals map[string]PerpetualSnapshot `json:"perpetuals"`
}

type StateStore struct {
	path string
}

func NewStateStore(path string) *StateStore {
	return &StateStore{path: path}
}

func (s *StateStore) Load() (*LocalState, error) {
	state := emptyLocalState()
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, state); err != nil {
		return emptyLocalState(), fmt.Errorf("decode state: %w", err)
	}
	if state.Version != stateVersion {
		return emptyLocalState(), fmt.Errorf("unsupported state version %d", state.Version)
	}
	if state.Perpetuals == nil {
		state.Perpetuals = make(map[string]PerpetualSnapshot)
	}
	for key, snapshot := range state.Perpetuals {
		if key == "" || snapshot.Symbol != key || snapshot.ObservedAt.IsZero() || math.IsNaN(snapshot.NotionalOpenInterest) || math.IsInf(snapshot.NotionalOpenInterest, 0) || snapshot.NotionalOpenInterest < 0 {
			delete(state.Perpetuals, key)
		}
	}
	return state, nil
}

func (s *StateStore) Save(state *LocalState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}
	state.Version = stateVersion
	if state.Perpetuals == nil {
		state.Perpetuals = make(map[string]PerpetualSnapshot)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	temp, err := os.CreateTemp(dir, ".coindaily-state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	tempPath := temp.Name()
	keepTemp := false
	defer func() {
		if !keepTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("set state permissions: %w", err)
	}
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		temp.Close()
		return fmt.Errorf("encode state: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync state: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close state: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	keepTemp = true
	return nil
}

func emptyLocalState() *LocalState {
	return &LocalState{Version: stateVersion, Perpetuals: make(map[string]PerpetualSnapshot)}
}
