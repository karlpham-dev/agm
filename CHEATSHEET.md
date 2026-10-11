# Antigravity CLI (`agy`) & `agm` Multi-Account Cheatsheet

A concise, step-by-step guide for setting up, managing multiple Google accounts, and running concurrent isolated `agy` CLI sessions.

---

## 1. Prerequisites (Linux)

Install Go compiler:

```bash
sudo snap install go --classic
```

Install OS keyring and D-Bus utilities:

```bash
sudo apt-get update && sudo apt-get install -y libsecret-tools dbus-user-session gnome-keyring
```

Ensure `~/.local/bin` is in your `PATH` (e.g., in `~/.bashrc` or `~/.zshrc`):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Install or update `agm`:

```bash
cd /path/to/agm
go build -o agm .
cp agm ~/.local/bin/agm
```

---

## 2. Managing Accounts & Quotas

### Step 2.1: Initialize the Local Store

```bash
agm init
```

*Data is stored encrypted at `~/.antigravity-agent/` (AES-256-GCM).*

### Step 2.2: Add Google Accounts

Run for each Google account you want to manage:

```bash
agm login
```

*Opens browser for Google OAuth. Repeat for each account.*

### Step 2.3: Set Friendly Aliases

> **Note:** Syntax is `agm alias <alias_name> <email>`.

```bash
agm alias acc1 user1@gmail.com
agm alias acc2 user2@gmail.com
```

To list or delete aliases:

```bash
agm alias              # List all aliases
agm unalias acc1       # Remove an alias
```

### Step 2.4: Check Quotas & Accounts

```bash
agm list               # Summary table of all accounts & remaining quotas
agm info acc1          # Detailed per-model breakdown (Gemini, Claude, GPT)
agm info acc2
```

---

## 3. Workflow 1: Single Session (Sequential Switching)

When using one account at a time:

```bash
# 1. Switch active credential for Antigravity CLI
agm switch acc1 --target agy

# 2. Launch agy with your preferred options
agy --dangerously-skip-permissions

# 3. When quota is depleted or you want to switch accounts:
agm switch acc2 --target agy
agy --dangerously-skip-permissions
```

---

## 4. Workflow 2: Concurrent Multi-Session (Native Parallel Mode)

### Why is this needed?
By default, `agy` shares a single OS keyring item (`gemini:antigravity`) and a single config directory (`~/.gemini/`). Running two `agy` instances concurrently in regular terminals causes credentials and sessions to collide.

### The Solution: Built-in Native Profile Isolation
`agm` natively creates and manages isolated sessions for any account or alias. Running `agm <alias>` launches an isolated D-Bus keyring sandbox under `~/.agy-profiles/<alias>`, auto-unlocks a headless keyring, and executes `agy` in your current working directory without touching your global keyring!

### Launch Concurrent Sessions Directly

Simply open separate terminal windows/tabs:

**Terminal 1 (Account 1):**
```bash
agm acc1
```

**Terminal 2 (Account 2):**
```bash
agm acc2
```

**Terminal 3 (Account 3):**
```bash
agm acc3
```

**Terminal 4 (Account 4):**
```bash
agm acc4
```

### Passing CLI Flags
Any additional flags are passed directly through to `agy`:
```bash
agm acc1 --dangerously-skip-permissions
agm acc2 -m gemini-2.5-flash
agm acc3 -m claude-3-5-sonnet
```

### Explicit Run Command
You can also use the explicit subcommand syntax:
```bash
agm run acc1 [flags...]
```

### What happens under the hood?
1. **Isolated Runtime & Keyring**: Runs inside a dedicated D-Bus session with an auto-unlocked ephemeral keyring.
2. **Current Directory Preserved**: Starts in whatever directory you ran `agm` from.
3. **Environment & Tools Shared**: Symlinks `~/.gitconfig`, `~/.ssh`, `~/.bashrc`, `~/.profile`, and global `~/.gemini/config` skills so all developer tools and configurations work seamlessly.
4. **Master Store Linked**: Points back to `~/.antigravity-agent/` so token refreshes and quota records stay synced across all instances.

---

## 5. Summary Comparison

| Method | Syntax | Multiple Terminals? | Global Keyring Changed? |
|---|---|---|---|
| **Sequential Switch** | `agm switch acc1 --target agy` then `agy` | No (switches all terminals) | Yes |
| **Direct Launch (Recommended)** | `agm acc1` / `agm acc2` | **Yes (Full concurrency)** | **No (Completely sandboxed)** |
| **Explicit Runner** | `agm run acc1` | **Yes (Full concurrency)** | **No (Completely sandboxed)** |

---

## 6. Common Issues & Quick Fixes

| Problem | Cause | Fix |
|---|---|---|
| `account matching "xyz" not found` | Inverted alias (`email alias` instead of `alias email`) | Run `agm unalias <corrupt_name>` then `agm alias <alias_name> <email>` |
| `install secret-tool (libsecret)` | D-Bus session not reachable or package missing | `sudo apt-get install libsecret-tools` or wrap with `dbus-run-session` |
| Keyring asks for password popup | Keyring daemon needs interactive unlock | Use `printf "\n" \| gnome-keyring-daemon --daemonize --login --components=secrets` |
| Token expired | Token lifetime exceeded | Run `agm validate` or `agm refresh <email>` |
| Both terminals show same email | Shared keyring conflict | Use `agy-profile <alias>` or D-Bus isolation |
