---
name: playwright-cli
description: Usa un navegador real para observar aplicaciones, comprobar interfaces y reproducir interacciones autorizadas con evidencias.
license: AGPL-3.0
compatibility: playwright-cli, Chromium y Node.js incluidos en la imagen Nexus.
---
# Navegador de Nexus

Usa Playwright para trabajar con la interfaz real del objetivo autorizado. Comparte evidencias con los otros agentes mediante los mecanismos existentes de Nexus.

## Flujo básico

1. Consulta `playwright-cli --help` para verificar las opciones de la versión instalada.
2. Abre la URL del alcance: `playwright-cli open URL`.
3. Obtén el estado con `playwright-cli snapshot`. Las referencias de elementos pertenecen a esa captura; actualízala después de una navegación o cambio importante.
4. Usa `click REFERENCIA`, `fill REFERENCIA TEXTO`, `type TEXTO`, `press TECLA` y las operaciones documentadas por `--help`.
5. Comprueba el resultado real antes de afirmar que una acción tuvo éxito. Usa capturas de pantalla cuando aporten evidencia visual.
6. Cierra el navegador al terminar con `playwright-cli close`.

## Sesiones y coordinación

Usa una sesión distinta por tarea mediante la opción `-s=NOMBRE` de la CLI. No compartas cookies ni perfiles entre tareas. Guarda capturas, trazas y resultados dentro del directorio autorizado de cada tarea. Para varias pestañas, consulta las órdenes `tab-list`, `tab-new`, `tab-select` y `tab-close`.

## Límites

No realices envíos de formularios, compras, borrados o cambios de configuración sin autorización para esa acción. Respeta las aprobaciones de Nexus. No uses respuestas simuladas como prueba de funcionamiento del servidor. No muestres cookies, credenciales o datos personales en los informes.

Redacta resultados en español; conserva selectores, URLs y mensajes de error exactos cuando sean necesarios para reproducir el problema.

Consulta los documentos de `references/` para sesiones, trazas, almacenamiento y pruebas. Origen: ARTEX/playwright-cli; adaptación de Nexus con atribución conservada.
