# f5tui

A [k9s](https://k9scli.io/)-style terminal UI for read-only navigation of F5 BIG-IP
resources: **Virtual Servers**, **Pools** (with members), **LTM Policies**, and
**ASM Policies**.

Built in Go with [tview](https://github.com/rivo/tview), talking to the BIG-IP
iControl REST API over HTTPS with basic auth.

---

## Features

- Browse Virtual Servers, Pools, LTM Policies, ASM Policies from a single TUI
- Drill from a pool into its members
- Filter rows with `/` (substring match, case-insensitive)
- Switch partition on the fly with `:part <name>` (or `:part *` for all)
- YAML config file so you don't have to type credentials on the command line
- Built-in mock BIG-IP (`--mock`) for demos and development without a lab device
- Read-only by design — no mutating endpoints are called

## Install

Requires Go 1.22 or later.

```bash
git clone https://github.com/Chup4Chups/f5tui.git
cd f5tui
go build -o f5tui ./cmd/f5tui
```

Or run directly:

```bash
go run ./cmd/f5tui --mock
```

## Quick start (mocked BIG-IP)

```bash
go run ./cmd/f5tui --mock
```

This starts an in-process mock server with fixture data — useful to try the UI
without any credentials.

## Connecting to a real BIG-IP

Either pass flags:

```bash
go run ./cmd/f5tui \
  --host https://bigip.example.com \
  --user admin \
  --pass '***' \
  --insecure \
  --partition Common
```

Or use a config file at `~/.config/f5tui/config.yaml` (or `$XDG_CONFIG_HOME/f5tui/config.yaml`):

```yaml
host: https://bigip.example.com
user: admin
pass: changeme
insecure: true
partition: Common
```

Then simply:

```bash
go run ./cmd/f5tui
```

CLI flags override values from the config file. A custom config path can be
passed with `--config /path/to/file.yaml`.

> **Credentials:** `config.yaml` is `.gitignore`d by default. Only
> `config.example.yaml` is committed.

## Keys and commands

| Key / command    | Action                                    |
|------------------|-------------------------------------------|
| `:`              | Open the command bar                      |
| `/`              | Filter rows in the current view           |
| `?`              | Show the help screen                      |
| `esc`            | Go back / clear filter input              |
| `enter`          | Drill into selection (pools → members)    |
| `:vs`            | Virtual Servers view                      |
| `:pools`         | Pools view                                |
| `:policies`      | LTM Policies view                         |
| `:asm`           | ASM Policies view                         |
| `:part <name>`   | Switch active partition (`*` = all)       |
| `:clear`         | Clear the current row filter              |
| `:q`             | Quit                                      |

The status bar at the bottom always shows the current partition and filter.

## Project layout

```
cmd/f5tui/              main entry point & CLI flag parsing
internal/config/        YAML config loader
internal/f5/            iControl REST client (read-only)
internal/mock/          in-process mock BIG-IP (fixtures embedded)
internal/ui/            tview application, views, navigation, filtering
```

## Development

```bash
go test ./...       # run the unit test suite
go vet ./...
go build ./...
```

The test suite covers:

- the REST client against an `httptest` server (auth header, JSON parsing,
  path encoding for pool members, error handling)
- the YAML config loader (valid, missing, invalid, XDG path)
- the embedded mock server (all endpoints return well-formed JSON)
- the UI filter / partition-matching helpers

## AI disclosure

This project was generated in collaboration with **Claude (Anthropic)** acting
as a pair-programming assistant. The architecture, implementation, tests and
documentation were produced through iterative conversation and reviewed by a
human before publication.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).

This project is not affiliated with or endorsed by F5, Inc. "F5", "BIG-IP",
"LTM", and "ASM" are trademarks of F5, Inc.
