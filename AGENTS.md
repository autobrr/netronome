# Repository Guidelines

## Project Structure & Module Organization
- Backend entrypoint: `cmd/netronome`.
- Feature packages: `internal/` (agent, monitor, speedtest, scheduler, notifications, tailscale, etc.).
- Reusable helpers: `pkg/`.
- Web UI: `web/src` (Vite + React), build output in `web/dist`.
- Tests and harnesses: `test/` plus `distrib/docker/docker-compose*.yml` for e2e stacks.
- Long-form docs: `docs/`. `ai_docs/` is gitignored and holds local notes only. Put each doc that the repo keeps in `docs/`.
- User docs site: `documentation/` (Astro Starlight, deployed to netrono.me by Netlify).

## Documentation Prose (Hard Rule)
Write every page in `documentation/`, `docs/`, and `README.md` in Simplified Technical English (ASD-STE100, plain mode). This covers new text and every text you change. If your harness has the `simple-english` and `unslop` skills, run both on that text before you commit. The rules:
- Procedures: imperative mood, one instruction per sentence, 20 words at most. Descriptions: simple tenses, 25 words at most, one topic per paragraph.
- Put the condition first, with a comma: "If the build fails, read the log."
- Use active voice with a named actor, and simple tenses ("completed", not "has completed").
- Use only the modals `can`, `will`, and `must`. A required "should" becomes "must". Delete an optional one.
- Write complete sentences: articles, "that", and verbs stay. Spell out contractions, arrows, and abbreviations.
- Punctuate with periods and commas. Use a colon only before a list, an example, or a code block. Use no semicolons and no dashes between clauses.
- Give each word one meaning per document: "make sure that" for check, verify, confirm, and ensure, and "configuration" for config, settings, and options. Keep one term per concept.
- Define a concept term at its first use, in under ten words. Name the host, the folder, the flag, or the prior step that a command needs.
- State the fact, the mechanism, or the number. Replace tone words with plain ones ("use" for leverage or utilize, "help" for facilitate, "is" for serves as). Cut simply, seamlessly, robust, powerful, crucial, delve, enhance, "in order to", "it is worth noting", "not just X, but Y", an "-ing" clause after a comma, forced groups of three, and closing summaries.
- Format: sentence-case headings, straight quotes, no emoji, no bold for emphasis. Use a vertical list for three or more parallel items.
- Warnings: the command or condition first, then the risk.
- Use American spelling. Keep code, commands, identifiers, error text, and product names exactly as they are.

## Build, Test, and Development Commands
- `make build`: installs web deps, builds the UI, embeds assets, compiles `bin/netronome`.
- `make run`: builds and runs `serve --config config/config.toml`.
- `make dev`: starts Vite + Go watcher (`air`) in tmux.
- `make watch`: runs Go watcher only (prompts to install `air` if missing).
- `make docker-build` / `make docker-run`: build and run a Docker image.
- `pnpm -C web dev|build|lint`: frontend dev server, production build, lint.
- `pnpm -C web test`: frontend unit tests (`node:test`, no test framework). CI runs this in the web job.
- `go test ./...`: backend unit tests.
- `./test/test-local.sh`: local dockerized scenario tests.
- If the server stops with `applied migration number on db cannot be greater than the defined migration list` after a branch switch, the local database has migrations that the branch does not have. Back up `config/netronome.db` and its `-wal` and `-shm` files. Then run `sqlite3 config/netronome.db "DELETE FROM schema_migrations WHERE id >= N;"`, where N is the number of migration files on the branch. The columns that those migrations added stay. If you switch back, the `ALTER TABLE ... ADD COLUMN` of those migrations fails with `duplicate column name`.

## Coding Style & Naming Conventions
- Go: `gofmt` or `goimports`, Go 1.27 target. If `go.mod` needs a newer Go, change these in the same PR:
  - `GO_VERSION` in `.github/workflows/release.yml`.
  - The `golang:` image in `distrib/docker/ci.Dockerfile`. The release workflow builds the Docker image from this file.
  - The `golang:` image in `distrib/docker/Dockerfile`.
  - The Go target on this line.
- Migrations: give each new file in `internal/database/migrations/sqlite/` and `postgres/` a number higher than every existing number. The migrator counts the applied rows and runs only the files after that count. An upgrade skips a file with a reused or lower number and gives no error. A fresh install runs every file, so the tests pass. After a rebase onto `develop`, make sure that no new migration on `develop` uses your number.
- TypeScript: 2-space indentation, PascalCase components, camelCase utilities.
- Tailwind tokens should align with the `@theme` block in `web/src/index.css`.
- Lint: `pnpm -C web lint` (ESLint).
- Before you write logic that a published spec or format defines (Markdown, HTML, URLs, CSV, semver, cron, time zones), use a well-proven library for it. Name the library in the PR body. If hand-written logic of this kind needs a second fix for an edge case, replace it with the library.

## Testing Guidelines
- Go tests live alongside code as `*_test.go`.
- Frontend tests live alongside code as `*.test.ts` and use `node:test`; do not add a test framework.
- Prefer table-driven tests for new backend logic.
- Scenario scripts live in `test/`; add fixtures under `test/data/`.
- `test/` also contains docker-compose scenarios, SSL cert scripts, and base-URL test harnesses.
- If you skip tests, note why in the PR description.

## Commit & Pull Request Guidelines
- Conventional Commits: `feat|fix|refactor|build|ci|chore|docs|style|perf|test`.
- Keep changes scoped; avoid unrelated refactors and mass formatting.
- Write the PR body with `.github/pull_request_template.md` and follow its comments.
- For issue-driven fixes, review attached screenshots/log images before coding.
- Use `pnpm` only; do not add `package-lock.json`.

## Security & Configuration Tips
- Keep secrets out of Git; use `.env` and redact `config/config.toml`.
- Avoid committing real GeoLite databases or generated logs.
- SQLite `.db` files are gitignored; do not commit them.
- Document any config changes in `docs/` when behavior changes.

## Agent skills

### Issue tracker

Issues live in GitHub Issues on `autobrr/netronome`, via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical triage roles, each label string equal to its name. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
