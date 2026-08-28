# sk2 — Session Key Keeper

Store, list, search, mark-as-used and delete session-ids for terminal agents
(Claude, opencode, kiro-cli, etc.), so you can resume sessions quickly from the command
line.

Written in Go, no CGO (portable binary). Data lives in a local SQLite database.

## Install

```sh
go build -o sk2 ./cmd/sk2
# optional: put the binary on your PATH, e.g.:
#   sudo install -m 0755 sk2 /usr/local/bin/sk2
```

Requires Go 1.21+.

## Quick start

```sh
# store a session-id
sk2 add abc123 -a claude -t "project ctx" -n "work on the payments module"

# list everything
sk2 list

# list only one agent
sk2 list -a claude

# search by id (partial) or by part of the note
sk2 get abc123
sk2 get payments

# mark as used and print the id (handy for piping)
sk2 use abc123
sk2 use abc123 | xclip -sel clip   # copy to clipboard

# delete
sk2 rm abc123
```

## Commands

### `add <session-key>`

Stores a new session-id. **Upsert:** if the `session-key` already exists, no error is
returned — it updates `last_used_at` and overwrites the provided fields (`created_at` is
preserved).

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--agent` | `-a` | Agent/tool that owns the session (required) |
| `--title` | `-t` | Short label for the session |
| `--note` | `-n` | Long description of the session |

### `list`

Lists records, ordered by `last_used_at` desc (fallback `created_at` desc).

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--agent` | `-a` | Filter by agent |
| `--fields` | `-f` | Title/session-key columns: `both` (default), `title`, `key` |
| `--note` | | Also show the `note` column (hidden by default) |

Examples:

```sh
sk2 list              # title + session-key
sk2 list -f title     # title only
sk2 list -f key       # session-key only
sk2 list --note       # include the note column
sk2 list -a opencode  # only the opencode agent
```

### `get <query>`

Shows the record detail (key, agent, title, note, timestamps) whose `session-key` **or**
`note` contains the given substring. If multiple rows match, all are shown. No match →
error + exit 1.

```sh
sk2 get abc        # by part of the key
sk2 get payments   # by part of the note
```

### `use <session-key>`

Sets `last_used_at` to now and prints the raw session-id to stdout — convenient for
piping/copy. Missing key → error + exit 1.

### `rm <session-key>`

Deletes the record. Missing key → error + exit 1.

## Where data lives

The SQLite database (`sk2.db`) is created at:

1. `$XDG_DATA_HOME/sk2/sk2.db` if `XDG_DATA_HOME` is set;
2. otherwise `~/.local/share/sk2/sk2.db`.

The parent directory is created on first use. The database runs in WAL mode (protection
against data loss).

## Project layout

```
cmd/sk2/            entry point (resolves path, opens the store, runs the CLI)
internal/store/     SQLite persistence (migration + CRUD/search)
internal/cli/       cobra commands (add/list/get/rm/use) and rendering
```

## Development

```sh
go build ./...   # build
go vet ./...     # static analysis
go test ./...    # unit + integration tests
```

External dependencies (justified): `github.com/spf13/cobra` (subcommands, shorthands,
help/autocomplete) and `modernc.org/sqlite` (pure-Go SQLite, no CGO). `mattn/go-sqlite3`
is intentionally avoided to keep the build portable.