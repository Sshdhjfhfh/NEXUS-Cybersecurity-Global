# Validación de la demostración NEXUS

Fecha de revisión: 25 de noviembre de 2026, zona America/Lima.

## Resultado observado

- Antes de corregir: cuatro respuestas HTTP 200, dos accesos indebidos confirmados y dos alertas.
- Después de corregir: respuestas 200, 200, 404 y 403; cero accesos indebidos confirmados y dos alertas de intento.
- En ambos lotes de cuatro peticiones: dos intentos detectados, dos controles normales, cero falsas alarmas y cero intentos omitidos.
- Evaluación separada de reglas: candidato mejorado promovido; candidato con falsas alarmas rechazado, conservando v1. Casos reservados del candidato mejorado: 4 a 10 aciertos sobre 10.

Evidencia completa de la autocomprobación: [self-check.json](self-check.json). Contiene únicamente el objetivo ficticio y los resultados medidos. El puerto efímero y la fecha UTC corresponden a esa ejecución, no a un servicio publicado.

## Comprobaciones ejecutadas

- `go test -race -count=1 -timeout 60s ./lab/fair`: aprobado. Cubre el ciclo HTTP, el destino y las rutas permitidas, rechazo de redirecciones, atribución de eventos, cancelación, integridad de evidencias, evaluación de reglas, sesión local, Host/Origin y exportaciones.
- `go test -count=1 -timeout 60s ./agents/blue/sigma ./agents/blue/telemetry ./agents/blue/detectbench ./evolve ./governance/report ./lab/engagement`: siete paquetes aprobados.
- Compilación del ejecutable y `-self-check`: correctos.
- Navegador real: se ejecutaron los tres botones de la consola y se observaron dos hallazgos antes, cero después, HTTP 404/403, dos alertas y mejora 4 a 10 en el experimento independiente.
- Revisión visual de la portada, comparación antes/después y evaluación de reglas: legibles en el viewport de escritorio inspeccionado.

No se certifica aquí la plataforma completa, la precisión en datos reales, la integración con el orquestador de producción, un ensayo en el equipo del asesor ni una auditoría completa de accesibilidad o dispositivos móviles. La descarga de informes se validó mediante pruebas HTTP; el flujo de impresión/PDF depende del navegador.
