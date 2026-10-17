// Package state stores which branches gitgud created, their type and parent.
// The journal, not this file, is the history record.
package state

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// SchemaVersion is written into state.json so readers can detect changes.
const SchemaVersion = "1.0"

// Branch is the metadata recorded for one gitgud branch.
type Branch struct {
	Type      string `json:"type"`
	Parent    string `json:"parent"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
	BaseSHA   string `json:"base_sha"`
	Status    string `json:"status"`
}

// State is the whole state.json document.
type State struct {
	SchemaVersion string            `json:"schema_version"`
	Branches      map[string]Branch `json:"branches"`
}

// Empty returns a state with no branches.
func Empty() State {
	return State{SchemaVersion: SchemaVersion, Branches: map[string]Branch{}}
}

// Load reads state from path. A missing file is an empty state.
func Load(path string) (State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Empty(), nil
	}
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	if s.Branches == nil {
		s.Branches = map[string]Branch{}
	}
	return s, nil
}

// Save writes state to path atomically: a temp file in the same directory is
// written and then renamed over the target.
func Save(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// OpenChildren lists the branches whose parent is parent and whose status is
// still "open", sorted by name. It is how an epic learns which features and
// fixes under it are not finished yet.
func OpenChildren(s State, parent string) []string {
	var open []string
	for name, b := range s.Branches {
		if b.Parent == parent && b.Status == "open" {
			open = append(open, name)
		}
	}
	sort.Strings(open)
	return open
}
