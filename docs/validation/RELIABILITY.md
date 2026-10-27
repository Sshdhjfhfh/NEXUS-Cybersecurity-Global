# Fiabilidad y escalabilidad del backend NEXUS

Resumen de las defensas del servicio para **no caerse** y **no saturarse**, y la base
para **muchos usuarios**. Todo aplicado en `nexus-real` y verificado en local.

## Que ya estaba (base de ARTEX)
- **Apagado ordenado** (graceful shutdown) ante señal (`cmd/nexus/main.go`): drena conexiones (5s).
- **ReadHeaderTimeout 10s**: corta ataques tipo slowloris en las cabeceras.
- **Pool de BD con cola**: `SetMaxOpenConns/Idle`, `ConnMaxLifetime 30m`, `ConnMaxIdleTime 5m`.
  Bajo carga, las consultas **esperan una conexión libre** en vez de reventar PostgreSQL con
  `too many clients`.

## Lo que añadimos (este sprint)
Middleware en `orchestrator/server/middleware.go`, cadena:
`recoverer → requestID → accessLog → loadShed → cors → requireAuth → mux`.

- **Recuperación de panics** (`recoverer`): un `panic` en cualquier handler se convierte en
  `500` + stack con `id`; **el proceso nunca cae** por una petición mala.
- **Request ID** (`requestID`): `X-Request-ID` por petición (honra el entrante), para
  correlacionar logs y errores.
- **Log de accesos útil** (`accessLog`): registra solo lo que importa — errores (`>=400`) y
  lentos (`>1s`) — para tener señal sin inundar disco.
- **Load shedding** (`loadShed`): limita las peticiones concurrentes; al superar el tope
  devuelve **503 + Retry-After** en vez de acumular memoria/conexiones. **Las SSE** (`…/stream`,
  `…/events`) quedan **exentas** (nunca ocupan un cupo), así que el streaming no se rompe.
- **Timeouts de servidor**: `IdleTimeout 120s` + `MaxHeaderBytes 1 MiB` (sin `WriteTimeout`
  para no cortar SSE ni subidas grandes).
- **Readiness** `GET /api/ready`: 200 solo si la BD responde (ping 2s); el balanceador deja de
  enviar tráfico a una instancia sin BD. `GET /api/health` sigue siendo *liveness*.

## Palancas por entorno (escalar sin recompilar)
| Variable | Default | Para qué |
|---|---|---|
| `NEXUS_MAX_INFLIGHT` | `512` | Tope de peticiones concurrentes (0 = sin límite). |
| `NEXUS_PG_MAX_CONNS` | `32` | Conexiones máximas del pool (1..200). |

## Verificado (local)
- `recoverer/requestID/loadShed/statusWriter/isStreaming` con tests unitarios (sin BD) — PASS.
- `/api/health` 200 · `/api/ready` `{"ok":true}` · cabecera `X-Request-ID` presente · POST de
  lead 201 (la cadena no rompe el flujo). `go build ./...` + `go vet` limpios.

## Siguiente para MUCHOS usuarios (recomendado, pendiente)
- **Escalado horizontal** detrás de un balanceador (el servicio es apto si el estado va en
  BD; revisar que no haya estado en memoria por-instancia crítico para multiusuario).
- **Métricas** (`/metrics` Prometheus: latencia, throughput, errores, in-flight, pool) y
  **tracing**; hoy hay logs con `request id`.
- **Rate limiting por IP global** (hoy solo en captación de leads) y **circuit breakers** hacia
  dependencias externas.
- **PostgreSQL**: réplicas de lectura / `PgBouncer` si el nº de instancias crece; índices por
  consulta caliente; backups y pruebas de restauración.
- **Pruebas de carga** (k6/Vegeta) para fijar `NEXUS_MAX_INFLIGHT`/`NEXUS_PG_MAX_CONNS` con datos.
- **CSP enforced + MFA** (ver `LAUNCH_CHECKLIST.md`).
