# NEXUS — Próximas prioridades (verificables)

**Fecha:** 2026-10-26 · Basado en
[CAPABILITY_SCORECARD.md](CAPABILITY_SCORECARD.md).

Cada ítem es un incremento pequeño, probable y con criterio de aceptación. No
se sube ningún nivel sin evidencia ejecutable.

## P0 — Bloqueante (decisión del dueño)
1. **Habilitar un proveedor LLM.** Desbloquea las dimensiones A (razonamiento) y
   B (autonomía). Sin esto, el loop plan→herramienta→verifica no se puede
   demostrar. Acción del dueño: proveer credencial/configuración. Yo no puedo
   ni debo hacerlo.

## P1 — Defensiva utilizable (Nivel 3→4, sin LLM)
2. **`nexus-detect` (CLI blue): ✅ HECHO.** Corre `telemetry`+`sigma` sobre
   eventos JSONL y emite alertas con tag ATT&CK, autorizando cada activo contra
   el alcance. Demo: 3 eventos→1 alerta HIGH (T1059.001); fuera de alcance
   denegado; SigmaHQ real carga 1000 reglas. Tests verdes.
3. **Métricas de detección:** precisión/recall/FP contra un corpus etiquetado.
   *Aceptación:* reporte con matriz de confusión reproducible. ← **siguiente**

## P2 — Ofensiva más completa (Nivel 4, sin LLM)
4. **Verificación/PoC de hallazgos:** confirmar no-falso-positivo antes de
   reportar (p. ej. revalidar el endpoint). *Aceptación:* hallazgo marcado
   "verificado" con traza.
5. **Enriquecimiento del informe:** CVSS, remediación y export HTML/PDF para
   entrega al cliente. *Aceptación:* informe del lab con severidad priorizada.
6. **nmap (red):** inventario de puertos/servicios dentro del alcance.
   *Aceptación:* demo en lab, gated.

## P3 — Purple team (Nivel 1→3)
7. **Ciclo cerrado:** ATT&CK → simulación en lab → eventos → evaluación de
   detección → brecha → regla → validación en muestras no vistas.
   *Aceptación:* una técnica demostrada de principio a fin con mejora medida.

## P4 — Seguridad empresarial e infraestructura (Nivel 3→5)
8. **Aislamiento multi-tenant bajo carga** y control de egreso de red.
9. **Pruebas de carga** (latencia/colas/CPU/RAM) con números y condiciones.

## P5 — SaaS internacional (Nivel 1→)
10. **Portal + multi-tenancy + i18n (EN/ES)**, tras validar autorización y
    aislamiento. No introducir pagos ni evaluaciones ofensivas antes de eso.

## Pendientes operativos
- Resolver obligaciones **AGPL** (procedencia ~68%) antes de SaaS propietario;
  el modelo de **servicios** es seguro con AGPL.
- Abrir PR de la rama `implementation/nexus-reliability-20261026` cuando se
  quiera revisión.

---
**Recomendado como siguiente incremento:** **P1 #2 (`nexus-detect`)** — alto
valor, no necesita LLM, y convierte el motor de detección ya probado en algo
usable sobre logs reales.
