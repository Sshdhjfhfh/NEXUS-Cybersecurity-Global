# Continuación de Nivel 1–2

La entrega de Nivel 0 está en el commit `6964ef9`, publicado en `main`.
Esta continuación añade una base funcional y probada; no declara terminados los
~17.000 LOC estimados ni la integración completa de Nivel 1–2.

## Implementado

| Paquete | Funcionalidad disponible |
|---|---|
| `agents/blue` | SENTINEL pasivo: analiza respuestas aportadas como evidencia, detecta HTTP sin TLS, ausencia de HSTS/CSP/nosniff, cookies débiles y fallos de autenticación. Cada hallazgo conserva el ID de evidencia. |
| `agents/prediction` | Modelo Beta(1,1) por clase, intervalos Wilson, al menos 50 sesiones antes de marcar una estimación lista, rechazo de feedback duplicado, persistencia atómica y validación del snapshot. |
| `swarm` | Registro obligatorio de ocho roles distintos, ejecución DAG con concurrencia acotada, serialización por agente, cancelación, aislamiento de fallos, validación de dependencias y autorización previa por trabajo. |
| `lab/network` | Alcance explícito RFC1918/ULA. Rechaza prefijos públicos o demasiado amplios, hostnames, direcciones IPv4 mapeadas y zonas IPv6; no realiza DNS ni tráfico de red. |
| `lab` | Puerta de autorización que requiere identidad, OPSEC, ética y ABM. Generación de planes Compose para DVWA, Metasploitable, VulnHub y AD con red interna, imágenes fijadas por digest, sin puertos publicados ni privilegios ni descargas implícitas. |

Los roles de swarm son reconocimiento, mapeo, análisis, validación, defensa,
predicción, informe y coordinación. El coordinador exige implementaciones
explícitas de los trabajadores; no introduce herramientas ofensivas por defecto.
Todos los mensajes internos nuevos se escribieron en inglés.

## Integración pasiva en el servidor

`orchestrator/level12` conecta los ocho trabajadores en un DAG de análisis de
tráfico ya registrado. No realiza reconocimiento activo ni peticiones al objetivo.
El servidor crea el servicio durante el arranque, con ejecución cerrada por defecto.

- PostgreSQL conserva informes sin cuerpos ni valores de cabeceras, un digest de
  evidencias y evaluaciones por clase. La repetición de evidencias idénticas y el
  feedback duplicado se rechazan mediante restricciones únicas. La predicción se
  reconstruye a partir de los resultados guardados y queda separada por tarea.
- La fuente de evidencias solo acepta direcciones IP privadas dentro del alcance
  del gate y presentes en los activos de la tarea. Comprueba que cada ID de tráfico
  corresponde al objetivo solicitado. Usa el almacén global existente: la atribución
  es por host asociado a tarea, no una prueba de que esa tarea generó la captura.
- Se verifican los controles antes de ejecutar cada trabajador, antes de analizar
  cada observación y antes de persistir o evaluar. Las ejecuciones del servicio se
  serializan para evitar identidades simultáneas entre schedulers.
- `/function/lab` muestra disponibilidad, bloqueo, selección de tarea, historial y
  estimaciones en español. El formulario y la evaluación quedan deshabilitados
  cuando faltan controles. Las estimaciones se ocultan hasta reunir 50 evaluaciones
  por clase; este umbral no demuestra independencia estadística ni precisión.

Rutas autenticadas: `GET /api/lab/status`, `GET|POST /api/lab/tasks/{id}/runs`,
`GET /api/lab/tasks/{id}/runs/{runID}`, `GET /api/lab/tasks/{id}/predictions` y
`POST /api/lab/tasks/{id}/runs/{runID}/feedback`. Las peticiones se validan de forma
estricta, con límites de tamaño, resultados y duración para el análisis.

## Pendientes reales de Nivel 1–2

- Conectar adaptadores reales de identidad ANS/NHI, OPSEC, ética y ABM a través de
  `Server.ConfigureLevel12`, con ocho identidades distintas. No hay un endpoint ni
  una variable que permita omitir controles. La configuración disponible actualmente
  no los aporta y el estado de producción devuelve `ready: false`.
- OPSEC y el evaluador ético requerido no están disponibles para esta conexión.
  No se sustituyen por funciones que permitan todo. El arranque integrado por sí
  solo no completa la ejecución de extremo a extremo.
- Proveer imágenes locales probadas y configuración propia de cada aplicación.
  Los perfiles son planes aislados, no appliances listos ni importadores de VMs.
  Metasploitable/VulnHub y AD necesitan adaptadores de virtualización cuando no
  sean contenedores compatibles.
- Medir precisión y rendimiento con sesiones de laboratorio reales. No se afirma
  el objetivo de >60% de precisión ni >40% de mejora de swarm.
- `watchdog/` sigue vacío e intacto por la regla de inmutabilidad del usuario.

## Validación

```sh
go build ./...
go test -race ./lab ./lab/network ./agents/blue ./agents/prediction ./swarm
# La suite con PostgreSQL se ejecuta contra una instancia temporal, nunca la de uso.
NEXUS_PG_DSN="$NEXUS_TEST_PG_DSN" go test -p 1 -count=1 ./...
```

La referencia de tests reveló además dos omisiones anteriores: un test de
reintentos omitía el cliente HTTP que espera la configuración normalizada, y el
resumen de grafo no incluía el alcance de empresa que exigía su prueba existente.
Se corrigió el cliente del fixture y se restauró el contexto de alcance de solo
lectura, acotado a 8.000 caracteres. No se amplía ninguna autorización.

Las pruebas de atribución de empresa también conservaban una regla CIDR entre
invocaciones. Se añadió limpieza explícita del fixture y la validación final usa
una base recién creada, además de ejecutar los paquetes en serie.
El fixture de lectura de actividades heredadas también inicia ahora el servicio
con contexto cancelado y espera a sus escritores antes de limpiar directorios;
sus pruebas no necesitan ejecutar agentes en segundo plano. La base del fixture
se cierra después de borrar sus tareas, no antes.
Los demás fixtures HTTP que usaban contextos sin cierre ahora comparten un helper
que cancela sus servicios, detiene los runtimes y espera a los escritores. Esto
impide que un notificador de una prueba anterior entregue mensajes durante la
siguiente. No cambia el ciclo de vida del servidor de producción.

Resultado final de esta entrega: `go build ./...`, pruebas nuevas con `-race`,
`go test -p 1 -count=1 ./...` sobre PostgreSQL temporal limpio y build Docker
correctos. La web se volvió a comprobar con Chromium en 20 rutas: sin texto chino,
sin claves sin resolver y sin errores JavaScript. Los módulos inmutables quedaron
sin cambios. Esto valida la base implementada, no los pendientes de integración.

La integración posterior añade pruebas del pipeline con autorización explícita de
fixture, cancelación y concurrencia; pruebas PostgreSQL de reinicio, aislamiento,
repetición y feedback concurrente; y pruebas HTTP de autenticación, alcance,
evidencias incompatibles, JSON estricto y bloqueo sin controles. Esto no sustituye
la validación de los adaptadores reales pendientes.

Validación de esta integración: build Docker completo (incluido TypeScript),
`go build ./...`, suite Go completa en PostgreSQL temporal limpio y pruebas del
pipeline con `-race` correctos. Chromium verificó 22 rutas, incluida la entrada
principal y el laboratorio: HTTP correcto, sin chino visible, claves sin resolver
ni errores JavaScript. El estado real del laboratorio se comprobó bloqueado y su
botón de análisis deshabilitado. Los dos contenedores quedaron saludables.

## Adaptadores de autorización ANS/NHI/ABM

`lab/security` aporta adaptadores explícitos de las interfaces existentes.
`Server.ConfigureLevel12Security` instala un gate únicamente cuando recibe los
servicios, una CA de confianza, evaluadores OPSEC/éticos y ocho identidades.
El arranque normal sigue cerrado: estos evaluadores no están disponibles todavía.

- Identidad: registro ANS activo, permisos exactos y alcance, identidad NHI vigente
  y no revocada, certificado firmado por la CA fijada, CN/OU/DID coherentes y JWT
  verificado con emisor, destinatario, sujeto, vencimiento y permisos coherentes.
  Los tres conjuntos de permisos y activos deben permitir la acción. No se aceptan
  comodines ni DNS en los permisos de activos; solo IPs y CIDRs privados.
- `PersistentDirectory` consulta ANS en PostgreSQL por acción. `PersistentIdentities`
  lee los permisos y el ciclo de vida actuales de NHI; obtiene el token del manager
  emisor solo si sus credenciales coinciden. Una actualización externa de permisos
  o cuarentena se aplica sin esperar a que expire una caché.
- ABM: requiere estar habilitado e inmutable, con parada y cuarentena automáticas.
  Rechaza agentes detenidos/cuarentenados y violaciones altas o críticas aunque el
  monitor todavía no haya cambiado el estado. Vuelve a verificar identidad antes
  de observar la acción. No modifica la implementación inmutable de ABM.
- Se corrigió `nhi_manager.Store.SaveIdentity`: persistía permisos y certificados,
  pero omitía `revoked_at` y `revoke_reason`. Ahora conserva la primera revocación
  incluso si después guarda una copia antigua de la identidad.

La conexión usa los servicios existentes; no crea identidades automáticamente ni
proporciona decisiones OPSEC/éticas permisivas. Las aprobaciones de fixture están
limitadas a los tests aislados. El guard heredado de interceptación no se considera
un evaluador ético obligatorio porque permite acciones sin reglas configuradas.

Siguen pendientes la composición de producción, la renovación y rotación operativa
de certificados y claves, y la recuperación de estados ABM. Los controles OPSEC
y éticos siguen sin implementación conectable bajo la regla de inmutabilidad.
El manager no se activa automáticamente sobre identidades persistidas.
Estas limitaciones impiden declarar la autorización de producción completada.

Validación de los adaptadores: pruebas con `-race` de ANS/NHI/ABM y del pipeline
HTTP con ocho identidades reales; retirada de permisos y cuarentena persistidas
fuera de caché; revocación persistida y preservada frente a guardados antiguos.
La suite Go completa pasó en PostgreSQL temporal limpio y el build Docker,
incluida la web, terminó correctamente. Los módulos inmutables siguen intactos.
Chromium verificó nuevamente 22 rutas después del despliegue, sin chino visible,
claves sin resolver ni errores JavaScript; confirmó el estado de ejecución cerrado
y el botón de análisis deshabilitado. Se conservó la clave de sesión del servidor
durante esta actualización para mantener los accesos existentes.


## Credenciales persistentes y decisiones firmadas

`nhi_manager.Config.KeyDir` permite conservar la CA, la clave JWT y la clave privada
Dilithium en un único archivo versionado fuera del espacio navegable. El directorio
requiere permisos privados y el archivo usa `0600`. La publicación es atómica, sin
sobrescritura entre inicializadores; se rechazan enlaces simbólicos finales,
archivos públicos, material corrupto y claves incompatibles. Un archivo dañado no
provoca una regeneración silenciosa. Sin `KeyDir`, se mantiene el comportamiento
efímero heredado para llamadores existentes; no equivale a despliegue persistente.

Las claves públicas de auditoría usan un ID derivado de su hash completo y no pueden
sobrescribirse con otra clave. Esto evita perder la verificabilidad de los nuevos
registros al reiniciar; no recupera claves históricas que ya se hayan sobrescrito.
Las marcas temporales se firman en UTC con precisión de microsegundos y se leen en
UTC, conforme a la precisión de PostgreSQL. Los registros nuevos sobreviven a una
ida y vuelta por la base de datos sin alterar el contenido firmado.

`security.NewAuditRecorder` y `lab.NewAuditedGate` registran cada decisión de
autorización antes de devolverla. La composición segura exige este recorder: si
falla el registro, no se autoriza la acción. Los rechazos y las cancelaciones usan
un contexto de auditoría independiente limitado a tres segundos. Los actores no
registrados quedan identificados expresamente como desconocidos, sin inventarles
un DID. Solo se guarda el digest de la acción y motivos fijos, nunca errores del
proveedor ni credenciales. Esto registra decisiones de autorización, no los
resultados posteriores de todos los trabajadores.

Las pruebas cubren reinicios con los mismos certificados y tokens, renovación de
token después del reinicio, revocaciones persistentes, verificación de firmas tras
leer PostgreSQL, protección de claves históricas, inicialización concurrente y
bloqueo al fallar la auditoría.

Validación de esta entrega: `go build ./...`, suite completa en PostgreSQL temporal
limpio y pruebas de credenciales, auditoría y composición segura con `-race`
correctos. Docker compiló y el despliegue conservó la clave de sesión existente.
Chromium verificó 22 rutas sin chino visible, claves sin resolver ni errores JS;
confirmó el laboratorio bloqueado. No se modificaron módulos inmutables.
