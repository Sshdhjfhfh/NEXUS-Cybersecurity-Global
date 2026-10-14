# ═══════════════════════════════════════════════════════════════
# NEXUS IA v6.0 — Dockerfile multi-stage
#
# Absorbido de ARTEX — Licencia AGPL-3.0
#
# Stages:
#   1. frontend: build Next.js static export
#   2. backend:  build Go binary con frontend embebido (-tags embedui)
#   3. runtime:  Alpine con el binario único
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
# STAGE 3: Runtime (Alpine mínimo)
# ───────────────────────────────────────────────────────────────
FROM alpine:3.20 AS runtime

# Herramientas mínimas que el backend necesita en runtime
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    python3 \
    py3-pip \
    ripgrep \
    curl \
    && update-ca-certificates

# Directorio de trabajo
WORKDIR /app

# Copiar el binario único
COPY --from=backend /nexus /app/nexus

# Directorio de datos (SQLite + skills)
RUN mkdir -p /app/data /app/skills

# Usuario no-root
RUN addgroup -g 1000 nexus && \
    adduser -u 1000 -G nexus -s /bin/sh -D nexus && \
    chown -R nexus:nexus /app

USER nexus

# Puerto HTTP + proxy MITM
EXPOSE 8787 8788

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -fsS http://localhost:8787/api/health || exit 1

# Variables de entorno por defecto
ENV ENV_STATE=production \
    LOG_LEVEL=INFO

# Arrancar
ENTRYPOINT ["/app/nexus"]
CMD ["-addr", ":8787", "-data", "/app/data", "-proxy", "127.0.0.1:8788"]
