# Contributing

## Before You Start

Please open an issue or discussion first for:

- new provider / platform integrations
- new routing or balancing strategies
- schema or API changes
- behavior changes that may affect compatibility

This helps avoid duplicate work and keeps feature direction aligned.

## Development Setup

Backend:

```bash
go run ./cmd/server -conf config/config.yaml
```

Frontend:

```bash
cd web
bun install
bun dev
```

Frontend production build:

```bash
cd web
bun run build
```

## Validation

Run these before submitting changes:

```bash
go test ./...
go build ./...
```

If the change touches the frontend, also run the relevant frontend checks in `web/`.

## Code Style

- follow the existing layering: `handler -> service -> repository`
- avoid unrelated refactors in feature PRs
- keep new provider behavior explicit and testable
- add tests for behavior changes

## Pull Requests

A good PR should include:

- what changed
- why it changed
- compatibility impact, if any
- screenshots for UI changes
- test/build results

## Issues

When opening a bug report, include:

- version or commit
- deployment method
- config details relevant to the bug
- reproduction steps
- expected result
- actual result
