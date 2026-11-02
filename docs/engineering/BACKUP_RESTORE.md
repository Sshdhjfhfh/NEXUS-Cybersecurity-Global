# Copias de seguridad y restauración de la base de datos

Toda la información de los clientes vive en PostgreSQL: organizaciones, usuarios,
solicitudes, hallazgos publicados, remediación, objetivos verificados, autorizaciones
(RoE) y el registro de auditoría encadenado. Perderla no es recuperable de otra forma.

## Qué hay

- El servicio `backup` de `docker-compose.yml` ejecuta `scripts/db-backup.sh --loop`:
  una copia cada `BACKUP_INTERVAL_HOURS` (24 h por defecto) en el volumen `nexus_backups`.
- Cada copia es un `pg_dump` comprimido (formato custom). Antes de guardarla se comprueba
  que `pg_restore` puede leerla; una copia fallida o ilegible nunca sustituye a una buena.
- Se conservan `BACKUP_KEEP_DAYS` días (14 por defecto). Solo se borran archivos
  `nexus-*.dump` del propio directorio.
- Los archivos tienen permisos `600`: contienen datos de clientes y hashes de credenciales.

## Lo que falta configurar en producción (no lo hace el stack)

1. **Copia fuera del servidor.** Un volumen en la misma máquina no protege de la pérdida
   del disco ni del servidor. Sincroniza `nexus_backups` a un almacenamiento externo cifrado
   (p. ej. un bucket S3 con cifrado y versionado, o `restic`/`borg` a otro proveedor).
2. **Alertas.** El servicio escribe `ERROR:` en su log si una copia falla. Revisa
   `docker compose logs backup` o conéctalo a tu monitorización.
3. **Prueba de restauración periódica** (al menos una vez al mes, y siempre antes del
   lanzamiento): sigue los pasos de abajo en una base de datos de prueba.

## La clave de cifrado (`jwt.key`): cópiala aparte

Los secretos de 2FA y de SSO se guardan cifrados en la base de datos con una clave
derivada de `jwt.key` (volumen `nexus_keys`, `/app/keys` en el contenedor). **Una copia
de la base de datos sin esa clave no basta**: al restaurarla, nadie con 2FA podría entrar
y el SSO dejaría de funcionar. Además, si la clave cambia, se cierran todas las sesiones.

- Guarda una copia de `jwt.key` en un lugar seguro y **distinto** de las copias de la base
  de datos (por ejemplo, el gestor de secretos de la empresa): quien tenga ambas puede
  descifrar los secretos de 2FA.
- No la regeneres ni borres el volumen `nexus_keys` (`make clean`/`make nuke` lo borran).

```bash
docker compose cp orchestrator:/app/keys/jwt.key ./jwt.key.backup   # luego muévela al gestor de secretos
```

Restaurar la clave (antes de arrancar `orchestrator` con una base de datos restaurada):

```bash
docker compose cp ./jwt.key.backup orchestrator:/app/keys/jwt.key
docker compose restart orchestrator
```

## Copia manual

```bash
docker compose exec backup /bin/sh /usr/local/bin/db-backup.sh
docker compose exec backup ls -lh /backups
```

## Restaurar

Restaura primero en una base de datos de prueba y compara antes de tocar la real.

```bash
# 1. Elegir la copia
docker compose exec backup ls -lh /backups

# 2. Restaurar en una base de datos de prueba
docker compose exec backup createdb nexus_restore_check
docker compose exec backup pg_restore --no-owner --no-privileges -d nexus_restore_check /backups/nexus-AAAAMMDD-HHMMSS.dump

# 3. Comparar (por ejemplo, número de organizaciones y usuarios)
docker compose exec backup psql -d nexus_restore_check -c "select count(*) from orgs; select count(*) from portal_users;"
```

Para sustituir la base de datos real (pérdida o corrupción confirmada):

```bash
docker compose stop orchestrator
docker compose exec backup dropdb nexus
docker compose exec backup createdb nexus
docker compose exec backup pg_restore --no-owner --no-privileges -d nexus /backups/nexus-AAAAMMDD-HHMMSS.dump
docker compose start orchestrator
```

Después de restaurar, comprueba la integridad del registro de auditoría desde la consola
(Auditoría → Verificar cadena): una restauración correcta mantiene la cadena intacta.

## Verificado

2026-10-30, en el entorno de desarrollo: copia de 768 KB, restaurada en una base de datos
nueva con los mismos conteos que la original (orgs 286, portal_users 474, portal_audit 2220,
portal_invites 236, portal_sessions 280); caducidad de copias antiguas y fallo de conexión
(sale con error, conserva las copias anteriores, no deja archivos parciales).
