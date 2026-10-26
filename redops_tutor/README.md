# redops_tutor — cerebro de ciberseguridad para Nexus

Especializa un modelo local de Ollama como **cerebro de razonamiento ofensivo
ético** de Nexus: analiza evidencia y propone pasos para auditorías en
laboratorios y sistemas reales expresamente autorizados. No ejecuta herramientas
por sí mismo. El alcance y los permisos deben aplicarse fuera del modelo.

Ver [SCOPE.md](SCOPE.md) para la barrera (coherente con los módulos inmutables
del repo). El `SYSTEM` describe el comportamiento esperado; no reemplaza los
controles externos ni garantiza seguridad. El perfil no constituye entrenamiento
de pesos ni demuestra especialización.

## Piezas

- `Modelfile` — define `nexus-redops:4b` sobre `qwen3:4b` (tools + thinking) con
  el `SYSTEM` del guía y su alcance incorporado.
- `knowledge/offensive_methodology.md` — base de conocimiento de metodología
  (recon → enumeración → hipótesis → validación controlada → impacto → reporte +
  contramedida), mapeada a MITRE ATT&CK. Sin payloads operativos.
- `eval/` — sonda de guía en texto libre y prueba de barrera. Lenta en CPU; la
  métrica rigurosa de capacidad es la **academia** (abajo).

## Crear / recrear el cerebro

```sh
# requiere Ollama local en 127.0.0.1:11434 y el modelo base qwen3:4b
ollama create nexus-redops:4b -f redops_tutor/Modelfile   # si tienes el CLI
# sin CLI (WSL): se crea vía POST /api/create (ver el historial del proyecto)
```

## Medir capacidad real (academia del repo)

La métrica fiable es `lab/academy`, que puntúa respuestas JSON estructuradas
(clasificación de hallazgo, categoría de corrección y selección de evidencia):

```sh
python3 lab/academy/run.py evaluate --out lab/academy/artifacts/redops-NN \
  --model nexus-redops:4b --limit 8 --timeout 120
python3 lab/academy/run.py evaluate --out lab/academy/artifacts/base-NN \
  --model qwen3:4b --limit 8 --timeout 120   # comparación
```

## Enseñar (aportar lecciones)

Siguiendo el contrato de `lab/academy/artifacts/claude-v1/prompt.txt`: lecciones
sintéticas originales, sin instrucciones de explotación, centradas en distinguir
evidencia de hipótesis y en controles de autorización. Las nuevas lecciones de
Claude están en `lab/academy/artifacts/claude-v1/lessons.jsonl`, marcadas
`seed-requires-review`: requieren revisión humana antes de promoverlas a
`curriculum.py`. No se tocan los casos reservados de EVAL.

## Conectar el cerebro a los agentes

Los agentes consumen el modelo local por el endpoint Ollama `127.0.0.1:11434`
(puenteado en el lab por `lab/benchmark/model_gateway.py`). Para usar el cerebro
especializado, apuntar el nombre de modelo del proveedor del lab a
`nexus-redops:4b` en lugar de `qwen3:4b`.
