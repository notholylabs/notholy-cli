# NotHoly CLI

**English** · [Português](README.pt-BR.md) · [Español](README.es.md)

`notholy` is the command line for [NotHoly](https://notholy.com): an agent in your terminal, CVE lookups, static analysis of files in our sandbox, and a one-command switch that points Claude Code, Codex or OpenCode at NotHoly models.

Commands run on your machine. By default the agent asks before each one. NotHoly's servers never scan or touch other systems.

## Install

macOS and Linux:

```bash
curl -fsSL https://notholy.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://notholy.com/install.ps1 | iex
```

The installers check the binary against `checksums.txt` before installing it. With Go 1.26 or newer you can also build from source:

```bash
go install github.com/notholylabs/notholy-cli/cmd/notholy@latest
```

## Sign in

```bash
notholy login
```

This opens a sign-in page in your browser and confirms a short code. The key is kept in your system keychain, or in a file only your user can read where no keychain exists. `notholy whoami` shows the account; `notholy logout` removes the key.

## Commands

| Command | What it does |
|---|---|
| `notholy` | Agent in this terminal. `--mode ask` (default) asks before each command; `plan` only plans, `auto-edit` approves file edits and asks for the rest, and `--mode yolo --yolo` asks nothing. `/mode` switches inside a session. |
| `notholy "question"` | One answer. Also reads from stdin: `cat app.log \| notholy "what failed?"` |
| `notholy models` | Models you can use, with prices. |
| `notholy credits` | Your balance. |
| `notholy cve CVE-2024-3094` | One CVE with CVSS, EPSS and KEV status. |
| `notholy cve search "openssl" --kev --min-cvss 7` | Search CVEs. |
| `notholy analyze ./app.apk` | Static analysis in an isolated sandbox. Your file is never executed. |
| `notholy connect claude` | Point Claude Code (or `codex`, `opencode`) at NotHoly. `--global` for every project, `--undo` to revert. |
| `notholy run claude [args...]` | Start the tool on NotHoly for this one process, without changing its settings. |
| `notholy mcp` | MCP server on stdio, for clients that speak MCP. |
| `notholy doctor` | Checks the setup and the connection. |
| `notholy version` | Prints the version. |

Add `--json` to any lookup for machine-readable output.

## Build from this repository

```bash
go build ./cmd/notholy
go test ./...
```

## Links

- Documentation: https://docs.notholy.com
- Models and prices: https://notholy.com/en/models
- Status: https://status.notholy.com
- Security reports: https://notholy.com/.well-known/security.txt

## License

See [LICENSE.md](LICENSE.md).
