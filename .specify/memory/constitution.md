<!--
Sync Impact Report
==================
Version change: (unversioned template) → 1.0.0
Modified principles (template placeholder → new title):
  - [PRINCIPLE_1_NAME] → I. Security-First (NON-NEGOTIABLE)
  - [PRINCIPLE_2_NAME] → II. Validate All Input (NON-NEGOTIABLE)
  - [PRINCIPLE_3_NAME] → III. One Small Application
  - [PRINCIPLE_4_NAME] → IV. Minimal Dependencies
  - [PRINCIPLE_5_NAME] → V. Document Non-Obvious Intent
Added sections:
  - Security Requirements (was [SECTION_2_NAME])
  - Development Workflow & Quality Gates (was [SECTION_3_NAME])
  - Governance (rules filled in)
Removed sections: none
Follow-up TODOs: none
-->

# Taskify Constitution

## Core Principles

### I. Security-First (NON-NEGOTIABLE)

Security outranks convenience, delivery speed, and features. When security conflicts with another
goal, security wins unless this constitution is amended.

- Every feature spec MUST state what data the feature exposes or changes and who is allowed to
  do so.
- Every operation that reads or changes data MUST check, on the server, that the caller is
  allowed to perform it on that specific record. Client-side checks are UX only and never count.
- Access is denied by default: anything not explicitly allowed is refused.
- Secrets (keys, passwords, tokens) MUST NOT appear in source code, version control, logs, or
  user-facing errors.
- Errors shown to users MUST NOT reveal internals such as stack traces, queries, or file paths.

Rationale: Taskify is defined as a Security-First application. A flaw prevented at design time
costs far less, and harms users far less, than one found after release.

### II. Validate All Input (NON-NEGOTIABLE)

All input from a source the application does not fully control MUST be validated before it is
used for logic, storage, or output. This includes request bodies, query strings, path
parameters, headers, cookies, form fields, uploaded files, and configuration values.

- Validation MUST run on the server at the trust boundary. Client-side validation MAY exist for
  user experience but never replaces server-side validation.
- Validation MUST use allowlists: check type, format, length or size, and allowed values or
  range. Input that does not match is rejected, not silently "cleaned".
- Every text field, collection, and upload MUST have an explicit maximum size.
- Rejected input MUST return an error that tells the user what to fix without echoing raw input
  unescaped or revealing internals.
- Every validation rule MUST have at least one automated test that proves bad input is rejected.

Rationale: Injection, cross-site scripting, and resource-exhaustion attacks all start with input
that was trusted without being checked.

### III. One Small Application

- Taskify MUST be built, deployed, and run as one application from one repository. Splitting it
  into microservices or other independently deployed services is prohibited.
- New layers, modules, and abstractions MUST serve a present requirement, not a predicted one.
  An interface with one implementation or a setting for a value that never changes needs a
  written justification.
- Any added complexity MUST be justified in the plan's Complexity Tracking table, or removed.

Rationale: Fewer moving parts means less to secure, less to break, and less to review. Every
extra service is extra attack surface and extra network traffic to protect.

### IV. Minimal Dependencies

- Solutions MUST be chosen in this order: the language standard library, built-in platform
  features, code already in the project, and only then a third-party dependency.
- A dependency MUST NOT be added for something a few lines of project code can do.
- Every new third-party dependency (runtime or development) MUST be justified in the plan: what
  it does, why the options above are not enough, and that it is actively maintained.
- Dependency versions MUST be pinned, and the lockfile MUST be committed.
- Dependencies that are no longer used MUST be removed.

Rationale: Each dependency is code the project did not write but must still trust. It adds
supply-chain risk and upgrade work for the life of the application.

### V. Document Non-Obvious Intent

- Code MUST first be made clear through naming and structure; comments are not a substitute for
  readable code.
- Where intent is not obvious from the code, a comment MUST explain *why*: a non-obvious
  decision, a trap that fails silently, a measured number, or a deliberate break from convention.
- Every non-obvious security decision (why a check exists, why a limit has its value) MUST have
  a comment, so it is not "simplified" away later.
- Comments MUST NOT restate what the code or a name already says. Banner and section-divider
  comments are not allowed.
- Explanations longer than a few lines belong in the feature's spec or plan; the comment links
  there once instead of repeating it.

Rationale: Comments that restate code go stale and bury the few that matter. A reader who does
not know why a security check exists is likely to remove it.

## Security Requirements

These apply to every feature in addition to Principles I and II.

- **Injection**: Database access MUST use parameterized queries. Building queries by joining
  strings with input is prohibited. Input MUST never reach a shell, `eval`, or any other
  dynamic code execution.
- **Output encoding**: Data rendered in HTML MUST be escaped for its context. If auto-escaping is
  turned off anywhere, a comment MUST explain why that spot is safe.
- **Accounts and sessions** (when Taskify has user accounts): passwords MUST be hashed with a slow,
  salted algorithm (Argon2id, scrypt, or bcrypt). Session cookies MUST be `HttpOnly`, `Secure`,
  and `SameSite`. State-changing requests MUST be protected against cross-site request forgery.
- **Transport**: Production traffic MUST use HTTPS.
- **Secrets**: Secrets MUST be loaded from the environment or from a file excluded from version
  control.
- **Logging**: Security events (failed sign-ins, denied access, rejected input) SHOULD be logged.
  Logs MUST NOT contain passwords, secrets, or session tokens.
- **Known vulnerabilities**: Dependencies MUST be scanned with the package manager's audit tool
  before a release. A known critical or high-severity vulnerability blocks the release.

## Development Workflow & Quality Gates

- **Spec gate**: Every feature spec MUST list each input the feature accepts and its validation
  rules, and MUST state who may access the feature.
- **Plan gate**: Every plan MUST pass the Constitution Check against all five principles before
  Phase 0 research and again after Phase 1 design. Each violation of Principles III–V is either
  fixed or recorded in Complexity Tracking with a justification.
- **Test gate**: Every validation rule MUST have a test proving bad input is rejected. Every
  access check MUST have a test proving an unauthorized caller is refused.
- **Review gate**: Before merge, each change MUST be checked for: validation at every new entry
  point, server-side access checks, justification for any new dependency, and comments where
  intent is not obvious. For a solo developer, self-review against this list counts.

## Governance

- This constitution overrides all other project practices and guidance. When a spec, plan, or
  task conflicts with it, the constitution wins until it is amended.
- **NON-NEGOTIABLE principles** (I and II) MUST NOT be waived through Complexity Tracking or any
  other exception. They change only by amendment.
- **Amendments** are made by running `/speckit-constitution` with the proposed change. Each
  amendment MUST record what changed, why, and the new version, and MUST be followed by a review
  of existing specs and plans that the change affects.
- **Versioning** follows semantic versioning:
  - MAJOR: a principle is removed or redefined in a way that breaks earlier compliance.
  - MINOR: a principle or section is added, or guidance is materially expanded.
  - PATCH: wording, clarification, or typo fixes with no change in meaning.
- **Compliance** is checked at the Constitution Check in every `/speckit-plan`, at every review,
  and by `/speckit-analyze`, which treats any constitution conflict as CRITICAL.

**Version**: 1.0.0 | **Ratified**: 2026-10-08 | **Last Amended**: 2026-10-08
