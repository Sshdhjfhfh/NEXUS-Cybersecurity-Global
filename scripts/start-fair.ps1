param([int]$Port = 8795)
$ErrorActionPreference = 'Stop'
if ($Port -lt 1 -or $Port -gt 65535) { throw 'Puerto no válido.' }
Write-Host "NEXUS: abre http://127.0.0.1:$Port cuando el servidor esté listo."
Write-Host 'Mantén esta terminal abierta. Ctrl+C para detener.'
wsl.exe -d Ubuntu-22.04 -u saul --cd /home/saul/nexus-real -- bash scripts/start-fair.sh -port $Port
if ($LASTEXITCODE -ne 0) { throw "NEXUS terminó con código $LASTEXITCODE" }
