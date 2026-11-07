# Diseño: aislamiento de las herramientas de los agentes (R13)

Estado: **en implementación** (2026-11-07). Pasos 1 y 3 completos en código
(protocolo/servidor/cliente, namespace de montaje y cableado en `cmd/nexus`); falta el
despliegue en contenedor con su prueba negativa (bloqueado por Docker–WSL) y los pasos 2,
4 y 6. Bloquea los encargos contra objetivos reales de clientes.

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

El paquete `core/runner` implementa un servidor (Go, el mismo módulo) que escucha en un
socket Unix compartido (`nexus_runner_sock`), no en TCP. Así no queda expuesto en ninguna
red. Protocolo real implementado (`core/runner/protocol.go`):

- Una petición `Request{Command, Dir, Env, TTY, Rows, Cols}` por conexión, enmarcada
  (`[tipo][len][payload]`, tope de trama 1 MiB). El cliente transmite stdin y recibe
  stdout/stderr, el código de salida, y reenvía señales y cambios de tamaño de terminal.
  En modo TTY el servidor abre un PTY para `shell_session.go`.
- **Autenticación por `SO_PEERCRED`, no por token firmado.** El servidor lee el uid real
  del proceso que se conecta por el socket y rechaza cualquiera que no sea el esperado
  (`PeerUID`). Combinado con los permisos del propio socket, esto evita Ed25519: no hay
  clave que custodiar ni que pueda filtrarse, y el orquestador y el runner comparten uid.
- **Confinamiento del directorio en el servidor, no por el cliente.** `resolveRoot`
  (`server_unix.go`) resuelve enlaces simbólicos con `EvalSymlinks` y solo acepta
  `tasks/<x>` o `sessions/<x>` bajo `WorkBase`; rechaza fugas por `../` y por symlink. El
  comando corre con `cwd` = esa raíz y `HOME` = esa raíz. Un comando no puede pedir
  ejecutar en el directorio de otra tarea aunque construya la ruta en tiempo de ejecución.
- **Mismo uid + namespace de montaje por comando, no un uid por tarea.** Se descartó un
  uid por tarea: los archivos que el orquestador crea en el directorio de la tarea (las
  herramientas Read/Write/Edit trabajan sobre ellos) quedarían ilegibles para otro uid
  sin ACLs. En su lugar, el hook `Isolate(cmd, root)` monta un namespace por comando que
  solo expone la raíz de esa tarea en su ruta absoluta; el runner necesita `CAP_SYS_ADMIN`
  para montar, y los procesos hijo sueltan todas las capacidades.
- Al desconectarse el cliente, el servidor mata el grupo de procesos (guard contra reuso
  de pgid tras `wait`), y solo reenvía señales INT/TERM/HUP/QUIT/KILL.

En `core/tool` se introdujo la interfaz `Launcher` (`LocalLauncher` hoy, `runner.Launcher`
para el runner) que reemplaza las tres llamadas a `exec.CommandContext`. El orquestador se
convierte en el cliente por el subcomando `runner-exec` (`runner.ClientMain`). `Local`
queda solo para desarrollo y se rechazará al arrancar cuando `NEXUS_PORTAL_BASE_URL` sea
`https://` (modo producción, igual que el watchdog de operaciones) — pendiente, paso 6.

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
3. 🟢 Servicio runner, protocolo, cliente e **isolación**. **Hecho (`core/runner`,
   commits f281bce + f078063; cableado en `cmd/nexus`):** protocolo enmarcado, servidor
   con PTY, verificación `SO_PEERCRED`, confinamiento de directorio con rechazo de
   symlinks, muerte del grupo al desconectar, reenvío de señales y de tamaño de terminal;
   `MountIsolate` (namespaces PID+montaje+usuario, `/proc` fresco, enmascarado de rutas,
   `no_new_privs`) con el subcomando `runner-init`; subcomandos `runner-exec`/`runner-serve`
   y `SetLauncher` (cuando `NEXUS_RUNNER_SOCKET` está definido) cableados en `cmd/nexus`.
   Tests con `-race`; `TestMountIsolateSeparatesPIDAndProc` cubre la propiedad (se salta si
   el kernel no tiene user namespaces sin privilegios); verificado end-to-end con el
   binario real. **Pendiente:** los **tests de integración en contenedor** (bloqueados por
   la integración Docker–WSL). Prueba negativa objetivo: un comando no puede leer
   `/app/keys`, ni `/proc/<pid>/environ` del orquestador, ni conectar con Postgres.
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
  compartido con **mismo uid** y un **namespace de montaje por comando** (ver 3.2) es el
  punto de partida, ya decidido e implementado en el protocolo.
- **Coste operativo:** dos contenedores más (runner y egress). Encaja con la Fase 7
  (resiliencia).
