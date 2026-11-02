# ═══════════════════════════════════════════════════════════════
# NEXUS IA v6.0 — Makefile
# ═══════════════════════════════════════════════════════════════

.PHONY: help build up up-https down restart logs ps shell clean nuke \
        test test-short lint fmt vet detect tier

# ───────────────────────────────────────────────────────────────
# Ayuda
# ───────────────────────────────────────────────────────────────
help:
	@echo "Nexus IA v6.0 — Comandos disponibles:"
	@echo ""
	@echo "  Stack Docker:"
	@echo "    make build       - Construye imagen Docker"
	@echo "    make up          - Arranca el stack (docker compose up -d)"
	@echo "    make up-https    - Arranca el stack publicado con HTTPS (NEXUS_DOMAIN)"
	@echo "    make down        - Para el stack"
	@echo "    make restart     - Reinicia el stack"
	@echo "    make logs        - Logs en vivo"
	@echo "    make ps          - Estado de contenedores"
	@echo "    make shell       - Shell interactivo en orchestrator"
	@echo ""
	@echo "  Limpieza:"
	@echo "    make clean       - Para contenedores y borra volúmenes"
	@echo "    make nuke        - Limpieza TOTAL (imágenes + volúmenes + redes)"
	@echo ""
	@echo "  Desarrollo:"
	@echo "    make test        - Ejecuta tests"
	@echo "    make test-short  - Tests sin integración"
	@echo "    make lint        - Linter (go vet + gofmt check)"
	@echo "    make fmt         - Formatea código (gofmt -w)"
	@echo "    make vet         - go vet"
	@echo ""
	@echo "  Hardware:"
	@echo "    make detect      - Detecta hardware"
	@echo "    make tier        - Selecciona tier óptimo"

# ───────────────────────────────────────────────────────────────
# Docker
# ───────────────────────────────────────────────────────────────
build:
	@echo "▶ Building Nexus IA image..."
	docker compose build

up:
	@echo "▶ Arrancando Nexus IA..."
	docker compose up -d
	@echo ""
	@echo "✅ Nexus IA corriendo"
	@echo "   Frontend + API: http://localhost:8787"
	@echo "   Logs: make logs"

up-https:
	@test -n "$$(grep -E '^NEXUS_DOMAIN=.+' .env 2>/dev/null)" || { echo "✖ Define NEXUS_DOMAIN en .env"; exit 1; }
	@echo "▶ Arrancando Nexus IA con HTTPS..."
	docker compose --profile https up -d
	@echo ""
	@echo "✅ Nexus IA publicado en https://$$(grep -E '^NEXUS_DOMAIN=' .env | cut -d= -f2-)"
	@echo "   Logs del certificado: docker compose logs https"

down:
	@echo "▶ Parando Nexus IA..."
	docker compose --profile https down

restart: down up

logs:
	docker compose logs -f

ps:
	docker compose ps

shell:
	docker compose exec orchestrator sh

# ───────────────────────────────────────────────────────────────
# Limpieza
# ───────────────────────────────────────────────────────────────
clean:
	@echo "▶ Parando contenedores y borrando volúmenes..."
	docker compose down -v
	@echo "✅ Limpieza completada"

nuke: clean
	@echo "▶ Eliminando imágenes, redes y todo..."
	-docker rmi nexus-ia/orchestrator:v6.0.0 2>/dev/null
	-docker volume rm nexus_pgdata nexus_data 2>/dev/null
	-docker network rm nexus_internal nexus_public 2>/dev/null
	@echo "✅ Nuke completado"

# ───────────────────────────────────────────────────────────────
# Desarrollo
# ───────────────────────────────────────────────────────────────
test:
	@echo "▶ Running tests..."
	go test ./...

test-short:
	@echo "▶ Running short tests..."
	go test -short ./...

lint: fmt vet

fmt:
	@echo "▶ Formateando código..."
	gofmt -w .
	@echo "✅ Código formateado"

vet:
	@echo "▶ go vet..."
	go vet ./...
	@echo "✅ Sin warnings"

# ───────────────────────────────────────────────────────────────
# Hardware
# ───────────────────────────────────────────────────────────────
detect:
	@python3 scripts/hardware_probe.py

tier: detect
	@python3 scripts/tier_selector.py
	@echo "---"
	@cat config/active_tier.yaml
