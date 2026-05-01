# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go-based LLM gateway backend, with an optional frontend in `web/`.

- `cmd/server`: main backend entrypoint.
- `cmd/server/wire`: dependency injection setup (`wire.go`) and generated injector (`wire_gen.go`).
- `internal/`: core backend modules (`handler`, `service`, `repository`, `models`, `tasks`, `loadbalancer`, `router`, etc.).
- `api/v1`: API request/response DTOs.
- `pkg/`: reusable provider/http tooling used by the backend.
- `config/`: runtime configuration (`config.yaml`).
- `storage/`: local runtime assets (SQLite DB, logs).
- `web/`: frontend app (has its own `web/AGENTS.md`).

## Build, Test, and Development Commands
- `go run ./cmd/server -conf config/config.yaml`  
  Run the backend locally.
- `go build ./cmd/server`  
  Build backend binary for validation.
- `go test ./internal/...`  
  Run backend-focused tests.
- `go test ./internal/tasks -run TestPriceSyncTask -count=1`  
  Example targeted test run.
- `go generate ./cmd/server/wire`  
  Regenerate `wire_gen.go` after editing DI wiring.

## Coding Style & Naming Conventions
- Use standard Go formatting (`gofmt`) before commit.
- Package names are lowercase; file names typically use `snake_case.go`.
- Exported identifiers: `CamelCase`; unexported: `camelCase`.
- Keep `context.Context` as the first parameter for request-scoped operations.
- Do not manually edit generated files (`cmd/server/wire/wire_gen.go`).

## Testing Guidelines
- Place tests next to implementation as `*_test.go`.
- Prefer descriptive names like `TestXxx_Scenario`.
- Cover boundary cases (e.g., pricing thresholds), fallback logic, and error paths.
- For changed backend logic, add or update targeted tests in the same module.

## Commit & Pull Request Guidelines
- Follow Conventional Commits seen in history, e.g.:
  - `feat: ...`
  - `feat(scope): ...`
  - `fix(scope): ...`
- Keep each commit focused on one concern (feature/fix/refactor).
- PRs should include:
  - what changed and why,
  - affected modules/files,
  - test commands and results,
  - migration/config notes if DB or config behavior changes.

## Security & Configuration Tips
- Never commit real secrets; use local `.env`/`.env.local` only.
- Validate DB-impacting changes against `storage/tudou.db` carefully before merge.
