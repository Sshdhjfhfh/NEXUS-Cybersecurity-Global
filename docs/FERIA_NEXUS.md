# NEXUS para la feria Innova Sys

NEXUS se presenta como un prototipo de evaluación y detección de amenazas para organizaciones. Esta entrega demuestra un ciclo ofensivo y defensivo acotado sobre una aplicación ficticia. Su uso en cada sector requiere una validación posterior; no se afirma cobertura universal ni preparación comercial.

## Iniciar la demostración

En WSL, desde `/home/saul/nexus-real`:

```sh
bash scripts/start-fair.sh
```

Abrir **http://127.0.0.1:8795**. Mantener la terminal abierta y usar Ctrl+C para detener. El script compila con las dependencias locales, sin descargarlas. Una vez preparado, también puede iniciarse directamente con `./build/nexus-fair`. No necesita PostgreSQL, Docker, un modelo, una clave de API ni Internet. El primer build necesita Go compatible con `go.mod` y sus dependencias descargadas previamente.

Desde PowerShell, el script `scripts/start-fair.ps1` inicia la instalación de WSL del equipo de Saúl. En otro equipo se debe adaptar la distribución, el usuario y la ruta o ejecutar el script de Linux desde el repositorio.

## Recorrido para el jurado

1. Explicar el problema: configuraciones expuestas y controles de acceso incorrectos pueden revelar información de una organización. Identificar un caso local real solo si se tiene autorización y evidencia para citarlo.
2. Pulsar **Evaluar aplicación**. NEXUS hace cuatro peticiones HTTP al objetivo creado para esta sesión. Dos son controles normales y dos comprueban fallos deliberados. Se esperan dos hallazgos con respuestas y huellas SHA-256.
3. Mostrar **Detección defensiva**. El motor Sigma/telemetría de NEXUS analiza los eventos derivados de esas peticiones. Las dos alertas se vinculan con los identificadores de evidencia.
4. Pulsar **Corregir y comprobar**. Se activan protecciones preimplementadas en el objetivo ficticio. La misma prueba debe obtener 404 para el archivo reservado y 403 para el registro ajeno. Los intentos siguen siendo detectables aunque ya no consigan acceso.
5. Pulsar **Comparar estrategias**. Es un experimento separado, basado en `detectbench` y `evolve`: compara reglas predefinidas sobre datos sintéticos, acepta una mejora y rechaza una alternativa con falsas alarmas. No modifica las reglas del paso anterior ni aprende nuevos ataques.
6. Descargar el informe y las evidencias JSON. La impresión del navegador permite guardar una copia PDF. Exportar antes de cerrar: las últimas 20 ejecuciones permanecen en memoria durante la sesión.

Distribución sugerida: problema 1 min; prueba ofensiva 2 min; evidencia defensiva 2 min; corrección y nueva prueba 2 min; evaluación de reglas 1 min; resultados y aporte propio 2 min. Preparar además cinco minutos de preguntas.

## Qué está implementado

- Objetivo HTTP efímero, exclusivamente en loopback, con datos ficticios y dos fallos deliberados.
- Comprobaciones activas con destino y rutas fijados por el código, sin parámetros de objetivo externos, proxies ni redirecciones.
- Eventos atribuibles a cada ejecución, hashes de las respuestas y detección usando los paquetes existentes `agents/blue/sigma` y `agents/blue/telemetry`.
- Matriz de detección de los cuatro casos del escenario. Las etiquetas del evaluador no se suministran a las reglas.
- Corrección de ambos fallos del objetivo de prueba y comprobación HTTP posterior.
- Evaluación separada de estrategias con `agents/blue/detectbench` y `evolve`.
- Consola en español, historial de sesión, exportación Markdown/JSON y presentación imprimible.
- Sesión local, comprobación de Host/Origin, cookies HttpOnly/SameSite y política de contenido del navegador.

La identidad `10.77.0.2` en los eventos es un identificador lógico del activo ficticio necesario para el contrato de telemetría; no se realiza ninguna conexión a esa IP. El destino efectivo es el puerto loopback que aparece en la consola. Las observaciones son registradas por el cliente de prueba; no se presentan como capturas independientes de un sensor de producción.

## Límites que deben explicarse

Las reglas del escenario reconocen rutas conocidas. Los cuatro casos no miden precisión general ni resistencia frente a ataques desconocidos. El experimento de evolución usa familias sintéticas relacionadas y estrategias registradas previamente; no demuestra reentrenamiento de un modelo ni mejora general de una IA.

La corrección es una conmutación preimplementada del objetivo de laboratorio, no un parche autónomo aplicado a código de clientes. Esta consola no activa el orquestador de producción, no desbloquea sus controles pendientes y no certifica sus módulos de aislamiento, agentes, multiusuario o laboratorio. La autorización local de eventos de feria no es un sustituto de la autorización de producción.

El estado y los informes no están firmados. Los hashes permiten comprobar correspondencia de contenido, pero no prueban por sí solos autoría o autenticidad contra quien controla el equipo.

## Verificación reproducible

Con una versión de Go compatible:

```sh
go test -race -count=1 ./lab/fair
go test -count=1 ./agents/blue/sigma ./agents/blue/telemetry ./agents/blue/detectbench ./evolve ./governance/report ./lab/engagement
go build -trimpath -o build/nexus-fair ./cmd/nexus-fair
./build/nexus-fair -self-check > build/fair-evidence.json
```

La autocomprobación devuelve error si no obtiene dos hallazgos antes, cero después, intentos correctamente detectados y rechazo de la estrategia con falsas alarmas. El archivo JSON contiene respuestas ficticias y resultados de esa ejecución; no deben sustituirse por cifras escritas a mano.

## Aportes y componentes externos

Revisar `CREDITS.md` y `LICENSE`: NEXUS deriva de ARTEX y norma. Esta entrega añade `lab/fair`, `cmd/nexus-fair` y los scripts de arranque; reutiliza los motores internos existentes. El equipo debe poder explicar su contribución concreta y declarar las herramientas y servicios utilizados, incluida la asistencia de IA cuando corresponda. No presentar toda la base upstream como desarrollo propio.

## Preparación pendiente del equipo

- Confirmar inscripción y categoría B: Plataformas de TIC. Según la ampliación comunicada por Saúl, el cierre es el viernes 27 de noviembre de 2026.
- Completar nombres, matrícula, asesor y firmas en los anexos exigidos por las bases; entregar el resumen de una página y la declaración de autoría.
- Preparar banner con título, integrantes, asesor, categoría y un esquema visual del ciclo.
- Validar la utilidad con un posible usuario de una organización local y registrar su comentario con permiso. No inventar adopción ni beneficiarios.
- Ensayar tres veces en el equipo que se llevará, incluida una sesión sin Internet. Guardar informe, ejecutable y video de respaldo antes de la feria del 1 de diciembre.
- Tener al menos dos integrantes presentes durante la evaluación y preparar respuestas sobre originalidad, límites, evidencia y uso autorizado.

## Resumen de inscripción propuesto

**Título:** NEXUS: evaluación ofensiva y detección defensiva de amenazas para organizaciones.

**Problema:** las organizaciones necesitan identificar fallos de acceso y exposición de información, relacionarlos con evidencia y comprobar la eficacia de sus correcciones.

**Objetivo:** demostrar un ciclo reproducible de evaluación, detección y verificación de correcciones mediante un prototipo de laboratorio.

**Solución:** NEXUS integra comprobaciones HTTP controladas, análisis de eventos con reglas Sigma, trazabilidad de evidencias y comparación de resultados antes y después de activar protecciones. Un experimento separado evalúa estrategias predefinidas y rechaza alternativas con falsas alarmas.

**Tecnologías:** Go, HTTP, reglas Sigma, HTML/CSS/JavaScript y los componentes de NEXUS declarados en sus créditos.

**Resultados:** completar con la ejecución exportada y revisada por el equipo. El alcance es un escenario sintético con dos fallos deliberados, no una medición de precisión en organizaciones reales.
