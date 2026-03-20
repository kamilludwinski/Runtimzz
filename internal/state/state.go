package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/kamilludwinski/runtimzzz/internal/logger"
	"github.com/kamilludwinski/runtimzzz/internal/meta"
)

func stateFilePath() string {
	return filepath.Join(meta.AppDir(), "state.json")
}

type State struct {
	mu sync.RWMutex `json:"-"`

	// active tells us which version is active
	// map of <runtime> : <version>
	// keep it hidden
	active map[string]string
}

func NewState() *State {
	return &State{
		active: make(map[string]string),
	}
}

func (s *State) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type state struct {
		Active map[string]string `json:"active"`
	}

	st := state{
		Active: s.active,
	}

	logger.Debug("saving state", "active", s.active)

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(stateFilePath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

func (s *State) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read state: %w", err)
	}

	var st struct {
		Active map[string]string `json:"active"`
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if st.Active != nil {
		s.active = st.Active
	}
	if s.active == nil {
		s.active = make(map[string]string)
	}

	logger.Debug("loaded state", "active", s.active)

	return nil
}

func (s *State) SetActive(runtime, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newActive := make(map[string]string, len(s.active))
	maps.Copy(newActive, s.active)

	newActive[runtime] = version

	type state struct {
		Active map[string]string `json:"active"`
	}

	st := state{
		Active: newActive,
	}

	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	tmp := stateFilePath() + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	if err := os.Rename(tmp, stateFilePath()); err != nil {
		return fmt.Errorf("failed to commit state: %w", err)
	}

	s.active = newActive
	logger.Debug("set active", "runtime", runtime, "version", version)

	return nil
}

func (s *State) IsActive(runtime string, version string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.active[runtime] == version
}

func (s *State) Active(runtime string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.active[runtime]
}

// ClearRuntime removes the runtime from the active map and persists state.
// Use after purging all versions for a runtime so the config entry is removed.
func (s *State) ClearRuntime(runtime string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	newActive := make(map[string]string, len(s.active))
	for k, v := range s.active {
		if k != runtime {
			newActive[k] = v
		}
	}

	type state struct {
		Active map[string]string `json:"active"`
	}
	st := state{Active: newActive}
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	tmp := stateFilePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}
	if err := os.Rename(tmp, stateFilePath()); err != nil {
		return fmt.Errorf("failed to commit state: %w", err)
	}

	s.active = newActive
	logger.Debug("clear runtime", "runtime", runtime)
	return nil
}
