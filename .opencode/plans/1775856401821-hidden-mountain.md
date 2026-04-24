# Use OpenCode Server For Autopilot

## Goal

Change the Go `workie autopilot` path so the `opencode` backend uses an already-running `opencode serve` instance over HTTP instead of shelling out to `opencode run`.

## Recommended Approach

Keep `autopilot` orchestration intact and replace backend execution at the `executeSession` boundary with a transport abstraction.

### 1. Introduce a runner boundary in `autopilot`

Add a small internal runner abstraction so `pipeline.go` does not care whether a backend is implemented by a subprocess or by the OpenCode server.

Recommended shape:
- `SessionRunner` interface with `Preflight(ctx)` and `Run(ctx, request, output)`
- `SessionRequest` carrying at least prompt, model, and working directory
- a factory in `autopilot` that returns:
  - a CLI runner for `claude` and `codex`
  - an OpenCode HTTP runner for `opencode`

This keeps prompt generation, backend parsing, and model resolution mostly unchanged.

### 2. Move current subprocess execution into a CLI runner

Refactor the current CLI-specific logic out of `autopilot/pipeline.go`.

Move into a new CLI runner:
- `exec.LookPath(...)`
- `BuildSessionCommand(...)`
- `exec.CommandContext(...)`
- current timeout and start/wait behavior

Keep current `claude` and `codex` behavior unchanged.

### 3. Implement an OpenCode server runner for `opencode`

Use `net/http` with small local request/response structs.

Planned server interaction:
- `GET /global/health` for health
- `POST /session` to create a fresh session per autopilot call
- `POST /session/:id/message` to send the prompt and wait for the reply
- `DELETE /session/:id` in `defer` cleanup

Use a fresh OpenCode session for each `executeSession(...)` call so the current per-phase isolation semantics remain intact.

### 4. Require an already-running OpenCode server

Do not make `workie autopilot` spawn or manage `opencode serve` in this change.

Reasons:
- smaller and safer scope
- avoids lifecycle issues like port selection, startup races, orphan cleanup, and auth bootstrapping
- keeps this change focused on replacing the transport for the `opencode` backend only

### 5. Add explicit autopilot config for OpenCode server connection

Extend autopilot config so the server URL and auth lookup are explicit.

Recommended config shape:

```yaml
autopilot:
  backend: opencode
  models:
    opencode: openai/gpt-5.4
  opencode:
    server_url: http://127.0.0.1:4096
    username_env: OPENCODE_SERVER_USERNAME
    password_env: OPENCODE_SERVER_PASSWORD
```

Also add a minimal CLI override:
- `--opencode-server-url`

Keep credentials out of YAML and load them from env names when needed.

### 6. Update preflight and logging semantics

For `opencode`, preflight should become a server check instead of a PATH check.

Planned preflight behavior:
- health probe with `GET /global/health`
- a minimal create-message-delete round trip using the configured model

Logging behavior:
- continue writing phase logs from `pipeline.go`
- for `opencode`, write the final response and useful request/response metadata to the log
- if `--show-output` is enabled, mirror the same content to stdout

Do not add SSE streaming in the first pass unless the synchronous message response turns out to be insufficient.

## Critical Files

- `autopilot/pipeline.go`
- `autopilot/backend.go`
- `cmd/autopilot.go`
- `config/config.go`
- `autopilot/backend_test.go`
- `autopilot/pipeline_test.go`
- `config/config_test.go`

Likely new files:
- `autopilot/session_runner.go`
- `autopilot/cli_runner.go`
- `autopilot/opencode_runner.go`

## File-Level Changes

### `autopilot/pipeline.go`

- add a `runner` field to `Pipeline`
- build the runner in `NewPipeline(...)`
- refactor `executeSession(...)` to open the log file and delegate execution to the runner
- refactor `preflightCheck()` to delegate to the runner
- remove direct subprocess assumptions from the pipeline for `opencode`

### `autopilot/backend.go`

- keep backend parsing, validation, and default model logic
- remove the `opencode` branch from `BuildSessionCommand(...)`
- keep `BuildSessionCommand(...)` only for CLI-backed backends, or move it into the CLI runner if that makes the transport split cleaner

### `cmd/autopilot.go`

- keep existing backend/model precedence
- add `--opencode-server-url`
- resolve OpenCode server URL from flag or config when backend is `opencode`
- pass the resolved value into autopilot options

### `config/config.go`

- extend `AutopilotConfig` with an OpenCode subsection for server settings
- keep the new settings autopilot-scoped rather than introducing global server config

### `autopilot/backend_test.go`

- stop asserting an `opencode` shell command if `opencode` is no longer CLI-backed
- keep `claude` and `codex` command construction coverage
- keep `opencode` backend/default-model coverage

### `autopilot/pipeline_test.go`

- add tests around a fake runner
- verify `runPhase(...)`, `runGitCommit(...)`, and `preflightCheck()` delegate correctly
- verify log file behavior still works

### `config/config_test.go`

- add YAML coverage for `autopilot.opencode.server_url`
- add YAML coverage for `username_env` and `password_env` if implemented

## Assumptions To Validate During Implementation

- `POST /session/:id/message` waits for the final response and returns enough content for autopilot logs
- the exact model request shape for the server API
  - docs appear inconsistent between string and structured model forms
- the server operates in the intended workspace
  - if there is no per-session working directory in the API, the server may need to be started from the repo root
- `DELETE /session/:id` is safe after failures/timeouts
- basic auth behavior matches `OPENCODE_SERVER_USERNAME` and `OPENCODE_SERVER_PASSWORD`
- synchronous responses are enough for `--show-output` without adding SSE

The most important validation is the workspace/cwd behavior of the server.

## Verification

### Unit tests

Run:

```bash
go test ./autopilot ./config ./cmd/...
```

Add tests for:
- OpenCode runner success path with `httptest.Server`
- health failure
- auth header behavior
- timeout/cancellation behavior
- session cleanup on success and failure
- pipeline delegation through a fake runner
- config loading for OpenCode server settings

### End-to-end check

1. Start the server manually from the target workspace:

```bash
opencode serve --hostname 127.0.0.1 --port 4096
```

2. Confirm health:

```bash
curl http://127.0.0.1:4096/global/health
```

3. Run autopilot against a known valid change:

```bash
workie autopilot <change> --backend opencode --opencode-server-url http://127.0.0.1:4096
```

4. Verify:
- preflight succeeds without `opencode run`
- phase logs contain the server-backed assistant response
- the flow still handles intermediate commit sessions
- sessions are deleted after each prompt

5. If auth is enabled, verify the client-side env-based auth path also works.
