# FinAI Backend

This is the **backend** repo (Go 1.24 + Gin + PostgreSQL via pgx/v5, hand-written SQL,
no ORM). It is one of two separate git repos in the FinAI project; the frontend is a
sibling `../frontend/` repo (`Leongt1/FinAI`). This repo is `Leongt1/go_backend` and
auto-deploys to Render on `main`. The database is Neon-hosted Postgres (migrations are
applied manually, not on deploy).

**Full project context, architecture, DB schema, and the complete Go coding rules live
in `../00-about.md`** (the single source of truth for the whole project). Read it first.
If you only have this repo checked out, ask for that file.

## Must-follow basics (see `../00-about.md` for the full set)

- **Verify before committing:** `go build ./... && go vet ./...` - 0 errors.
- **Never work on `main`** (it auto-deploys to Render prod). Branch per task
  (`feature/<name>` or `fix/<name>`), push, open a PR with `gh pr create`; the user
  reviews and merges. Work is issue-driven - backlog is GitHub issues on
  `Leongt1/go_backend`; reference `Closes #N` in the PR.
- **Layering is one-way:** handler -> service -> repository -> DB. Handlers never touch
  the DB; repositories hold no business rules; domain validation lives in domain
  constructors (`NewX`) and `Update` methods. Feature layout:
  `internal/<feature>/{domain,repository,service,handler,routes.go}`.
- **Ownership checks are mandatory** on every user-scoped read/write: compare the
  resource's `UserID` to the JWT caller ID. Non-admins access only their own data.
  Never trust client-supplied role/user IDs.
- **Money:** store integer **paisa** (`int64`); convert at the handler edge with
  `int64(math.Round(rupees*100))` - never a bare cast (truncation bug). Response DTOs
  divide by 100.
- **Errors:** sentinel `DomainError`s per feature; handlers report via `c.Error(err)` -
  never write ad-hoc JSON error bodies. Never ignore a returned `err`.
- **SQL:** parameterized only; every user-scoped query has `WHERE user_id = $n`; check
  `rows.Err()`; return `[]T{}` (never nil) from list queries.
- **Never run repo-wide `gofmt -w`** (rewrites CRLF across every file) - format only
  files you touch.
- No AI attribution in commits/PRs. Use `-`, never em/en dashes.

See `../00-about.md` for the full Go rules, DB schema, API surface, and current state.
