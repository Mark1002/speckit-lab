# speckit-lab

A practice project for spec-driven development with [GitHub Spec Kit](https://github.com/github/spec-kit). It builds Taskify, the app from the official [Quickstart](https://github.github.io/spec-kit/quickstart.html), with Claude Code as the coding agent instead of Copilot.

## What Taskify is

A team Kanban board web app. Phase 1 covers:

- 5 predefined users: 1 product manager and 4 engineers. There is no login; you pick yourself from a list.
- 3 sample projects, and you can create more.
- Tasks live in 4 columns: To Do, In Progress, In Review, Done. Move them by drag and drop or with a drop-down.
- Tasks can be assigned and edited. Anyone can comment; only a comment's author can edit or delete it.

## The Spec Kit workflow

One full pass in the Quickstart's order:

| Step | Command | Output |
| --- | --- | --- |
| 1 | `/speckit-constitution` | [`constitution.md`](.specify/memory/constitution.md): 5 project principles |
| 2 | `/speckit-specify` | [`spec.md`](specs/001-taskify-kanban/spec.md): what to build and what is out of scope |
| 3 | `/speckit-clarify` | Clarifications section in `spec.md`: answers to 5 questions |
| 4 | `/speckit-plan` | [`plan.md`](specs/001-taskify-kanban/plan.md), [`research.md`](specs/001-taskify-kanban/research.md), [`data-model.md`](specs/001-taskify-kanban/data-model.md), [`contracts/api.md`](specs/001-taskify-kanban/contracts/api.md), [`quickstart.md`](specs/001-taskify-kanban/quickstart.md) |
| 5 | `/speckit-checklist` | [`checklists/security.md`](specs/001-taskify-kanban/checklists/security.md): 15 requirement-quality checks |
| 6 | `/speckit-tasks` | [`tasks.md`](specs/001-taskify-kanban/tasks.md): 36 tasks |
| 7 | `/speckit-analyze` | Consistency report across the documents (read-only); 9 issues fixed |
| 8 | `/speckit-implement` | Built in 3 runs: MVP, the remaining user stories, final checks |
| 9 | `/speckit-converge` | First run added 4 tasks (T037–T040); second run reported Converged |

## Differences from the official Quickstart

| Item | Official | This project | Why |
| --- | --- | --- | --- |
| Init | `specify init taskify --integration copilot` | `specify init --here --integration claude` | Claude Code, in an existing folder |
| Architecture | Microservices | One application | Keep it simple; constitution principle III forbids microservices |
| Backend | .NET Aspire | Go 1.25 standard library, 0 third-party modules | Constitution principle IV: as few dependencies as possible |
| Database | Postgres | One JSON file, written atomically | Nothing to install |
| Frontend | Blazor Server | Plain HTML, CSS and JavaScript, no build step | Same reason |
| Real-time updates, notifications | Yes | No; refresh to see changes | The spec puts them out of scope |
| Network access | Not specified | Listens on `127.0.0.1` only | There is no login, so only this computer may connect |

The command names are the same for both agents: `/speckit-*`.

## Running it

Requires Go 1.25 or newer.

```sh
go test ./...
go run .
```

Open <http://127.0.0.1:8080/>. The first start creates `taskify.json` with sample data (git-ignored); delete it to start over. Use `-port` to change the port and `-data` to change the data file.

More checks are in [`quickstart.md`](specs/001-taskify-kanban/quickstart.md).

## Layout

```text
.specify/                    Spec Kit templates, scripts and the constitution
specs/001-taskify-kanban/    All spec documents for this feature
main.go  api.go  store.go  validate.go  seed.go   Backend
*_test.go                    Tests
web/                         Frontend (embedded into the binary)
```

`.claude/` (the `/speckit-*` skills Spec Kit installs for Claude Code) is not in the repo. To use them after cloning, start with `specify integration --help`.

## Known issues

- Task T036 (the `govulncheck` scan) is left open on purpose. The project stays on `go1.25.0`, where the scan finds 20 Go standard library vulnerabilities; `go1.25.13` fixes them. Under the constitution, this version is not released.
- Scan results depend on the Go version that runs the scan. Scan with the same Go that builds the release.
