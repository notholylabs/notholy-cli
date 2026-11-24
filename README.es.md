# NotHoly CLI

[English](README.md) · [Português](README.pt-BR.md) · **Español**

`notholy` es un agente en tu terminal que funciona con los modelos de [NotHoly](https://notholy.com): modelos sin censura y abliterated para ingeniería inversa, pentest, bug bounty, red team, análisis de malware e investigación de vulnerabilidades. Abliterated significa que el comportamiento de rechazo se quitó de los pesos.

Desde la 1.0 el CLI está construido sobre Codex CLI, de código abierto (Apache-2.0). Conserva la interfaz de terminal, las teclas, las sesiones, el cliente MCP, las skills, el sandbox y las aprobaciones, y solo se conecta a `api.notholy.com` y `notholy.com`.

## Instalar

macOS y Linux:

```bash
curl -fsSL https://notholy.com/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://notholy.com/install.ps1 | iex
```

Los instaladores comprueban el binario con `checksums.txt` antes de instalarlo. Los mismos archivos están en cada [release](https://github.com/notholylabs/notholy-cli/releases). `notholy update` instala versiones nuevas desde el canal firmado.

| Plataforma | Versión |
|---|---|
| Linux x64, Windows x64, macOS con Intel | 1.0.0 |
| Linux arm64, Windows on ARM, Apple Silicon | 0.9.4, hasta la 1.0.1 |

## Iniciar sesión

```bash
notholy login
```

Confirmas un código corto en el navegador. La clave queda en el llavero del sistema. Una clave guardada por la 0.9 sigue sirviendo, así que no hace falta iniciar sesión otra vez después de actualizar. `notholy whoami` muestra la cuenta y el saldo; `notholy logout` borra la clave.

## Uso

| Comando | Qué hace |
|---|---|
| `notholy` | Agente en esta terminal. Enter agrega el mensaje al turno en curso, Tab lo deja en cola, Esc interrumpe. |
| `notholy "tarea"` | Abre una sesión con esa tarea. |
| `notholy exec "tarea"` | Corre sin interfaz; `--json` imprime los eventos como líneas JSON. |
| `notholy resume` | Elige una sesión guardada. `--last` o `notholy --continue` abre la más reciente. |
| `notholy review` | Revisa los cambios del repositorio actual. |
| `notholy mcp` | Administra servidores MCP adicionales. |
| `notholy doctor` | Revisa la instalación y la conexión. |

Dentro de una sesión, `/model` cambia el modelo y el esfuerzo de razonamiento, `/permissions` cambia lo que el agente puede hacer y `/resume` cambia de sesión. La lista de modelos viene de tu cuenta.

## Herramientas

El servidor MCP de NotHoly viene configurado. Le da al agente una base de conocimiento de seguridad ofensiva, consultas de CVE con CVSS, EPSS y KEV, advisories, búsqueda web, un lector de páginas y recon pasivo. Puedes apagar herramientas en `~/.notholy/config.toml`.

## Qué corre dónde

Los comandos corren en tu máquina, nunca en los servidores de NotHoly, y los servidores de NotHoly no escanean ni tocan otros sistemas.

- Linux: los comandos corren en un sandbox del workspace cuando bubblewrap (`bwrap`) está instalado. Sin él, el agente pregunta antes de cada comando que no sea de solo lectura. `notholy exec` no puede preguntar, así que sin bubblewrap corre los comandos sin sandbox.
- macOS: los comandos corren en el sandbox de macOS.
- Windows: el agente pregunta antes de los comandos. Puedes permitir uno por el resto de la sesión o elegir acceso total en `/permissions`.
- Los comandos de shell que buscan claves o tokens en la máquina, los dotfiles o el entorno, o que leen archivos de credenciales, se rechazan en todos los modos. El agente te pide la clave.

El binario de macOS tiene firma ad hoc, sin notarización. El instalador y `notholy update` no se ven afectados. Si lo descargas con el navegador y macOS lo bloquea, ejecuta `xattr -d com.apple.quarantine <archivo>`.

## Desde la 0.9

En el primer inicio, las sesiones de la 0.9 pasan a `~/.notholy/sessions-legacy`, y tus servidores MCP, el modelo guardado y la configuración de herramientas se copian a `~/.notholy/config.toml`. Los comandos `cve`, `analyze`, `connect` y `run` de la 0.9 no están en la 1.0: las consultas y el análisis de archivos ahora son herramientas del agente. El código de la 0.9 está en la rama [`go-legacy`](https://github.com/notholylabs/notholy-cli/tree/go-legacy).

## Enlaces

- Documentación: https://docs.notholy.com
- Modelos y precios: https://notholy.com/es/models
- Estado: https://status.notholy.com
- Reportes de seguridad: https://notholy.com/.well-known/security.txt

## Licencia

El código propio de NotHoly y los binarios siguen [LICENSE.md](LICENSE.md). El agente se basa en [OpenAI Codex](https://github.com/openai/codex), bajo la Apache License 2.0 ([LICENSE-APACHE](LICENSE-APACHE), [NOTICE-UPSTREAM](NOTICE-UPSTREAM)).
