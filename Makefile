.PHONY: help detect tier up down logs ps clean

help:
	@echo "Nexus IA v6.0 — Comandos:"
	@echo "  make detect  - Detecta hardware"
	@echo "  make tier    - Selecciona tier"
	@echo "  make up      - Arranca Nexus"
	@echo "  make down    - Detiene Nexus"
	@echo "  make logs    - Ver logs"

detect:
	@python3 scripts/hardware_probe.py

tier: detect
	@python3 scripts/tier_selector.py
	@echo "---"
	@cat config/active_tier.yaml

up:
	@docker compose up -d

down:
	@docker compose down

logs:
	@docker compose logs -f

ps:
	@docker ps

clean:
	@docker compose down -v
