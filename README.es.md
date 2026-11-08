# NotHoly CLI

[English](README.md) · [Português](README.pt-BR.md) · **Español**

`notholy` es la línea de comandos de [NotHoly](https://notholy.com): un agente en tu terminal, consulta de CVE, análisis estático de archivos en nuestro sandbox y un comando que apunta Claude Code, Codex u OpenCode a los modelos de NotHoly.

Los comandos se ejecutan en tu máquina. Por defecto, el agente pide confirmación antes de cada uno. Los servidores de NotHoly nunca escanean ni tocan otros sistemas.

## Instalación

macOS y Linux:

```bash
curl -fsSL https://notholy.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://notholy.com/install.ps1 | iex
```

Los instaladores comprueban el binario con `checksums.txt` antes de instalarlo. Con Go 1.26 o posterior también puedes compilar desde el código:

```bash
go install github.com/notholylabs/notholy-cli/cmd/notholy@latest
```

## Iniciar sesión

```bash
notholy login
```

Abre una página de inicio de sesión en el navegador y confirma un código corto. La clave se guarda en el llavero del sistema o, donde no hay llavero, en un archivo que solo tu usuario puede leer. `notholy whoami` muestra la cuenta; `notholy logout` borra la clave.

## Comandos

| Comando | Qué hace |
|---|---|
| `notholy` | Agente en esta terminal. `--mode ask` (por defecto) pide confirmación antes de cada comando; `plan` solo planifica, `auto-edit` aprueba las ediciones de archivos y pregunta en el resto y `--mode yolo --yolo` no pregunta nada. `/mode` cambia el modo durante la sesión. |
| `notholy "pregunta"` | Una respuesta. También lee de la entrada estándar: `cat app.log \| notholy "¿qué falló?"` |
| `notholy models` | Modelos disponibles, con precios. |
| `notholy credits` | Tu saldo. |
| `notholy cve CVE-2024-3094` | Un CVE con CVSS, EPSS y estado en KEV. |
| `notholy cve search "openssl" --kev --min-cvss 7` | Búsqueda de CVE. |
| `notholy analyze ./app.apk` | Análisis estático en un sandbox aislado. Tu archivo nunca se ejecuta. |
| `notholy connect claude` | Apunta Claude Code (o `codex`, `opencode`) a NotHoly. `--global` para todos los proyectos, `--undo` para deshacer. |
| `notholy run claude [args...]` | Abre la herramienta con NotHoly solo en este proceso, sin cambiar su configuración. |
| `notholy mcp` | Servidor MCP en stdio, para clientes que hablan MCP. |
| `notholy doctor` | Revisa la instalación y la conexión. |
| `notholy version` | Muestra la versión. |

Añade `--json` a cualquier consulta para obtener la salida en JSON.

## Compilar desde este repositorio

```bash
go build ./cmd/notholy
go test ./...
```

## Enlaces

- Documentación: https://docs.notholy.com
- Modelos y precios: https://notholy.com/es/models
- Estado: https://status.notholy.com
- Reportes de seguridad: https://notholy.com/.well-known/security.txt

## Licencia

Consulta [LICENSE.md](LICENSE.md).
