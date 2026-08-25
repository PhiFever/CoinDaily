package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateStoreFirstRunAndAtomicReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".coindaily-state.json")
	store := NewStateStore(path)
	state, err := store.Load()
	if err != nil || len(state.Perpetuals) != 0 {
		t.Fatalf("first load state=%#v err=%v", state, err)
	}

	firstTime := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	state.Perpetuals["xyz:ZHIPU"] = PerpetualSnapshot{Symbol: "xyz:ZHIPU", ObservedAt: firstTime, NotionalOpenInterest: 100}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	state.Perpetuals["xyz:ZHIPU"] = PerpetualSnapshot{Symbol: "xyz:ZHIPU", ObservedAt: firstTime.Add(time.Hour), NotionalOpenInterest: 120}
	if err := store.Save(state); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.Perpetuals["xyz:ZHIPU"].NotionalOpenInterest != 120 {
		t.Fatalf("loaded=%#v err=%v", loaded, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".coindaily-state-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary state files remain: %v, err=%v", matches, err)
	}
}

func TestStateStoreCorruptAndSymbolMismatchDegradeGracefully(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".coindaily-state.json")
	if err := os.WriteFile(path, []byte(`{broken`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStateStore(path)
	state, err := store.Load()
	if err == nil || len(state.Perpetuals) != 0 {
		t.Fatalf("corrupt state=%#v err=%v", state, err)
	}

	if err := os.WriteFile(path, []byte(`{"version":1,"perpetuals":{"xyz:ZHIPU":{"symbol":"xyz:OTHER","observed_at":"2026-08-25T10:00:00Z","notional_open_interest":100}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err = store.Load()
	if err != nil || len(state.Perpetuals) != 0 {
		t.Fatalf("symbol mismatch should be ignored: state=%#v err=%v", state, err)
	}
}
