# Repository Guidelines

## Project Structure & Module Organization
- Backend entrypoint: `cmd/netronome`.
- Feature packages: `internal/` (agent, monitor, speedtest, scheduler, notifications, tailscale, etc.).
- Reusable helpers: `pkg/`.
- Web UI: `web/src` (Vite + React), build output in `web/dist`.
- Tests and harnesses: `test/` plus `distrib/docker/docker-compose*.yml` for e2e stacks.
- Long-form docs: `docs/` and `ai_docs/`.

## Build, Test, and Development Commands
- `make build`: installs web deps, builds the UI, embeds assets, compiles `bin/netronome`.
- `make run`: builds and runs `serve --config config/config.toml`.
- `make dev`: starts Vite + Go watcher (`air`) in tmux.
- `make watch`: runs Go watcher only (prompts to install `air` if missing).
- `make docker-build` / `make docker-run`: build and run a Docker image.
- `pnpm -C web dev|build|lint`: frontend dev server, production build, lint.
- `go test ./...`: backend unit tests.
- `./test/test-local.sh`: local dockerized scenario tests.

## Coding Style & Naming Conventions
- Go: `gofmt` or `goimports`, Go 1.25 target.
- TypeScript: 2-space indentation, PascalCase components, camelCase utilities.
- Tailwind tokens should align with `web/tailwind.config.*`.
- Lint: `pnpm -C web lint` (ESLint).

## Testing Guidelines
- Go tests live alongside code as `*_test.go`.
- Prefer table-driven tests for new backend logic.
- Scenario scripts live in `test/`; add fixtures under `test/data/`.
- `test/` also contains docker-compose scenarios, SSL cert scripts, and base-URL test harnesses.
- If you skip tests, note why in the PR description.

## Commit & Pull Request Guidelines
- Conventional Commits: `feat|fix|refactor|build|ci|chore|docs|style|perf|test`.
- Keep changes scoped; avoid unrelated refactors and mass formatting.
- PRs must include: summary, test commands run, and screenshots for UI changes.
- For issue-driven fixes, review attached screenshots/log images before coding.
- Use `pnpm` only; do not add `package-lock.json`.

## Security & Configuration Tips
- Keep secrets out of Git; use `.env` and redact `config/config.toml`.
- Avoid committing real GeoLite databases or generated logs.
- SQLite `.db` files are gitignored; do not commit them.
- Document any config changes in `docs/` when behavior changes.
