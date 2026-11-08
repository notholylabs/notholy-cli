# NotHoly CLI

[English](README.md) · **Português** · [Español](README.es.md)

`notholy` é a linha de comando da [NotHoly](https://notholy.com): um agente no seu terminal, consulta de CVEs, análise estática de arquivos no nosso sandbox e um comando que aponta o Claude Code, o Codex ou o OpenCode para os modelos da NotHoly.

Os comandos rodam na sua máquina. Por padrão, o agente pede confirmação antes de cada um. Os servidores da NotHoly nunca escaneiam nem acessam outros sistemas.

## Instalação

macOS e Linux:

```bash
curl -fsSL https://notholy.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://notholy.com/install.ps1 | iex
```

Os instaladores conferem o binário com o `checksums.txt` antes de instalar. Com Go 1.26 ou mais novo, dá para compilar a partir do código:

```bash
go install github.com/notholylabs/notholy-cli/cmd/notholy@latest
```

## Entrar na conta

```bash
notholy login
```

Abre uma página de login no navegador e confirma um código curto. A chave fica no chaveiro do sistema ou, onde não houver chaveiro, num arquivo que só o seu usuário lê. `notholy whoami` mostra a conta; `notholy logout` remove a chave.

## Comandos

| Comando | O que faz |
|---|---|
| `notholy` | Agente neste terminal. `--mode ask` (padrão) pede confirmação antes de cada comando; `plan` só planeja, `auto-edit` aprova edições de arquivo e pergunta no resto e `--mode yolo --yolo` não pergunta nada. `/mode` troca o modo durante a sessão. |
| `notholy "pergunta"` | Uma resposta. Também lê da entrada padrão: `cat app.log \| notholy "o que falhou?"` |
| `notholy models` | Modelos disponíveis, com preço. |
| `notholy credits` | Seu saldo. |
| `notholy cve CVE-2024-3094` | Um CVE com CVSS, EPSS e status no KEV. |
| `notholy cve search "openssl" --kev --min-cvss 7` | Busca de CVEs. |
| `notholy analyze ./app.apk` | Análise estática num sandbox isolado. Seu arquivo nunca é executado. |
| `notholy connect claude` | Aponta o Claude Code (ou `codex`, `opencode`) para a NotHoly. `--global` vale para todos os projetos, `--undo` desfaz. |
| `notholy run claude [args...]` | Abre a ferramenta na NotHoly só neste processo, sem mexer na configuração dela. |
| `notholy mcp` | Servidor MCP em stdio, para clientes que falam MCP. |
| `notholy doctor` | Confere a instalação e a conexão. |
| `notholy version` | Mostra a versão. |

Use `--json` em qualquer consulta para receber a saída em JSON.

## Compilar a partir deste repositório

```bash
go build ./cmd/notholy
go test ./...
```

## Links

- Documentação: https://docs.notholy.com
- Modelos e preços: https://notholy.com/pt-br/models
- Status: https://status.notholy.com
- Relatar falhas de segurança: https://notholy.com/.well-known/security.txt

## Licença

Veja [LICENSE.md](LICENSE.md).
