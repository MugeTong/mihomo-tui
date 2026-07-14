package local

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const shellEnvironment = `# Managed by Mihomo TUI.
mhmt() {
    local bin="$HOME/.local/bin/mhmt"
    case "${1-}" in
        on|off)
            local proxy_commands
            proxy_commands="$("$bin" "$1")" || return
            eval "$proxy_commands"
            case "$1" in
                on) echo "Proxy enabled." ;;
                off) echo "Proxy disabled." ;;
            esac
            ;;
        *)
            "$bin" "$@"
            ;;
    esac
}

_mhmt_proxy_commands="$("$HOME/.local/bin/mhmt" on 2>/dev/null)" && eval "$_mhmt_proxy_commands"
unset _mhmt_proxy_commands
`

func installShellIntegration(layout Layout) error {
	envPath := filepath.Join(layout.DataDir, "env")
	if err := os.WriteFile(envPath, []byte(shellEnvironment), 0o644); err != nil {
		return fmt.Errorf("write shell integration: %w", err)
	}
	return addEnvToShellRC(envPath)
}

func addEnvToShellRC(envPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locate user home directory: %w", err)
	}
	sourceLine := `. "` + envPath + `"`
	for _, name := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, name)
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if hasLine(string(data), sourceLine) {
			continue
		}
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		_, writeErr := fmt.Fprintf(file, "\n%s\n", sourceLine)
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("update %s: %w", path, writeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %s: %w", path, closeErr)
		}
	}
	return nil
}

func hasLine(content, target string) bool {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}
