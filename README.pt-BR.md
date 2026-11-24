# NotHoly CLI

[English](README.md) · **Português** · [Español](README.es.md)

`notholy` é um agente no seu terminal que roda nos modelos da [NotHoly](https://notholy.com): modelos sem censura e abliterated para engenharia reversa, pentest, bug bounty, red team, análise de malware e pesquisa de vulnerabilidade. Abliterated quer dizer que o comportamento de recusa foi removido dos pesos.

Desde a 1.0 o CLI é construído sobre o Codex CLI, de código aberto (Apache-2.0). Ele mantém a interface de terminal, as teclas, as sessões, o cliente MCP, as skills, o sandbox e as aprovações, e só se conecta a `api.notholy.com` e `notholy.com`.

## Instalar

macOS e Linux:

```bash
curl -fsSL https://notholy.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://notholy.com/install.ps1 | iex
```

Os instaladores conferem o binário com o `checksums.txt` antes de instalar. Os mesmos arquivos estão em cada [release](https://github.com/notholylabs/notholy-cli/releases). `notholy update` instala versões novas pelo canal assinado.

| Plataforma | Versão |
|---|---|
| Linux x64, Windows x64, macOS com Intel | 1.0.0 |
| Linux arm64, Windows on ARM, Apple Silicon | 0.9.4, até a 1.0.1 |

## Entrar

```bash
notholy login
```

Você confirma um código curto no navegador. A chave fica no chaveiro do sistema. Uma chave salva pela 0.9 continua valendo, então não precisa entrar de novo depois de atualizar. `notholy whoami` mostra a conta e o saldo; `notholy logout` apaga a chave.

## Usar

| Comando | O que faz |
|---|---|
| `notholy` | Agente neste terminal. Enter acrescenta a mensagem ao turno em andamento, Tab deixa na fila, Esc interrompe. |
| `notholy "tarefa"` | Abre uma sessão com essa tarefa. |
| `notholy exec "tarefa"` | Roda sem a interface; `--json` imprime os eventos em linhas JSON. |
| `notholy resume` | Escolhe uma sessão salva. `--last` ou `notholy --continue` abre a mais recente. |
| `notholy review` | Revisa as mudanças do repositório atual. |
| `notholy mcp` | Gerencia servidores MCP extras. |
| `notholy doctor` | Confere a instalação e a conexão. |

Dentro de uma sessão, `/model` troca o modelo e o esforço de raciocínio, `/permissions` muda o que o agente pode fazer e `/resume` troca de sessão. A lista de modelos vem da sua conta.

## Ferramentas

O servidor MCP da NotHoly já vem configurado. Ele dá ao agente uma base de conhecimento de segurança ofensiva, consulta de CVE com CVSS, EPSS e KEV, advisories, busca na web, leitor de páginas e recon passivo. Dá para desligar ferramentas em `~/.notholy/config.toml`.

## O que roda onde

Os comandos rodam na sua máquina, nunca nos servidores da NotHoly, e os servidores da NotHoly não fazem scan nem tocam outros sistemas.

- Linux: os comandos rodam num sandbox do workspace quando o bubblewrap (`bwrap`) está instalado. Sem ele, o agente pergunta antes de cada comando que não é só leitura. `notholy exec` não tem como perguntar, então sem bubblewrap roda os comandos sem sandbox.
- macOS: os comandos rodam no sandbox do macOS.
- Windows: o agente pergunta antes dos comandos. Dá para liberar um comando pelo resto da sessão ou escolher acesso total em `/permissions`.
- Comandos de shell que vasculham a máquina, os dotfiles ou o ambiente atrás de chaves e tokens, ou que leem arquivos de credencial, são recusados em todos os modos. O agente pede a chave para você.

O binário de macOS tem assinatura ad hoc, sem notarização. O instalador e o `notholy update` não são afetados. Se você baixar pelo navegador e o macOS bloquear, rode `xattr -d com.apple.quarantine <arquivo>`.

## Vindo da 0.9

Na primeira execução, as sessões da 0.9 vão para `~/.notholy/sessions-legacy`, e os seus servidores MCP, o modelo salvo e as escolhas de ferramentas são copiados para `~/.notholy/config.toml`. Os comandos `cve`, `analyze`, `connect` e `run` da 0.9 não existem na 1.0: consultas e análise de arquivos agora são ferramentas do agente. O código da 0.9 está no branch [`go-legacy`](https://github.com/notholylabs/notholy-cli/tree/go-legacy).

## Links

- Documentação: https://docs.notholy.com
- Modelos e preços: https://notholy.com/pt-BR/models
- Status: https://status.notholy.com
- Relatos de segurança: https://notholy.com/.well-known/security.txt

## Licença

O código próprio da NotHoly e os binários seguem o [LICENSE.md](LICENSE.md). O agente é baseado no [OpenAI Codex](https://github.com/openai/codex), sob a Apache License 2.0 ([LICENSE-APACHE](LICENSE-APACHE), [NOTICE-UPSTREAM](NOTICE-UPSTREAM)).
