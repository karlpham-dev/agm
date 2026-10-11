package paths

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const DBName = "cloud_accounts.db"
const MasterKeyFile = ".mk"

// AgentDir is the local data directory for agm.
// Override: AGM_DATA_DIR or ANTIGRAVITY_AGENT_DIR.
func AgentDir() string {
	if v := os.Getenv("AGM_DATA_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("ANTIGRAVITY_AGENT_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".antigravity-agent")
}

// EnsureAgentDir creates the standalone data directory.
func EnsureAgentDir() (string, error) {
	dir := AgentDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create data dir %s: %w", dir, err)
	}
	return dir, nil
}

// CloudAccountsDBPath is the DB path used by this CLI (always under AgentDir unless AGM_DB_PATH).
func CloudAccountsDBPath() string {
	if db := os.Getenv("AGM_DB_PATH"); db != "" {
		return db
	}
	return filepath.Join(AgentDir(), DBName)
}

// MasterKeyPath is where the standalone CLI stores its AES key (plain hex, mode 0600).
func MasterKeyPath() string {
	return filepath.Join(AgentDir(), MasterKeyFile)
}

// ElectronUserDataDirs returns legacy Antigravity Electron userData candidates (optional key sources).
func ElectronUserDataDirs() []string {
	home, _ := os.UserHomeDir()
	var dirs []string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		dirs = append(dirs,
			filepath.Join(appData, "Antigravity Manager"),
			filepath.Join(appData, "AntigravityManager"),
			filepath.Join(appData, "antigravity-manager"),
		)
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support")
		dirs = append(dirs,
			filepath.Join(base, "Antigravity Manager"),
			filepath.Join(base, "AntigravityManager"),
			filepath.Join(base, "antigravity-manager"),
		)
	default:
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			xdg = filepath.Join(home, ".config")
		}
		dirs = append(dirs,
			filepath.Join(xdg, "Antigravity Manager"),
			filepath.Join(xdg, "AntigravityManager"),
			filepath.Join(xdg, "antigravity-manager"),
		)
	}
	return uniqueStrings(dirs)
}

// ManagerDataDirs returns all dirs that may hold DB / .mk (standalone first).
func ManagerDataDirs() []string {
	dirs := []string{AgentDir()}
	dirs = append(dirs, ElectronUserDataDirs()...)
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, ".config"), cwd)
	}
	return uniqueStrings(dirs)
}

func uniqueStrings(dirs []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

// FindDBPath returns an existing DB, or empty if none (use EnsureStore to create).
func FindDBPath() string {
	if db := os.Getenv("AGM_DB_PATH"); db != "" && fileExists(db) {
		return db
	}
	canonical := CloudAccountsDBPath()
	if fileExists(canonical) {
		return canonical
	}
	for _, d := range ManagerDataDirs() {
		p := filepath.Join(d, DBName)
		if fileExists(p) {
			return p
		}
	}
	return ""
}

// FindMasterKeyPath returns first existing .mk (standalone agent dir preferred).
func FindMasterKeyPath() string {
	p := MasterKeyPath()
	if fileExists(p) {
		return p
	}
	for _, d := range ElectronUserDataDirs() {
		c := filepath.Join(d, MasterKeyFile)
		if fileExists(c) {
			return c
		}
	}
	for _, d := range ManagerDataDirs() {
		c := filepath.Join(d, MasterKeyFile)
		if fileExists(c) {
			return c
		}
	}
	return ""
}

// FindLocalStatePath returns Chromium Local State for legacy Electron OSCrypt keys.
func FindLocalStatePath() string {
	for _, d := range ElectronUserDataDirs() {
		p := filepath.Join(d, "Local State")
		if fileExists(p) {
			return p
		}
	}
	return ""
}

// PrimaryManagerDir is the writable standalone data dir.
func PrimaryManagerDir() string {
	return AgentDir()
}

// productFolderNames returns Application Support / Roaming folder names for a product.
// product is "ide".
func productFolderNames(product string) []string {
	switch product {
	case "ide":
		return []string{"Antigravity IDE", "AntigravityIDE"}
	default:
		return []string{"Antigravity IDE"}
	}
}

// FindStateDB locates state.vscdb for ide (empty product = either/IDE/Code).
func FindStateDB(product string) string {
	home, _ := os.UserHomeDir()
	names := productFolderNames(product)
	if product == "" {
		names = []string{"Antigravity IDE", "Code"}
	}
	var bases []string
	switch runtime.GOOS {
	case "windows":
		bases = []string{os.Getenv("APPDATA")}
	case "darwin":
		bases = []string{filepath.Join(home, "Library", "Application Support")}
	default:
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			xdg = filepath.Join(home, ".config")
		}
		bases = []string{xdg}
	}
	for _, base := range bases {
		for _, name := range names {
			for _, rel := range []string{
				filepath.Join(name, "User", "globalStorage", "state.vscdb"),
				filepath.Join(name, "User", "state.vscdb"),
				filepath.Join(name, "state.vscdb"),
			} {
				p := filepath.Join(base, rel)
				if fileExists(p) {
					return p
				}
			}
		}
	}
	return ""
}

// FindIDEStateDB locates Antigravity IDE state.vscdb.
func FindIDEStateDB() string {
	return FindStateDB("ide")
}

// ResolveStateDB resolves a file or user-data-dir path to a state.vscdb path.
func ResolveStateDB(p string) string {
	if p == "" {
		return ""
	}
	// Expand home ~
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	if fileExists(p) {
		return p
	}
	candidates := []string{
		filepath.Join(p, "User", "globalStorage", "state.vscdb"),
		filepath.Join(p, "globalStorage", "state.vscdb"),
		filepath.Join(p, "state.vscdb"),
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

// FindAllIDEStateDBs discovers all state.vscdb files across all Antigravity IDE profiles.
func FindAllIDEStateDBs() []string {
	home, _ := os.UserHomeDir()
	var bases []string
	switch runtime.GOOS {
	case "windows":
		if app := os.Getenv("APPDATA"); app != "" {
			bases = append(bases, app)
		}
	case "darwin":
		bases = append(bases, filepath.Join(home, "Library", "Application Support"))
	default:
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			xdg = filepath.Join(home, ".config")
		}
		bases = append(bases, xdg)
	}

	var dbs []string
	for _, base := range bases {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(strings.ToLower(name), "antigravity") {
				profileDir := filepath.Join(base, name)
				if resolved := ResolveStateDB(profileDir); resolved != "" {
					dbs = append(dbs, resolved)
				}
			}
		}
	}
	return uniqueStrings(dbs)
}

// FindExecutableForProduct returns a GUI executable for ide.
func FindExecutableForProduct(product string) string {
	if product != "ide" {
		return ""
	}
	return FindAntigravityExecutable()
}

// FindRunningIDEExecutable dynamically inspects running processes to locate the active Antigravity IDE binary.
func FindRunningIDEExecutable() string {
	switch runtime.GOOS {
	case "linux":
		// Fast pure-Go /proc inspection without spawning subprocesses
		entries, err := os.ReadDir("/proc")
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				if _, err := strconv.Atoi(e.Name()); err != nil {
					continue
				}
				exeLink, err := os.Readlink(filepath.Join("/proc", e.Name(), "exe"))
				if err != nil {
					continue
				}
				base := strings.ToLower(filepath.Base(exeLink))
				if base == "antigravity-ide" || base == "antigravity" || (strings.Contains(base, "antigravity") && strings.Contains(base, "ide")) {
					if !strings.Contains(base, "language_server") && !strings.Contains(base, "crashpad") {
						// Prefer launcher script bin/antigravity-ide if present alongside binary
						binWrapper := filepath.Join(filepath.Dir(exeLink), "bin", filepath.Base(exeLink))
						if fileExists(binWrapper) {
							return binWrapper
						}
						return exeLink
					}
				}
			}
		}
	case "darwin":
		out, err := exec.Command("ps", "-ax", "-o", "command=").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				lower := strings.ToLower(line)
				if strings.Contains(lower, "antigravity") && (strings.Contains(lower, "ide") || strings.Contains(lower, "electron")) {
					fields := strings.Fields(strings.TrimSpace(line))
					if len(fields) > 0 && fileExists(fields[0]) {
						return fields[0]
					}
				}
			}
		}
	case "windows":
		out, err := exec.Command("powershell", "-NoProfile", "-Command",
			"(Get-Process | Where-Object { $_.ProcessName -like '*antigravity*' -and $_.Path } | Select-Object -First 1 -ExpandProperty Path)").Output()
		if err == nil {
			path := strings.TrimSpace(string(out))
			if fileExists(path) {
				return path
			}
		}
	}
	return ""
}

// FindAntigravityExecutable dynamically checks running processes, environment variables, PATH, and common install locations.
func FindAntigravityExecutable() string {
	// 1. Dynamic check: inspect running processes to find active IDE path
	if p := FindRunningIDEExecutable(); p != "" {
		return p
	}

	home, _ := os.UserHomeDir()
	var candidates []string

	// 2. Environment variable overrides
	for _, env := range []string{"ANTIGRAVITY_IDE_EXE", "ANTIGRAVITY_IDE_PATH", "AGM_IDE_EXE"} {
		if v := os.Getenv(env); v != "" {
			candidates = append(candidates, v)
			candidates = append(candidates, filepath.Join(v, "antigravity-ide"))
			candidates = append(candidates, filepath.Join(v, "bin", "antigravity-ide"))
		}
	}

	// 3. LookPath
	for _, bin := range []string{"antigravity-ide", "antigravity"} {
		if p, err := exec.LookPath(bin); err == nil {
			candidates = append(candidates, p)
		}
	}

	// 4. Standard platform directories
	switch runtime.GOOS {
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		prog := os.Getenv("ProgramFiles")
		prog86 := os.Getenv("ProgramFiles(x86)")
		candidates = append(candidates,
			filepath.Join(local, "Programs", "Antigravity IDE", "Antigravity IDE.exe"),
			filepath.Join(prog, "Antigravity IDE", "Antigravity IDE.exe"),
			filepath.Join(prog86, "Antigravity IDE", "Antigravity IDE.exe"),
		)
	case "darwin":
		candidates = append(candidates,
			"/Applications/Antigravity IDE.app/Contents/MacOS/Electron",
			"/Applications/Antigravity IDE.app/Contents/MacOS/Antigravity",
			filepath.Join(home, "Applications", "Antigravity IDE.app", "Contents", "MacOS", "Electron"),
			filepath.Join(home, "Applications", "Antigravity IDE.app", "Contents", "MacOS", "Antigravity"),
		)
	default:
		candidates = append(candidates,
			filepath.Join(home, ".local", "bin", "antigravity-ide"),
			filepath.Join(home, ".local", "share", "AntigravityIDE", "antigravity-ide"),
			"/opt/AntigravityIDE/antigravity-ide",
			"/opt/google/AntigravityIDE/antigravity-ide",
			"/usr/bin/antigravity-ide",
			"/usr/local/bin/antigravity-ide",
		)
	}

	for _, p := range candidates {
		if resolved, err := filepath.EvalSymlinks(p); err == nil {
			p = resolved
		}
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// MatchEmail reports whether haystack contains needle (case-insensitive).
// It also matches dot-insensitive variations (e.g. 'johnsmith' matching 'john.smith@gmail.com').
func MatchEmail(haystack, needle string) bool {
	h := strings.ToLower(haystack)
	n := strings.ToLower(needle)
	if strings.Contains(h, n) {
		return true
	}
	hNoDots := strings.ReplaceAll(h, ".", "")
	nNoDots := strings.ReplaceAll(n, ".", "")
	return strings.Contains(hNoDots, nNoDots)
}
