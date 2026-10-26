---
name: api-recon
description: Inventaría rutas y API de una aplicación autorizada mediante análisis estático y observación del navegador, con evidencias reproducibles.
license: AGPL-3.0
compatibility: Python 3, Node.js, Chromium y puppeteer-core; aplicación dentro del alcance autorizado.
---
# Inventario de API de Nexus

Usa esta habilidad para comprender las rutas y llamadas de una aplicación del laboratorio. Trabaja con el planificador y comparte resultados mediante las herramientas de hechos y evidencias de Nexus.

## Alcance y evidencias

- Confirma la URL y el alcance de la tarea antes de navegar. No sigas destinos externos ni amplíes permisos.
- Distingue **rutas observadas**, **rutas inferidas del código** y **hallazgos reproducidos**. Una ruta encontrada no demuestra una vulnerabilidad.
- El inventario no autoriza acciones que modifiquen datos. Excluye cierre de sesión, eliminación, subida, importación y otras operaciones con efectos.
- Usa únicamente credenciales y sesiones proporcionadas para la tarea. Los datos simulados del navegador sirven para estudiar el frontend y deben identificarse como simulaciones: no acreditan acceso real al servidor.
- No incluyas cookies, tokens ni contraseñas en informes. Conserva las evidencias sensibles solo en el almacenamiento autorizado.
- No edites los scripts instalados. Si necesitas adaptarlos, copia los archivos necesarios al directorio de trabajo de la tarea y registra las diferencias.

## Procedimiento

1. Clasifica la aplicación: páginas tradicionales, SPA, autenticación requerida y recursos públicos disponibles.
2. Descarga y analiza únicamente recursos del alcance. Extrae rutas, métodos, parámetros, formularios, módulos y dependencias desde HTML y JavaScript.
3. Crea un inventario inicial con origen y nivel de confianza. Conserva el archivo y fragmento donde aparece cada ruta.
4. Si se autorizó navegación, abre la aplicación con Playwright. Registra las solicitudes realmente realizadas, sus respuestas y las acciones que las originaron.
5. Para una SPA, visita rutas conocidas y comprueba cobertura de menús y submódulos. Evita bucles y limita cantidad de páginas y duración.
6. Los scripts de runtime permiten estudiar comportamiento del frontend con respuestas simuladas. Usa el modo sin reenvío para esa exploración y separa sus resultados del tráfico real.
7. Fusiona el inventario estático y el observado. Deduplica por método y ruta conservando las fuentes. Marca lo que no pudo verificarse.
8. Devuelve al planificador un resumen con cobertura, evidencias, límites y próximos pasos. Guarda hechos mediante las herramientas existentes; no declares vulnerabilidades sin reproducción.

## Recursos

Los scripts están en `${SKILL_DIR}/scripts`. Consulta [la referencia](reference.md) antes de ejecutarlos. La imagen Nexus incluye sus dependencias; no descargues paquetes durante una tarea sin necesidad y autorización.

Mantén las claves técnicas de JSON y los nombres de las herramientas. Escribe explicaciones e informes en español.

Origen: habilidad api-recon de ARTEX, adaptada para Nexus. Se conservan los créditos y la licencia.
