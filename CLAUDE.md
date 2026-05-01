# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DS2API is a Go-based proxy that converts DeepSeek Web chat capabilities into OpenAI, Claude, and Gemini compatible APIs. It also includes a React admin web UI. The project exposes HTTP endpoints that accept standard API requests, translates them into DeepSeek's internal web chat format, and returns protocol-compatible responses.

- **Language**: Go 1.26+
- **Frontend**: React (Vite) in `webui/`, served from `static/admin/`
- **Default port**: 5001

## Build, Test, and Development Commands

```bash
# Run the server locally
go run ./cmd/ds2api

# Run a single Go test file
go test ./internal/config/config_test.go

# Run all Go tests
go test ./...

# Run a single Go test function
go test -run TestFunctionName ./internal/...

# Lint (includes auto-bootstrap of golangci-lint v2.11.4 if needed)
./scripts/lint.sh

# Full PR gate (lint + unit tests + webui build)
./scripts/lint.sh
./tests/scripts/check-refactor-line-gate.sh
./tests/scripts/run-unit-all.sh
npm run build --prefix webui

# End-to-end live tests (requires real accounts)
./tests/scripts/run-live.sh

# Build webui only
./scripts/build-webui.sh

# Run webui dev server
npm run dev --prefix webui
```

## Architecture

### Request Flow

```
Client (OpenAI/Claude/Gemini SDK)
  → chi Router (internal/server/router.go)
  → HTTP API surface (internal/httpapi/{openai,claude,gemini})
  → promptcompat (API → DeepSeek web chat text context)
  → prompt assembly (internal/prompt/)
  → Auth resolver (internal/auth/)
  → Account pool + queue (internal/account/)
  → DeepSeek client (internal/deepseek/client/)
  → DeepSeek upstream
```

### Key Internal Packages

- **`internal/server`** — chi router, middleware (CORS, logging, request ID), health/readiness probes
- **`internal/httpapi/openai`** — OpenAI-compatible endpoints: chat completions, responses, files, embeddings
- **`internal/httpapi/claude`** — Claude-compatible messages API
- **`internal/httpapi/gemini`** — Gemini-compatible generateContent API; reuses OpenAI prompt builder
- **`internal/promptcompat`** — Core compatibility layer: normalizes API messages into DeepSeek web chat format (single prompt string + file references + control flags). This is the most critical package — see `docs/prompt-compatibility.md` for detailed semantics.
- **`internal/prompt`** — DeepSeek-specific prompt assembly with role markers, tool history XML
- **`internal/deepseek/client`** — Upstream HTTP calls: login, session management, completion, file upload/delete
- **`internal/deepseek/protocol`** — DeepSeek URL constants, skip paths, PoW challenge handling
- **`internal/account`** — Multi-account pool with concurrency slots (in-flight limits) and waiting queues
- **`internal/config`** — Configuration loading from `config.json` + env vars, hot-reloadable settings
- **`internal/toolcall`** / **`internal/toolstream`** — Canonical XML tool call parsing and leak prevention
- **`internal/stream`** / **`internal/sse`** — Go stream processing and SSE parsing
- **`internal/format/{openai,claude}`** — Response formatting into protocol-specific structures
- **`internal/translatorcliproxy`** — Multi-protocol message translation (Claude/Gemini ↔ OpenAI)
- **`internal/chathistory`** — Server-side chat history persistence
- **`internal/js`** / **`api/chat-stream.js`** — Vercel Node.js streaming bridge (Go prepares auth/session, Node handles real-time SSE)
- **`internal/httpapi/admin`** — Admin API: config management, account testing, proxy settings, session cleanup, dev capture

### Vercel Deployment

On Vercel, `/v1/chat/completions` is handled by `api/chat-stream.js` (Node runtime) for real-time SSE. Auth, account selection, and session/PoW preparation are done by Go internal endpoints; Node handles streaming output with tool sieve semantics aligned with the Go implementation.

## Configuration

- Primary config: `config.json` (copy from `config.example.json`)
- Docker/Vercel: pass `DS2API_CONFIG_JSON` env var (Base64-encoded config)
- Common env vars: `DS2API_ADMIN_KEY`, `DS2API_HOST_PORT`, `DS2API_DEV_PACKET_CAPTURE`
- Config supports: API keys, DeepSeek accounts (email/mobile), model aliases, runtime concurrency limits, auto-delete mode, history split settings, embeddings provider

## Code Style

- Run `gofmt -w` on every changed Go file before commit
- Do not ignore error returns from cleanup calls (`Close`, `Flush`, `Sync`, etc.)
- Keep changes additive and tightly scoped — don't mix unrelated refactors
- When modifying business logic or user-visible behavior, update corresponding docs:
  - `docs/prompt-compatibility.md` is the source of truth for API→web-chat compatibility
  - Changes to message normalization, tool prompt injection, tool history, file handling, history split, or completion payload assembly must update that file
- Lint config: `.golangci.yml` (golangci-lint v2, enables errcheck, govet, staticcheck, unused, gofmt formatter)

## Documentation

- `README.MD` — Overview, quick start, model support matrix
- `docs/ARCHITECTURE.md` — Directory structure and module responsibilities
- `docs/prompt-compatibility.md` — Detailed API→web chat compatibility flow (critical reference)
- `API.md` — Full API documentation with request/response examples
- `docs/DEPLOY.md` — Deployment guide (local, Docker, Vercel, systemd)
- `docs/toolcall-semantics.md` — Tool call protocol details
