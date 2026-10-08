# REST API Contract: Taskify Phase 1

Base URL: `http://127.0.0.1:{port}` (default port 8080). Field rules are defined once in
[data-model.md](../data-model.md); this file does not repeat them.

## Request rules (all `/api/` routes)

| Rule | Failure |
| ---- | ------- |
| `Host` is `127.0.0.1:{port}` or `localhost:{port}` (applies to every route, static files too) | 421 |
| Not a cross-origin browser request with an unsafe method (`http.CrossOriginProtection`) | 403 |
| Method and path match a route below | 404 / 405 |
| `X-Taskify-User` is `1`–`5` (every route except `GET /api/users`) | 401 |
| `POST`/`PATCH`: `Content-Type` is `application/json` (parameters allowed) | 415 |
| Body ≤ 64 KiB | 413 |
| Body is one JSON object, known fields only, correct JSON types | 400 |
| Path IDs name existing records, nested correctly (task in project, comment in task) | 404 |
| Comment edit/delete: `X-Taskify-User` equals the comment's `authorId` | 403 |
| Every field passes its rule in data-model.md | 400 |

Checks run top to bottom; the first failure is returned and nothing is saved. Ownership (403)
is checked before field rules, so a non-author learns nothing about validation.

In request bodies, an optional field that is absent or `null` means "not provided".

## Error body

```json
{ "error": { "field": "title", "message": "title must be 1 to 200 characters" } }
```

- `field` is present only when one request field caused the error.
- `message` is a fixed string naming the field and rule. It never contains the submitted value,
  an unknown field's name, or internal details (paths, stack traces, Go error text).
- 500 always returns `{ "error": { "message": "something went wrong; nothing was saved" } }`.

## Headers on every response

`Content-Security-Policy: default-src 'self'; frame-ancestors 'none'`,
`X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`. JSON responses use
`Content-Type: application/json; charset=utf-8`. No CORS headers are ever sent.

## Routes

### `GET /api/users` → 200

No `X-Taskify-User` needed (the picker calls it before a user is chosen).

```json
[ { "id": 1, "name": "Avery Chen", "role": "product_manager" }, … ]
```

### `GET /api/projects` → 200

Array of Project, ID order.

### `POST /api/projects` → 201

```json
{ "name": "Website", "description": "optional" }
```

Returns the new Project. Duplicate name (ignoring case) → 400, `field: "name"`.

### `GET /api/projects/{pid}` → 200

The Project plus its tasks in ID order (the board):

```json
{ "id": 1, "name": "…", "description": "…", "createdBy": 1, "createdAt": "…",
  "tasks": [ { "id": 4, "projectId": 1, "title": "…", "description": "…", "status": "todo",
               "assigneeId": 0, "createdBy": 1, "createdAt": "…" } ] }
```

### `POST /api/projects/{pid}/tasks` → 201

```json
{ "title": "Fix header", "description": "optional", "assigneeId": 3 }
```

`description` defaults to `""`, `assigneeId` to `0`. `status` is not accepted here; new tasks
are always `todo` (sending `status` → 400). Returns the new Task.

### `GET /api/projects/{pid}/tasks/{tid}` → 200

The Task plus its comments in ID order:

```json
{ "id": 4, …task fields…,
  "comments": [ { "id": 20, "taskId": 4, "authorId": 2, "text": "…", "createdAt": "…" } ] }
```

### `PATCH /api/projects/{pid}/tasks/{tid}` → 200

Any of `title`, `description`, `status`, `assigneeId`; at least one must be provided (`{}` →
400). Any of the five users may call it (spec Clarifications Q1, Q2).

```json
{ "status": "in_review" }
{ "assigneeId": 0 }
{ "title": "New title", "description": "" }
```

`assigneeId: 0` clears the assignee. Changing `status` never changes `assigneeId`. Returns the
updated Task.

### `POST /api/projects/{pid}/tasks/{tid}/comments` → 201

```json
{ "text": "Looks good to me" }
```

`authorId` is the `X-Taskify-User` value. Returns the new Comment.

### `PATCH /api/projects/{pid}/tasks/{tid}/comments/{cid}` → 200

```json
{ "text": "Updated text" }
```

Author only (else 403). `createdAt` does not change. Returns the updated Comment.

### `DELETE /api/projects/{pid}/tasks/{tid}/comments/{cid}` → 204

Author only (else 403). No body.

## Static files

| Path | Serves |
| ---- | ------ |
| `GET /` | `index.html` |
| `GET /app.js` | `app.js` |
| `GET /style.css` | `style.css` |

Served from the embedded `web/` directory. No other paths exist; directory listings are off.
