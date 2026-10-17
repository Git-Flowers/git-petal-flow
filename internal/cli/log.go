package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"

	"github.com/josepalala-dev/gitgud/internal/events"
)

// log prints a branch journal as raw JSONL so it can be piped into jq. Lines
// are never reformatted; malformed lines are passed through and counted.
func (a *app) log(args []string) error {
	branch := ""
	limit := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--branch":
			if i+1 >= len(args) {
				return errors.New("--branch needs a name")
			}
			branch = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return errors.New("--limit needs a number")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 1 {
				return errors.New("--limit must be a positive number")
			}
			limit = n
			i++
		default:
			return fmt.Errorf("unknown argument %q", args[i])
		}
	}
	if branch == "" {
		current, err := a.repo.CurrentBranch()
		if err != nil {
			return err
		}
		branch = current
	}
	lines, err := events.ReadLines(events.Path(a.stateDir, branch))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no journal for %s", branch)
	}
	if err != nil {
		return err
	}
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	malformed := 0
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			malformed++
		}
		fmt.Fprintln(a.out, line)
	}
	if malformed > 0 {
		fmt.Fprintf(a.errw, "gitgud: warning: %d malformed line(s) passed through\n", malformed)
	}
	return nil
}
