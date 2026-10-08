# Research: Taskify Kanban Board (Phase 1)

All Technical Context items were fixed by the user's stack choice, so there were no open
unknowns. Each entry records a decision the stack left open. Stdlib APIs were checked with
`go doc` against the local toolchain (go1.26.2).

## R1. Go version and toolchain

- **Decision**: `go.mod` declares `go 1.25`. No `toolchain` line. Zero `require` lines, so no
  `go.sum`.
- **Rationale**: User asked for Go 1.25. A newer local toolchain (1.26.2) builds a `go 1.25`
  module with 1.25 language semantics.
- **Alternatives considered**: Pinning `toolchain go1.25.x` — rejected; forces a toolchain
  download for no gain.

## R2. HTTP routing

- **Decision**: `net/http.ServeMux` with method and wildcard patterns
  (`"PATCH /api/projects/{pid}/tasks/{tid}"`, `r.PathValue("tid")`).
- **Rationale**: Built in since Go 1.22. Returns 405 for wrong methods and 404 for unknown paths
  without extra code, which gives deny-by-default routing.
- **Alternatives considered**: Hand-written path splitting — more code, easier to get wrong.

## R3. Storage: one JSON file

- **Decision**: All data lives in memory behind one `sync.Mutex` and is written to one JSON file
  after every change.
  - Write = `os.CreateTemp` in the same directory → write → `Sync` → `Close` → `os.Rename`.
    Rename is atomic on the same filesystem, so a crash leaves the old or new file, never half.
  - `os.CreateTemp` creates the file with mode `0600`, so only the owner can read it.
  - The store keeps the bytes of the last successful save. If a save fails, it restores memory
    from those bytes and the request returns 500. This satisfies FR-027 (no partial change).
  - On start: file missing → seed sample data and save (FR-009, FR-029). File present but not
    valid JSON → exit with an error and **do not** overwrite it, so a damaged file is never
    replaced by sample data.
  - Path set by `-data` flag, default `taskify.json`; the file is git-ignored.
- **Rationale**: User asked for one JSON file. Atomic rename plus rollback keeps the file and
  memory consistent with ~20 lines of stdlib code.
- **Alternatives considered**: Write in place with `os.WriteFile` — a crash mid-write corrupts
  the only copy. `RWMutex` — no measurable gain for 5 users on one computer.
- **Known ceiling**: Whole file rewritten on every change; fine to a few thousand tasks. Upgrade
  path is an embedded database, which needs a constitution-justified dependency.

## R4. IDs and ordering

- **Decision**: One integer counter `nextId` in the file, shared by projects, tasks, and
  comments. IDs start at 1, only go up, and are never reused.
- **Rationale**: Creation order equals ID order, so FR-012 and FR-021 ("oldest first") are a sort
  by ID. A deleted comment's ID can never point at new data.
- **Alternatives considered**: Random UUIDs — need `crypto/rand` formatting code and give no
  order; not needed when nothing outside the app creates IDs.

## R5. The five users and "who am I"

- **Decision**: The five users are constants in Go code with IDs 1–5, not stored in the file.
  The browser keeps the picked user ID in `localStorage` and sends it on every API call in the
  header `X-Taskify-User`. The server rejects a missing or unknown ID with 401.
- **Rationale**: Users can never change (Access Rules), so code constants make that structural.
  A custom header, unlike a cookie, is never attached automatically by the browser to requests
  from other sites.
- **Alternatives considered**: A cookie — sent automatically on cross-site requests, so it needs
  CSRF handling for identity too. A `POST /api/session` endpoint — extra state for no security
  gain, since there is no password.

## R6. Keeping it local-only (FR-006, SC-009)

- **Decision**: Three layers.
  1. Listen on `127.0.0.1` only. The host is hard-coded; only the port is a flag (`-port`,
     default 8080, must be 1–65535). No flag can make it listen on another interface.
  2. Reject any request whose `Host` header is not `127.0.0.1:<port>` or `localhost:<port>`
     (421). This blocks DNS rebinding, where a web page re-points its own domain at 127.0.0.1
     to make the browser treat Taskify as same-origin.
  3. Wrap the handler in `http.NewCrossOriginProtection()` (Go 1.25) so browsers cannot send
     state-changing requests from other sites (CSRF). No CORS headers are ever sent.
- **Rationale**: Layer 1 stops other computers. Layers 2 and 3 stop websites open in a browser
  on the same computer from reaching the API, which binding alone does not stop.
- **Alternatives considered**: An `-addr` flag — one typo exposes the unauthenticated app to the
  network. Hand-written Origin checks — `CrossOriginProtection` is the stdlib version.

## R7. Input validation

- **Decision**:
  - Request bodies: `http.MaxBytesReader` at 64 KiB (largest valid body is ~20 KB: 5,000
    four-byte characters); `Content-Type` must be `application/json`; `json.Decoder` with
    `DisallowUnknownFields`; exactly one JSON value (a second `Decode` must return `io.EOF`).
  - "Characters" = Unicode code points (`utf8.RuneCountInString`).
  - "Only spaces counts as empty" = `strings.TrimSpace(v) == ""`. Stored text is never trimmed
    or changed (constitution: reject, do not clean).
  - "No control characters" = reject any rune where `unicode.IsControl` is true, except `\n` in
    multi-line fields. Browsers normalise textarea values to `\n`, so `\r` is rejected too.
  - Project name uniqueness: `strings.EqualFold` against every other project.
  - Path IDs: `strconv.Atoi`; anything not a positive integer naming an existing record → 404.
  - Error messages are fixed strings naming the field and rule; they never include the input.
- **Rationale**: Every rule in the spec's Input Validation table maps to one stdlib call, and
  each is testable on its own.
- **Alternatives considered**: Counting grapheme clusters (what a person sees as one character)
  — needs a third-party library; code points are close enough for limits this large. Rejecting
  Unicode format characters (category Cf) too — would break emoji that use zero-width joiners.

## R8. Output safety (FR-026, SC-007)

- **Decision**: The frontend sets user text only through `textContent` / `value`, never
  `innerHTML`. Every response carries `Content-Security-Policy: default-src 'self';
  frame-ancestors 'none'`, `X-Content-Type-Options: nosniff`, and `Referrer-Policy: no-referrer`.
  JSON responses use `encoding/json`, which escapes `<`, `>`, `&`.
- **Rationale**: `textContent` makes markup display as text. The CSP blocks inline scripts as a
  second line of defence, which also means no inline `<script>` or `style=` in the HTML.
- **Alternatives considered**: An HTML sanitiser — needs a dependency and is unnecessary when
  no user HTML is ever rendered.

## R9. Frontend

- **Decision**: One `index.html`, one `app.js`, one `style.css`, embedded with `//go:embed` and
  served by `http.FileServerFS`. Hash routes (`#/`, `#/projects/{id}`) switch views. Task
  details open in a native `<dialog>`. Moving uses native drag-and-drop (`draggable`,
  `dragover`, `drop`) plus a column `<select>` in the task dialog for keyboard users (FR-015).
  Board data is fetched again after every change and on every navigation; there is no polling.
  Any 401 response clears the stored user ID and returns to the user picker.
- **Rationale**: No build step, no framework, one binary to run.
- **Alternatives considered**: Server-rendered `html/template` pages — would also work, but the
  user asked for a REST API consumed by plain JavaScript.

## R10. Testing

- **Decision**: `testing` + `net/http/httptest` only. Table-driven tests: one case or more per
  rule in the Input Validation table, plus an at-limit and a limit-plus-one case for every
  length limit (SC-005), access tests for comment edit/delete by a
  non-author (SC-006), missing/unknown user header, bad `Host`, cross-origin `POST`, restart
  persistence and no re-seed (SC-003, FR-029), corrupt file refusal, rollback on failed save.
  One test reads the embedded `app.js` and fails if it contains `innerHTML` (SC-007 guard).
  Each test is run and seen failing before the code it covers is written.
- **Rationale**: Constitution test gate; user asked for the standard `testing` package.
- **Alternatives considered**: Browser automation for the UI — a dependency; UI flows are
  checked by hand with [quickstart.md](./quickstart.md).

## R11. Logging

- **Decision**: `log/slog` text handler to stderr. Log: rejected input (field and rule, never
  the value), 401/403/421 refusals, cross-origin rejections, save failures, startup address.
- **Rationale**: Constitution Security Requirements (security events SHOULD be logged).
  Logging the rule rather than the value keeps user text out of logs.

## R12. Vulnerability scanning

- **Decision**: Before a release, run `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.
  It is not added to `go.mod` and ships nothing in the binary.
- **Rationale**: The constitution requires an audit before release. With zero third-party
  modules, the remaining risk is the Go standard library itself, which `govulncheck` (the Go
  team's official scanner) checks. The `go` command has no built-in audit.
- **Alternatives considered**: Skipping the scan because there are no dependencies — misses
  stdlib CVEs fixed in Go patch releases.

## R13. HTTPS

- **Decision**: Plain HTTP on the loopback interface.
- **Rationale**: The constitution requires HTTPS for production traffic. Phase 1 is not a
  production deployment and traffic never leaves the computer (FR-006). A self-signed
  certificate would add browser warnings and key handling with no protection gained.
- **Revisit when**: FR-006 changes to allow other computers. HTTPS then becomes mandatory.
