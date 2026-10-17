// Package mcpserver exposes gitgud's commands as MCP tools. Each tool runs the
// gitgud CLI with an argument list, never a shell string, and reports the exit
// code so the caller can tell a refusal from a conflict.
package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Result is the outcome of one gitgud invocation.
type Result struct {
	Code   int
	Stdout string
	Stderr string
}

// Runner runs gitgud with args inside dir. An empty dir means the server's
// working directory. A non-nil error means gitgud could not be started; a
// non-zero exit is reported through Result.Code instead.
type Runner func(ctx context.Context, dir string, args []string) (Result, error)

// ExecRunner runs the gitgud binary named by GITGUD_BIN, or "gitgud" from PATH.
// Stdin is left empty, so a command that would prompt reads EOF and fails
// instead of hanging.
func ExecRunner(ctx context.Context, dir string, args []string) (Result, error) {
	bin := os.Getenv("GITGUD_BIN")
	if bin == "" {
		bin = "gitgud"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return res, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.Code = exitErr.ExitCode()
		return res, nil
	}
	return res, err
}

// New returns an MCP server with every gitgud tool registered. run performs
// the actual gitgud invocations; tests pass a fake or an in-process runner.
func New(version string, run Runner) *server.MCPServer {
	s := server.NewMCPServer("gitgud", version, server.WithToolCapabilities(false))

	s.AddTool(mcp.NewTool("gitgud_init",
		mcp.WithDescription("Create the develop and main branches if they are missing, and the gitgud state directory. Safe to run more than once."),
		repoOption(),
	), handler(run, initArgs, false))

	s.AddTool(mcp.NewTool("gitgud_new",
		mcp.WithDescription("Start a branch and check it out. feature, fix and epic branch from develop (or --parent). release creates release/<version> from develop. Refuses when the working tree is dirty. Use this instead of git checkout -b so the parent is recorded."),
		repoOption(),
		mcp.WithString("type", mcp.Required(), mcp.Enum("feature", "fix", "epic", "release"),
			mcp.Description("Kind of branch to create.")),
		mcp.WithString("description", mcp.Description("What the branch is for. It becomes the branch slug: 'readline from user' gives fix/readline-from-user. Required for feature, fix and epic.")),
		mcp.WithString("version", mcp.Description("Release version as MAJOR.MINOR.PATCH, for example 1.4.0. Required for release.")),
		mcp.WithString("parent", mcp.Description("Optional parent branch. develop, or an open epic/<name> for feature and fix.")),
	), handler(run, newArgs, false))

	s.AddTool(mcp.NewTool("gitgud_record",
		mcp.WithDescription("Journal commits made with plain git on the current branch. Run it after committing with git. Running it again records nothing new."),
		repoOption(),
	), handler(run, recordArgs, false))

	s.AddTool(mcp.NewTool("gitgud_finish",
		mcp.WithDescription("Merge the current feature, fix or release branch into its recorded parent with --no-ff. Releases are tagged vX.Y.Z on main and merged back to develop. On a merge conflict the merge is left in progress and the result is an error; resolve the files and commit on the target branch, then check out the branch and call gitgud_finish again. Epics cannot be finished yet."),
		repoOption(),
		mcp.WithBoolean("delete", mcp.Description("Delete the branch after a successful finish. Defaults to false.")),
		mcp.WithDestructiveHintAnnotation(true),
	), handler(run, finishArgs, false))

	s.AddTool(mcp.NewTool("gitgud_log",
		mcp.WithDescription("Read a branch's journal. Returns one JSON object per line, oldest first: branch.created, commit.recorded (with commit SHA, parents, subject and Conventional Commit type), merged, merge.conflict, tag.created, branch.finished."),
		repoOption(),
		mcp.WithString("branch", mcp.Description("Branch whose journal to read. Defaults to the current branch.")),
		mcp.WithNumber("limit", mcp.Description("Return only the last N records.")),
		mcp.WithReadOnlyHintAnnotation(true),
	), handler(run, logArgs, true))

	return s
}

func repoOption() mcp.ToolOption {
	return mcp.WithString("repo", mcp.Description("Absolute path to the git repository. Defaults to the server's working directory."))
}

// argsFunc turns tool arguments into a gitgud argv. It never returns a shell
// string.
type argsFunc func(req mcp.CallToolRequest) ([]string, error)

// handler wraps an argsFunc into a tool handler. raw means a successful run
// returns stdout unchanged, which the log tool needs for JSONL.
func handler(run Runner, build argsFunc, raw bool) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo := req.GetString("repo", "")
		if repo != "" && !filepath.IsAbs(repo) {
			return mcp.NewToolResultError("repo must be an absolute path"), nil
		}
		args, err := build(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := run(ctx, repo, args)
		if err != nil {
			return mcp.NewToolResultError("could not run gitgud: " + err.Error()), nil
		}
		if raw && res.Code == 0 {
			return mcp.NewToolResultText(res.Stdout), nil
		}
		return formatResult(res), nil
	}
}

func formatResult(res Result) *mcp.CallToolResult {
	var b strings.Builder
	fmt.Fprintf(&b, "exit: %d\n", res.Code)
	if out := strings.TrimRight(res.Stdout, "\n"); out != "" {
		b.WriteString(out + "\n")
	}
	if errOut := strings.TrimRight(res.Stderr, "\n"); errOut != "" {
		b.WriteString(errOut + "\n")
	}
	text := strings.TrimRight(b.String(), "\n")
	if res.Code != 0 {
		return mcp.NewToolResultError(text)
	}
	return mcp.NewToolResultText(text)
}

func initArgs(mcp.CallToolRequest) ([]string, error) {
	return []string{"init"}, nil
}

func recordArgs(mcp.CallToolRequest) ([]string, error) {
	return []string{"record"}, nil
}

func finishArgs(req mcp.CallToolRequest) ([]string, error) {
	args := []string{"finish"}
	if req.GetBool("delete", false) {
		args = append(args, "--delete")
	}
	return args, nil
}

func newArgs(req mcp.CallToolRequest) ([]string, error) {
	kind := req.GetString("type", "")
	var args []string
	switch kind {
	case "release":
		version := req.GetString("version", "")
		if version == "" {
			return nil, errors.New("version is required for release, as MAJOR.MINOR.PATCH")
		}
		if err := safeValue("version", version); err != nil {
			return nil, err
		}
		args = []string{"new", "release", version}
	case "feature", "fix", "epic":
		desc := strings.TrimSpace(req.GetString("description", ""))
		if desc == "" {
			return nil, fmt.Errorf("description is required for %s", kind)
		}
		if err := safeValue("description", desc); err != nil {
			return nil, err
		}
		args = []string{"new", kind, desc}
	case "":
		return nil, errors.New("type is required: feature, fix, epic or release")
	default:
		return nil, fmt.Errorf("unknown type %q: want feature, fix, epic or release", kind)
	}
	if parent := req.GetString("parent", ""); parent != "" {
		if err := safeValue("parent", parent); err != nil {
			return nil, err
		}
		args = append(args, "--parent", parent)
	}
	return args, nil
}

func logArgs(req mcp.CallToolRequest) ([]string, error) {
	args := []string{"log"}
	if branch := req.GetString("branch", ""); branch != "" {
		if err := safeValue("branch", branch); err != nil {
			return nil, err
		}
		args = append(args, "--branch", branch)
	}
	if limit := req.GetInt("limit", 0); limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	return args, nil
}

// safeValue rejects values that gitgud would read as a flag, so a value cannot
// change what the command does.
func safeValue(name, v string) error {
	if strings.HasPrefix(v, "-") {
		return fmt.Errorf("%s must not start with '-'", name)
	}
	return nil
}
