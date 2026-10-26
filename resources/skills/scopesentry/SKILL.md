---
name: scopesentry-mcp
description: Consulta e importa activos de una instancia ScopeSentry conectada mediante MCP, respetando el alcance de la tarea.
license: AGPL-3.0
compatibility: Servidor ScopeSentry MCP configurado y asignado al agente por el operador.
---
# Integración de ScopeSentry con Nexus

Esta habilidad no crea una conexión por sí sola. El operador debe configurar el MCP, sus credenciales y su visibilidad en Nexus. Usa los nombres y esquemas de las herramientas realmente disponibles; no inventes endpoints ni parámetros.

1. Comprueba que el MCP autorizado está disponible. Si falta, indica qué conexión necesita el operador.
2. Consulta los proyectos y tareas permitidos y selecciona los del alcance actual.
3. Consulta activos con filtros y paginación. Verifica el esquema de `list_assets` y `count_assets` antes de usarlos.
4. Clasifica dominios, subdominios, IP, puertos, sitios y endpoints. Conserva procedencia y evita duplicados.
5. Incorpora únicamente los activos autorizados mediante las herramientas de Nexus. Una importación no amplía el alcance de las tareas existentes.
6. Crear o ejecutar escaneos requiere una instrucción explícita y controles del laboratorio. Consultar activos no implica autorización para escanearlos.
7. Informa en español del resultado, cantidad de activos, filtros y limitaciones. No incluyas claves API ni material de autenticación.

Origen: ARTEX/scopesentry, adaptado para Nexus. Se conservan créditos y licencia.
