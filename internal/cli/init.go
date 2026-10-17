package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/josepalala-dev/gitgud/internal/state"
	"github.com/josepalala-dev/gitgud/internal/workflow"
)

// initRepo makes sure develop and main exist and that the gitgud directories
// are in place. Running it again changes nothing.
func (a *app) initRepo() error {
	var created []string
	if !a.repo.RefExists("refs/heads/" + workflow.Main) {
		if _, err := a.repo.RevParse("HEAD"); err != nil {
			return errors.New("repository has no commits yet; make an initial commit first")
		}
		if err := a.repo.CreateBranch(workflow.Main, "HEAD"); err != nil {
			return err
		}
		created = append(created, workflow.Main)
	}
	if !a.repo.RefExists("refs/heads/" + workflow.Develop) {
		if err := a.repo.CreateBranch(workflow.Develop, workflow.Main); err != nil {
			return err
		}
		created = append(created, workflow.Develop)
	}
	if err := os.MkdirAll(filepath.Join(a.stateDir, "journal"), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(a.statePath); errors.Is(err, fs.ErrNotExist) {
		if err := state.Save(a.statePath, state.Empty()); err != nil {
			return err
		}
	}
	if len(created) == 0 {
		a.say("already initialized")
	} else {
		a.say("created %s", strings.Join(created, ", "))
	}
	return nil
}
