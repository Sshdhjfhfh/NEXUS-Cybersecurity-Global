# NEXUS: revisión local y puente de sesión

Esta entrega incorpora un transporte de mensajes compatible con Chat Completions. Un operador de la conversación activa lee la solicitud y entrega la respuesta del asistente. El proceso Python no es un modelo, no interpreta solicitudes por sí solo y no ejecuta herramientas. NEXUS mantiene la ejecución y sus controles de alcance.

## Arranque en el entorno del proyecto

En WSL Ubuntu-22.04, desde `/home/saul/nexus-real`:

```sh
python3 scripts/start-review.py
```

Requiere las dependencias ya instaladas en este equipo: PostgreSQL en `~/.nexus-dev/pg`, herramientas en `~/.nexus-dev/labtools/bin`, Go 1.26.10 en la caché de toolchains y Node con las dependencias de `web`. Este script es un lanzador de revisión local, no un instalador portable.

- Web: http://127.0.0.1:3000
- API: http://127.0.0.1:8790/api/health
- Puente: http://127.0.0.1:11435/v1

La base `nexus_review` está separada del entorno anterior para no reanudar sus tareas. PostgreSQL acepta solamente el socket Unix privado. Los datos, claves, cola y registros están bajo `build/review`, excluido de Git. Los servicios web y puente escuchan solamente en loopback. El primer acceso requiere que el propietario establezca su contraseña en la pantalla de configuración.

## Configuración pendiente en el panel

Después de establecer la contraseña, crear un perfil LLM con formato `openai`, modelo `codex-session-relay`, URL `http://127.0.0.1:11435/v1` y streaming desactivado. La clave está en `build/review/relay/api.key`; no subirla al repositorio ni incluirla en capturas. Activar el perfil y comprobar un mensaje desde el chat antes de usarlo ante el jurado.

El puente necesita un operador activo. Las solicitudes aparecen en `build/review/relay/queue`; `lab/benchmark/relay_queue.py --help` describe el protocolo de lectura y respuesta. La respuesta debe corresponder a la solicitud real, respetar su alcance y distinguir observación, inferencia y prueba pendiente. No responder con resultados inventados. El plazo predeterminado es 600 segundos; sin respuesta la solicitud falla. No hay continuidad autónoma garantizada al cerrar la conversación.

## Evidencia obtenida el 26 de noviembre de 2026

- El backend y la web respondieron HTTP 200 en la instancia aislada.
- `scripts/probe-session-relay.py` envió una solicitud HTTP autenticada y recibió una respuesta redactada en la conversación activa. Prueba el transporte, no todavía la integración del chat de la interfaz.
- Los 10 tests existentes de `lab/benchmark/test_session_bridge.py` pasaron: incluyen validación y ciclo de vida del transporte.
- `TestHTTPXAuthorizesThenProbes` ejecutó httpx instalado contra un servidor local. La denegación de destinos fuera de alcance también pasó.
- `TestNucleiLocalExposureAndRetest` ejecutó Nuclei instalado con una plantilla local GET contra un servidor propio: 1 hallazgo confirmado antes, 0 después de cerrar la ruta, 0 métodos de escritura observados. El contenido del servidor es sintético y está identificado como tal.
- La demostración defensiva documentada en `VALIDACION.md` usa eventos del laboratorio. No acredita sensores conectados a una organización real ni vigilancia permanente.

## Repetir las comprobaciones

```sh
python3 -m unittest discover -s scripts -p test_session_relay_entrypoint.py -v
python3 -m unittest discover -s lab/benchmark -p test_session_bridge.py -q
export PATH="$HOME/.nexus-dev/labtools/bin:$PATH"
GOPROXY=off GOTOOLCHAIN=local "$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.linux-amd64/bin/go" test -count=1 -timeout 60s -v ./agents/red/scan ./lab/engagement
```

El test Nuclei se omite explícitamente si el ejecutable falta: un resultado SKIP no es evidencia de ejecución. Los valores por defecto limitan httpx y Nuclei a 2 peticiones/segundo, concurrencia 1 y timeout 5 segundos. Las comprobaciones de actualización están desactivadas y Nuclei no usa Interactsh en este recorrido. Estos límites no sustituyen una autorización de alcance ni un control de salida de red.

## Antes de la presentación

Pendiente: verificar el chat completo con el perfil activo, ensayar las acciones autorizadas desde la interfaz y revisar los textos comerciales de la web contra la evidencia disponible. Una prueba del laboratorio no demuestra auditoría universal de dominios. Para un dominio del jurado se necesita identificar el objetivo autorizado, rutas, intensidad y criterio de detención; no se ha probado ningún dominio público en esta entrega.

Presentar el producto como prototipo de seguridad asistido por IA con comprobaciones locales reproducibles. Explicar que el modelo de razonamiento es externo y que NEXUS aporta la orquestación, herramientas, evidencias y controles implementados.
