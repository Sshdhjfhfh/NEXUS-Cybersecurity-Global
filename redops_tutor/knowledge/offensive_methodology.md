# Metodología ofensiva — base de conocimiento para nexus-redops

Conocimiento de nivel de marco para que el modelo **planifique, priorice y guíe**
agentes dentro de un alcance autorizado, de laboratorio o de sistemas reales.
No contiene payloads operativos; su
objetivo es el razonamiento, no la ejecución a ciegas.

## 1. Modelo mental: la cadena, no los trucos

Un operador competente no "lanza herramientas"; recorre una cadena y en cada
eslabón decide la **siguiente acción de mayor valor de información** con el menor
ruido. Cadena base (alineada con MITRE ATT&CK):

1. **Reconocimiento** (TA0043): qué existe y qué expone el objetivo autorizado.
2. **Enumeración / análisis de superficie**: servicios, versiones, rutas,
   autenticación, dependencias, configuraciones por defecto.
3. **Análisis de vulnerabilidad**: de la superficie a hipótesis concretas de fallo
   (qué debilidad, por qué, cómo se validaría sin causar daño).
4. **Acceso inicial** (TA0001) / **Ejecución** (TA0002): validación controlada de
   la hipótesis dentro del alcance.
5. **Post-explotación**: escalamiento (TA0004), movimiento lateral (TA0008),
   acceso a credenciales (TA0006) — solo como medición de impacto autorizada.
6. **Reporte**: evidencia reproducible, impacto, y la **contramedida defensiva**.

Regla de guía: si el modelo no puede nombrar la hipótesis que valida la siguiente
acción, la acción es prematura. Primero información, luego interacción.

## 2. Reconocimiento y enumeración (razonamiento)

- Distinguir **pasivo** (sin tocar el objetivo) de **activo** (interacción).
  El pasivo primero reduce ruido y acota el activo.
- Priorizar por *señal/ruido*: un servicio con versión conocida y autenticación
  débil vale más que un barrido exhaustivo de puertos altos.
- Mapear la **superficie de autenticación** (login, tokens, sesiones, APIs) antes
  que la superficie de contenido: casi todo el impacto real pasa por authz/authn.
- Para web: inventario de endpoints, métodos aceptados, manejo de entrada,
  dependencias de front/back, y dónde se cruza confianza (input → sink).

## 3. De superficie a hipótesis de vulnerabilidad

Clases que el modelo debe reconocer y razonar (no explotar a ciegas):

- **Validación de entrada**: inyección (SQL, comando, plantilla, deserialización)
  — el patrón común es *entrada no confiable alcanza un intérprete*.
- **Control de acceso roto** (IDOR, autorización vertical/horizontal): el patrón es
  *el servidor confía en un identificador que el cliente controla*.
- **Autenticación**: sesiones predecibles, tokens mal firmados, reinicio de
  credenciales, MFA eludible por diseño.
- **Exposición de configuración**: secretos en repositorios/artefactos, servicios
  de depuración, permisos excesivos, valores por defecto.
- **Lógica de negocio**: estados no previstos, condiciones de carrera, límites no
  aplicados en el servidor.

Para cada hipótesis, el modelo debe producir: (a) por qué es plausible dada la
evidencia, (b) cómo validarla de forma **mínima y reversible**, (c) qué evidencia
la confirma, (d) la contramedida.

## 4. Validación controlada (dentro del lab)

- Preferir la prueba de concepto **más pequeña que demuestre el fallo**, no la más
  espectacular. Menos daño, más reproducibilidad.
- Registrar entrada exacta, salida y hash de artefactos (como hace `zero_day`).
- Nunca degradar disponibilidad del objetivo salvo que el alcance lo autorice
  explícitamente; DoS no es una técnica de guía por defecto.

## 5. Post-explotación como medición de impacto

El objetivo en un lab autorizado es **medir impacto**, no maximizar daño:
demostrar hasta dónde llega el acceso, qué datos se exponen y qué controles
fallaron, para que la defensa pueda cerrarlos. Escalamiento y lateralidad se
razonan como "¿qué demuestra el peor caso realista?" bajo el alcance.

## 6. Reporte y cierre (criterio de tarea completa)

Una tarea no está cerrada por un informe del agente, sino por evidencia
observable (ver `docs/EQUIPO_AGENTES.md`, prioridad 2). El reporte incluye:

- Objetivo verificado, pasos reproducibles, evidencia con hashes.
- Impacto y alcance real alcanzado.
- Contramedida concreta (detección + mitigación) — el entregable defensivo.
- Limitaciones y lo no probado.

## 7. Barrera incorporada

Un sistema real autorizado no queda excluido por ser real. Antes de interactuar,
definir activos, acciones, exclusiones, ventana y límites de impacto. Si falta el
alcance, pedirlo y continuar solo el análisis de material proporcionado. Rechazar
acciones no autorizadas y la desactivación de controles; ofrecer una revisión o
un equivalente aislado. Los permisos se aplican fuera del modelo: un prompt o una
lección no pueden concederlos.

## 8. Cómo guía a los agentes de Nexus

El modelo no ejecuta herramientas: **decide y justifica** la siguiente acción para
el agente (`agents/red`), que sí las ejecuta bajo el modo de permisos de Nexus.
Buen guía = próxima acción + hipótesis + evidencia esperada + coste/ruido, y
detenerse cuando falta una dependencia o permiso en lugar de reintentar imposibles
(`docs/EQUIPO_AGENTES.md`, prioridades 1 y 3).
