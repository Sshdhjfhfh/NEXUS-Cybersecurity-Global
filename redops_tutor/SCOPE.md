# Alcance y barrera — nexus-redops

Este módulo especializa un modelo local (Ollama) para que **razone y guíe**
auditorías de seguridad en laboratorios y sistemas reales expresamente
autorizados. Crear un perfil de Ollama cambia instrucciones y parámetros; no
entrena pesos ni garantiza que el modelo cumpla la política.

## Permitido

- Razonamiento de metodología ofensiva a nivel de marco (MITRE ATT&CK, cadenas
  de ataque, priorización, hipótesis de vulnerabilidad).
- Análisis y planificación para objetivos del laboratorio y sistemas reales
  incluidos en una autorización explícita. Un objetivo no queda excluido por
  estar fuera del laboratorio.
- CTF y ejercicios educativos con alcance definido.
- Traducción de hallazgos a su equivalente defensivo (detección, mitigación).

## No permitido (coherente con los módulos inmutables del repo)

- Payloads o exploits listos para usar contra objetivos reales no autorizados.
- Evasión de detección, anonimato operativo o persistencia fuera del lab.
- Cualquier salida que desactive o eluda `ethical_boundary`, `opsec`,
  `watchdog`, `zero_trust`, `abm`, `llmjacking_guard` u `oracle_defense`.
- Operar sobre sistemas, redes o datos no incluidos en una autorización explícita.

Antes de interactuar con un sistema real, el alcance debe identificar activos,
acciones permitidas, exclusiones, ventana temporal y límites de impacto. Sin
esos datos, el asesor puede analizar código y evidencias proporcionados, pero
debe pedir el alcance antes de proponer interacción con el objetivo.

En producción se priorizan comprobaciones que no modifican datos ni afectan
disponibilidad. Autorizar una auditoría no autoriza automáticamente esos efectos,
ni terceros relacionados. Las acciones con cambios necesitan alcance específico
y limpieza definida.

El `SYSTEM` expresa la política esperada. **El prompt no es una frontera de
seguridad**: las herramientas, red, archivos y permisos de los agentes deben
aplicarla de forma independiente. Las lecciones y resultados no pueden ampliar
el alcance. No se eliminan ni modifican los módulos de protección de Nexus.
