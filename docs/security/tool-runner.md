# Diseño: aislamiento de las herramientas de los agentes (R13)

Estado: **propuesta** (2026-11-02). Bloquea los encargos contra objetivos reales de clientes.

## 1. Problema

Las herramientas que ejecutan comandos (Bash en primer y segundo plano, el shell
interactivo, las herramientas personalizadas y los servidores MCP stdio) lanzan
procesos **dentro del contenedor del orquestador**: `exec.CommandContext(shell, "-c", cmd)`
en `core/tool/bash.go`, `task.go` y `shell_session.go`.

Ese contenedor corre como `node` (`Dockerfile`, `USER node`) y tiene:

| Recurso | Dónde | Qué permite a un comando |
| --- | --- | --- |
| Clave JWT y clave de cifrado de 2FA/SSO | volumen `nexus_keys` en `/app/keys` | Firmar sesiones de consola y de portal y descifrar los secretos TOTP |
| Credenciales de la BD | entorno del proceso del orquestador | Leer `/proc/<ppid>/environ` y conectarse a Postgres con los datos de todos los clientes |
| Red `nexus_internal` | compose | Alcanzar Postgres directamente |
| Datos de todas las tareas | volumen `nexus_data` en `/app/data` | Leer la evidencia de otros clientes |
| Herramientas de archivos (Read, Write, Edit, Grep, Glob) | proceso del orquestador | Lo mismo que Bash para todo lo legible por `node`, sin ejecutar nada (ver 3.4) |

En la Fase 1 se quitaron los secretos del **entorno heredado** (`core/tool/childenv.go`),
pero eso no es una frontera: el proceso hijo sigue siendo el mismo usuario en el mismo
contenedor. Un prompt injection desde un objetivo hostil ("ejecuta `cat /proc/1/environ`")
basta para robar las credenciales.

La puerta de alcance (`ethical_boundary/scope`, R12) inspecciona el **texto** de la
llamada. Un script que construye el host, lo codifica o lo resuelve en tiempo de
ejecución la evita. Solo una frontera de red la hace efectiva.

## 2. Objetivo

Un comando lanzado por un agente:

1. no puede leer secretos, claves ni datos de otras tareas;
2. no alcanza la BD ni otros servicios internos;
3. solo alcanza por red los destinos del alcance bloqueado de su tarea;
4. sigue pasando por el proxy de captura, para que la evidencia se conserve;
5. ve un directorio de trabajo propio de su tarea, que el orquestador también lee
   (las herramientas Read/Write/Grep trabajan sobre los mismos archivos).

## 3. Diseño propuesto

### 3.1 Contenedor `tool-runner`

- Un servicio de compose aparte, con imagen propia (herramientas de pentest, sin el
  binario del orquestador), usuario sin privilegios, `cap_drop: [ALL]` (más `NET_RAW`
  solo si se aprueban escaneos SYN), `no-new-privileges`, raíz de solo lectura y
  `tmpfs` para `/tmp`.
- **Volúmenes:** solo `nexus_work`, con un subdirectorio por tarea. Ni `nexus_keys` ni
  `nexus_data`.
- **Red:** una red `nexus_tools` propia, sin acceso a `nexus_internal` (Postgres) ni
  ruta por defecto a Internet. La única salida es la pasarela de 3.3.

### 3.2 Protocolo orquestador → runner

Un servicio pequeño dentro del runner (Go, el mismo módulo) escucha en un socket Unix
compartido (`nexus_runner_sock`), no en TCP. Así no queda expuesto en ninguna red.

- `Exec{task_id, cmd, timeout, env}` devuelve la salida en streaming, el código de
  salida y señales de cancelación. Cubre Bash y las tareas en segundo plano.
- `Session{task_id}` abre un PTY persistente para `shell_session.go`.
- Cada petición lleva un token por tarea, firmado por el orquestador con una clave que
  **no** está en el runner: el runner solo verifica (clave pública Ed25519). Así un
  comando no puede pedir ejecutar en otra tarea ni usar el directorio de otra.
- El runner ejecuta cada comando con `cwd = /work/<task_id>` y un usuario por tarea o,
  como mínimo, un `setpriv` con `--no-new-privs`.

En `core/tool` se introduce una interfaz `Launcher` (`Local` hoy y `Runner` nuevo) que
reemplaza las tres llamadas a `exec.CommandContext`. `Local` queda solo para desarrollo
y se rechaza al arrancar cuando `NEXUS_PORTAL_BASE_URL` es `https://` (modo producción,
igual que el watchdog de operaciones).

### 3.3 Salida de red limitada al alcance

- Una pasarela (contenedor `egress`) en `nexus_tools` con nftables: permite solo las IP
  del alcance bloqueado de cada tarea, más el proxy de captura. El orquestador publica
  esas listas por tarea; cada tarea sale con su propia IP de origen (una por
  subdirectorio de trabajo) o por un puerto del proxy por tarea.
- El tráfico HTTP sigue yendo por el proxy de captura, que se mueve de `127.0.0.1:8788`
  a una interfaz alcanzable solo desde `nexus_tools`. Además, el proxy aplica también
  el alcance (hoy solo captura).
- Las resoluciones DNS pasan por un resolutor de la pasarela que solo responde nombres
  del alcance. Esto cierra la exfiltración por DNS.

### 3.4 Lo que no cambia

- Read, Write, Edit, Grep y Glob siguen ejecutándose en el orquestador, pero
  restringidos a `nexus_work/<task_id>`. **Hoy no tienen ninguna restricción:**
  `resolvePath` (`core/tool/helpers.go`) acepta cualquier ruta absoluta y ningún hook
  la filtra. Por eso Read puede abrir `/app/keys/jwt.key` o los datos de otra tarea sin
  pasar por Bash. Es el paso 2 del plan y no depende del runner.
- La puerta de alcance en texto (R12) se mantiene como primera barrera y para dar
  mensajes claros al agente.

## 4. Plan por pasos (cada uno con pruebas)

1. ✅ `Launcher` en `core/tool` (`launcher.go`), con la implementación `LocalLauncher`
   actual y sin cambio funcional. Bash en primer y segundo plano y las sesiones de
   shell ya arrancan por ahí. `TestCommandsGoThroughLauncher` comprueba que las tres
   rutas pasan por el launcher. Las herramientas personalizadas y los MCP stdio
   (`customtool.go`, `core/mcp/stdio.go`) todavía no; van en el paso 4.
2. Raíz de trabajo por tarea para Read/Write/Edit/Grep/Glob, con tests negativos de
   recorrido de rutas (`../`, enlaces simbólicos).
3. Servicio runner, protocolo `Exec`, tokens por tarea y tests de integración en un
   contenedor real. Prueba negativa: un comando no puede leer `/app/keys`, ni
   `/proc/<pid>/environ` del orquestador, ni conectar con Postgres.
4. `Session` (PTY) y MCP stdio sobre el runner.
5. Pasarela de salida y proxy con alcance. Prueba negativa en el Cyber Range (Fase 2):
   un host fuera de alcance no recibe ni un paquete.
6. Rechazo de `Local` en producción y actualización de `docs/security/security-audit.md`.

## 5. Decisiones abiertas

- **Escaneos SYN o UDP:** necesitan `NET_RAW`. Se puede empezar sin ellos (escaneos
  `connect`) y añadirlos detrás de una opción por encargo.
- **Un runner por tarea o compartido:** un contenedor por tarea aísla mejor, pero
  requiere orquestar contenedores. Hoy eso exigiría montar el socket de Docker en el
  orquestador, y montarlo equivale a root en el host: queda descartado. El runner
  compartido con usuario y directorio por tarea es el punto de partida.
- **Coste operativo:** dos contenedores más (runner y egress). Encaja con la Fase 7
  (resiliencia).
