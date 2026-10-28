# NEXUS — Endurecimiento de seguridad

Medidas de defensa aplicadas (alineadas con OWASP ASVS/Top 10) para que NEXUS sea
robusto frente a atacantes y no filtre información. Todo en `nexus-real`, verificado.

## Principio clave: la autorización vive en el servidor
Manipular rutas en el navegador **no** da acceso. El API aplica JWT en
`requireAuth` (HS256, clave de 32 bytes fuera del workspace, expiración 7 días,
método verificado). El middleware del frontend es solo UX; la frontera real es el
backend. Rutas desconocidas → **404 de marca**, no exponen la consola.

## Aplicado
- **Cabeceras de seguridad del API** (`secureHeaders`): `X-Content-Type-Options: nosniff`,
  `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Cross-Origin-Resource-Policy:
  same-origin`, `Cache-Control: no-store`. (La web ya tenía HSTS, nosniff, etc.)
- **CSP con nonce (anti-XSS)**: el middleware genera un nonce por petición y fija la
  `Content-Security-Policy` **estricta en producción** (`script-src 'self' 'nonce-…'
  'strict-dynamic'`, `object-src 'none'`, `frame-ancestors 'none'`, `upgrade-insecure-requests`).
  En dev va en **report-only** para no romper el HMR. El script inline de arranque usa el nonce.
- **CORS con allowlist** (`corsHandler`): se elimina el `Access-Control-Allow-Origin: *`.
  Por defecto **no** se permite cross-origin; se configura con `NEXUS_CORS_ORIGINS`.
- **Anti-fuerza-bruta en login**: bloqueo por IP tras 5 fallos en 15 min, por 15 min
  (`429 + Retry-After`). Verificado.
- **Rate limiting global por IP** (token bucket) + **load shedding** (503) — ya existentes.
- **Sin fuga de información**: los errores internos se registran en el servidor pero al
  usuario se le devuelve un mensaje genérico (se quitó `err.Error()` de las respuestas de
  auth; los panics devuelven `500` genérico). `X-Powered-By` desactivado.
- **`/.well-known/security.txt`** (RFC 9116) apuntando a la política de divulgación `/legal/vdp`.
- **Contraseñas**: bcrypt; validación de longitud (8–72). Cookie de sesión `SameSite=Lax`.
- **Autenticación de API**: Bearer token; SSE por `?token=` sobre mismo origen.
- **MFA (2FA TOTP, RFC 6238)** para la consola: enrolamiento en `/system/mfa`, código de 6
  dígitos en el login, **secreto cifrado en reposo** (AES-256-GCM con clave derivada de la
  clave JWT). Endpoints `/api/auth/mfa/{status,setup,enable,disable}`. Unit-tested.

## Pendiente (recomendado, honesto)
- **Verificar la CSP estricta con un build de producción** (`next build && next start`) y
  resolver cualquier violación residual antes de publicar.
- **Revocación de tokens** (lista de revocados / rotación de clave) y expiración más corta.
- **Defensa CSRF explícita** para métodos que mutan si se admite auth por cookie
  (hoy el API usa Bearer; la cookie es para el guard de rutas).
- **Escaneo en CI**: `gitleaks` (secretos) + `osv-scanner` (dependencias) + `govulncheck`.
  (En esta sesión no estaban en el PATH; integrarlos en el pipeline.)
- **Auditoría de endpoints de archivos** (workspace) contra path traversal y SSRF.
- **Pruebas de seguridad** automatizadas (ZAP/nuclei) sobre un entorno de staging autorizado.
- **WAF / protección DDoS** a nivel de infraestructura para "muchos usuarios".

> No se declara cumplimiento de ASVS ni certificaciones sin pruebas trazables; esto es una
> base sólida, no un sustituto de una auditoría externa.
