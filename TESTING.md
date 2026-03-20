# Testing

## Unit tests

Run all unit tests:

```bash
go test ./...
```

Tests use `RTZ_APP_DIR` (and optionally `DEV_APP_NAME`) to isolate app data under a temporary directory so that no real `~/.runtimz` data is used.

### Test layout

- **`internal/*`** – Unit tests live next to the code (e.g. `internal/state/state_test.go`, `internal/update/update_test.go`, `internal/meta/meta_test.go`, `internal/runtime/helpers_test.go`, `internal/utils/*_test.go`).
- **`cmd/`** – CLI and purge behaviour tests in `cmd/purge_test.go` (global purge, per-runtime purge, state clear).

### Running specific packages

```bash
go test ./cmd/...
go test ./internal/update/...
go test ./internal/state/...
go test ./internal/meta/...
go test ./internal/runtime/...
```

## Integration tests

The same test binary runs both unit and integration-style tests. Integration tests in `cmd/purge_test.go` exercise the full purge flow (e.g. `TestHandleRuntime_Purge`) with a temporary app dir and registered runtimes.

No extra flags are required; run:

```bash
go test ./cmd -v
```

## E2E tests (Windows)

End-to-end tests run the real `rtz.exe` binary on a Windows host (e.g. GitHub Actions or a local Windows dev environment).

### CI (GitHub Actions)

The workflow [`.github/workflows/e2e.yml`](.github/workflows/e2e.yml) runs on `windows-latest` on push/PR to `main`/`master`. It:

1. Builds the Windows shim and `rtz.exe`.
2. Sets `RTZ_APP_DIR` to a runner temp directory.
3. Runs [`scripts/e2e/windows-e2e.ps1`](scripts/e2e/windows-e2e.ps1).

### Running E2E locally (Windows)

1. Build from repo root:

   ```powershell
   go build -o internal/shimembed/shim.exe ./cmd/shim
   go build -o rtz.exe .
   ```

2. Set a dedicated app dir (optional, to avoid touching real data):

   ```powershell
   $env:RTZ_APP_DIR = "$env:TEMP\rtz-e2e"
   ```

3. Run the E2E script from repo root:

   ```powershell
   powershell -ExecutionPolicy Bypass -File ./scripts/e2e/windows-e2e.ps1
   ```

The script expects `rtz.exe` in the current directory (or under the repo when run from `scripts/e2e`). It checks:

- `rtz version` prints a version string.
- `rtz` (no args) prints help including version, purge, update.
- `rtz purge` when no app dir exists (or empty) completes without error.
- `rtz go purge` with no installed versions prints an appropriate message.
- `rtz go ls` runs and prints Go version info (smoke test).

### Future: Linux/macOS E2E

When release artifacts for Linux and macOS are published, add corresponding jobs in `.github/workflows/e2e.yml` and shell scripts under `scripts/e2e/` (e.g. `linux-e2e.sh`, `darwin-e2e.sh`) that run the built binary with a temp app dir.
