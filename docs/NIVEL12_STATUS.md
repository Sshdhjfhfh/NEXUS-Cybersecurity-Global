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

## Pendiente para integrar Nivel 1–2

- Conectar los adaptadores de autorización reales del host y registrar los
  trabajadores en el runtime de `cmd/nexus`/orchestrator. Si faltan controles,
  los constructores rechazan la configuración; no existe un modo permisivo.
- Conectar observaciones y resultados persistidos con BLUE y predicción.
- Proveer imágenes locales probadas y configuración propia de cada aplicación.
  Los perfiles son planes aislados, no appliances listos ni importadores de VMs.
  Las máquinas originales de Metasploitable/VulnHub y escenarios AD necesitan
  adaptadores de virtualización cuando no sean contenedores compatibles.
- Medir precisión y rendimiento con sesiones de laboratorio reales. No se afirma
  todavía el objetivo de >60% de precisión ni >40% de mejora de swarm.
- `watchdog/` está vacío y no se modificó: la regla absoluta del usuario lo
  declara inmutable. No se añade una implementación alternativa que lo sustituya.

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
