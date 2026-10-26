# Validación de ZERO_day

Todos los archivos añadidos están dentro de `zero_day/`. Se retiraron los cambios
de credenciales iniciados antes de precisar el alcance; `git status --short` solo
muestra este directorio nuevo. La imagen y los servicios existentes de Nexus no
se actualizaron.

## Resultados

- Imagen `nexus-zero-day:local` construida correctamente con AFL++ 4.04c.
- Once pruebas Python correctas: guía estricta, límites, escapes del diccionario,
  rechazo de semillas fallantes, clasificación de crashes/cuelgues, informes de
  error, rutas de sesión, escape HTML y adaptador local de Ollama.
- Campaña real de 15 segundos: **49.031 ejecuciones**, una entrada fallante encontrada
  y reproducida. Las semillas iniciales no fallaban.
- Reproducción posterior independiente: la misma entrada produjo `SIGABRT` (6).
- Cancelación comprobada con `docker stop`: informe `cancelled`, código de salida
  130, sin perder los archivos de la sesión.
- Configuración del contenedor verificada con `docker inspect`: red `none`, ningún
  puerto publicado, raíz de solo lectura, sin privilegios, capacidades eliminadas,
  `no-new-privileges`, 512 MB, un CPU y 64 procesos.
- Nexus permaneció con PostgreSQL y orchestrator saludables, en el puerto 8789.

Sesión de prueba: `361c5dec5f4c45e6852be42ffedf40f3`.

SHA-256 de la entrada reproducida:
`0f1ffa4807681e92ca059c5a51e9a9eee3b3cab8ab0e4e176231830d6f757b3d`.

El informe real está en
`artifacts/361c5dec5f4c45e6852be42ffedf40f3/report.html`.
Los artefactos se conservan localmente y están excluidos de Git.

## Límites comprobados

El fallo del parser es sintético e intencional. Esta prueba valida el circuito de
instrumentación, fuzzing, evidencia y reproducción; no acredita descubrimiento de
vulnerabilidades nuevas, rendimiento en parsers reales ni integración autónoma.

En la primera prueba Ollama no respondía en el loopback de Linux y el demo usó
la guía incluida. La continuación encontró el servidor en Windows y añadió un
transporte local para WSL. La generación real con `qwen3:1.7b` produjo una guía
validada que se ejecutó desde la terminal Nexus: **45.073 ejecuciones en 15 segundos
y una entrada fallante reproducida**, sesión `cd6270ee319749048f1402db9bf7b51b`.
Se conserva su informe en `artifacts/cd6270ee319749048f1402db9bf7b51b/report.html`.
Los intentos iniciales de generación con más contexto excedieron el tiempo o el
presupuesto de salida; no se presentan como pruebas exitosas.
