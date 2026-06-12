# harvest

A command-line tool for managing your Harvest time entries, projects, and tasks, built on the [Harvest API v2](https://help.getharvest.com/api-v2/). It also includes an [interactive TUI](#interactive-tui) (`harvest -ui`) for browsing and editing your timesheet from the terminal.

## Installation

### Download a Release Binary (Recommended)

Download the latest binary for your platform from the [Releases page](https://github.com/jc00ke/harvest/releases).

Or use curl to download directly (example for macOS Apple Silicon):

```bash
curl -sL https://github.com/jc00ke/harvest/releases/latest/download/harvest_darwin_arm64.tar.gz | tar xz
sudo mv harvest /usr/local/bin/
```

### Install with Go

```bash
go install github.com/jc00ke/harvest/cmd/harvest@latest
```

The binary is installed to `~/go/bin`. If that directory isn't already on your `$PATH`, add it:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Otherwise, you can run it directly with `~/go/bin/harvest`.

### Build from Source

1. Clone this repository:

   ```bash
   git clone https://github.com/jc00ke/harvest.git
   cd harvest
   ```

2. Build the application:

   ```bash
   mise run build
   ```

The binary will be created at `bin/harvest`.

## Updating

- **Release binary:** Download the latest version from the [Releases page](https://github.com/jc00ke/harvest/releases) and replace the existing binary.
- **Go install:** Run `go install github.com/jc00ke/harvest/cmd/harvest@latest` again.
- **Build from source:** Pull the latest changes and rebuild:

  ```bash
  git pull
  mise run build
  ```

### Shell Completions

`harvest completion <shell>` prints a completion script that completes subcommands and flags
by calling back into the binary, so it always matches the installed version.
Release archives also ship the scripts in a `completions/` directory.

#### fish

```fish
harvest completion fish > ~/.config/fish/completions/harvest.fish
```

Completions are available in new fish sessions immediately.

#### bash

Requires the [bash-completion](https://github.com/scop/bash-completion) package (v2).

```bash
# Linux, current user only
mkdir -p ~/.local/share/bash-completion/completions
harvest completion bash > ~/.local/share/bash-completion/completions/harvest

# Linux, all users
harvest completion bash | sudo tee /etc/bash_completion.d/harvest > /dev/null

# macOS with Homebrew's bash-completion@2
harvest completion bash > "$(brew --prefix)/etc/bash_completion.d/harvest"
```

Open a new shell to pick up the completions.

#### zsh

```zsh
# Place the script in any directory on your $fpath, e.g.
harvest completion zsh > "${fpath[1]}/_harvest"

# or with Homebrew
harvest completion zsh > "$(brew --prefix)/share/zsh/site-functions/_harvest"
```

If completions aren't already enabled in your shell, also add `autoload -U compinit; compinit` to your `~/.zshrc`. Then open a new shell.

PowerShell is also supported; run `harvest completion powershell --help` for details.

## Configuration

### Getting Harvest API Credentials

1. Log into your Harvest account
2. Go to Settings → Integrations → Developers
3. Create a new Personal Access Token
4. Note your Account ID and Access Token

### Storing Credentials

Store your credentials in the OS keyring:

```bash
harvest auth login
```

You'll be prompted for your Account ID and Access Token. Use `harvest auth status` to verify, and `harvest auth logout` to remove them.

## Usage

### CLI

```bash
harvest entries list              # list time entries for a date (defaults to today)
harvest entries list --date 2026-06-08 --week   # a 7-day window starting at --date
harvest entries list --summary    # aggregate the week per client per day
harvest entries create            # log time
harvest entries start             # start the timer on an existing entry
harvest entries stop              # stop a running timer
harvest entries edit              # edit an existing entry
harvest entries delete            # delete an entry
harvest projects                  # list projects and their tasks
harvest me                        # show the authenticated user
```

Every command accepts `--json` to output results as JSON instead of a table, which makes the CLI easy to script with tools like `jq`:

```bash
harvest --json entries list | jq '.[].hours'
```

Run `harvest --help` or `harvest <command> --help` for full details.

## Interactive TUI

Launch the TUI to browse and edit your timesheet interactively:

```bash
harvest -ui
```

### Time Sheet view

<img width="807" height="611" alt="image" src="https://github.com/user-attachments/assets/c2700762-410e-41f5-9a50-d28836ff7242" />

### Add/Edit Time Entry

<img width="807" height="611" alt="image" src="https://github.com/user-attachments/assets/7d7c8537-c668-4c77-8692-42472ccaa889" />

### Help Menu

<img width="807" height="611" alt="image" src="https://github.com/user-attachments/assets/dda8f9db-c3dc-455e-ba90-898d9bb003e9" />

### Keybindings

#### Navigation

| Key | Action |
|-----|--------|
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `←` / `h` | Previous day |
| `→` / `l` | Next day |
| `t` | Jump to today |

#### Time Entry Actions

| Key | Action |
|-----|--------|
| `n` | Create new time entry |
| `e` | Edit selected entry |
| `d` | Delete selected entry |
| `s` | Start/stop timer on selected entry |

#### General

| Key | Action |
|-----|--------|
| `?` | Toggle help overlay |
| `q` / `Esc` | Quit / go back |
| `Ctrl+C` | Force quit |

## Development

### Running Tests

```bash
mise run test
```

### Building Locally

```bash
mise run build
```

### Full Check (Format, Lint, Test)

```bash
mise run check
```

### Releases

```bash
mise run release v0.1.0
```

Releases now support attestations.

## Disclaimer

[Harvest](https://www.getharvest.com/) is a registered trademark of [Bending Spoons US Inc](https://bendingspoons.com/).
This project has no direct affiliation with Harvest or Bending Spoons.
It is an independent open source project that integrates with the [Harvest API v2](https://help.getharvest.com/api-v2/).

## Credit

I forked this from [Planet Argon's harvest-tui](https://github.com/planetargon/harvest-tui).
If you need Ruby, Rails, AI Safety consulting, you should check out their [services](https://www.planetargon.com/services).

## License

This project is licensed under the [MIT License](LICENSE).
