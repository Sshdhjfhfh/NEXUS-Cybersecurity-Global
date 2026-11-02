# Auditoría de seguridad: línea base (Fase 0)

Fecha: 2026-11-02. Commit `d218b3b`. Es una revisión interna con herramientas
automáticas y lectura de código. **No es una prueba de intrusión externa ni una
certificación.** NEXUS no tiene certificación ISO 27001 ni SOC 2.

## 1. Cadena de suministro

### Go

- 188 módulos en el grafo (`go list -m all`); 74 tienen código descargado.
- **osv-scanner:** 1 aviso, GO-2026-5932. Dice que `golang.org/x/crypto/openpgp` no tiene mantenimiento (`x/crypto` v0.58.0).
- **govulncheck:** el código de NEXUS **no llama** a ninguna función vulnerable ("vulnerability in modules you require, but your code doesn't appear to call").
- **Acción:** ninguna urgente. Vigilarlo en la CI, que ya ejecuta govulncheck.

### npm (web)

725 entradas en el árbol completo y 669 en el árbol de producción
(`npm ls --omit=dev`).

- **osv-scanner:** 20 paquetes con avisos y 84 avisos en total.
- **Severidad máxima por grupo:** 1 crítica, 32 altas, 44 medias, 7 bajas.

**Origen de los avisos** (verificado con `npm explain`):

- **18 de 20 paquetes** solo llegan a través de `shadcn` (CLI de generación de componentes): `undici`, `hono`, `@hono/node-server`, `@modelcontextprotocol/sdk`, `fast-uri`, `systeminformation`, `ip-address`, `proxy-addr` (CVSS 9.1), `body-parser`, `qs`, `js-yaml`, `deepmerge`, `brace-expansion`, `minimatch`, `postcss-selector-parser` y `browserslist`.
  - `shadcn` está declarado en `dependencies` aunque solo se usa al desarrollar.
- `baseline-browser-mapping` y `source-map-js` también llegan vía `next`, `geist` y `postcss`. Son herramientas de build.
- `zod` (dependencia directa 4.4.3; 3.25.76 vía shadcn): GHSA-mrhx-c7xc-m3w5, agotamiento de memoria. **`src/` no importa `zod`**.

**Exposición real:**
- El producto sirve la web como export estático embebido en el binario Go. Ningún proceso Node corre en producción.
- Ninguno de estos paquetes se ejecuta en el servidor de producción.
- Ningún paquete afectado aparece en el bundle del navegador: `zod` no se importa y el resto son herramientas de CLI o de build.
- El riesgo queda en el equipo del desarrollador y en la CI.

**Acción (Fase 1):**
- Mover `shadcn` a `devDependencies`.
- Quitar `zod` si sigue sin uso, o actualizarlo a una versión corregida.
- Actualizar `next`, `geist` y `postcss` a versiones con `baseline-browser-mapping` y `source-map-js` corregidos.

### SBOM

No se regeneró hoy. El SBOM CycloneDX histórico (1.021 componentes) está desactualizado.
**Acción (Fase 1):** generarlo en la CI para cada release, tanto del binario como de la imagen.

## 2. Licencias

| Ecosistema | Resultado |
|---|---|
| Repositorio | **AGPL-3.0** (`LICENSE`), con unos 449 archivos derivados de ARTEX (AGPL) |
| Go (74 módulos con código) | MIT 34, BSD 34, Apache-2.0 4, ISC 1 y MPL-2.0 1 (`hashicorp/golang-lru/v2`: copyleft débil por archivo, compatible si no se modifica) |
| npm, producción (669) | MIT 576, ISC 50, Apache-2.0 15, BSD 15, BlueOak 2, CC0, 0BSD, Python-2.0 |
| npm, a revisar | `@img/sharp-libvips-*` (LGPL-3.0, binario de `sharp`, solo de build) · `caniuse-lite` (CC-BY-4.0, datos de build) · `geist` (SIL OFL, fuente: permitida embebida con su licencia, ya incluida en `reportassets/`) |

**Riesgo principal: AGPL-3.0.**
- Un SaaS basado en código AGPL debe ofrecer el código fuente correspondiente a los usuarios que interactúan con él por red (sección 13).
- Para vender a empresas y gobiernos, el titular debe elegir una de estas vías:
  - (a) cumplir la AGPL y publicar el código;
  - (b) obtener una licencia comercial del titular upstream;
  - (c) reescribir los componentes derivados.
- **Es una decisión legal del titular con su abogado.** Bloquea el lanzamiento comercial.

## 3. Controles verificados

Implementados en commits previos y cubiertos por la batería E2E (591/591):

- **Aislamiento multitenant:** `org_id` resuelto en el servidor en cada petición; el rol se lee de la BD y no del JWT.
- **Separación de tokens:** un token de portal (`typ=portal`) no autentica la consola y viceversa (con test de regresión).
- **Sesiones en servidor:** revocación, cierre de otras sesiones y aviso por dispositivo nuevo.
- **2FA TOTP de un solo uso, SSO y políticas por organización:** MFA obligatoria, allowlist de IP, caducidad de sesión.
- **Auditoría encadenada por hash,** verificable.
- **Puerta de alcance:** propiedad del objetivo verificada (DNS TXT, archivo `.well-known` o aprobación del operador) más RoE firmadas antes de promover una solicitud al motor.
- **Cabeceras de seguridad en la web embebida:** CSP con hashes, `nosniff`, `X-Frame-Options: DENY`, COOP, HSTS vía Caddy.
- **IP de cliente:** solo se confía en `X-Forwarded-For` desde loopback o proxies declarados.
- **Límite de tasa:** 50 req/s por IP.
- **`jwt.key` persistente** fuera de la capa del contenedor.

## 4. Riesgos abiertos (por prioridad)

| ID | Riesgo | Impacto | Prioridad | Fase |
|---|---|---|---|---|
| R1 | Licencia AGPL en un SaaS comercial | Legal, bloquea la venta | P0 (decisión) | — |
| R2 | El módulo `phantom` (onion/garlic routing, mixnet, ofuscación, esteganografía, tráfico de cobertura) está en el repositorio | Para un proveedor de gobiernos, poseer herramientas de evasión y anonimato es un riesgo reputacional y de cumplimiento. Hoy no se compila en el producto | P1 (decisión) | 1 |
| R3 | No se ha probado de extremo a extremo que el motor RED no toque objetivos fuera de alcance | Ataque no autorizado contra terceros | P1 | 1–2 (prueba negativa en el Cyber Range) |
| R4 | `ethical_boundary` (guard e intercept) tiene ~34 % de cobertura | Un control de seguridad con poca verificación | P1 | 1 |
| R5 | Deadlock entre el esquema y el tráfico (F1 en current-state) | Errores 5xx en despliegues o reinicios | P1 | 1 |
| R6 | Sin backups cifrados off-site ni HA | Pérdida de datos o indisponibilidad | P1 | 7 |
| R7 | Sin pentest externo independiente | Vulnerabilidades no detectadas | P1 | 7–8 |
| R8 | Dependencias npm vulnerables en las herramientas de build | Compromiso del equipo o de la CI | P2 | 1 |
| R9 | Errores de accesibilidad (Biome a11y) | Requisitos WCAG en compras públicas | P2 | 6 |
| R10 | Sin emergency stop para el cliente ni política de retención y destrucción de datos | Contractual y normativo | P2 | 5/7 |
| R11 | Sin modo de "IA solo local" por organización (soberanía de datos) | Requisito de gobiernos | P2 | 5 |

## 5. Lo que esta auditoría no cubre

- Revisión manual línea por línea.
- Fuzzing del API.
- Pruebas de concurrencia con `-race` en todo el árbol.
- La imagen Docker de producción: no se escaneó la imagen, solo los manifiestos.
- La infraestructura de despliegue real, que aún no existe.
