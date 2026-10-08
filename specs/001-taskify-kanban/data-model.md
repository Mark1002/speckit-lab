# Data Model: Taskify Kanban Board (Phase 1)

Source: [spec.md](./spec.md) Key Entities, Access Rules, Input Validation. Storage decisions:
[research.md](./research.md) R3–R5.

## Entities

### User (fixed in code, not stored)

| Field | JSON | Type | Notes |
| ----- | ---- | ---- | ----- |
| ID | `id` | int | 1–5 |
| Name | `name` | string | Display name |
| Role | `role` | string | `product_manager` or `engineer`; a label only, grants nothing |

| id | name | role |
| -- | ---- | ---- |
| 1 | Avery Chen | product_manager |
| 2 | Blake Rivera | engineer |
| 3 | Casey Patel | engineer |
| 4 | Devon Kim | engineer |
| 5 | Emery Novak | engineer |

### Project

| Field | JSON | Type | Set by | Rules |
| ----- | ---- | ---- | ------ | ----- |
| ID | `id` | int | System | From `nextId` |
| Name | `name` | string | User | Required; 1–100 chars; single line; no control chars; unique ignoring case |
| Description | `description` | string | User | Optional (`""`); ≤ 1,000 chars; `\n` allowed; no other control chars |
| Created by | `createdBy` | int | System | Picked user ID |
| Created at | `createdAt` | string | System | RFC 3339, UTC |

Immutable after creation in phase 1 (no edit, no delete).

### Task

| Field | JSON | Type | Set by | Rules |
| ----- | ---- | ---- | ------ | ----- |
| ID | `id` | int | System | From `nextId` |
| Project | `projectId` | int | System | From the URL; must exist |
| Title | `title` | string | User | Required; 1–200 chars; single line; no control chars |
| Description | `description` | string | User | Optional (`""`); ≤ 5,000 chars; `\n` allowed; no other control chars |
| Status | `status` | string | User | One of `todo`, `in_progress`, `in_review`, `done`; new tasks are `todo` |
| Assignee | `assigneeId` | int | User | `0` = unassigned, or 1–5 |
| Created by | `createdBy` | int | System | Picked user ID |
| Created at | `createdAt` | string | System | RFC 3339, UTC |

Not deletable in phase 1. No record is kept of who moved, edited, or reassigned it (FR-004).

### Comment

| Field | JSON | Type | Set by | Rules |
| ----- | ---- | ---- | ------ | ----- |
| ID | `id` | int | System | From `nextId` |
| Task | `taskId` | int | System | From the URL; must exist and belong to the URL's project |
| Author | `authorId` | int | System | Picked user ID at creation; never changes |
| Text | `text` | string | User | Required; 1–2,000 chars; `\n` allowed; no other control chars |
| Created at | `createdAt` | string | System | RFC 3339, UTC; unchanged by edits (FR-022) |

Edit and delete allowed only when the picked user equals `authorId`. Delete removes the record
from the file; nothing remains.

## Relationships

```text
Project 1 ── * Task 1 ── * Comment
User    1 ── * Project.createdBy, Task.createdBy, Comment.authorId
User    1 ── * Task.assigneeId   (0..1 per task)
```

## Validation definitions

These apply to every text rule above ([research.md](./research.md) R7):

- **chars** = Unicode code points.
- **Required** = after ignoring surrounding whitespace, at least one character remains. The
  stored value is exactly what was sent; nothing is trimmed.
- **Single line** = no `\n`.
- **No control chars** = no rune with `unicode.IsControl` true, except `\n` where allowed.
- Unknown JSON fields, wrong JSON types, a second JSON value, or a body over 64 KiB are rejected
  before any field is checked.

## Task status transitions

```text
todo ⇄ in_progress ⇄ in_review ⇄ done     (and any other pair, both ways)
```

Any status may change to any other status, by any of the five users, assigned or not (spec
Clarifications Q1, Q4). Changing status never changes `assigneeId`. Setting the current status
again is a successful no-op.

## Storage file

One JSON object at the `-data` path (default `taskify.json`, mode `0600`):

```json
{
  "nextId": 42,
  "projects": [ { "id": 1, "name": "…", "description": "…", "createdBy": 1, "createdAt": "…" } ],
  "tasks":    [ { "id": 4, "projectId": 1, "title": "…", "description": "", "status": "todo",
                  "assigneeId": 0, "createdBy": 1, "createdAt": "…" } ],
  "comments": [ { "id": 20, "taskId": 4, "authorId": 2, "text": "…", "createdAt": "…" } ]
}
```

Arrays are kept in ID order, which is creation order.

## Sample data (first start only)

Written when the data file does not exist (FR-009, FR-029):

- Three projects, created by user 1: "Website Refresh", "Mobile App Beta", "Internal Tools".
- Each project has 4–6 tasks with at least one in every status, at least one unassigned, and
  the rest spread across all five users.
- No comments.
