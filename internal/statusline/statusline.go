// Package statusline renders the one-line gitgud status shown in Claude Code's
// status bar. It only reads: it never changes the repository or the journal.
package statusline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/josepalala-dev/gitgud/internal/events"
	"github.com/josepalala-dev/gitgud/internal/git"
	"github.com/josepalala-dev/gitgud/internal/state"
	"github.com/josepalala-dev/gitgud/internal/workflow"
)

// input is the part of the JSON Claude Code sends on stdin that we use.
type input struct {
	CWD       string `json:"cwd"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
}

// DirFromInput returns the working directory Claude Code reported, preferring
// workspace.current_dir. It returns "" when the JSON has neither field.
func DirFromInput(data []byte) string {
	var in input
	if json.Unmarshal(data, &in) != nil {
		return ""
	}
	if in.Workspace.CurrentDir != "" {
		return in.Workspace.CurrentDir
	}
	return in.CWD
}

// Render returns the status line for the repository containing dir. It returns
// "" outside a git repository, so the bar stays empty in other projects.
func Render(dir string) string {
	repo, err := git.Open(dir)
	if err != nil {
		return ""
	}
	common, err := repo.CommonDir()
	if err != nil {
		return ""
	}
	stateDir := filepath.Join(common, "gitgud")
	branch, err := repo.CurrentBranch()
	if err != nil {
		return dim("gitgud · detached HEAD")
	}
	st, err := state.Load(filepath.Join(stateDir, "state.json"))
	if err != nil {
		return ""
	}
	// A conflicted merge leaves HEAD on the target branch, so check for it
	// before the current branch's own state.
	if repo.MergeInProgress() {
		return yellow(fmt.Sprintf("gitgud ⚠ merge in progress into %s · %s · resolve, commit, then gitgud record",
			branch, conflictedBranch(st, stateDir)))
	}
	b, ok := st.Branches[branch]
	if !ok {
		return dim(fmt.Sprintf("gitgud · %s · not a gitgud branch", branch))
	}

	journal := events.Path(stateDir, branch)
	if b.Status == "finished" {
		return dim(fmt.Sprintf("gitgud ✓ %s finished into %s", branch, targets(b.Type, b.Parent)))
	}

	recorded, _ := events.Commits(journal)
	toRecord := unrecorded(repo, b.BaseSHA, recorded)
	line := fmt.Sprintf("gitgud ● %s · from %s · %d recorded · %d to record · last: %s",
		branch, b.Parent, len(recorded), toRecord, lastEvent(journal))
	if b.Type == string(workflow.Epic) {
		// An epic cannot finish while a child is open, so say so before the user tries.
		if open := state.OpenChildren(st, branch); len(open) > 0 {
			return yellow(fmt.Sprintf("%s · ⚠ %d open %s, finish them before the epic",
				line, len(open), plural(len(open), "child", "children")))
		}
	}
	return green(line)
}

// plural picks the singular or plural noun for n.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// conflictedBranch names the open branch whose journal ends in merge.conflict,
// which is the branch being merged. It returns "unknown branch" if none matches.
func conflictedBranch(st state.State, stateDir string) string {
	for name, b := range st.Branches {
		if b.Status == "open" && lastEvent(events.Path(stateDir, name)) == "merge.conflict" {
			return name
		}
	}
	return "unknown branch"
}

// targets names where a finished branch went. A release goes to main and then
// develop; everything else goes to its recorded parent.
func targets(kind, parent string) string {
	if kind == string(workflow.Release) {
		return workflow.Main + " then " + workflow.Develop
	}
	return parent
}

// unrecorded counts first-parent commits on the branch that are not journaled.
func unrecorded(repo git.Repo, base string, recorded map[string]bool) int {
	commits, err := repo.Commits(base, "HEAD")
	if err != nil {
		return 0
	}
	n := 0
	for _, c := range commits {
		if !recorded[c.SHA] {
			n++
		}
	}
	return n
}

// lastEvent returns the event name of the journal's final record, or "none".
func lastEvent(path string) string {
	lines, err := events.ReadLines(path)
	if err != nil || len(lines) == 0 {
		return "none"
	}
	var r events.Record
	if json.Unmarshal([]byte(lines[len(lines)-1]), &r) != nil || r.Event == "" {
		return "unreadable"
	}
	return r.Event
}

// colour wraps s in an ANSI code, unless NO_COLOR is set (no-color.org).
func colour(code, s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func green(s string) string  { return colour("32", s) }
func yellow(s string) string { return colour("33", s) }
func dim(s string) string    { return colour("2", strings.TrimSpace(s)) }
