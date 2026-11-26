"""Request one supervised, text-only response from the local session relay."""
import json
from pathlib import Path
from urllib.request import Request, ProxyHandler, build_opener

ROOT = Path(__file__).resolve().parents[1]
STATE = ROOT / 'build/review/relay'
key = (STATE / 'api.key').read_text().strip()
payload = {
    'model': 'codex-session-relay', 'stream': False,
    'messages': [{'role': 'user', 'content': 'Prueba de enlace asistido. Explica en español el papel del puente entre esta conversación y Nexus, y los límites de una evaluación de dominio autorizado. No ejecutes herramientas ni afirmes pruebas no realizadas.'}],
    'tools': [],
}
request = Request('http://127.0.0.1:11435/v1/chat/completions', data=json.dumps(payload).encode(), headers={'Content-Type':'application/json','Authorization':'Bearer '+key})
print('Waiting for an active conversation operator to answer the private queue.', flush=True)
with build_opener(ProxyHandler({})).open(request, timeout=610) as response:
    result = json.load(response)
(STATE / 'probe-response.json').write_text(json.dumps(result, ensure_ascii=False, indent=2))
print(json.dumps({'status':'delivered','message':result['choices'][0]['message']['content']}, ensure_ascii=False), flush=True)
