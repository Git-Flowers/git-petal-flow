package cli

import (
	"fmt"

	"github.com/josepalala-dev/gitgud/internal/events"
	"github.com/josepalala-dev/gitgud/internal/state"
)

// record journals commits made with plain git on the current branch. Running
// it again journals nothing new.
func (a *app) record() error {
	branch, err := a.repo.CurrentBranch()
	if err != nil {
		return err
	}
	st, err := state.Load(a.statePath)
	if err != nil {
		return err
	}
	if _, ok := st.Branches[branch]; !ok {
		return fmt.Errorf("branch %s is not recorded; create it with gitgud new", branch)
	}
	opID, err := events.NewOpID(a.now())
	if err != nil {
		return err
	}
	n, err := a.capture(branch, opID)
	if err != nil {
		return err
	}
	a.say("recorded %d commit(s) on %s", n, branch)
	return nil
}
