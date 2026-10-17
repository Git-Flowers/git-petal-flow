package cli

import (
	"errors"
	"fmt"
)

// status prints the short git status of the working tree. It is a thin wrapper
// over `git status -s`, so the output is exactly git's.
func (a *app) status(args []string) error {
	if len(args) > 0 {
		return errors.New("usage: gitgud status")
	}
	out, err := a.repo.Run("status", "-s")
	if err != nil {
		return err
	}
	if out != "" {
		fmt.Fprintln(a.out, out)
	}
	return nil
}
