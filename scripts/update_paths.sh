#!/usr/bin/env bash
# Put ~/.local/bin on PATH so a gitgud installed there is found. Linux, WSL.
#
#   scripts/update_paths.sh          add ~/.local/bin to your shell startup file
#   scripts/update_paths.sh check    report whether it is on PATH, change nothing;
#                                    exits 1 if it is not on PATH or gitgud is missing
#
# It edits ~/.bashrc (and ~/.zshrc if you use zsh), once. A marker comment makes
# a second run a no-op. Nothing outside your home directory is touched.
#
# Environment:
#   HOME   where ~/.local/bin and the startup files live

set -euo pipefail

bin="$HOME/.local/bin"
marker="# added by gitgud scripts/update_paths.sh"

log() { printf '%s\n' "$*"; }
die() { printf 'update_paths.sh: %s\n' "$*" >&2; exit 1; }

on_path() {
	case ":$PATH:" in
	*":$bin:"*) return 0 ;;
	*) return 1 ;;
	esac
}

# rc_files lists the startup files to edit: bashrc always, zshrc when zsh is in use.
rc_files() {
	echo "$HOME/.bashrc"
	if [[ "${SHELL:-}" == */zsh || -f "$HOME/.zshrc" ]]; then
		echo "$HOME/.zshrc"
	fi
}

do_check() {
	local ok=0
	if on_path; then
		log "$bin is on PATH"
	else
		log "$bin is not on PATH in this shell"
		ok=1
	fi
	if command -v gitgud >/dev/null 2>&1; then
		log "gitgud: $(command -v gitgud)"
	else
		log "gitgud not found on PATH"
		ok=1
	fi
	return "$ok"
}

do_update() {
	mkdir -p "$bin"
	local rc
	while IFS= read -r rc; do
		if [[ -f "$rc" ]] && grep -qF "$marker" "$rc"; then
			log "already set in $rc"
			continue
		fi
		printf '\n%s\nexport PATH="$HOME/.local/bin:$PATH"\n' "$marker" >>"$rc"
		log "added ~/.local/bin to PATH in $rc"
	done < <(rc_files)

	if on_path; then
		log "$bin is already on PATH in this shell"
	else
		log "open a new terminal, or run: export PATH=\"\$HOME/.local/bin:\$PATH\""
	fi
	if ! [[ -x "$bin/gitgud" ]]; then
		log "note: $bin/gitgud not found yet; see the Install section of the README"
	fi
}

case "${1:-}" in
"") do_update ;;
check) do_check ;;
*) die "usage: $0 [check]" ;;
esac
