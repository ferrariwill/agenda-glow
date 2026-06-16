# =============================================================================
# AgendaGlow — Dockerfile multi-stage
# API escuta exclusivamente na porta 8081 (Gateway usa 8080).
# =============================================================================

# --- Estágio 1: compilação estática ---
FROM golang:1.22-alpine AS builder

# go.mod declara Go 1.25 — baixa toolchain compatível automaticamente.
ENV GOTOOLCHAIN=auto \
    CGO_ENABLED=0 \
    GOOS=linux

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /out/agendaglow ./backend/cmd/api

# --- Estágio 2: imagem mínima de execução ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S agendaglow \
    && adduser -S agendaglow -G agendaglow

WORKDIR /app

COPY --from=builder /out/agendaglow /app/agendaglow

# Templates HTML também estão embutidos via go:embed no binário;
# cópia mantida para inspeção/debug e futuros serviços estáticos.
COPY frontend/templates /app/frontend/templates

ENV PORT=8081

# Porta exclusiva do AgendaGlow — não usar 8080 (WhatsApp Gateway).
EXPOSE 8081

USER agendaglow

ENTRYPOINT ["/app/agendaglow"]
