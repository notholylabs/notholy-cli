# NotHoly CLI

**English** · [Português](README.pt-BR.md) · [Español](README.es.md)

`notholy` is an agent in your terminal that runs on [NotHoly](https://notholy.com) models: uncensored and abliterated models for reverse engineering, pentest, bug bounty, red team, malware analysis and vulnerability research. Abliterated means the refusal behavior was removed from the weights.

Since 1.0 the CLI is built on the open-source Codex CLI (Apache-2.0). It keeps its terminal interface, keys, sessions, MCP client, skills, sandbox and approvals, and connects only to `api.notholy.com` and `notholy.com`.

## Install

macOS and Linux:

```bash
curl -fsSL https://notholy.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://notholy.com/install.ps1 | iex
```

The installers check the binary against `checksums.txt` before installing it. The same files are attached to each [release](https://github.com/notholylabs/notholy-cli/releases). `notholy update` installs new versions from the signed channel.

| Platform | Version |
|---|---|
| Linux x64, Windows x64, macOS on Intel | 1.0.0 |
| Linux arm64, Windows on ARM, Apple Silicon | 0.9.4, until 1.0.1 |

## Sign in

```bash
notholy login
```

You confirm a short code in the browser. The key goes to your system keychain. A key saved by 0.9 keeps working, so you do not need to sign in again after updating. `notholy whoami` shows the account and balance; `notholy logout` removes the key.

## Use

| Command | What it does |
|---|---|
| `notholy` | Agent in this terminal. Enter adds a message to the running turn, Tab queues it, Esc interrupts. |
| `notholy "task"` | Starts a session with that task. |
| `notholy exec "task"` | Runs without the interface; `--json` prints events as JSON lines. |
| `notholy resume` | Picks a saved session. `--last` or `notholy --continue` opens the most recent one. |
| `notholy review` | Reviews the changes in the current repository. |
| `notholy mcp` | Manages extra MCP servers. |
| `notholy doctor` | Checks the setup and the connection. |

Inside a session, `/model` changes the model and its reasoning effort, `/permissions` changes what the agent may do, and `/resume` switches sessions. The model list comes from your account.

## Tools

The NotHoly MCP server is set up by default. It gives the agent a knowledge base of offensive security material, CVE lookups with CVSS, EPSS and KEV, security advisories, web search, a page reader and passive recon. You can turn tools off in `~/.notholy/config.toml`.

## What runs where

Commands run on your machine, never on NotHoly's servers, and NotHoly's servers do not scan or touch other systems.

- Linux: commands run in a workspace sandbox when bubblewrap (`bwrap`) is installed. Without it, the agent asks before each command that is not read-only. `notholy exec` cannot ask, so without bubblewrap it runs commands with no sandbox.
- macOS: commands run in the macOS sandbox.
- Windows: the agent asks before commands. You can allow one for the rest of the session or choose full access in `/permissions`.
- Shell commands that search the machine, dotfiles or the environment for keys or tokens, or that read credential files, are refused in every mode. The agent asks you for the key instead.

The macOS binary is signed ad hoc, not notarized. The installer and `notholy update` are not affected. If you download it with a browser and macOS blocks it, run `xattr -d com.apple.quarantine <file>`.

## From 0.9

The first start moves 0.9 sessions to `~/.notholy/sessions-legacy` and copies your MCP servers, saved model and tool settings into `~/.notholy/config.toml`. The 0.9 commands `cve`, `analyze`, `connect` and `run` are not in 1.0: lookups and file analysis are agent tools now. The 0.9 source is in the [`go-legacy`](https://github.com/notholylabs/notholy-cli/tree/go-legacy) branch.

## Links

- Documentation: https://docs.notholy.com
- Models and prices: https://notholy.com/en/models
- Status: https://status.notholy.com
- Security reports: https://notholy.com/.well-known/security.txt

## License

NotHoly's own code and the binaries are under [LICENSE.md](LICENSE.md). The agent is based on [OpenAI Codex](https://github.com/openai/codex), licensed under the Apache License 2.0 ([LICENSE-APACHE](LICENSE-APACHE), [NOTICE-UPSTREAM](NOTICE-UPSTREAM)).
