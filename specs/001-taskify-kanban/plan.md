# Implementation Plan: Taskify Kanban Board (Phase 1)

**Branch**: `main` (no feature branch; feature dir `specs/001-taskify-kanban/`) | **Date**: 2026-10-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-taskify-kanban/spec.md`

## Summary

Five fixed users, picked without a password, manage projects, tasks, assignments, and comments
on a four-column Kanban board. One Go 1.25 binary using only the standard library serves a
plain HTML/CSS/JavaScript frontend and a REST API on `127.0.0.1`, and saves everything to one
JSON file with an atomic replace after every change. Security rests on loopback-only listening,
a `Host` allowlist, Go 1.25's `CrossOriginProtection`, strict server-side validation of every
input, and `textContent`-only rendering ([research.md](./research.md) R5–R8).

## Technical Context

**Language/Version**: Go 1.25 (`go.mod` declares `go 1.25`); JavaScript (ES2020, no build
step), HTML, CSS

**Primary Dependencies**: None. Go standard library only (`net/http`, `encoding/json`, `embed`,
`log/slog`, `sync`, `os`, `unicode`, `unicode/utf8`, `strings`, `strconv`, `flag`)

**Storage**: One JSON file (`-data`, default `taskify.json`, mode `0600`), rewritten atomically
after every change ([research.md](./research.md) R3)

**Testing**: `go test ./...` with `testing` and `net/http/httptest`; browser flows checked by
hand with [quickstart.md](./quickstart.md)

**Target Platform**: One desktop computer (macOS, Linux, or Windows) running the binary, with a
modern desktop browser on the same computer

**Project Type**: Web application — one Go binary serving the REST API and static frontend

**Performance Goals**: Board with 200 tasks fully shown within 2 seconds (SC-004); every change
on disk before its response is sent

**Constraints**: Listen on `127.0.0.1` only (host not configurable); zero third-party modules;
no JavaScript build step; request bodies ≤ 64 KiB

**Scale/Scope**: 5 fixed users taking turns on one computer; 3 sample projects plus any created;
up to ~200 tasks per project; 4 views (user picker, project list, board, task dialog)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / gate | How the plan meets it | Pre-research | Post-design |
| ---------------- | --------------------- | ------------ | ----------- |
| I. Security-First | Spec Access Rules enforced in handlers for every request; comment author check (403); deny by default via exact `ServeMux` routes, `Host` allowlist, no CORS; no secrets exist; fixed error messages, generic 500 ([contracts/api.md](./contracts/api.md)) | PASS | PASS |
| II. Validate All Input | Every input in the spec's table has a rule in [data-model.md](./data-model.md); headers, path IDs, body size, content type, JSON shape, and the `-port`/`-data` flags are validated; one test or more per rule (R10) | PASS | PASS |
| III. One Small Application | One Go module, one `main` package, one binary serving UI and API; no services, no layers beyond handler → store | PASS | PASS |
| IV. Minimal Dependencies | Zero `require` lines, so no `go.sum` to pin. `govulncheck` is a dev-only tool run with `go run`, never added to `go.mod`; justified in R12 | PASS | PASS |
| V. Document Non-Obvious Intent | Comments planned only where code cannot say why: `Host` check (DNS rebinding), header identity (no login), atomic rename, save rollback, `assigneeId` 0 = unassigned, 64 KiB cap | PASS | PASS |
| Sec: Injection | No SQL, no shell, no `eval`; data goes through `encoding/json` only | PASS | PASS |
| Sec: Output encoding | `textContent` only; CSP `default-src 'self'`; test fails if `app.js` contains `innerHTML` | PASS | PASS |
| Sec: Accounts and sessions | No accounts in phase 1 (spec FR-002) | N/A | N/A |
| Sec: Transport | Loopback-only and not a production deployment, so plain HTTP; HTTPS becomes mandatory if FR-006 ever allows other computers (R13) | N/A | N/A |
| Sec: Secrets | None exist | PASS | PASS |
| Sec: Logging | `log/slog` logs refusals and rejected input by field and rule, never the value (R11) | PASS | PASS |
| Sec: Known vulnerabilities | `govulncheck` before release; any finding blocks it ([quickstart.md](./quickstart.md)) | PASS | PASS |
| Spec gate | Spec has the Input Validation and Access Rules tables | PASS | PASS |
| Test gate | Tests per validation rule and per access check (R10) | PASS | PASS |

No violations. Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-taskify-kanban/
├── plan.md              # This file
├── research.md          # Phase 0 decisions
├── data-model.md        # Entities, rules, storage file, sample data
├── quickstart.md        # Run and validation guide
├── contracts/
│   └── api.md           # REST contract
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (/speckit-tasks, not created here)
```

### Source Code (repository root)

```text
go.mod              # module taskify, go 1.25, no requires
main.go             # flags (-port, -data), open store, listen on 127.0.0.1, slog setup
api.go              # handler: Host check, security headers, CrossOriginProtection, routes
validate.go         # text/ID/status/assignee rules from data-model.md
store.go            # in-memory data, mutex, JSON load, atomic save, rollback
seed.go             # five users, sample projects and tasks
web/
├── index.html      # one page: picker, project list, board, task <dialog>
├── app.js          # hash routing, fetch with X-Taskify-User, drag-and-drop, textContent only
└── style.css
api_test.go         # routes, access rules, Host/cross-origin, innerHTML guard
validate_test.go    # one case or more per Input Validation rule
store_test.go       # persistence, no re-seed, corrupt file, rollback
.gitignore          # add taskify.json
```

**Structure Decision**: One Go `main` package at the repository root, next to `.specify/` and
`specs/`. Tests sit beside the code they cover in the same package. The frontend is embedded
into the binary with `//go:embed web`, so the app ships and runs as one file.

## Complexity Tracking

No constitution violations to justify.
