'use strict';
const byId = (id) => document.getElementById(id);
let currentState = null;
let busy = false;
const when = (date) => new Intl.DateTimeFormat('es-PE', { timeZone: 'America/Lima', hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(date));
function element(tag, text, className) {
  const result = document.createElement(tag);
  if (text !== undefined) result.textContent = text;
  if (className) result.className = className;
  return result;
}
function status(message, error = false) {
  byId('status').textContent = message;
  byId('status').classList.toggle('error', error);
}
function buttons() {
  byId('baseline').disabled = busy || !currentState;
  byId('evaluate').disabled = busy || !currentState;
  byId('secured').disabled = busy || !currentState?.runs.some((run) => !run.secured);
}
async function request(path, options) {
  const response = await fetch(path, { ...options, credentials: 'same-origin', cache: 'no-store' });
  if (!response.ok) throw new Error((await response.text()).slice(0, 250));
  return response.json();
}
function render(state) {
  currentState = state;
  byId('target-url').textContent = state.target;
  const last = state.runs.at(-1);
  const before = state.runs.findLast((run) => !run.secured);
  const after = state.runs.findLast((run) => run.secured);
  byId('before-count').textContent = before ? before.findings.length : '—';
  byId('after-count').textContent = after ? after.findings.length : '—';
  if (before && after) byId('comparison-note').textContent = `Últimas comprobaciones: ${before.findings.length} → ${after.findings.length} hallazgos. Antes: ejecución ${before.id}; después: ejecución ${after.id}.`;
  if (last) {
    byId('finding-count').textContent = last.findings.length;
    byId('finding-caption').textContent = last.secured ? 'Tras activar las correcciones' : 'En la aplicación vulnerable';
    byId('alert-count').textContent = last.detection.alerts.length;
    byId('event-count').textContent = last.observations.length;
    byId('target-state').textContent = state.secured ? 'Corregido' : 'Vulnerable';
    const alertContainer = byId('alerts');
    alertContainer.replaceChildren();
    if (!last.detection.alerts.length) alertContainer.append(element('div', 'No hubo coincidencias de reglas en esta ejecución.', 'empty'));
    for (const alert of last.detection.alerts) {
      const card = element('article', undefined, 'panel alert-card');
      card.append(element('span', 'ALERTA · ' + ({high:'ALTA',medium:'MEDIA',low:'BAJA'}[alert.severity] || alert.severity), 'alert-label'));
      card.append(element('h3', alert.title));
      card.append(element('p', 'Evidencia: ' + alert.evidence_ids.join(', ')));
      card.append(element('small', 'Regla: ' + alert.rule_id));
      if (last.secured) card.append(element('p', 'Intento detectado después de la corrección. Consulta el estado HTTP para comprobar el bloqueo.', 'fine'));
      alertContainer.append(card);
    }
    const metrics = last.metrics;
    byId('confusion').replaceChildren();
    for (const [label, value] of [['Detectados',metrics.true_positive],['Falsas alarmas',metrics.false_positive],['Omitidos',metrics.false_negative],['Normales',metrics.true_negative]]) {
      const item = element('span', label); item.append(element('b', value)); byId('confusion').append(item);
    }
    byId('observations').replaceChildren();
    for (const observation of last.observations) {
      const row = element('tr');
      row.append(element('td', observation.id), element('td', observation.path), element('td', observation.status));
      row.append(element('td', observation.succeeded ? 'Acceso indebido confirmado' : observation.expected_attack ? 'Acceso no conseguido' : 'Control normal'));
      const detailsCell = element('td');
      const details = element('details'); details.append(element('summary', 'Ver evidencia'));
      details.append(element('pre', observation.body), element('small', 'SHA-256: ' + observation.sha256, 'digest'));
      detailsCell.append(details); row.append(detailsCell); byId('observations').append(row);
    }
    for (const id of ['export-md','export-json']) {byId(id).classList.remove('disabled');byId(id).setAttribute('aria-disabled','false');}
  }
  if (state.evaluation) {
    const evaluation = state.evaluation;
    const comparison = evaluation.improvement.holdout;
    const area = byId('evaluation-result'); area.replaceChildren();
    area.append(element('span', `ACIERTOS / ${comparison.cases} CASOS RESERVADOS`, 'eyebrow'));
    area.append(element('strong', `${comparison.baseline_passed} → ${comparison.candidate_passed}`));
    area.append(element('p', evaluation.improvement.decision === 'promoted' ? 'Mejora aceptada por el evaluador.' : 'La mejora no fue aceptada.'));
    area.append(element('p', evaluation.noisy.active === 'v1' ? 'Alternativa con falsas alarmas: rechazada.' : 'Revisar resultado de la alternativa.', 'mint'));
  }
  const history = byId('history');history.replaceChildren();
  if (!state.runs.length) history.textContent = 'Sin ejecuciones.';
  for (const run of [...state.runs].reverse()) history.append(element('p', `#${run.id} · ${when(run.at)} (Lima) · ${run.secured ? 'Después de corregir' : 'Antes de corregir'} · ${run.findings.length} hallazgos · ${run.detection.alerts.length} alertas · ${run.elapsed_ms} ms`));
  buttons();
}
async function execute(mode) {
  if (busy) return;
  busy = true; buttons();
  status(mode === 'evaluate' ? 'Comparando estrategias con el evaluador de NEXUS…' : mode === 'secured' ? 'Activando las correcciones del objetivo y repitiendo las peticiones…' : 'Realizando peticiones al objetivo y analizando sus eventos…');
  try {
    await request('/api/run', { method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({mode}) });
    render(await request('/api/state'));
    status(mode === 'evaluate' ? 'Evaluación de reglas completada. Resultados disponibles abajo.' : 'Evaluación completada. Evidencias e informe disponibles.');
  } catch (error) {status('No se completó la operación: ' + error.message, true);}
  finally {busy = false;buttons();}
}
byId('baseline').addEventListener('click', () => execute('baseline'));
byId('secured').addEventListener('click', () => execute('secured'));
byId('evaluate').addEventListener('click', () => execute('evaluate'));
byId('print').addEventListener('click', () => window.print());
for (const id of ['export-md','export-json']) byId(id).addEventListener('click', (event) => {if(byId(id).getAttribute('aria-disabled') === 'true')event.preventDefault();});
request('/api/state').then((state) => {render(state);status('Laboratorio preparado. Empieza con «Evaluar aplicación».');}).catch((error) => {status('No se pudo conectar: ' + error.message, true);buttons();});
