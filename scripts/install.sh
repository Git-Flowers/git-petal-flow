#!/usr/bin/env bash
# Install gitgud, its MCP server and the Claude skill on this machine.
#
#   scripts/install.sh install     build and install both binaries, link the skill
#   scripts/install.sh register    register gitgud-mcp with Claude Code (user scope)
#   scripts/install.sh all         install, then register
#   scripts/install.sh uninstall   remove the skill link, binaries and registration
#   scripts/install.sh statusline  set the Claude Code status bar to gitgud statusline
#   scripts/install.sh check       run the CI gate: gofmt, vet, test, build
#
# statusline backs up settings.json first, and will not replace a status line
# that is not gitgud's unless FORCE=1.
#
# Environment:
#   CLAUDE_HOME   Claude config directory (default: $HOME/.claude)
#   GOBIN         where go install puts binaries (default: go env GOBIN, else GOPATH/bin)
#
# Nothing here pushes, tags, or touches a remote. `register` and `uninstall`
# change Claude Code's user configuration, so they only run when asked.

set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
claude_home="${CLAUDE_HOME:-$HOME/.claude}"
skill_src="$repo/skills/gitgud"
skill_dst="$claude_home/skills/gitgud"

bin_dir() {
	if [[ -n "${GOBIN:-}" ]]; then
		echo "$GOBIN"
	elif [[ -n "$(go env GOBIN)" ]]; then
		go env GOBIN
	else
		echo "$(go env GOPATH)/bin"
	fi
}

log() { printf '%s\n' "$*"; }
die() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

require() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required but not on PATH"
}

do_install() {
	require go
	cd "$repo"
	log "building and installing gitgud and gitgud-mcp to $(bin_dir)"
	go install ./cmd/gitgud ./cmd/gitgud-mcp

	mkdir -p "$claude_home/skills"
	if [[ -L "$skill_dst" ]]; then
		ln -sfn "$skill_src" "$skill_dst"
	elif [[ -e "$skill_dst" ]]; then
		die "$skill_dst exists and is not a symlink; move it aside and re-run"
	else
		ln -s "$skill_src" "$skill_dst"
	fi
	log "linked skill: $skill_dst -> $skill_src"
	case ":$PATH:" in
	*":$(bin_dir):"*) ;;
	*) log "note: $(bin_dir) is not on your PATH; add it so 'gitgud' is found:"
	   log "  export PATH=\"$(bin_dir):\$PATH\"" ;;
	esac
	log "installed. Run '$0 register' to add the MCP server to Claude Code."
}

do_register() {
	require claude
	local bin
	bin="$(bin_dir)"
	[[ -x "$bin/gitgud-mcp" && -x "$bin/gitgud" ]] || die "binaries not found in $bin; run install first"

	# Remove any earlier registration so re-running does not fail or duplicate.
	claude mcp remove --scope user gitgud >/dev/null 2>&1 || true
	# The name must come before --env: --env takes several values, so placing it
	# first would consume the server name.
	claude mcp add --scope user gitgud --env "GITGUD_BIN=$bin/gitgud" -- "$bin/gitgud-mcp"
	log "registered gitgud MCP server (user scope) -> $bin/gitgud-mcp"
}

# statusline_cmd is the shell command Claude Code runs for the status bar. The
# binary path is quoted, so an install path with spaces still works.
statusline_cmd() {
	printf '%q statusline' "$(bin_dir)/gitgud"
}

# edit_settings merges statusLine into settings.json, keeping every other key.
# It backs the file up first and refuses to replace a status line that is not
# ours unless FORCE=1.
edit_settings() {
	local action="$1" settings="$claude_home/settings.json"
	require python3
	mkdir -p "$claude_home"
	if [[ -f "$settings" ]]; then
		cp "$settings" "$settings.bak-$(date +%Y%m%d%H%M%S)"
	fi
	python3 - "$settings" "$action" "$(statusline_cmd)" "${FORCE:-0}" <<'PY'
import json, os, sys
path, action, ours, force = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4]
data = {}
if os.path.exists(path):
    with open(path) as f:
        data = json.load(f)
current = data.get("statusLine")
mine = isinstance(current, dict) and str(current.get("command", "")).endswith("gitgud statusline")
if action == "set":
    if current and not mine and force != "1":
        sys.exit(f"settings.json already has a statusLine ({current.get('command')}); re-run with FORCE=1 to replace it")
    data["statusLine"] = {"type": "command", "command": ours}
elif action == "unset":
    if mine:
        del data["statusLine"]
    else:
        sys.exit(0)
with open(path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY
}

do_statusline() {
	[[ -x "$(bin_dir)/gitgud" ]] || die "gitgud not found in $(bin_dir); run install first"
	edit_settings set
	log "status line set in $claude_home/settings.json: $(statusline_cmd)"
}

do_uninstall() {
	if [[ -f "$claude_home/settings.json" ]]; then
		edit_settings unset && log "removed gitgud status line (if it was ours)"
	fi
	if [[ -L "$skill_dst" ]]; then
		rm "$skill_dst"
		log "removed skill link $skill_dst"
	fi
	local bin
	bin="$(bin_dir)"
	for b in gitgud gitgud-mcp; do
		if [[ -f "$bin/$b" ]]; then
			rm "$bin/$b"
			log "removed $bin/$b"
		fi
	done
	if command -v claude >/dev/null 2>&1; then
		claude mcp remove --scope user gitgud >/dev/null 2>&1 && log "removed MCP registration" || true
	fi
}

do_check() {
	require go
	cd "$repo"
	[[ -z "$(gofmt -l .)" ]] || die "gofmt reports unformatted files: $(gofmt -l .)"
	go vet ./...
	go test -count=1 ./...
	go build -o /dev/null ./cmd/gitgud
	go build -o /dev/null ./cmd/gitgud-mcp
	log "check passed"
}

case "${1:-}" in
	install)   do_install ;;
	register)  do_register ;;
	all)       do_install && do_register ;;
	statusline) do_statusline ;;
	uninstall) do_uninstall ;;
	check)     do_check ;;
	*)
		sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'
		exit 2
		;;
esac
