package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/shyim/agm/internal/aliases"
	"github.com/shyim/agm/internal/api"
	"github.com/shyim/agm/internal/db"
	"github.com/shyim/agm/internal/process"
	"github.com/shyim/agm/internal/target"
)

var runCmd = &cobra.Command{
	Use:                "run <account|alias> [agy-flags...]",
	Aliases:            []string{"exec"},
	Short:              "Launch an isolated Antigravity CLI (agy) session for an account",
	DisableFlagParsing: true,
	ValidArgsFunction:  completeAccountsAndAliases,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: agm <account|alias> [agy-flags...] or agm run <account|alias> [agy-flags...]")
		}
		targetArg := args[0]
		agyArgs := args[1:]
		return RunIsolatedSession(targetArg, agyArgs)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)
}

// RunIsolatedSession launches agy with an isolated profile and keyring for the given account.
func RunIsolatedSession(targetArg string, agyArgs []string) error {
	resolved := aliases.Resolve(targetArg)
	acc, err := db.FindAccount(resolved)
	if err != nil {
		return fmt.Errorf("account matching %q not found: %w", targetArg, err)
	}

	if acc.Token == nil {
		return fmt.Errorf("could not decrypt token for %s", acc.Email)
	}

	if db.IsTokenExpired(acc.Token) {
		fmt.Fprintf(os.Stderr, "Token expired for %s; refreshing…\n", acc.Email)
		if _, _, err := api.ValidateAccount(acc); err != nil {
			return fmt.Errorf("token refresh: %w", err)
		}
		acc, err = db.FindAccount(acc.Email)
		if err != nil {
			return err
		}
	}

	agyExe := process.FindAgyExecutable()
	if agyExe == "" {
		return fmt.Errorf("Antigravity CLI (agy) executable not found on PATH")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Use alias name or sanitized email as profile folder name
	profileName := targetArg
	safeName := strings.ReplaceAll(strings.ReplaceAll(profileName, "@", "_at_"), "/", "_")
	profileDir := filepath.Join(home, ".agy-profiles", safeName)

	runtimeDir := filepath.Join(profileDir, ".runtime")
	keyringsDir := filepath.Join(profileDir, ".local", "share", "keyrings")
	geminiCliDir := filepath.Join(profileDir, ".gemini", "antigravity-cli")

	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		return fmt.Errorf("create runtime dir: %w", err)
	}
	if err := os.MkdirAll(keyringsDir, 0o700); err != nil {
		return fmt.Errorf("create keyrings dir: %w", err)
	}
	if err := os.MkdirAll(geminiCliDir, 0o700); err != nil {
		return fmt.Errorf("create gemini cli dir: %w", err)
	}

	// Symlink .antigravity-agent so agm commands inside profile find the master DB
	_ = symlinkIfMissing(filepath.Join(home, ".antigravity-agent"), filepath.Join(profileDir, ".antigravity-agent"))

	// Symlink developer essentials (.gitconfig, .ssh, .bashrc, .profile) if present
	_ = symlinkIfMissing(filepath.Join(home, ".gitconfig"), filepath.Join(profileDir, ".gitconfig"))
	_ = symlinkIfMissing(filepath.Join(home, ".ssh"), filepath.Join(profileDir, ".ssh"))
	_ = symlinkIfMissing(filepath.Join(home, ".bashrc"), filepath.Join(profileDir, ".bashrc"))
	_ = symlinkIfMissing(filepath.Join(home, ".profile"), filepath.Join(profileDir, ".profile"))

	// Symlink global gemini config (customizations, rules, skills)
	_ = symlinkIfMissing(filepath.Join(home, ".gemini", "config"), filepath.Join(profileDir, ".gemini", "config"))

	// Symlink builtin skills and helper bin
	_ = symlinkIfMissing(filepath.Join(home, ".gemini", "antigravity-cli", "builtin"), filepath.Join(geminiCliDir, "builtin"))
	_ = symlinkIfMissing(filepath.Join(home, ".gemini", "antigravity-cli", "bin"), filepath.Join(geminiCliDir, "bin"))

	// Copy baseline settings if not present in profile
	baselineSettings := filepath.Join(home, ".gemini", "antigravity-cli", "settings.json")
	targetSettings := filepath.Join(geminiCliDir, "settings.json")
	if fileExists(baselineSettings) && !fileExists(targetSettings) {
		_ = copyFile(baselineSettings, targetSettings)
	}

	baselineJetski := filepath.Join(home, ".gemini", "antigravity-cli", "jetski_state.pbtxt")
	targetJetski := filepath.Join(geminiCliDir, "jetski_state.pbtxt")
	if fileExists(baselineJetski) && !fileExists(targetJetski) {
		_ = copyFile(baselineJetski, targetJetski)
	}

	baselineInstallID := filepath.Join(home, ".gemini", "antigravity-cli", "installation_id")
	targetInstallID := filepath.Join(geminiCliDir, "installation_id")
	if fileExists(baselineInstallID) && !fileExists(targetInstallID) {
		_ = copyFile(baselineInstallID, targetInstallID)
	}

	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("dbus-run-session"); err == nil {
			return runLinuxIsolated(profileDir, runtimeDir, acc.Email, agyExe, agyArgs)
		}
	}

	// Fallback for non-Linux or systems without dbus-run-session:
	// switch account in OS store and execute agy
	if err := switchOne(acc, target.Agy); err != nil {
		return fmt.Errorf("switch account: %w", err)
	}
	cmd := exec.Command(agyExe, agyArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runLinuxIsolated(profileDir, runtimeDir, accountEmail, agyExe string, agyArgs []string) error {
	home, _ := os.UserHomeDir()

	dbusWrapper := filepath.Join(runtimeDir, "dbus-daemon-quiet")
	if !fileExists(dbusWrapper) {
		_ = os.WriteFile(dbusWrapper, []byte("#!/bin/sh\nexec dbus-daemon \"$@\" 2>/dev/null\n"), 0o755)
	}

	bashScript := fmt.Sprintf(`
eval $(printf "\n" | gnome-keyring-daemon --daemonize --login --components=secrets 2>/dev/null) 2>/dev/null
agm switch "%s" --target agy >/dev/null 2>&1
exec "%s" "$@"
`, accountEmail, agyExe)

	cmdArgs := []string{
		"--dbus-daemon=" + dbusWrapper,
		"--",
		"bash",
		"-c",
		bashScript,
		"--",
	}
	cmdArgs = append(cmdArgs, agyArgs...)

	cmd := exec.Command("dbus-run-session", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"XDG_RUNTIME_DIR="+runtimeDir,
		"HOME="+profileDir,
		"PATH="+filepath.Join(home, ".local", "bin")+":"+os.Getenv("PATH"),
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func symlinkIfMissing(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	return os.Symlink(src, dst)
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
