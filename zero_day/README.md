# ZERO_day — AFL++ de laboratorio

También disponible desde la terminal del sistema:
`nexus` → `/zero-day help`. Ver [la terminal Nexus](../terminal/README.md).

Módulo independiente: no cambia el backend, la interfaz ni los módulos existentes.
Ejecuta AFL++ sobre un parser sintético instrumentado, acepta semillas y diccionario
propuestos por un modelo local, reproduce las entradas fallantes y conserva informes
JSON/HTML. La campaña y la reproducción ocurren dentro del mismo contenedor sin red.

## Probar

Desde la raíz del repositorio:

```sh
mkdir -p zero_day/artifacts
docker compose -f zero_day/compose.yml build fuzz
docker compose -f zero_day/compose.yml run --rm fuzz
```

La prueba predeterminada dura 30 segundos. La primera compilación descarga las
dependencias de AFL++; las campañas posteriores funcionan sin red. El usuario del
contenedor es `1000:1000`, correspondiente al propietario actual del proyecto.

El programa de ejemplo tiene un `abort()` deliberado al leer `NEXUS!`. El corpus
inicial contiene `NEXUS?` y `HELLO`, que no fallan. AFL++ debe descubrir una mutación
fallante: la prueba no alimenta al fuzzer una entrada que ya falle. No es un zero-day
real ni una prueba del objetivo de un crash por diez horas de la especificación.

Cada ejecución imprime un identificador y guarda en
`zero_day/artifacts/<identificador>/`:

- `report.html`: informe en español que se puede abrir en el navegador.
- `report.json`: estado, duración, versión de AFL++ cuando está en sus estadísticas,
  hash del binario, guía utilizada, estadísticas y resultados de reproducción.
- `output/default/crashes/`: entradas encontradas por AFL++.
- `afl.log`: diagnóstico completo, también si la campaña falla.
- `guidance.json`, `input/` y `dictionary.txt`: corpus y decisiones reproducibles.
- `rejected-seeds/`: entradas sugeridas que fallaban antes de empezar; se registran
  aparte y no se cuentan como descubrimientos del fuzzer.

Las entradas se deduplican por SHA-256: varias entradas distintas pueden activar
el mismo fallo. Los hallazgos requieren revisión humana. Los cuelgues y las salidas
ordinarias con error no se clasifican como crashes reproducidos.

## Consultar y reproducir

```sh
docker compose -f zero_day/compose.yml run --rm fuzz list
docker compose -f zero_day/compose.yml run --rm fuzz report IDENTIFICADOR
docker compose -f zero_day/compose.yml run --rm fuzz replay IDENTIFICADOR SHA256
```

La reproducción comprueba el hash del binario y el de la entrada antes de ejecutarla.
Conserva la misma imagen si necesitas reproducir un resultado anterior.

Para cambiar la duración, entre 1 y 600 segundos:

```sh
docker compose -f zero_day/compose.yml run --rm fuzz run --seconds 60 \
  --guidance /opt/zero-day/guidance.example.json
```

`Ctrl+C` cancela la campaña y conserva su informe. Una ejecución fallida devuelve
código 1; una cancelada devuelve 130. Los logs y resultados permanecen en el host
aunque el contenedor de la campaña se elimine.

## Guía mediante un LLM local

```sh
python3 zero_day/campaign.py prompt
```

Si Ollama ya está activo en el host con el modelo descargado, genera una guía real:

```sh
python3 zero_day/guide.py --model qwen3:4b
```

El adaptador solo acepta un servidor HTTP en loopback, no sigue redirecciones ni
usa proxies. No descarga modelos. Opcionalmente acepta `--stats` con un archivo
`fuzzer_stats` anterior para guiar la siguiente campaña. La generación ocurre en
el host; el contenedor de fuzzing continúa sin red.

En WSL detecta el servidor en el loopback de Windows mediante `curl.exe` cuando
el de Linux rechaza la conexión. Se conserva el mismo endpoint local; no se abren
puertos ni se cambia la configuración de Ollama. La prueba real de esta entrega
usó `qwen3:1.7b`, ya instalado, con salida JSON acotada:

```sh
python3 zero_day/guide.py --model qwen3:1.7b
```

También puedes entregar el contrato, el código del parser y las estadísticas al
modelo local manualmente. Guarda su respuesta JSON como
`zero_day/artifacts/guidance.json` y ejecútala:

```sh
docker compose -f zero_day/compose.yml run --rm fuzz run --seconds 30 \
  --guidance /work/guidance.json
```

La guía se importa como datos: no puede modificar comandos, imágenes, límites ni
el binario. Se rechazan campos desconocidos, claves repetidas, base64 inválido y
entradas excesivas. El ejemplo incluido es un fixture, no una respuesta real de un
modelo. La guía de ejemplo permite probar el motor aunque Ollama no esté activo.

## Aislamiento y alcance

El contenedor tiene red deshabilitada, raíz de solo lectura, usuario sin privilegios,
capacidades eliminadas, límite de 512 MB, un CPU y 64 procesos. Solo se monta su
directorio de resultados; no recibe el socket Docker, credenciales ni datos de Nexus.
Los subprocesos usan argumentos explícitos, nunca un shell. La duración y el tamaño
de las entradas están acotados.

Esta entrega permite probar el circuito de fuzzing y triage de un archivo local
sintético. No conecta objetivos reales, no construye exploits, no publica servicios
vulnerables y no modifica ni sustituye controles OPSEC/éticos/ABM. La integración con
el pipeline autónomo y sus controles de autorización queda fuera de esta prueba
independiente; no se declara completada por ejecutar este demo.

## Pruebas

```sh
python3 -m unittest discover -s zero_day -p 'test_*.py' -v
```

Documentación del motor: [fuzzing con AFL++](https://aflplus.plus/docs/fuzzing_in_depth/).
La guía local utiliza la [API de generación de Ollama](https://docs.ollama.com/api/generate).
Resultados comprobados: [VALIDATION.md](VALIDATION.md).
