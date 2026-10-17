package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Record is one journal line. Fields not relevant to an event are omitted so
// jq can read every line without a schema.
type Record struct {
	Seq     int      `json:"seq"`
	TS      string   `json:"ts"`
	OpID    string   `json:"op_id"`
	Event   string   `json:"event"`
	Branch  string   `json:"branch"`
	Commit  string   `json:"commit,omitempty"`
	Parents []string `json:"parents,omitempty"`
	Subject string   `json:"subject,omitempty"`
	Type    *string  `json:"type"`
	Source  string   `json:"source,omitempty"`
	Target  string   `json:"target,omitempty"`
	Tag     string   `json:"tag,omitempty"`
	Files   []string `json:"files,omitempty"`
}

// Path returns the journal file for a branch under stateDir. A "/" in the
// branch name becomes "__", so feature/login maps to feature__login.jsonl.
func Path(stateDir, branch string) string {
	name := strings.ReplaceAll(branch, "/", "__") + ".jsonl"
	return filepath.Join(stateDir, "journal", name)
}

// ReadLines returns the non-empty lines of a journal. A missing file returns
// an error wrapping fs.ErrNotExist.
func ReadLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, sc.Err()
}

// Append assigns seq and ts to r and writes it as one line at the end of the
// journal. The file is never truncated or rewritten.
func Append(path string, r *Record, now time.Time) error {
	lines, err := ReadLines(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	r.Seq = len(lines) + 1
	r.TS = now.UTC().Format(time.RFC3339)
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Commits returns the set of commit SHAs already journaled in path. A missing
// journal yields an empty set.
func Commits(path string) (map[string]bool, error) {
	lines, err := ReadLines(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, line := range lines {
		var r Record
		if json.Unmarshal([]byte(line), &r) == nil && r.Commit != "" {
			seen[r.Commit] = true
		}
	}
	return seen, nil
}
