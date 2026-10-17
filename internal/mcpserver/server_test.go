package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/josepalala-dev/gitgud/internal/cli"
)

// fakeRunner records the argv it was asked to run and returns a canned result.
type fakeRunner struct {
	dir  string
	args []string
	res  Result
	err  error
	n    int
}

func (f *fakeRunner) run(_ context.Context, dir string, args []string) (Result, error) {
	f.n++
	f.dir = dir
	f.args = args
	return f.res, f.err
}

func request(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func call(t *testing.T, run Runner, build argsFunc, raw bool, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := handler(run, build, raw)(context.Background(), request(args))
	if err != nil {
		t.Fatalf("handler returned a Go error: %v", err)
	}
	return res
}

func resultText(r *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestNewArgvPerType(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want []string
	}{
		{"feature", map[string]any{"type": "feature", "description": "login form"},
			[]string{"new", "feature", "login form"}},
		{"fix with parent", map[string]any{"type": "fix", "description": "typo", "parent": "epic/auth"},
			[]string{"new", "fix", "typo", "--parent", "epic/auth"}},
		{"release", map[string]any{"type": "release", "version": "1.4.0"},
			[]string{"new", "release", "1.4.0"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := &fakeRunner{res: Result{Code: 0}}
			call(t, fake.run, newArgs, false, c.args)
			if !reflect.DeepEqual(fake.args, c.want) {
				t.Errorf("argv = %q; want %q", fake.args, c.want)
			}
		})
	}
}

// Missing or unsafe inputs must be refused before gitgud is started, so a
// prompt can never be reached and a value can never become a flag.
func TestNewRefusesBadInput(t *testing.T) {
	cases := []map[string]any{
		{},
		{"type": "hotfix", "description": "x"},
		{"type": "feature"},
		{"type": "feature", "description": "   "},
		{"type": "feature", "description": "--delete"},
		{"type": "fix", "description": "x", "parent": "--force"},
		{"type": "release"},
		{"type": "release", "version": "-1.0.0"},
	}
	for _, c := range cases {
		fake := &fakeRunner{}
		res := call(t, fake.run, newArgs, false, c)
		if fake.n != 0 {
			t.Errorf("input %v reached gitgud with argv %q", c, fake.args)
		}
		if !res.IsError {
			t.Errorf("input %v was not refused", c)
		}
	}
}

func TestFinishDeleteFlag(t *testing.T) {
	fake := &fakeRunner{}
	call(t, fake.run, finishArgs, false, map[string]any{"delete": true})
	if !reflect.DeepEqual(fake.args, []string{"finish", "--delete"}) {
		t.Errorf("argv = %q", fake.args)
	}
	call(t, fake.run, finishArgs, false, map[string]any{})
	if !reflect.DeepEqual(fake.args, []string{"finish"}) {
		t.Errorf("argv without delete = %q", fake.args)
	}
}

func TestLogArgvAndLimit(t *testing.T) {
	fake := &fakeRunner{}
	call(t, fake.run, logArgs, true, map[string]any{"branch": "fix/typo", "limit": float64(2)})
	want := []string{"log", "--branch", "fix/typo", "--limit", "2"}
	if !reflect.DeepEqual(fake.args, want) {
		t.Errorf("argv = %q; want %q", fake.args, want)
	}
}

func TestRelativeRepoRefused(t *testing.T) {
	fake := &fakeRunner{}
	res := call(t, fake.run, initArgs, false, map[string]any{"repo": "relative/dir"})
	if !res.IsError || fake.n != 0 {
		t.Errorf("relative repo was not refused (calls=%d)", fake.n)
	}
}

func TestRepoPassedAsWorkingDirectory(t *testing.T) {
	fake := &fakeRunner{}
	call(t, fake.run, recordArgs, false, map[string]any{"repo": "/abs/project"})
	if fake.dir != "/abs/project" {
		t.Errorf("dir = %q", fake.dir)
	}
}

// A conflict exits 2. The caller must see the exit code and the error flag.
func TestNonZeroExitIsToolError(t *testing.T) {
	fake := &fakeRunner{res: Result{Code: 2, Stderr: "gitgud: merge stopped with conflicts"}}
	res := call(t, fake.run, finishArgs, false, map[string]any{})
	text := resultText(res)
	if !res.IsError || !strings.Contains(text, "exit: 2") || !strings.Contains(text, "conflicts") {
		t.Errorf("conflict result = error:%v text:%q", res.IsError, text)
	}
}

func TestLogSuccessReturnsRawJSONL(t *testing.T) {
	line := `{"seq":1,"event":"branch.created"}`
	fake := &fakeRunner{res: Result{Code: 0, Stdout: line + "\n"}}
	res := call(t, fake.run, logArgs, true, map[string]any{"branch": "x"})
	if res.IsError || resultText(res) != line+"\n" {
		t.Errorf("log text = %q; want raw JSONL", resultText(res))
	}
}

func TestRunnerFailureIsToolError(t *testing.T) {
	fake := &fakeRunner{err: errors.New("exec: gitgud: not found")}
	res := call(t, fake.run, initArgs, false, map[string]any{})
	if !res.IsError || !strings.Contains(resultText(res), "could not run gitgud") {
		t.Errorf("runner failure = %q", resultText(res))
	}
}

// End to end through the real CLI, in a temp repo. Nothing touches the real
// repository.
func TestEndToEndThroughCLI(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "chore: init")

	inProcess := func(_ context.Context, d string, args []string) (Result, error) {
		var out, errb bytes.Buffer
		code := cli.Run(d, args, strings.NewReader(""), &out, &errb)
		return Result{Code: code, Stdout: out.String(), Stderr: errb.String()}, nil
	}

	if r := call(t, inProcess, initArgs, false, map[string]any{"repo": dir}); r.IsError {
		t.Fatalf("init: %s", resultText(r))
	}
	r := call(t, inProcess, newArgs, false, map[string]any{"repo": dir, "type": "fix", "description": "readline from user"})
	if r.IsError || !strings.Contains(resultText(r), "created fix/readline-from-user from develop") {
		t.Fatalf("new fix: %s", resultText(r))
	}
	if got := gitIn(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "fix/readline-from-user" {
		t.Errorf("checked-out branch = %s", got)
	}
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "fix: readline prompt")
	if r := call(t, inProcess, recordArgs, false, map[string]any{"repo": dir}); r.IsError {
		t.Fatalf("record: %s", resultText(r))
	}
	r = call(t, inProcess, finishArgs, false, map[string]any{"repo": dir, "delete": true})
	if r.IsError {
		t.Fatalf("finish: %s", resultText(r))
	}
	logRes := call(t, inProcess, logArgs, true, map[string]any{"repo": dir, "branch": "fix/readline-from-user"})
	if logRes.IsError || !strings.Contains(resultText(logRes), `"event":"branch.finished"`) {
		t.Errorf("log after finish: %s", resultText(logRes))
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
