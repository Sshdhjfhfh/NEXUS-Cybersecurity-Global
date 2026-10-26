# Nexus desde la terminal

Abre el sistema con un solo comando, desde cualquier directorio:

```sh
nexus
```

El acceso ya está instalado en `~/go/bin/nexus`, incluido en el PATH de este host.
Apunta al cliente de este repositorio. Para instalarlo de nuevo sin sobrescribir
otro ejecutable: `python3 terminal/install.py`.

Nexus detecta su backend local, incluido el puerto real publicado por Docker,
restaura la conversación y muestra un menú numerado. La primera vez pide **la
misma contraseña de la aplicación**. Guarda únicamente el token de sesión en
`~/.config/nexus-terminal/session.json` con permisos `0600`; vuelve a pedir acceso
cuando caduque. No lee claves de firma del servidor ni guarda la contraseña.

## Hablar con la IA y manejar el sistema

Escribe tu petición normalmente. El cliente usa el agente **Auto** existente, que
gestiona tareas, activos, skills, herramientas y MCP mediante sus herramientas de
plataforma. Las conversaciones aparecen también en la web. La selección de IA
respeta las prioridades del backend: perfil del agente, perfil de la conversación
y perfil global activo. No se pide otra API key ni se sustituye la configuración.

El menú acepta números. También funcionan `tareas`, `activos`, `hallazgos`,
`estado`, `ayuda`, `menu` y `salir`. TAB completa los comandos; Ctrl+C cancela la
respuesta en curso. Las listas principales se muestran como tablas compactas.

```text
/status                     Estado e IA seleccionada
/tasks                      Tareas
/task 123                   Detalle de una tarea
/assets                     Activos
/findings                   Hallazgos
/agents                     Agentes
/tools                      Herramientas
/skills                     Skills
/mcp                        Conexiones MCP
/logs                       Logs
/lab                        Estado del laboratorio
/models                     Perfiles configurados
/conversations              Conversaciones
/new                        Nueva conversación con Auto
/new CLAVE_AGENTE           Usar otro agente existente
/use 123                    Continuar una conversación
/profile auto               Volver a la selección automática de perfil
/profile 2                  Elegir perfil solo para esta conversación
/stop                       Detener la conversación seleccionada
```

La terminal utiliza las APIs existentes, con la misma autenticación y controles
del backend. Para operaciones que no tienen un atajo, `/api` admite los endpoints
JSON autorizados, incluidas operaciones de escritura explícitas:

```text
/api GET /api/settings
/api GET /api/tasks/123/goals
/api POST /api/tasks/123/control {"action":"pause"}
```

La terminal no desactiva bloqueos del laboratorio. Las subidas multipart y los
flujos SSE no tienen comandos propios en este cliente; no se afirma paridad de
todos los formatos de transferencia de la web. Las funciones de administración
también se pueden usar mediante el agente Auto y su catálogo de herramientas.

## ZERO_day y Phantom

```text
/zero-day help
/zero-day guide qwen3:1.7b
/zero-day run 30 guided
/zero-day list
/zero-day report IDENTIFICADOR
/zero-day replay IDENTIFICADOR SHA256
/phantom status
/phantom check --fault
/phantom vpn-test
/phantom audit
```

ZERO_day ejecuta el parser sintético aislado de su módulo. Su guía usa Ollama local,
con soporte para Windows desde WSL, sin descargar modelos. `/phantom audit` ejecuta
las pruebas Go; `/phantom check --fault` prueba Tor y su recuperación;
`/phantom vpn-test` prueba WireGuard real. [Alcance y resultados](PHANTOM_AUDIT.md).

## Comandos de una sola ejecución

```sh
nexus --doctor
nexus --command /status
nexus --command '/zero-day run 15 guided'
nexus phantom status
nexus phantom check --fault
nexus phantom vpn-test
nexus phantom audit
```

`--doctor` verifica la conexión sin credenciales. Los comandos autenticados
requieren la sesión ya iniciada o `NEXUS_TOKEN`. Un servidor concreto puede
indicarse con `--url` o `NEXUS_URL`; HTTP solo se admite en loopback y los servidores
remotos requieren HTTPS. El token guardado está asociado al origen exacto del
servidor y no se reutiliza automáticamente para otro origen.

## Validación

Doce pruebas de cliente y sesión correctas. Se comprobaron el comando instalado,
el descubrimiento real de Nexus en 8789, el arranque hasta la contraseña sin
modificarla y la salida limpia al cancelar. Se probaron la conversación asíncrona,
el uso de perfiles, las operaciones API y la cancelación contra fixtures; no se
inició una sesión real con la contraseña del usuario durante esta entrega.

ZERO_day sí completó una campaña con la guía real de `qwen3:1.7b`: 45.073
ejecuciones en 15 segundos y una entrada con crash reproducido. Phantom pasó 60
pruebas con `-race`, incluidas las tres sondas de transporte añadidas. El detalle
de esa validación **no** acredita una red de anonimato operativa.

```sh
python3 -m unittest discover -s terminal -p 'test_*.py' -v
python3 -m unittest discover -s zero_day -p 'test_*.py' -v
go test -race -count=1 ./terminal/checks ./phantom/...
```
