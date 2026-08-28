<!-- mlc-dochub:begin — auto-managed, do not edit between these markers -->
## MLC Doc Hub — Project Rules

This project keeps structured documentation in `.mlcai/`, maintained
through the `mlc-dochub` MCP server.

- **Project ID:** `localembed` — LocalEmbed — Local Embedding Server
- **Source directory:** `/mnt/data2tb/localembed` (read source files with native Read — MCP tools only touch `.mlcai/`)

### Existing docs

You can read any of these directly (native Read is fine, or
`mcp__mlc-dochub__get_doc`) to gather context before you work:

- `.mlcai/INTEGRATION.md` — How this project fits into the larger system
- `.mlcai/TECH_STACK.md` — Stack & dependencies
- `.mlcai/API_CONTRACT.md` — API contract / endpoints
- `.mlcai/DECISION_LOG.md` — Architecture decisions
- `.mlcai/BACKLOG.md` — Bugs / ideas / tasks
- `.mlcai/INFRASTRUCTURE.md` — Infrastructure & deployment
### Two modes: local-only vs. service-backed

`dochub-mcp` runs in one of two modes, chosen at startup:

1. **Local-only** — no `-dochub-url` / `DOCHUB_API_URL`. The MCP reads
   and writes the brain directory and templates directly from the
   filesystem. Fine for a single-host, solo-developer setup.
2. **Service-backed** — `-dochub-url http://host:port/dochub` points at
   a running Doc Hub HTTP service. **Registry mutations** (registering
   projects, editing metadata) go via HTTP so there is always a single
   writer to `registry.yaml`. **Template reads** and **global-context
   reads** also hit the service (useful on a remote checkout). Local
   `.mlcai/*.md` writes stay on disk — `.mlcai` belongs to whichever
   host the project lives on.

If `-dochub-url` is set but the service is unreachable, registry-mutating
tools fail loudly rather than silently writing a stale local copy.

### Do NOT re-register a known project

If you are working in a remote checkout (Mac, laptop, another Linux
box) and notice that the project's server-side path (e.g.
`/mnt/data2tb/new_mlcterm`) does not exist on your filesystem, **this
is expected**. The MCP auto-detects your local clone via `.mcp.json`
and `.mlcai/.project-id` and maps `.mlcai/*.md` operations to your cwd
automatically. `list_projects` / `get_project_context` / `get_doc` /
`create_doc` / backlog tools all work against the local clone while
registry mutations round-trip through the central service.

→ **Use the existing `project_id` (`localembed`). Never call
`register_project` with a new id like `<name>-mac` or `<name>-local`,
and never call `update_project` to "fix" the path to your local
filesystem** — the central registry is shared across every host, and
the MCP's cwd detection already handles per-host path mapping. Writing
Mac-specific paths into `registry.yaml` corrupts it for the xeon
service and every other client.

### Writes: exclusively through `mcp-dochub`

Every create/update/delete of a `.mlcai/` file goes through the MCP
server. It stamps the `## 📋 Meta` footer (author, timestamp,
base-modified), appends to the activity log, and protects against
concurrent edits via optimistic locking. A native write would skip all
of that — **please don't**.

| Purpose | Tool |
|---------|------|
| Begin a session (once) | `mcp__mlc-dochub__start_session` |
| List open doc todos | `mcp__mlc-dochub__get_doc_todos` |
| Read a doc before editing | `mcp__mlc-dochub__get_doc` (returns `last_modified` → pass as `base_modified` on the next write) |
| Template for a new doc | `mcp__mlc-dochub__get_template` |
| Create or replace a doc | `mcp__mlc-dochub__create_doc` (with `author`) |
| Patch one section | `mcp__mlc-dochub__update_section` (with `base_modified`) |
| Mark a todo done | `mcp__mlc-dochub__complete_doc_todo` |
| Append to the session log | `mcp__mlc-dochub__update_worklog` |
| Project / style context | `mcp__mlc-dochub__get_project_context` / `get_global_context` |

On every write, pass `author="<your-model-name>"` (e.g. `"claude-sonnet-4-6"`,
`"gemini-2.5-pro"`, `"qwen3.5:35b"`).

### If the MCP server isn't available

If any `mcp__mlc-dochub__*` tool call returns **"tool not found"** or the
server does not start:

- **Don't write.** No native write/edit for `.mlcai/` files — a half-
  written doc without the meta stamp is worse than none at all.
- **Reading via native Read is fine** — use the doc listing above as a
  context index.
- **Point the user at the installation:**

  ```bash
  git clone https://github.com/mlc911/mlcintegration
  cd mlcintegration && task install-all
  ```

  `install-all` builds binaries, symlinks them to `~/.local/bin/`, and
  symlinks `brain/` + `templates/` into `~/.local/share/mlc-dochub/` so
  `dochub-mcp` auto-discovers them from any project directory. The
  `.mcp.json` in this repo carries no machine-specific paths — same
  config works on every machine where `task install-all` ran.

  Then restart the CLI (claude / gemini / opencode) — the `.mcp.json`
  in the project root registers the server automatically.

- **Remote checkout (Mac, laptop, …)** — if you are working against a
  project cloned from a private bare while the Doc Hub service runs on
  another host, add `-dochub-url http://<dochub-host>:8600/dochub` to
  the MCP args in `.mcp.json` (or export `DOCHUB_API_URL`). The MCP
  binary itself still has to be installed locally; only registry
  writes and template reads round-trip to the remote service.

### Global context & style guide

The MCP call `mcp__mlc-dochub__get_global_context` returns project-type-
specific style guides and a language-filtered code style. Call it once
before creating new docs.
<!-- mlc-dochub:end -->

## 🧠 Codebase Memory & Intelligence Engine (`cbm`)

Dieses Projekt unterstützt den `codebase-memory` MCP-Server (`cbm-server`) für blitzschnelle strukturelle Codebase-Navigation (<50µs In-Memory Graph, Tree-Sitter ASTs, Call-Hierarchien und Qdrant-Vektorsuche).

### 🛠️ Empfohlener Agent-Workflow:
1. **Orientierung bei Session-Start:** Rufe `get_repo_map` auf, um die Paketstruktur, Interfaces und Monorepo-Subprojekte kompakt (<400 Tokens) zu erfassen.
2. **Falls Repository noch nicht indexiert ist:** Rufe einmalig `index_repository(path=".")` auf.
3. **Funktions- & Aufrufketten analysieren:** Nutze `get_callers`, `get_callees` oder `trace_call_path` anstelle manueller Datei-Suchen.
4. **Vor Refactorings / Änderungen:** Prüfe Abhängigkeiten mit `simulate_refactoring` oder `get_impact_radius`.
5. **Git Diff Impact:** Nutze `detect_changes`, um den Blast Radius ungespeicherter Änderungen zu analysieren.
