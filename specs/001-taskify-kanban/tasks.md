---

description: "Task list for Taskify Kanban Board (Phase 1)"
---

# Tasks: Taskify Kanban Board (Phase 1)

**Input**: Design documents from `/specs/001-taskify-kanban/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md, quickstart.md

**Tests**: Requested. The constitution's Test gate, spec SC-005/SC-006, and research R10 require
them. Every test task is written first, then run with `go test -run <Name> ./...` and seen
failing **on its assertions** (not on a compile error) before its implementation task starts.

**Organization**: Tasks are grouped by user story. All Go files are `package main` at the
repository root; the frontend is in `web/` (plan §Project Structure).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

## Shared strings (used by tests and code)

Field errors (`validate.go`), where `<f>` is the JSON field name and `<max>` is plain digits:

- `<f> must be 1 to <max> characters` — required text that is empty, only whitespace, or too long
- `<f> must be at most <max> characters` — optional text that is too long
- `<f> must be a single line`
- `<f> must not contain control characters`
- `status must be one of todo, in_progress, in_review, done`
- `assigneeId must be 0 or a user ID from 1 to 5`
- `name is already used by another project`
- `<f> has the wrong type or is null` — from `decodeBody` in `api.go`, only for a key in `allowed`

Request errors (`api.go`), sent as `{"error":{"message":"…"}}`:

| Status | Message |
| ------ | ------- |
| 400 | `request body must be one JSON object` |
| 400 | `request body has a field that is not allowed or has the wrong type` |
| 400 | `request body must change at least one field` |
| 401 | `pick a user first` |
| 403 | `cross-site requests are not allowed` |
| 403 | `only the comment's author can change it` |
| 404 | `not found` |
| 413 | `request body is too large` |
| 415 | `request body must be JSON` |
| 421 | `requests must use 127.0.0.1 or localhost` |
| 500 | `something went wrong; nothing was saved` |

---

## Phase 1: Setup

- [X] T001 Create `go.mod` at the repository root containing exactly `module taskify` and `go 1.25`, with no `require` or `toolchain` lines (research R1)
- [X] T002 [P] Create `.gitignore` at the repository root with the lines `taskify.json`, `.taskify-*.tmp`, and `/taskify`
- [X] T003 [P] Create `web/index.html`: `<!doctype html>`, `<meta charset="utf-8">`, `<title>Taskify</title>`, `<link rel="stylesheet" href="/style.css">`, `<header id="bar"></header>`, `<main id="app"></main>`, an empty `<dialog id="task-dialog"></dialog>`, and `<script src="/app.js" defer></script>`. No inline `<script>`/`<style>` content, no `style=` or `on…=` attributes (CSP, research R8). Also create empty `web/app.js` and `web/style.css`

---

## Phase 2: Foundational (blocks every user story)

**Goal**: Validation rules, crash-safe storage with sample data, the secured HTTP shell, the
user picker, and the project list.

- [X] T004 Add compile-only stubs (zero values / `http.NotFoundHandler()`) so every later test compiles and fails on assertions: in `validate.go` — `type fieldError struct{ Field, Message string }` with `Error() string`, `checkText(field, v string, required bool, maxChars int, multiline bool) *fieldError`, `checkStatus(v string) *fieldError`, `checkAssignee(id int) *fieldError`, `parseConfig(args []string) (port int, dataPath string, err error)`; in `store.go` — `User`, `Project`, `Task`, `Comment`, `data` structs with the JSON tags in data-model §Storage file and §User (`nextId`, `projects`, `tasks`, `comments`, `id`, `name`, `role`, `description`, `createdBy`, `createdAt`, `projectId`, `title`, `status`, `assigneeId`, `taskId`, `authorId`, `text`), `type store struct`, `openStore(path string, now time.Time) (*store, error)`, `(s *store) view(fn func(d *data))`, `(s *store) update(fn func(d *data) error) error`, `(d *data) newID() int`; in `seed.go` — `var users []User`, `seedData(now time.Time) data`; in `api.go` — `//go:embed web` `var webFS embed.FS`, `newHandler(s *store, port int, log *slog.Logger) http.Handler`, `decodeBody(w http.ResponseWriter, r *http.Request, allowed []string, dst any) bool`; in `main.go` — `listenAddr(port int) string` and empty `func main() {}`
- [X] T005 [P] Write table-driven tests in `validate_test.go` (one row or more per rule; each length limit has an exactly-at-limit row that passes and a limit-plus-one row that fails, SC-005). Build long strings from `é` so code points ≠ bytes. Rules, quoted from data-model: Project name "Required; 1–100 chars; single line; no control chars"; Project description "Optional (`""`); ≤ 1,000 chars; `\n` allowed; no other control chars"; Task title "Required; 1–200 chars; single line; no control chars"; Task description "Optional (`""`); ≤ 5,000 chars; `\n` allowed; no other control chars"; Comment text "Required; 1–2,000 chars; `\n` allowed; no other control chars"; Status "One of `todo`, `in_progress`, `in_review`, `done`"; Assignee "`0` = unassigned, or 1–5". Cases per text rule: `""`, `"   "`, `" \n "` (required → fail), `"a\nb"`, `"a\r\nb"`, `"a\tb"`, `"a\x00b"`, and `"👨‍👩‍👧"` (zero-width joiners allowed, research R7); status `Done`, `"todo "`, `""` fail; assignee `-1`, `6` fail. Assert exact messages from "Shared strings". `parseConfig`: no args → `8080`, `taskify.json`; `-port 1` and `-port 65535` pass; `-port 0`, `-port 65536`, `-port abc`, `-data ""`, `-data <t.TempDir()>` (a directory), `-data <missing dir>/x.json`, an extra positional argument, and an unknown flag `-addr x` each return an error whose text names the flag (`-port`/`-data`) or says the argument is not allowed, never echoing the value
- [X] T006 [P] Write tests in `store_test.go` (each uses a file in `t.TempDir()`): first `openStore` on a missing file seeds — 3 projects named "Website Refresh", "Mobile App Beta", "Internal Tools", all `createdBy` 1; each project has 4–6 tasks, at least one task in every status and at least one with `assigneeId` 0; across all tasks each user 1–5 is an assignee at least once; no comments; file mode is `0600`. Reopening the same file returns equal data and does not seed again (add a project via `update`, reopen, count is 4) (FR-028, FR-029). A file containing `{not json` makes `openStore` return an error and leaves the file bytes unchanged. An `update` whose `fn` changes data and then returns an error leaves `view` and the file unchanged. Save failure: open a store, `os.RemoveAll` its directory, call `update` with a change → returns an error and `view` shows the old data (FR-027). `newID` only increases and is not reused after a comment is removed (research R4)
- [X] T007 [P] Write foundation tests in `api_test.go` with a helper `newTestServer(t) (http.Handler, *store)` (store in `t.TempDir()`, port 8080, `slog` to `io.Discard`) and requests via `httptest.NewRecorder` with `Host: 127.0.0.1:8080`: Host `127.0.0.1:8080` and `localhost:8080` reach `GET /api/users`; Host `evil.example:8080`, `127.0.0.1:9999`, and `localhost` → 421, also for `GET /` (FR-006). `POST /api/projects` with `Sec-Fetch-Site: cross-site` → 403 JSON `cross-site requests are not allowed`. Responses with status 200, 404, and 421 all carry `Content-Security-Policy: default-src 'self'; frame-ancestors 'none'`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, and no `Access-Control-Allow-Origin`. `GET /api/users` with no `X-Taskify-User` → 200 with exactly the five users in data-model §User. `GET /api/projects` with `X-Taskify-User` missing, `0`, `6`, `abc`, `"1 "`, `"+1"`, or `"01"` → 401 `pick a user first`; with `2` → 200, the three projects in ID order. `GET /api/nope` → 404 JSON `not found`. `decodeBody` called directly: no `Content-Type` → 415; body of 64 KiB + 1 byte → 413; `[]` and `{"a":1}{"b":2}` → 400 `request body must be one JSON object`; a field not in `allowed` → 400 with no `field` and a message that does not contain that field's name; with `allowed` = `["title"]`, `{"title":null}` and `{"title":5}` (into a `*string`) → 400 with `field` `title` and message `title has the wrong type or is null`. `GET /` → 200 HTML; `GET /app.js` and `GET /style.css` → 200; `GET /index.html` and `GET /x.js` → 404. `listenAddr(8080)` == `127.0.0.1:8080` (SC-009). Guard (SC-007): read `web/app.js` from `webFS` and fail if it contains `innerHTML`, `outerHTML`, `insertAdjacentHTML`, or `document.write` — app.js is empty now, so prove the guard can fail by adding a temporary `innerHTML` line, running it red, then removing the line
- [X] T008 [P] Implement `validate.go` to pass T005, per data-model §Validation definitions: chars = `utf8.RuneCountInString`; required = `strings.TrimSpace(v) != ""`; single line = no `\n`; reject any rune with `unicode.IsControl` true except `\n` when `multiline`; never trim or change the value. Messages exactly as in "Shared strings". `parseConfig` uses `flag.NewFlagSet("taskify", flag.ContinueOnError)` with output `io.Discard`, defaults `-port 8080` and `-data taskify.json`, then enforces data-model §Configuration inputs: "`-port` Integer from 1 to 65535", "`-data` Not empty; must not name a directory; its parent directory must already exist", and "Any other command-line argument is refused"; returned errors name the flag and never contain the value
- [X] T009 [P] Implement `store.go` to pass T006: `openStore` reads the file; if it does not exist, `data = seedData(now)` and save; if JSON is invalid, return an error without writing anything. `save` = `os.CreateTemp(filepath.Dir(path), ".taskify-*.tmp")` → write `json.MarshalIndent` → `Sync` → `Close` → `os.Rename` onto `path`, removing the temp file on any error; keep the last saved bytes in the store. `update` locks the mutex, runs `fn`, saves; if `fn` or the save fails, restore by unmarshalling the last saved bytes into a fresh `data` value (not into the existing one, which would merge slices) and return the error. `view` locks and runs `fn`. `newID` returns `nextId` and increments it. Comment only the non-obvious why: rename is atomic so a crash never leaves half a file; restore keeps memory equal to disk (research R3)
- [X] T010 [P] Implement `seed.go`: `users` = the five rows in data-model §User (IDs 1–5, exact names and roles); `seedData(now)` returns projects 1–3 ("Website Refresh", "Mobile App Beta", "Internal Tools"), `createdBy` 1, empty descriptions or one short line, followed by 4–6 tasks per project with short realistic titles meeting every T006 seed assertion, all `createdAt` = `now` in UTC RFC 3339, `nextId` one past the last ID, no comments
- [X] T011 Implement the HTTP shell in `api.go` to pass T007 (after T009, T010): wrap as `securityHeaders(hostCheck(cop.Handler(mux)))`. `hostCheck` allows exactly `127.0.0.1:<port>` and `localhost:<port>`, else 421 (comment: blocks DNS rebinding, research R6). `cop := http.NewCrossOriginProtection()` with `SetDenyHandler` writing the 403 JSON error. Helpers: `writeJSON`, `writeError(w, status, field, message)` (omit `field` when empty; never include request data), `requireUser` (parse `X-Taskify-User` with `strconv.Atoi`, then also require `strconv.Itoa(n) == s` so `+1` and `01` are refused, must be 1–5, else 401; comment: identity is a claim because phase 1 has no login), `pathID(r, name) (int, bool)` (positive integers only, with the same `strconv.Itoa(n) == s` check), and `decodeBody`: `Content-Type` media type must be `application/json` (415); `http.MaxBytesReader` at `64 << 10` (413); read the body as `map[string]json.RawMessage` with exactly one JSON value (400 one JSON object); a key not in `allowed` → 400 `request body has a field that is not allowed or has the wrong type` with no `field`; a key in `allowed` whose value is `null` → 400 with `field` = that key and message `<key> has the wrong type or is null`; then `json.Unmarshal` into `dst`, and on a `*json.UnmarshalTypeError` whose `Field` is in `allowed` return that same per-field 400 (any other unmarshal error → the general 400 with no `field`). Routes: `/api/` catch-all → 404 JSON; `GET /api/users` (no user needed); `GET /api/projects` (requireUser; projects in ID order). Static files: register exactly `GET /{$}`, `GET /app.js`, `GET /style.css` with `http.FileServerFS(sub)` where `sub, _ := fs.Sub(webFS, "web")` — do not register `GET /`, which conflicts with `/api/` and panics at start-up. Any failure from `store.update` → 500 with the fixed message. Log with `slog` (research R11): 401/403/421 refusals, rejected input by field and rule only, 500 errors with the Go error; never log request values
- [X] T012 Implement `main.go` (after T008, T009, T011): `parseConfig(os.Args[1:])` → on error print it to stderr and exit 2; `slog.New(slog.NewTextHandler(os.Stderr, nil))`; `openStore(dataPath, time.Now().UTC())` → on error log and exit 1; `net.Listen("tcp", listenAddr(port))`; serve with `http.Server{ReadHeaderTimeout: 5s, ReadTimeout: 10s, WriteTimeout: 10s, IdleTimeout: 60s, MaxHeaderBytes: 16 << 10}`; log the listening address. `listenAddr` returns `fmt.Sprintf("127.0.0.1:%d", port)` with a one-line comment that the host is fixed on purpose (FR-006)
- [X] T013 [P] Implement the frontend shell in `web/app.js`: `el(tag, attrs, ...children)` creates elements and sets strings only via `textContent` (never `innerHTML`); `currentUser()` reads `localStorage.taskifyUser` inside try/catch; `api(method, path, body)` calls `fetch` with `X-Taskify-User` and, when there is a body, `Content-Type: application/json`, parses the JSON error body, and on 401 removes `taskifyUser` and sets `location.hash = '#/'` (research R9); `showError(target, error)` shows `error.message` by text. Router on `load` and `hashchange`: `#/` user picker, `#/projects` project list, `#/projects/{id}` board placeholder; any route other than `#/` with no picked user goes to `#/` (FR-002). Picker: `GET /api/users`, one button per user showing name and role; choosing stores the ID and goes to `#/projects`. Header `#bar` shows the current user's name and a "Switch user" button that goes to `#/` (FR-003). Project list: `GET /api/projects`, one link per project to `#/projects/{id}`
- [X] T014 [P] Write base styles in `web/style.css`: page layout, header bar, buttons and links with visible keyboard focus outlines, error text style

**Checkpoint**: `go test ./...` passes; `go run .` serves the picker and project list on 127.0.0.1.

---

## Phase 3: User Story 1 - Pick a user and move tasks on a project board (Priority: P1) 🎯 MVP

**Goal**: Open a project's board, see four columns, move tasks by drag or by keyboard, and see
your own tasks stand out.

**Independent Test**: Start with only sample data, pick a user, open a sample project, move a
task to another column, reload, and see it still in the new column (quickstart steps 1–4).

- [X] T015 [P] [US1] Write tests in `api_test.go`: `GET /api/projects/{pid}` returns the project fields plus `tasks` holding only that project's tasks in ID order (FR-012); `pid` of `999`, `abc`, `0`, `-1`, `+1`, `01` → 404; no user → 401; a project with no tasks (insert one directly with `store.update`) returns `"tasks":[]`, not `null`. `PATCH /api/projects/{pid}/tasks/{tid}` `{"status":"in_review"}` → 200 with `status` `in_review`, still `in_review` after reopening the store from the same file; same status again → 200, no change; `{"status":"Done"}` → 400 with `field` `status`; user 5 moves a task assigned to user 2 → 200 (Clarifications Q1); an unassigned task moved to `done` keeps `assigneeId` 0 and an assigned task keeps its `assigneeId` (FR-011, Q4); `{}` → 400 `request body must change at least one field`; `{"status":null}` and `{"createdBy":1}` → 400; a task ID belonging to another project → 404; after every refused request the data file bytes are unchanged. Save failure (FR-027): remove the store's data folder with `os.RemoveAll`, send a valid `PATCH {"status":"done"}` → 500 with `{"error":{"message":"something went wrong; nothing was saved"}}`, and a following `GET` still shows the old status
- [X] T016 [US1] Implement in `api.go` to pass T015: `GET /api/projects/{pid}` (requireUser; `pathID`; 404 if no such project; respond with the project fields plus `tasks` in ID order, where an empty list is written as `[]`, not `null` — start from a non-nil slice). `PATCH /api/projects/{pid}/tasks/{tid}` (requireUser; `decodeBody` with `allowed` = `["status"]` into a struct with `Status *string`; 404 unless the project exists and the task's `projectId` equals `pid`; 400 if no field given; `checkStatus`; in one `store.update` set only `status` — never touch `assigneeId`; respond 200 with the task). Status values per data-model: "One of `todo`, `in_progress`, `in_review`, `done`"
- [X] T017 [US1] Implement the board view in `web/app.js` for `#/projects/{id}`: `GET /api/projects/{id}`; heading with the project name; four columns in the order To Do, In Progress, In Review, Done mapped to `todo`, `in_progress`, `in_review`, `done` (contract §Field values); cards in ID order showing the title and the assignee's name or "Unassigned" (FR-013); cards whose `assigneeId` equals the current user get class `mine` (FR-014). Native drag-and-drop: cards are `draggable="true"` and put the task ID in `dataTransfer` on `dragstart`; columns call `preventDefault` on `dragover` and, on `drop` into a different column, send `PATCH {"status": …}` and then reload the board; errors go through `showError`
- [X] T018 [US1] Add the keyboard move path in `web/app.js` (FR-015): each card is a focusable `<button>`; activating it opens `#task-dialog` showing the task title and a `<select>` of the four columns with the current one selected; changing the selection sends `PATCH {"status": …}`, reloads the board, and keeps the dialog open; a Close button closes it
- [X] T019 [P] [US1] Style the board in `web/style.css`: four equal-width columns, card style, `.mine` cards with a thicker border and a different background so they differ by more than color, and a highlighted drop target during `dragover`

**Checkpoint**: US1 works on its own with the sample data; quickstart manual steps 1–4 pass.

---

## Phase 4: User Story 2 - Create, edit, and assign tasks (Priority: P2)

**Goal**: Add tasks, open full task details, edit title and description, and assign or
unassign anyone.

**Independent Test**: Create a task in a sample project, assign it to an engineer, edit its
title, and see it in To Do with the new title and the engineer's name (quickstart steps 5–7).

- [X] T020 [P] [US2] Write tests in `api_test.go`: `POST /api/projects/{pid}/tasks` `{"title":"Fix header"}` → 201 with `status` `todo`, `assigneeId` 0, `description` `""`, `createdBy` = the `X-Taskify-User` value, `createdAt` parseable as RFC 3339 UTC, and an ID higher than every existing ID; with `"assigneeId":3` and `"description":"a\nb"` → stored exactly as sent. Title rule "Required; 1–200 chars; single line; no control chars": 200 chars → 201, 201 chars, `"   "`, and `"a\nb"` → 400 `field` `title`. Description rule "Optional (`""`); ≤ 5,000 chars; `\n` allowed; no other control chars": 5,000 → 201, 5,001 and `"a\rb"` → 400. `"assigneeId":6` → 400 `field` `assigneeId`; `"assigneeId":"3"`, `"status":"done"`, `"projectId":2`, and `"title":null` → 400; a 64 KiB + 1 body → 413; `Content-Type: text/plain` → 415; unknown project → 404; every refusal leaves the task count unchanged. Title `<script>alert(1)</script>` → 201 and reads back byte-for-byte equal (SC-007). `GET /api/projects/{pid}/tasks/{tid}` → the task fields plus `comments` as `[]` (not `null`); a task from another project → 404. `PATCH` by user 2 `{"assigneeId":4}` → 200 `assigneeId` 4 (Clarifications Q2); `{"assigneeId":0}` clears it; `{"assigneeId":null}` → 400; `{"title":"New","description":""}` → 200; changing the assignee leaves `status` unchanged; an invalid title → 400 and nothing changes
- [X] T021 [US2] Implement in `api.go` to pass T020: `POST /api/projects/{pid}/tasks` (requireUser; `decodeBody` with `allowed` = `["title","description","assigneeId"]`; 404 if no project; `checkText("title", v, true, 200, false)`, `checkText("description", v, false, 5000, true)`, `checkAssignee`; inside one `store.update` create the task with `newID`, `status` `todo`, `createdBy` = user, `createdAt` = now UTC RFC 3339; respond 201). `GET /api/projects/{pid}/tasks/{tid}` (requireUser; 404 rules as in T016; respond with the task plus that task's comments in ID order, `[]` when none). Extend the T016 `PATCH` to `allowed` = `["title","description","status","assigneeId"]` with pointer fields, validate each field that is present with the same rules, and apply all of them in one `store.update`. Assignee per data-model: "`0` = unassigned, or 1–5"
- [X] T022 [US2] Add task creation to the board in `web/app.js`: a "New task" button opens a form with a title input, a description textarea, and an assignee `<select>` ("Unassigned" = 0 plus the five users); submit sends `POST /api/projects/{id}/tasks`, shows a 400 message next to the field named in `error.field`, and on success reloads the board. Do not set `maxlength` (it counts UTF-16 units, not code points; the server is the authority)
- [X] T023 [US2] Expand `#task-dialog` in `web/app.js` (FR-019): load `GET /api/projects/{pid}/tasks/{tid}`; show title, description (keeping line breaks), assignee, the column `<select>` from T018, and the creator's name; edit fields for title, description, and assignee with a Save button that sends `PATCH` with only the changed fields; show a 400 message next to its field; on success reload the dialog and the board
- [X] T024 [P] [US2] Style the task form and dialog in `web/style.css`: labelled fields, field-level error text, description shown with `white-space: pre-wrap`

**Checkpoint**: US1 and US2 both work; quickstart steps 5–7 pass.

---

## Phase 5: User Story 3 - Discuss tasks with comments (Priority: P3)

**Goal**: Everyone reads all comments on a task; only the author can edit or delete theirs.

**Independent Test**: As one user add and edit a comment; as another user see it with no edit or
delete controls and get 403 from the API; as the author delete it (quickstart step 8).

- [X] T025 [P] [US3] Write tests in `api_test.go`: `POST /api/projects/{pid}/tasks/{tid}/comments` `{"text":"Looks good"}` by user 3 → 201 with `authorId` 3 and an RFC 3339 `createdAt`; `GET` task lists comments in ID order. Text rule "Required; 1–2,000 chars; `\n` allowed; no other control chars": 2,000 chars and `"a\nb"` → 201; 2,001 chars, `"  \n "`, and `"a\rb"` → 400 `field` `text`. `{"authorId":2,"text":"x"}` → 400; a task in another project → 404. `PATCH .../comments/{cid}` by its author → 200, new text, `createdAt` unchanged, and no `edited`-style field in the response (FR-022). By another user → 403 `only the comment's author can change it` and the comment is unchanged; by another user with 2,001-char text → 403, not 400 (contract check order). `DELETE` by another user → 403 and the comment remains (SC-006); `DELETE` by the author → 204, gone from `GET` task and still gone after reopening the store; a comment ID under the wrong task → 404; the deleted ID → 404; the next new comment gets a higher ID than the deleted one (research R4)
- [X] T026 [US3] Implement in `api.go` to pass T025: `POST .../comments` (requireUser; `decodeBody` with `allowed` = `["text"]`; 404 unless project, task, and nesting match; `checkText("text", v, true, 2000, true)`; create with `newID`, `authorId` = user, `createdAt` = now UTC RFC 3339; 201). `PATCH .../comments/{cid}` in contract order — `decodeBody` (`["text"]`), then 404 lookups, then 403 if user ≠ `authorId`, then `checkText`; update only `text`; 200. `DELETE .../comments/{cid}`: 404 lookups, 403 if user ≠ `authorId`, remove it from `comments` in one `store.update`, 204. Log every 403 with `slog`
- [X] T027 [US3] Add comments to `#task-dialog` in `web/app.js`: list comments oldest first with the author's name, the posted time (`new Date(createdAt).toLocaleString()`), and the text; show Edit and Delete only on comments whose `authorId` equals the current user (US3-4); Edit swaps in a textarea with Save (`PATCH`); Delete sends `DELETE`; an "Add comment" textarea with Post sends `POST`; show 400 and 403 messages; reload the dialog after each change
- [X] T028 [P] [US3] Style the comment list and comment form in `web/style.css`

**Checkpoint**: US1–US3 work; quickstart step 8 passes.

---

## Phase 6: User Story 4 - Create a new project (Priority: P4)

**Goal**: Any user creates a project that everyone sees with an empty board.

**Independent Test**: Create a project and see it in another user's project list with four
empty columns; a name differing only in case is refused (quickstart step 9).

- [X] T029 [P] [US4] Write tests in `api_test.go`: `POST /api/projects` `{"name":"Website"}` → 201 with `description` `""` and `createdBy` = the user; it appears in `GET /api/projects`; `GET` on it returns `tasks` as `[]`. `{"name":"website"}` after that, and `{"name":"WEBSITE REFRESH"}` (seeded name), → 400 `field` `name`, message `name is already used by another project`. Name rule "Required; 1–100 chars; single line; no control chars": 100 chars → 201; 101 chars and `"a\nb"` → 400. Description rule "Optional (`""`); ≤ 1,000 chars; `\n` allowed; no other control chars": 1,000 → 201; 1,001 → 400. `{"id":9,"name":"x"}` → 400. Every refusal leaves the project count unchanged
- [X] T030 [US4] Implement `POST /api/projects` in `api.go` to pass T029: requireUser; `decodeBody` with `allowed` = `["name","description"]`; `checkText("name", v, true, 100, false)`, `checkText("description", v, false, 1000, true)`; inside one `store.update`, refuse a name equal to any existing name under `strings.EqualFold`, then create with `newID`, `createdBy` = user, `createdAt` = now UTC RFC 3339; respond 201. The duplicate check runs inside `update` so check-then-insert is atomic
- [X] T031 [US4] Add a "New project" form to the project list in `web/app.js`: name input and description textarea; submit sends `POST /api/projects`; show a 400 message next to its field; on success reload the list

**Checkpoint**: All four stories work; quickstart step 9 passes.

---

## Phase 7: Polish & cross-cutting

- [X] T032 Run `gofmt -l .` (must print nothing), `go vet ./...`, and `go test ./...` over all `*.go` files at the repository root; fix every finding
- [X] T033 Constitution review gate on all `*.go` and `web/*` files: every route validates its inputs and checks access server-side; `go.mod` still has no `require`; comments exist only for the planned non-obvious reasons (DNS rebinding `Host` check, header identity without login, atomic rename, save rollback, `assigneeId` 0, 64 KiB cap, fixed `127.0.0.1`); no banner or restating comments; compare `grep -cE '^\s*//' *.go` against line counts to keep density low
- [X] T034 Check SC-004 using `specs/001-taskify-kanban/quickstart.md` setup: create 200 tasks in one project with a `curl` loop against `POST /api/projects/{pid}/tasks`, open that board in the browser, and confirm it is fully shown within 2 seconds (browser dev tools network/performance timing)
- [X] T035 Run every check in `specs/001-taskify-kanban/quickstart.md` on a fresh `taskify.json`: build and tests, manual browser steps 1–10, the terminal security checks, and the second-computer connection check (SC-009)
- [ ] T036 Run `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` for the module in `go.mod` at the repository root; any finding blocks the release (research R12)
  - Blocked by choice: this project stays on go1.25.0; govulncheck reports 20 standard library vulnerabilities (all fixed in go1.25.13). Not released.

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup (Phase 1)**: none
- **Foundational (Phase 2)**: after Setup; blocks every story. T004 first; then T005–T007 in
  parallel; then T008–T010 in parallel; then T011; then T012. T013 and T014 can run any time
  after T003
- **US1 (Phase 3)**: after Foundational
- **US2 (Phase 4)**: after US1 (adds to the board and the task dialog from US1)
- **US3 (Phase 5)**: after US2 (uses the task details route and full dialog from US2)
- **US4 (Phase 6)**: after US1 (T029 and the US4 independent test use the board route
  `GET /api/projects/{pid}` from T016); independent of US2 and US3
- **Polish (Phase 7)**: after the stories you plan to ship

### Within each story

- The test task runs red on its assertions before the implementation task starts
- `api.go` work before the `web/app.js` work that calls it
- `web/app.js` tasks in one story run in order (same file)

### Parallel opportunities

- Setup: T002 ∥ T003
- Foundational: T005 ∥ T006 ∥ T007; then T008 ∥ T009 ∥ T010; T013 ∥ T014 alongside the Go work
- In every story, the test task and the `web/style.css` task are [P]
- Once US1 is done, US4 can run alongside US2–US3 by a second person, but both edit `api.go`, `api_test.go`, and
  `web/app.js`, so expect merge work

---

## Parallel Example: Foundational

```text
Task: "T005 Write table-driven tests in validate_test.go"
Task: "T006 Write tests in store_test.go"
Task: "T007 Write foundation tests in api_test.go"
# after all three are red:
Task: "T008 Implement validate.go"
Task: "T009 Implement store.go"
Task: "T010 Implement seed.go"
```

## Parallel Example: User Story 1

```text
Task: "T015 Write board and move tests in api_test.go"
Task: "T019 Style the board in web/style.css"
```

## Parallel Example: User Story 2

```text
Task: "T020 Write create/details/assign tests in api_test.go"
Task: "T024 Style the task form and dialog in web/style.css"
```

## Parallel Example: User Story 3

```text
Task: "T025 Write comment tests in api_test.go"
Task: "T028 Style the comment list in web/style.css"
```

---

## Implementation Strategy

### MVP first (User Story 1)

1. Phase 1 Setup → Phase 2 Foundational
2. Phase 3 US1
3. **Stop and validate**: quickstart steps 1–4 and `go test ./...`
4. Demo: the team can pick a user and move sample tasks

### Incremental delivery

1. US2 → quickstart steps 5–7
2. US3 → quickstart step 8
3. US4 → quickstart step 9
4. Polish → full quickstart run, `govulncheck`

---

## Notes

- [P] = different files, no dependency on an unfinished task
- Never weaken an existing green test to make new code pass; change a test only with a stated
  reason
- Commit after each task or each red/green pair
