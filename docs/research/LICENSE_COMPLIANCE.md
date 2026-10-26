# Licencias, procedencia y dependencias

2026-10-26. Análisis técnico de obligaciones observadas; las decisiones de
licenciamiento comercial requieren revisar contratos y titularidad por componente.
No se cambió ninguna licencia, atribución ni copyright.

## Hallazgo comercial principal

El `LICENSE` raíz declara AGPL v3 o posterior y reconoce ARTEX/norma. Numerosas
cabeceras identifican código absorbido en `core`, `agents/red`, `db`, `orchestrator`,
`evidence`, etc. `web/LICENSE` es MIT, pero eso no relicencia el backend ni prueba
que cada archivo web tenga esa única procedencia. `docs/ABSORPTION_PLAN.md` describe
copia/adaptación y cambio de imports: esas operaciones no eliminan derechos de
terceros. La frase de independencia del diseño no equivale a independencia jurídica.

AGPL permite uso comercial y propiedad de aportaciones originales, pero impone
obligaciones sobre obras cubiertas y código fuente correspondiente, incluida la
oferta de fuente a usuarios que interactúan remotamente con una versión modificada
en las condiciones de su sección 13. No es correcto prometer que el núcleo derivado
actual pueda distribuirse o prestarse como servicio cerrado sin resolver esos
derechos. Tampoco implica automáticamente publicar todo software independiente de
la compañía. Referencia: [texto AGPL en SPDX](https://spdx.org/licenses/AGPL-3.0-or-later.html).

## Matriz de decisiones

| Componente/fuente | Obligaciones observadas | Decisión |
|---|---|---|
| NEXUS derivado de ARTEX/norma | Mantener avisos; determinar fuente correspondiente, modificaciones y oferta remota aplicable | Registrar commit upstream por archivo y revisar titularidad antes de oferta cerrada. No retirar avisos. |
| MIT: PentAGI, HexStrike, SED y otros | Copyright/licencia; componentes y herramientas auxiliares pueden diferir | Reutilización selectiva posible tras inventario; preferir código original integrado donde aporte independencia. |
| Apache-2.0: Strix, Caldera actual, AgentEvolver, OSS-Fuzz | Licencia, avisos/NOTICE aplicables, marcas y cambios; condiciones de patentes | Conservar metadatos y revisar dependencias/submódulos; no asumir licencia histórica de Caldera. |
| CAI | MIT para partes indicadas; añadidos Alias Robotics restringidos a investigación no comercial | No integrar añadidos para servicios comerciales sin licencia separada. |
| CyberEvolver | Sin licencia general raíz identificada en el SHA consultado; corpus con licencias diversas | Estudiar paper; no copiar implementación sin aclarar permiso. |
| SigmaHQ reglas | DRL-1.1: autores, origen/licencia al compartir y autor en mensajes derivados de coincidencias cuando se suministra | Añadir procedencia por regla y atribución en alertas antes de incorporar contenido externo. |
| pySigma | LGPL-2.1 del motor; plugins pueden diferir | Uso como herramienta de validación separado, documentando cambios/distribución; no asumir que proceso separado elimina obligaciones. |
| Elastic detection-rules | ELv2, incluida limitación sobre ofrecer funcionalidad sustancial como servicio gestionado, además de avisos | Estudiar procesos; revisión comercial previa a copiar reglas o redistribuir servicio. |
| ATT&CK STIX | Licencia MITRE propia permite uso comercial con copyright/licencia; marcas separadas | Fijar versión y conservar aviso. No etiquetarlo como licencia MIT. |
| Atomic Red Team | MIT principal; inspeccionar payloads y auxiliares | Escenarios mínimos con procedencia, sin importar ejecutables en masa. |
| Wazuh | Root describe engine AGPL, bibliotecas listadas duales y resto GPLv2; archivos/submódulos adicionales | Evaluar conector a instalación del operador; no incorporar engine al núcleo cerrado. |
| AutoGen | LICENSE-CODE MIT, docs CC-BY-4.0; API etiqueta solo docs | Inventariar por archivo; además resolver estado de mantenimiento. |
| OWASP WSTG | Licencia de contenido CC-BY-SA-4.0 indicada por el proyecto | Referenciar metodología; preservar atribución y condiciones cuando se reproduzca contenido. |

Fuentes de las licencias y sus SHA están en
[GITHUB_REPOSITORY_RESEARCH.md](GITHUB_REPOSITORY_RESEARCH.md),
[manifiesto de fuentes](evidence/github-source-manifest.json) y
[DRL texto](https://github.com/SigmaHQ/Detection-Rule-License/blob/main/LICENSE.Detection.Rules.md).

## SBOM y límites

Se generó [SBOM.cdx.json](evidence/SBOM.cdx.json), formato CycloneDX 1.6:
183 módulos externos del build list Go y 838 identidades npm versión/paquete
únicas de `web/package-lock.json` y
`resources/skills/api-recon/scripts/package-lock.json`; total 1.021 componentes.
Incluye dependencias de desarrollo, opcionales y herramientas de compilación.
El grafo es agregado al producto, no reconstruye cada arista transitiva.
Se validó contra el esquema oficial CycloneDX 1.6 mediante `jsonschema`
Draft7: cero errores. También se comprobó unicidad de referencias y correspondencia
con los inventarios. Las URLs y hashes de los esquemas están en
[sbom-schema-validation.json](evidence/sbom-schema-validation.json).

Licencias npm provienen del lockfile, sin concluir que esos metadatos sean un
dictamen jurídico. Para Go se registraron nombres/hashes de archivos de licencia
locales en [go-license-inventory.json](evidence/go-license-inventory.json), con
conclusión `NOASSERTION`: encontrar LICENSE no determina su alcance ni compatibilidad.
No se inventaron SPDX para licencias aún sin revisión.

Este SBOM **no cubre** paquetes del SO de la imagen, navegadores/toolchains
instalados globalmente, entornos Python, modelos/datasets, plugins externos ni
todos los archivos absorbidos que ya no figuran como módulos. Es un inventario
inicial de manifiestos, no un SBOM completo del producto distribuido.

## Comprobación de vulnerabilidades

Consulta pública OSV por nombre/ecosistema/versión para los 1.021 componentes,
sin enviar código ni configuración. Respuesta: **29 componentes con avisos**,
4 Go y 25 npm; sin errores de consulta en ese lote. Se conserva el detalle completo
en [osv-manifest-results.json](evidence/osv-manifest-results.json).
Un mismo problema puede tener aliases GO/GHSA/CVE; no sumar IDs como incidentes.
No se ejecutó `govulncheck`, análisis de símbolos ni un escaneo de la imagen activa.

| Ejemplo | Comprobación adicional | Consecuencia |
|---|---|---|
| `klauspost/compress` v1.18.6, GO-2026-5841 | [Aviso Go](https://pkg.go.dev/vuln/GO-2026-5841) afecta `s2.NewDict` antes de v1.18.7 | Revisar import/uso antes de actualizar; no afirmar explotación del backend. |
| `x/crypto` v0.58.0, GO-2026-5932 | [Aviso Go](https://pkg.go.dev/vuln/GO-2026-5932) afecta OpenPGP, sin arreglo conocido en ese paquete | Subir la versión por sí solo no resuelve uso de una API obsoleta; comprobar si se utiliza. |
| `utls` v1.7.1 y `goldmark` v1.7.4 | Coincidencias registradas por OSV en dependencias transitivas del build list | Evaluar alcanzabilidad, versión corregida y regresiones en cambio separado. |
| `next` 16.2.9 | OSV devuelve varios avisos; ejemplo [GHSA-2464-fhr4-w9xw](https://github.com/advisories/GHSA-2464-fhr4-w9xw) | La imagen construye export estático servido por Go; la API de optimización de imágenes de Next no queda demostrada como expuesta por ese despliegue. Revisar cada aviso y uso dev/build. |
| SDK MCP y librerías npm auxiliares | Dependencias de UI/skills y herramientas, algunas de desarrollo | No asumir que todas forman parte del servidor activo ni que las de build carecen de riesgo. |

`go list -deps ./cmd/nexus ./cmd/nexus-scan` no mostró los paquetes afectados
`compress/s2`, `crypto/openpgp`, `utls` o `goldmark` en esas dos compilaciones
predeterminadas. Esto reduce la evidencia de exposición en esas rutas, pero no
certifica builds con otros tags, librerías, herramientas externas o imagen final.
El listado está en [runtime-package-analysis.json](evidence/runtime-package-analysis.json).

Próximo paso: priorizar avisos según superficie real, datos no fiables y uso en
build/runtime; abrir cambios de dependencias pequeños con regresión y reversión.
No se actualizaron todas las dependencias durante una auditoría ni se ocultaron
avisos sin justificación.

## Construcción de tecnología propia

Mantener un registro por componente: propietario, origen URL/SHA, ruta local,
tipo de incorporación (copia/adaptación/interfaz/dato), licencia, modificaciones,
obligaciones y responsable. Para código original, registrar autores/cesiones y
separar los módulos independientes del código derivado. Eliminar una atribución,
traducir o reordenar funciones no convierte una adaptación en creación independiente.

Opciones a evaluar: comercializar servicios compatibles con AGPL; negociar derechos
adicionales con titulares; o crear componentes independientes originalmente, con
especificaciones y procedencia documentadas. No se propone reescribir todo el
producto ni afirmar que un wrapper o microservicio eluda licencias automáticamente.

Aceptación empresarial: fuente correspondiente/avisos disponibles cuando proceda,
SBOM por artefacto de release, inventario de contenido/datasets, revisión de
dependencias y obligaciones aprobadas para cada modalidad comercial. Pendientes:
trazabilidad histórica ARTEX/norma por archivo, licencias Go completas, SO/Python,
submódulos, modelos y validación formal del SBOM.
