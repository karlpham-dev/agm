# agm — Antigravity Account Manager

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](https://github.com/shyim/agm)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

`agm` is a fast, standalone, multi-account manager CLI for [Antigravity](https://antigravity.google/) developer tools:

- **Antigravity CLI** (`agy`)
- **Antigravity IDE**
- **Antigravity Desktop**

Manage multiple Google accounts, inspect live model quotas (Gemini Pro, Gemini Flash, Claude), switch sessions with zero friction, and run parallel isolated sessions from a single static binary.

---

## Key Features

- **Standalone & Pure Go**: Built with `modernc.org/sqlite` — zero CGO dependencies, compiles into a single portable binary.
- **Client-Side Encryption**: All OAuth tokens and quota snapshots are encrypted at rest with **AES-256-GCM** using a local master key (`.mk`).
- **Unified Account Switching**: Switch Antigravity CLI, IDE, or all surfaces simultaneously (`agm switch <email|alias>`).
- **Live Quota Tracking**: View remaining capacity and reset schedules across Gemini 2.5 Pro, Flash, and Claude models.
- **Intelligent Auto-Switching**: Automatically select and switch to the account with the highest available quota (`agm auto-switch`).
- **Safe State Injection**: Automatic database backup (`state.vscdb.backup`) prior to IDE OAuth token injection.
- **Direct Isolated Sessions**: Run parallel `agy` sessions directly using `agm <alias>` or `agm run <alias>` with per-account sandboxed keyrings.
- **Interactive Shell Completion**: Built-in tab-completion for account emails, friendly aliases, targets, and models.

---

## Architecture & How It Works

> For in-depth system architecture, component breakdowns, sequence diagrams, and cryptographic specifications, see **[ARCHITECTURE.md](./ARCHITECTURE.md)**.

```
                     ┌───────────────────────────────┐
                     │ Google OAuth / Cloud APIs     │
                     └───────────────┬───────────────┘
                                     │
                   agm login         │  agm refresh / validate
                                     ▼
                     ┌───────────────────────────────┐
                     │     ~/.antigravity-agent/     │
                     │  ├── cloud_accounts.db (AES)  │
                     │  ├── .mk (Master Key, 0600)   │
                     │  └── aliases.json             │
                     └───────────────┬───────────────┘
                                     │
                                     │  agm switch --target <agy|ide|all>
                                     │
                   ┌─────────────────┴─────────────────┐
                   │                                   │
                   ▼                                   ▼
        ┌─────────────────────┐             ┌─────────────────────┐
        │   Antigravity CLI   │             │   Antigravity IDE   │
        │       (`agy`)       │             │   (`state.vscdb`)   │
        ├─────────────────────┤             ├─────────────────────┤
        │ macOS Keychain      │             │ SQLite injection:   │
        │ Linux secret-tool   │             │   • oauthToken      │
        │ Windows Credential  │             │   • authStatus      │
        │ (no restart needed) │             │ (auto-restart IDE)  │
        └─────────────────────┘             └─────────────────────┘
```

---

## Installation

### From Go Package

```bash
go install github.com/shyim/agm@latest
```

### From Source

```bash
git clone https://github.com/shyim/agm.git
cd agm
go build -o agm .
mkdir -p ~/.local/bin
cp agm ~/.local/bin/agm
```

Ensure `~/.local/bin` is in your `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Platform Prerequisites (Linux)

Linux systems require `libsecret` to interact with the system keyring:

```bash
sudo apt-get update && sudo apt-get install -y libsecret-tools dbus-user-session gnome-keyring
```

> **macOS** and **Windows** work out of the box using native APIs (macOS Keychain via `/usr/bin/security`, Windows Credential Manager via `advapi32.dll`).

---

## Shell Autocompletion

Enable fast tab-completion for commands, account emails, aliases, targets, and models:

### Bash

```bash
# Current session:
source <(agm completion bash)

# Permanent:
agm completion bash > ~/.local/share/bash-completion/completions/agm
```

### Zsh

```bash
# Add to ~/.zshrc:
autoload -U compinit; compinit
source <(agm completion zsh)
```

### Fish

```bash
agm completion fish > ~/.config/fish/completions/agm.fish
```

---

## Quick Start

```bash
# 1. Initialize local store (creates ~/.antigravity-agent/)
agm init

# 2. Add your Google accounts via browser OAuth
agm login

# 3. Add friendly aliases for quick switching
agm alias work user1@gmail.com
agm alias personal user2@gmail.com

# 4. View accounts and remaining quota scores
agm list

# 5. Switch global active session
agm switch work --target agy

# 6. Or run agy directly for any account in parallel terminals (no switching required!)
agm work
agm personal
```

---

## Adding & Importing Accounts

### 1. Browser OAuth Login (Recommended)

```bash
agm login
# Alias: agm add
```

- Binds to `127.0.0.1:8888` (automatically falls back through `8892` if port is in use).
- Opens default browser for Google OAuth with `access_type=offline` and `prompt=consent`.
- Retrieves OAuth refresh token, user profile, project ID, and initial quota snapshot.
- Securely stores encrypted credentials into `cloud_accounts.db`.

### 2. Import Active Antigravity CLI Session

```bash
agm import-cli
# Alias: agm import-agy
```

Pulls active credentials already saved in your OS credential store (`gemini:antigravity`).

### 3. Import Antigravity IDE Sessions

```bash
# Auto-detect and import all profiles (default + custom user-data-dir profiles):
agm import-ide

# Explicitly import all detected profiles across ~/.config/Antigravity*:
agm import-ide --all

# Import from a specific user-data-dir or state.vscdb path:
agm import-ide ~/.config/AntigravityIDE-Account2

# Import multiple specified profiles at once:
agm import-ide ~/.config/AntigravityIDE-Account2 ~/.config/AntigravityIDE-Account3
```

- Extracts and decodes the protobuf unified OAuth token (`antigravityUnifiedStateSync.oauthToken`).
- Automatically scans and discovers multi-instance IDE profiles (e.g. `~/.config/Antigravity IDE`, `~/.config/AntigravityIDE-Account*`).
- Resolves account identity, project ID, and live quotas automatically via Google APIs.

### 4. Backup & Restore

```bash
# Export all accounts (tokens stay encrypted with master key)
agm export accounts_backup.json

# Import accounts from backup file
agm import-backup accounts_backup.json
# Alias: agm import accounts_backup.json
```

> [!NOTE]
> Backup files preserve AES-GCM ciphertext. Restoring on another system requires the same master key (`~/.antigravity-agent/.mk`).

---

## Switching Accounts

```bash
agm switch <email|alias|partial> [--target agy|ide|all]
```

### Target Options (`--target`, `-t`)

| Target | Description | Behavior |
|--------|-------------|----------|
| `agy` (`cli`) | Antigravity CLI | Updates OS credential store (`gemini:antigravity`). No GUI process restart required. |
| `ide` | Antigravity IDE | Injects protobuf token into `state.vscdb` (`state.vscdb.backup` created), terminates and restarts IDE process. |
| `all` *(default)* | All surfaces | Applies switch to `agy` first, then updates `ide`. |

### Flexible Account Matching

`agm` supports fuzzy and forgiving matching:
- **By alias**: `agm switch work`
- **By full email**: `agm switch you@gmail.com`
- **By substring**: `agm switch you`
- **Dot-insensitive**: `agm switch johnsmith` matches `john.smith@gmail.com`

Expired tokens are automatically refreshed via Google's token endpoint prior to switching.

---

## Quota Inspection & Account Management

### Account Overview (`agm list` / `agm ls`)

```bash
agm list
```

Example output:
```text
EMAIL                                STATUS             GEM-PRO  GEM-FLASH     CLAUDE
------------------------------------------------------------------------------------
work@company.com                     cli,ide               100%       100%        85%
personal@gmail.com                   active                 40%        90%        20%
backup@gmail.com                     token-exp                -          -          -
```

**Status Tags:**
- `cli`: Account is currently active in the Antigravity CLI credential store.
- `ide`: Account is currently active in the Antigravity IDE database.
- `active`: Account was most recently marked active in `agm`.
- `token-exp`: Access token is currently expired (run `agm validate` or `agm switch`).

### Detailed Model Quotas (`agm info`)

```bash
agm info work
```

Shows token expiration time and a detailed breakdown of models, sorted by remaining score:

```text
Account: work@company.com
Token expiry: 2026-10-09T22:15:00+07:00

PROVIDER     MODEL                                                SCORE  RESET
------------------------------------------------------------------------------------------
GOOGLE       gemini-2.5-pro                                        100%  2026-10-10T02:00:00Z
GOOGLE       gemini-2.5-flash                                      100%  2026-10-10T02:00:00Z
ANTHROPIC    claude-3-7-sonnet                                      85%  2026-10-10T05:00:00Z
ANTHROPIC    claude-3-5-sonnet                                      85%  2026-10-10T05:00:00Z
```

### Active Snapshot (`agm status`)

```bash
agm status
```

Displays active account and primary model quotas at a glance.

### Live Dashboard (`agm watch`)

```bash
agm watch --interval 10
# Alias: agm watch -n 10
```

Clears the screen and provides real-time monitoring of all accounts and quota scores (default interval: 10 seconds).

### Automated Best-Quota Switch (`agm auto-switch`)

```bash
# Switch to account with highest quota (min 50% threshold)
agm auto-switch

# Filter by minimum score and model family
agm auto-switch --min 60 --model claude
agm auto-switch --min 80 --model gemini
```

Prompts for confirmation before applying the switch.

### Quota & Token Maintenance

```bash
# Refresh live quotas for a single account
agm refresh work

# Bulk refresh quotas for all accounts
agm refresh-all

# Validate all accounts and refresh expired tokens only
agm validate

# Compare local DB against IDE and CLI credential store
agm sync

# Remove an account from local database (prompts confirmation)
agm remove work
# Alias: agm rm work
```

---

## Aliases

Create convenient shortcuts for long email addresses:

```bash
# Set alias: agm alias <name> <email>
agm alias acc1 dev1@gmail.com
agm alias acc2 dev2@gmail.com

# List all aliases
agm alias

# Delete an alias
agm unalias acc1
```

> [!TIP]
> `agm alias` automatically detects inverted arguments (e.g. `agm alias dev1@gmail.com acc1`) and swaps them appropriately to prevent corrupted lookups.

---

## Diagnostics (`agm doctor`)

Run `agm doctor` anytime to verify environment paths, binaries, keys, and stores:

```bash
agm doctor
```

Example diagnostics output:
```text
Running diagnostics...
✓ DATA_DIR: /home/user/.antigravity-agent
✓ DATABASE: /home/user/.antigravity-agent/cloud_accounts.db
✓ MASTER_KEY: /home/user/.antigravity-agent/.mk
✓ IDE_DB: /home/user/.config/Antigravity IDE/User/globalStorage/state.vscdb
✓ AGY_EXE: /home/user/.local/bin/agy
✓ CLI_CREDS: dev1@gmail.com
✓ IDE_EXE: /usr/bin/antigravity-ide
✓ ACCOUNTS: 3
```

---

## Concurrent Multi-Session Mode

Need to run multiple `agy` CLI agents simultaneously in different terminals without token conflicts or manual switching?

`agm` provides **direct, zero-config isolated sessions**. Simply run `agm <alias>` in any terminal:

```bash
# Terminal 1 (Account 1):
agm acc1

# Terminal 2 (Account 2):
agm acc2

# Pass arbitrary flags directly to agy:
agm acc1 --dangerously-skip-permissions
agm acc2 -m claude-3-5-sonnet
```

Under the hood:
- Automatically creates an isolated profile in `~/.agy-profiles/<name>`
- Spawns an isolated D-Bus keyring session (`dbus-run-session` with headless auto-unlocking `gnome-keyring-daemon`)
- Symlinks your developer configuration (`~/.gitconfig`, `~/.ssh`, `~/.bashrc`, `~/.profile`, and global `~/.gemini/config` skills)
- Preserves your current working directory (`pwd`) and full terminal TTY control
- No global credentials or default keyrings are overwritten!

You can also use the explicit syntax:
```bash
agm run <account|alias> [agy-flags...]
```

👉 See **[CHEATSHEET.md](./CHEATSHEET.md)** for a deep dive and tips on running multiple agents in parallel.

---

## Storage & Configuration

### Data Directory Structure

By default, `agm` stores all data under `~/.antigravity-agent/`:

```text
~/.antigravity-agent/
├── cloud_accounts.db      # SQLite database (AES-256-GCM encrypted tokens & quotas)
├── .mk                    # 32-byte AES-256 master key (hex-encoded, chmod 0600)
└── aliases.json           # User-defined email shortcuts
```

### Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `AGM_DATA_DIR` | Custom base directory for `agm` data | `~/.antigravity-agent` |
| `ANTIGRAVITY_AGENT_DIR` | Alias for `AGM_DATA_DIR` | `~/.antigravity-agent` |
| `AGM_DB_PATH` | Explicit path to `cloud_accounts.db` | `$AGM_DATA_DIR/cloud_accounts.db` |

---

## Command Reference

| Command | Aliases | Description |
|---------|---------|-------------|
| `agm <alias>` | — | **Direct launcher**: Launch isolated `agy` CLI session directly for an account/alias |
| `run` | `exec` | Launch an isolated Antigravity CLI (`agy`) session (`agm run <account> [flags]`) |
| `init` | — | Initialize local data directory, master key, and database schema |
| `login` | `add` | Authorize a Google account via browser OAuth |
| `import-cli` | `import-agy` | Import active account from OS credential store |
| `import-ide` | — | Import accounts from Antigravity IDE (`state.vscdb`, user-data-dir, or `--all`) |
| `list` | `ls` | List accounts with status flags and quota summaries |
| `info` | — | Show detailed per-model quota scores and token expiration |
| `status` | — | Display active account and primary quota snapshot |
| `switch` | — | Switch active credentials for target surface(s) (`--target agy\|ide\|all`) |
| `auto-switch` | — | Automatically select account with highest quota (`--min`, `--model`) |
| `refresh` | — | Refresh token and live model quotas for a specific account |
| `refresh-all` | — | Refresh tokens and quotas across all accounts |
| `validate` | — | Check all tokens and refresh expired ones |
| `watch` | — | Live real-time dashboard of accounts and quotas (`-n`, `--interval`) |
| `sync` | — | Verify synchronization between Local DB, IDE DB, and CLI store |
| `alias` | — | List or create account shortcuts |
| `unalias` | — | Delete an alias |
| `export` | — | Export accounts to JSON (`0600` permissions, encrypted tokens) |
| `import-backup` | `import` | Restore accounts from an exported JSON file |
| `remove` | `rm` | Remove an account from local database |
| `doctor` | — | Run system and environment diagnostics |
| `completion` | — | Generate shell autocompletion script (bash, zsh, fish, powershell) |
| `help` | — | Display help for any command |

---

## Troubleshooting

| Symptom | Cause | Solution |
|---------|-------|----------|
| `account matching "xyz" not found` | Typo, non-existent alias, or inverted alias | Run `agm list` and `agm alias` to verify; use `agm alias <name> <email>` |
| `install secret-tool (libsecret)` | Missing D-Bus keyring utility on Linux | Run `sudo apt-get install -y libsecret-tools` |
| `no free OAuth callback port` | Ports `8888`–`8892` bound by another process | Close conflicting processes or check `lsof -i :8888` |
| `IDE database not found` | Antigravity IDE has never been launched | Launch Antigravity IDE at least once so `state.vscdb` is generated |
| `Token expired` | Google OAuth access token expired (1h lifetime) | Run `agm validate` or switch accounts (auto-refreshes) |
| `Empty quotas` | Quota API call timed out or account freshly added | Run `agm refresh <email\|alias>` |
| Keyring prompts for password on Linux | Keyring not unlocked in headless/terminal environment | Use `agy-profile` helper script or unlock via `gnome-keyring-daemon` |

---

## Project Structure

```text
.
├── cmd/                      # Cobra CLI command definitions
│   ├── root.go               # Root command, PersistentPreRunE, db initialization
│   ├── accounts.go           # list, info, status, switch, refresh, validate, sync, remove
│   ├── extras.go             # alias, unalias, export, import-backup, auto-switch, doctor, watch
│   ├── setup.go              # init, login, import-cli, import-ide
│   └── completion_test.go    # Autocompletion unit tests
├── internal/
│   ├── aliases/              # Alias persistence & inverted argument safeguards
│   ├── api/                  # Google OAuth2 login, token refresh, and quota fetching
│   ├── credstore/            # OS-specific keyring wrappers (Keychain, libsecret, DPAPI)
│   ├── crypto/               # AES-256-GCM encryption and master key management
│   ├── db/                   # SQLite schema, queries, and IDE state.vscdb injection
│   ├── paths/                # Data directory and application path resolution
│   ├── process/              # Process detection, termination, and relaunching
│   ├── proto/                # Pure Go protobuf encoding for IDE unified OAuth state
│   └── target/               # Target surface abstractions (agy, ide, all)
├── CHEATSHEET.md             # Guide for concurrent agy multi-session isolation
├── README.md                 # Project documentation
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
└── main.go                   # Main entrypoint
```

---

## Development & Testing

```bash
# Run unit tests
go test -v ./...

# Static analysis
go vet ./...

# Build binary
go build -o agm .
```

---

## License

This project is open source and available under the terms of the [MIT License](./LICENSE).
