# Quickstart: Taskify Phase 1

How to run Taskify and prove it meets the spec. Routes and error codes:
[contracts/api.md](./contracts/api.md). Fields and rules: [data-model.md](./data-model.md).

## Prerequisites

- Go 1.25 or newer (`go version`). Nothing else; there are no third-party modules.
- A desktop browser (Chrome, Firefox, Safari, or Edge, 2023 or newer).

## Build, test, run

```sh
go vet ./...
go test ./...                       # must pass before any manual check
go run . -port 8080 -data taskify.json
```

Expected log line: the server is listening on `127.0.0.1:8080`. Open
<http://127.0.0.1:8080/>.

To start over with fresh sample data, stop the server and delete `taskify.json`.

## Automated checks (go test)

| Spec item | Covered by |
| --------- | ---------- |
| SC-005, FR-024, FR-025 | One or more failing values per Input Validation rule → 400, nothing saved; every length limit: exactly the limit accepted, limit + 1 → 400 |
| SC-006, FR-023 | Non-author comment `PATCH`/`DELETE` → 403, comment unchanged |
| SC-007, FR-026 | Text with `<script>` round-trips unchanged; `app.js` contains no `innerHTML` |
| SC-003, FR-028, FR-029 | Reopen the store from the same file: data kept, sample data not added again |
| FR-027 | Failed save → 500, memory and file unchanged |
| FR-002, FR-005 | Missing or unknown `X-Taskify-User` → 401 |
| FR-006 | Wrong `Host` → 421; cross-site `POST` → 403 |
| FR-011, Q4 | Moving an unassigned task to `done` keeps `assigneeId` 0 |
| Startup | Corrupt data file → exits with an error, file left untouched |

## Manual checks in the browser

Run these in order on a fresh `taskify.json`.

1. **Pick a user (FR-002, US1-1)**: The first screen lists the five users and no projects.
   Open `http://127.0.0.1:8080/#/projects/1` in a private window → you are sent to the picker.
2. **Move a task (SC-001, US1-4)**: Pick Blake Rivera, open "Website Refresh", drag a To Do
   card to In Review. Reload the page → it is still in In Review. Time it: under 1 minute.
3. **Keyboard move (FR-015)**: Open a card, change the column with the drop-down only. The
   card moves.
4. **Own tasks stand out (FR-014, US1-5/6)**: Cards assigned to Blake look different. Switch to
   Casey Patel → Casey's cards stand out instead.
5. **Create and assign (SC-002, US2)**: Create a task with a title and assignee Devon Kim. It
   appears in To Do with Devon's name. Time it: under 30 seconds.
6. **Validation message (US2-5)**: Try a 201-character title → nothing is saved; the message
   names the title field and the 200-character limit.
7. **Plain text (SC-007)**: Create a task titled `<img src=x onerror=alert(1)>`. The card shows
   those exact characters; no alert, no image.
8. **Comments (US3)**: As Casey, comment, then edit it → new text, no "(edited)" mark, same
   time. Switch to Devon → no edit or delete controls on Casey's comment. Switch back to Casey
   and delete it → it is gone.
9. **New project (US4)**: Create "Website" → it appears. Create "website" → refused as a
   duplicate.
10. **Restart (SC-003)**: Stop the server (Ctrl-C), start it again, reload → everything from
    steps 2–9 is still there and no second copy of the sample projects appears.

## Security checks from a terminal

With the server running on port 8080:

```sh
# FR-002: no picked user → 401
curl -si http://127.0.0.1:8080/api/projects | head -1

# FR-006: wrong Host header (DNS rebinding) → 421
curl -si -H 'Host: evil.example:8080' http://127.0.0.1:8080/ | head -1

# FR-006: cross-site browser POST → 403
curl -si -X POST http://127.0.0.1:8080/api/projects \
  -H 'Sec-Fetch-Site: cross-site' -H 'X-Taskify-User: 1' \
  -H 'Content-Type: application/json' -d '{"name":"x"}' | head -1

# SC-006: user 3 deletes a comment written by user 2 → 403 (use real IDs from step 8)
curl -si -X DELETE -H 'X-Taskify-User: 3' \
  http://127.0.0.1:8080/api/projects/1/tasks/4/comments/20 | head -1

# SC-009: only the loopback address is listening
lsof -nP -iTCP:8080 -sTCP:LISTEN     # expect 127.0.0.1:8080, never *:8080
```

From a second computer on the same network, `curl http://<this-computer-ip>:8080/` must fail
to connect (SC-009).

## Before a release

```sh
go run golang.org/x/vuln/cmd/govulncheck@latest ./...   # any finding blocks the release
```
