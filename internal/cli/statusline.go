package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/josepalala-dev/gitgud/internal/statusline"
)

// maxStatusInput caps how much of stdin is read. Claude Code's payload is small.
const maxStatusInput = 1 << 20

// statusLine prints the gitgud status for Claude Code's status bar. It never
// fails: outside a gitgud repo it prints nothing, and any problem reading stdin
// falls back to the directory it was started in. The exit code is always 0 so
// a status-bar hiccup cannot show as an error.
func statusLine(dir string, in io.Reader, out io.Writer) int {
	target := dir
	if data := readPiped(in); len(data) > 0 {
		if d := statusline.DirFromInput(data); d != "" {
			target = d
		}
	}
	if line := statusline.Render(target); line != "" {
		fmt.Fprintln(out, line)
	}
	return exitOK
}

// statusStdinWait bounds how long we wait for stdin to reach EOF. Claude Code
// writes its JSON and closes stdin at once. A shell that leaves stdin open
// would otherwise block the command forever.
const statusStdinWait = 500 * time.Millisecond

// readPiped reads stdin only when it is a pipe or file. On a terminal it returns
// nil, so running `gitgud statusline` by hand does not block. If stdin stays
// open past statusStdinWait, it returns nil and the caller uses the working
// directory.
func readPiped(in io.Reader) []byte {
	if f, ok := in.(*os.File); ok {
		info, err := f.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice != 0 {
			return nil
		}
	}
	done := make(chan []byte, 1)
	go func() {
		data, err := io.ReadAll(io.LimitReader(in, maxStatusInput))
		if err != nil {
			data = nil
		}
		done <- data
	}()
	select {
	case data := <-done:
		return data
	case <-time.After(statusStdinWait):
		return nil
	}
}
