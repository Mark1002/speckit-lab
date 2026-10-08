# Feature Specification: Taskify Kanban Board (Phase 1)

**Feature Branch**: `main` (no feature branch was created; the spec lives in `specs/001-taskify-kanban/`)

**Created**: 2026-10-08

**Status**: Draft

**Input**: User description: "Develop Taskify, a team productivity platform where predefined users create projects, assign tasks, comment, and move tasks across Kanban columns (To Do, In Progress, In Review, Done). Five users (one product manager, four engineers), three sample projects, no login for this first phase."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pick a user and move tasks on a project board (Priority: P1)

A team member opens Taskify and picks who they are from a list of five people. They see the list
of projects, open one, and see its tasks laid out in four columns: To Do, In Progress, In Review,
and Done. They move a task from one column to another to show its progress. Tasks assigned to
them stand out from the rest.

**Why this priority**: The board is the heart of the product. With the sample data already in
place, this story alone lets the team see and track work, so it is a usable first version.

**Independent Test**: Start Taskify with only the sample data. Pick a user, open a sample
project, move a task to another column, reopen the board, and confirm the task is still in its
new column.

**Acceptance Scenarios**:

1. **Given** no user has been picked yet, **When** a person opens Taskify, **Then** they see the
   five users to choose from and no project data.
2. **Given** a user has been picked, **When** they view the project list, **Then** they see all
   three sample projects.
3. **Given** a user opens a project, **When** the board appears, **Then** it shows the four
   columns in the order To Do, In Progress, In Review, Done, with every task of that project in
   exactly one column.
4. **Given** a task is in To Do, **When** the user moves it to In Review, **Then** it appears in
   In Review and stays there after the board is reopened.
5. **Given** some tasks are assigned to the current user, **When** the board appears, **Then**
   those tasks are visibly marked as theirs and other tasks are not.
6. **Given** a user is working as one person, **When** they switch to another user, **Then** the
   board marks the new user's tasks as theirs.

---

### User Story 2 - Create, edit, and assign tasks (Priority: P2)

A team member adds a new task to a project with a title and an optional description. They give it
to one teammate, change who it is given to, or leave it unassigned. They open a task to read its
full details and fix its title or description.

**Why this priority**: Without new tasks the board only shows sample data. This turns the board
into a tool the team can use for real work.

**Independent Test**: Pick a user, create a task in a sample project, assign it to an engineer,
edit its title, and confirm it shows in To Do with the new title and the engineer's name.

**Acceptance Scenarios**:

1. **Given** a user is on a project board, **When** they create a task with a valid title,
   **Then** the task appears in the To Do column, unassigned unless they chose an assignee.
2. **Given** a task exists, **When** a user assigns it to one of the five users, **Then** the
   card shows that user's name.
3. **Given** a task is assigned, **When** a user removes the assignee, **Then** the card shows it
   as unassigned.
4. **Given** a task exists, **When** a user changes its title or description to valid text,
   **Then** the new text is shown to every user.
5. **Given** a user enters a title that breaks a rule in the Input Validation table, **When** they
   save, **Then** nothing is saved and they see which field is wrong and how to fix it.

---

### User Story 3 - Discuss tasks with comments (Priority: P3)

A team member opens a task and adds a comment. Everyone can read all comments on a task, with who
wrote each one and when. A person can edit or delete their own comments, but never anyone
else's.

**Why this priority**: Comments let the team talk about work in one place. The board is still
useful without them, so they come after tasks.

**Independent Test**: As one user, add a comment and edit it. Switch to another user and confirm
the comment is visible but cannot be edited or deleted. Switch back and delete it.

**Acceptance Scenarios**:

1. **Given** a user opens a task, **When** they add a comment with valid text, **Then** it shows
   on that task with their name and the time it was posted, after any older comments.
2. **Given** a user wrote a comment, **When** they edit it to valid text, **Then** the new text
   replaces the old text for every user.
3. **Given** a user wrote a comment, **When** they delete it, **Then** it no longer shows for any
   user.
4. **Given** a comment was written by someone else, **When** the current user views it, **Then**
   they are not offered a way to edit or delete it.
5. **Given** a comment was written by someone else, **When** the current user tries to edit or
   delete it anyway, even through a request that does not come from Taskify's own screens,
   **Then** the system refuses and the comment is unchanged.

---

### User Story 4 - Create a new project (Priority: P4)

A team member creates a new project with a name and an optional description. It appears in the
project list for every user with an empty board, ready for tasks.

**Why this priority**: Three sample projects already exist, so the team can work without this.
It matters once the team starts work that does not fit the sample projects.

**Independent Test**: Pick a user, create a project, and confirm it shows in the project list for
a different user with four empty columns.

**Acceptance Scenarios**:

1. **Given** a user is on the project list, **When** they create a project with a valid, unused
   name, **Then** it appears in the list for every user.
2. **Given** a new project was just created, **When** a user opens it, **Then** it shows the four
   columns with no tasks.
3. **Given** a project named "Website" exists, **When** a user tries to create a project named
   "website", **Then** nothing is saved and they are told the name is already used.

---

### Edge Cases

- A field is empty, only spaces, too long, or has a line break where one is not allowed: nothing
  is saved, and the message names the field and the rule it broke.
- Text looks like formatting or code (for example `<b>hi</b>` or `<script>`): it is saved and
  shown exactly as typed, as plain text, and never acts as formatting or code.
- A task is moved to the column it is already in: nothing changes and no error is shown.
- A request names a column that is not one of the four, or a user who is not one of the five: it
  is refused and nothing changes.
- A request names a project, task, or comment that does not exist, or a task under the wrong
  project: the user sees a "not found" message with no internal details.
- Two users change the same task at almost the same time: the change saved last wins, and the
  other user sees it the next time they open or refresh the board.
- Someone on another computer tries to open Taskify: the attempt fails and no data is shown.
- A person opens a board link directly without having picked a user: they are asked to pick a
  user first.
- Taskify is restarted: all projects, tasks, and comments are still there, and the sample data is
  not added a second time.
- Something unexpected goes wrong on the system side: the user sees a short, general error
  message with no internal details, and no partial change is saved.

## Requirements *(mandatory)*

### Functional Requirements

#### Users and access

- **FR-001**: System MUST include exactly five predefined users: one product manager and four
  engineers. Each has a display name and a role.
- **FR-002**: System MUST ask a person to pick one of the five users before showing or changing
  any project data. No password is asked for in phase 1.
- **FR-003**: Users MUST be able to switch to a different user at any time.
- **FR-004**: System MUST record every change (project, task, assignment, move, comment) as made by
  the currently picked user.
- **FR-005**: System MUST apply the Access Rules table below to every request, including requests
  that do not come from Taskify's own screens.
- **FR-006**: System MUST be reachable only from the computer it runs on. Any attempt to reach it
  from another computer MUST fail. Because there is no login, this is what stops someone else
  from acting as one of the five users.

#### Projects

- **FR-007**: System MUST show all projects to every user.
- **FR-008**: Users MUST be able to create a project with a name and an optional description.
- **FR-009**: System MUST include three sample projects the first time Taskify starts.

#### Board and tasks

- **FR-010**: Each project MUST have a board with exactly four columns, always in this order:
  To Do, In Progress, In Review, Done.
- **FR-011**: Every task MUST be in exactly one column at a time. A new task MUST start in To Do.
- **FR-012**: Within a column, tasks MUST be listed by creation time, oldest first.
- **FR-013**: Each task card MUST show the task title and the assignee's name, or "Unassigned".
- **FR-014**: Tasks assigned to the currently picked user MUST look visibly different from other
  tasks.
- **FR-015**: Users MUST be able to move a task from any column to any other column. They MUST be
  able to do this by choosing the target column directly, not only by dragging.
- **FR-016**: Users MUST be able to create a task with a title, an optional description, and an
  optional assignee.
- **FR-017**: Users MUST be able to edit a task's title and description.
- **FR-018**: Users MUST be able to assign a task to one of the five users, change the assignee,
  or remove the assignee. A task has at most one assignee.
- **FR-019**: Users MUST be able to open a task to see its title, description, assignee, column,
  creator, and comments.

#### Comments

- **FR-020**: Users MUST be able to add a comment to any task.
- **FR-021**: System MUST show each comment with its author's name and the time it was posted,
  oldest first.
- **FR-022**: Users MUST be able to edit and delete their own comments.
- **FR-023**: System MUST refuse any attempt to edit or delete a comment by anyone other than its
  author, and leave the comment unchanged.

#### Input handling

- **FR-024**: System MUST check every input against the Input Validation table below by itself,
  even when a screen has already checked it.
- **FR-025**: When input breaks a rule, System MUST save nothing and MUST show a message that
  names the field and the rule. The message MUST NOT reveal internal details.
- **FR-026**: System MUST show all text that users enter exactly as typed, as plain text. Entered
  text MUST never be treated as formatting or code.
- **FR-027**: When an unexpected error happens, System MUST show a general error message with no
  internal details and MUST NOT save a partial change.

#### Data

- **FR-028**: System MUST keep all projects, tasks, and comments when Taskify is restarted.
- **FR-029**: System MUST add the sample data only on the very first start, and MUST NOT add it
  again or overwrite changes on later starts.

### Access Rules

Because there is no login in phase 1, Taskify cannot prove which person is using which user.
Anyone who can reach it can act as any of the five users. This is accepted for phase 1 only and
is limited by FR-006.

| Data | Who can see it | Who can change it |
| ---- | -------------- | ----------------- |
| Users (name, role) | All five users | Nobody; the five users are fixed |
| Projects | All five users | Any user can create one. Editing and deleting are out of scope. |
| Tasks | All five users | Any user can create, edit, assign, and move any task |
| Comments | All five users | Any user can add one. Only its author can edit or delete it. |

### Input Validation

"Single line" means no line breaks. "No control characters" means no invisible control
characters, except line breaks where they are allowed. A value made only of spaces counts as
empty.

| Input | Required | Rules |
| ----- | -------- | ----- |
| Picked user | Yes | Must be one of the five predefined users |
| Project name | Yes | 1 to 100 characters; single line; no control characters; not already used by another project, ignoring upper/lower case |
| Project description | No | Up to 1,000 characters; line breaks allowed; no other control characters |
| Task title | Yes | 1 to 200 characters; single line; no control characters |
| Task description | No | Up to 5,000 characters; line breaks allowed; no other control characters |
| Task assignee | No | Empty, or one of the five predefined users |
| Task column | Yes | Exactly one of: To Do, In Progress, In Review, Done |
| Comment text | Yes | 1 to 2,000 characters; line breaks allowed; no other control characters |
| Reference to a project, task, or comment | Yes | Must name a record that exists; a task must belong to the named project; a comment must belong to the named task |

### Key Entities

- **User**: One of the five fixed team members. Has a display name and a role (product manager or
  engineer). Cannot be added, changed, or removed in phase 1.
- **Project**: A body of work with a unique name and an optional description. Records who created
  it and when. Has many tasks.
- **Task**: A piece of work inside one project. Has a title, an optional description, a column
  (To Do, In Progress, In Review, or Done), and at most one assignee. Records who created it and
  when. Has many comments.
- **Comment**: A note on one task. Has text, an author, and the time it was posted. Only its
  author can edit or delete it.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time user can pick a user, open a project, and move a task to another
  column in under 1 minute, with no instructions.
- **SC-002**: A user can create a task with a title and an assignee in under 30 seconds.
- **SC-003**: 100% of saved changes are still there after Taskify restarts, and other users see
  them the next time they open or refresh the board.
- **SC-004**: A board with up to 200 tasks is fully shown within 2 seconds of opening it.
- **SC-005**: For every rule in the Input Validation table, at least one breaking value is tested,
  and 100% of them are refused with a message that names the field, with nothing saved.
- **SC-006**: 0 successful attempts to edit or delete another user's comment, including attempts
  that do not use Taskify's own screens.
- **SC-007**: 0 cases where user-entered text is shown as formatting or acts as code.
- **SC-008**: In a team walkthrough, each of the five people completes the full flow on the first
  try: pick a user, create a task, assign it, comment on it, and move it to Done.
- **SC-009**: 0 successful attempts to reach Taskify from any computer other than the one it runs
  on.

## Assumptions

- Phase 1 is a first working version to prove the basic features. It is not meant to hold
  sensitive real-world data.
- Picking a user works on trust, like writing your name on a sign-in sheet. Proper login comes in
  a later phase.
- All five users have the same abilities. The product manager and engineer roles are labels only.
- Each sample project starts with a few sample tasks spread across all four columns, some of them
  assigned, so the board can be tried right away.
- Changes made by other users show up when a board is opened or refreshed. Live, automatic updates
  are out of scope.
- Tasks can move between any two columns in either direction; there is no required order.
- Taskify is used from a desktop web browser, in English.
- In phase 1 the team shares the one computer Taskify runs on and takes turns, switching users as
  they go.
- Out of scope for phase 1: login and passwords; adding, changing, or removing users; editing or
  deleting projects; deleting tasks; reordering tasks inside a column; attachments; due dates;
  labels; notifications; search; and layouts made for phones.
