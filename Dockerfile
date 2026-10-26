# ═══════════════════════════════════════════════════════════════
# NEXUS IA v6.0 — Dockerfile multi-stage
#
# Absorbido de ARTEX — Licencia AGPL-3.0
#
# Stages:
#   1. frontend: build Next.js static export
#   2. backend:  build Go binary con frontend embebido (-tags embedui)
#   3. runtime:  Debian con el binario único y herramientas de los agentes
# ═══════════════════════════════════════════════════════════════

# ───────────────────────────────────────────────────────────────
# STAGE 1: Frontend (Next.js static export)
# ───────────────────────────────────────────────────────────────
FROM node:22-alpine AS frontend

WORKDIR /app/web

# Copiar package.json + lockfile primero (mejor caching)
COPY web/package.json web/package-lock.json ./

# Instalar dependencias
RUN npm ci --no-audit --no-fund

# Copiar el resto del frontend
COPY web/ ./

# Build static export → produce ./out/
RUN NEXT_EXPORT=1 npm run build:static

# ───────────────────────────────────────────────────────────────
# STAGE 2: Backend (Go binary con frontend embebido)
# ───────────────────────────────────────────────────────────────
FROM golang:1.26.3-alpine AS backend

WORKDIR /app

# Instalar herramientas de build
RUN apk add --no-cache git ca-certificates

# Copiar go.mod + go.sum primero (mejor caching)
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar el resto del código fuente
COPY . .

# Copiar el frontend buildeado al directorio esperado por go:embed
COPY --from=frontend /app/web/out ./orchestrator/server/webui/dist

# Build del binario con frontend embebido
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags embedui \
    -ldflags="-s -w -X main.version=v6.0.0" \
    -o /nexus ./cmd/nexus

# ───────────────────────────────────────────────────────────────
# STAGE 3: Runtime (herramientas de los agentes)
# ───────────────────────────────────────────────────────────────
FROM node:22-bookworm-slim AS runtime

# Herramientas mínimas que el backend necesita en runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash ca-certificates tzdata python3 python3-pip ripgrep curl git jq unzip \
    dnsutils iputils-ping netcat-openbsd whois nmap \
    && rm -rf /var/lib/apt/lists/*

# Fixed versions: browser tools are ready without downloading at task runtime.
ENV PLAYWRIGHT_BROWSERS_PATH=/opt/ms-playwright \
    NODE_PATH=/usr/local/lib/node_modules \
    CHROMIUM=/usr/local/bin/chromium
RUN npm install -g @playwright/cli@0.1.22 @playwright/mcp@0.0.83 playwright@1.64.1 puppeteer-core@23.11.1 \
    && playwright install --with-deps chromium \
    && node -e "require('fs').symlinkSync(require('playwright').chromium.executablePath(), '/usr/local/bin/chromium')" \
    && chmod -R a+rX /opt/ms-playwright \
    && npm cache clean --force \
    && rm -rf /var/lib/apt/lists/*

# Directorio de trabajo
WORKDIR /app

# Copiar el binario único
COPY --from=backend /nexus /app/nexus
COPY resources/skills/ /app/bundled-skills/
COPY scripts/seed_skills.py /app/seed_skills.py
COPY scripts/container-entrypoint.sh /app/entrypoint.sh

# Directorio de datos (SQLite + skills)
RUN mkdir -p /app/data /app/skills && chmod +x /app/entrypoint.sh

# Usuario no-root
RUN chown -R node:node /app

USER node

# Fail the image build if a required execution dependency is missing.
RUN /bin/bash -lc 'for tool in bash python3 rg curl git jq node npm nmap dig whois chromium; do command -v "$tool" || exit 1; done'

# Puerto HTTP + proxy MITM
EXPOSE 8787 8788

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -fsS http://localhost:8787/api/health || exit 1

# Variables de entorno por defecto
ENV ENV_STATE=production \
    LOG_LEVEL=INFO \
    NEXUS_SKILL_DIR=/app/data/skills \
    NEXUS_MEMORY_ENABLED=true

# Arrancar
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["-addr", ":8787", "-data", "/app/data", "-proxy", "127.0.0.1:8788"]
