package machine

import (
	"fmt"
	"path"
)

// shellCwdHookScript 写入远端 ~/.flashdock_cwd.sh，由 bashrc/zshrc source。
const shellCwdHookScript = `# FlashDock: report cwd via OSC 777 + sidecar file
__fd_cwd() {
  printf '\033]777;cwd;%s\007' "$PWD"
  [ -n "$HOME" ] && printf '%s' "$PWD" > "$HOME/.flashdock_pwd" 2>/dev/null
}
if [ -n "$ZSH_VERSION" ]; then
  precmd_functions+=(__fd_cwd)
  cd() { builtin cd "$@" && __fd_cwd; }
elif [ -n "$BASH_VERSION" ]; then
  case ":${PROMPT_COMMAND}:" in
    *":__fd_cwd:"*) ;;
    *) PROMPT_COMMAND="__fd_cwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
  esac
  cd() { builtin cd "$@" && __fd_cwd; }
fi
__fd_cwd
`

const (
	shellCwdHookFilename = ".flashdock_cwd.sh"
	shellCwdPwdFilename  = ".flashdock_pwd"
	shellRcMarker        = "flashdock-cwd-hook"
)

// UninstallShellCwdHook 清理历史版本写入远端的 cwd hook（可选，恢复 FinalShell 等工具）。
func UninstallShellCwdHook(aux *ShellAuxManager) error {
	script := `set +e
for f in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$HOME/.zshrc"; do
  [ -f "$f" ] && sed -i '/flashdock-cwd-hook/d' "$f"
done
rm -f "$HOME/.flashdock_cwd.sh" "$HOME/.flashdock_pwd"
`
	_, err := aux.ExecBash(script)
	return err
}

// InstallShellCwdHook 经辅助 SSH 静默写入 hook（不经 PTY，终端无回显），并在 rc 追加 source。
func InstallShellCwdHook(aux *ShellAuxManager) error {
	home, err := aux.Home()
	if err != nil || home == "" {
		home = "/tmp"
	}
	hookPath := path.Join(home, shellCwdHookFilename)
	pwdPath := path.Join(home, shellCwdPwdFilename)
	rcLine := `[ -f "$HOME/.flashdock_cwd.sh" ] && . "$HOME/.flashdock_cwd.sh" # flashdock-cwd-hook`
	script := fmt.Sprintf(`set +e
HOOK=%q
PWD_FILE=%q
MARKER=%q
RC_LINE=%q
installed=0
for rc in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$HOME/.zshrc"; do
  if [ -f "$rc" ] && grep -q "$MARKER" "$rc" 2>/dev/null; then
    installed=1
    break
  fi
done
if [ "$installed" = 1 ] && [ -f "$HOOK" ]; then
  exit 0
fi
cat > "$HOOK" << 'ENDOFFD'
%s
ENDOFFD
chmod 600 "$HOOK"
: > "$PWD_FILE" 2>/dev/null || true
for rc in "$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.profile" "$HOME/.zshrc"; do
  if [ -f "$rc" ] && ! grep -q "$MARKER" "$rc" 2>/dev/null; then
    printf '\n%%s\n' "$RC_LINE" >> "$rc"
  fi
done
if [ ! -f "$HOME/.bashrc" ] && [ -z "$ZSH_VERSION" ]; then
  printf '%%s\n' "$RC_LINE" > "$HOME/.bashrc"
  chmod 600 "$HOME/.bashrc" 2>/dev/null || true
fi
`, hookPath, pwdPath, shellRcMarker, rcLine, shellCwdHookScript)
	_, err = aux.ExecBash(script)
	return err
}
