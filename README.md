## Runtimz (`rtz`)

Runtimz is a small cross‑platform version manager for **Go**, **Node.js**, and **Python**.

### Quickstart

Convenience command to install (or update to newer version):

- Windows:
```Powershell
irm "https://raw.githubusercontent.com/kamilludwinski/runtimzzz/master/scripts/install.ps1" | iex
```

### OS support:
- Windows AMD64 (and presumably ARM64, not validated)
- Planned: Linux AMD64
- Planned: Darwin (macos) ARM64 (apple silicon)

### Capabilities

- **Install** runtime versions
- **Switch** runtime active version
- **Uninstall** versions you no longer need
- **List** available and installed versions
- **Purge** all installations globally or per-runtime

The CLI binary is called `rtz` (short for **Runtimz**).

---

## Installation

### Build from source

Prerequisites:

- **Go 1.21+** installed
- Git

Clone the repo and build:

```bash
git clone https://github.com/kamilludwinski/runtimzzz.git
cd runtimz
go build -o rtz .
```

On Windows this will produce `rtz.exe`.

### Directory layout

Runtimz uses a shim directory under your home:

- App data directory: `~/.runtimz`
- Installations: `~/.runtimz/installations`
- Shims: `~/.runtimz/shims`

When you activate a version, Runtimz:

- Writes shims into `~/.runtimz/shims` (such as `go`, `gofmt`, `node`, `npm`, `npx`)
- Ensures `~/.runtimz/shims` is on your **user PATH**
- Marks the version as active in the config

---

## Usage

Global help (no args):

```bash
rtz
```

Print Runtimz version:

```bash
rtz version
```

Update rtz to the latest release:

```bash
rtz update
```

Purge all Runtimz data (keeps logs):

```bash
rtz purge
```

Runtime commands (Go, Node, Python):

```bash
rtz <runtime> ls
rtz <runtime> install <version>
rtz <runtime> use <version>
rtz <runtime> uninstall <version>
rtz <runtime> purge
```

Supported runtimes:

- `go`
- `node`
- `python`

### Examples

- **List Go versions** (top 10 LTS/stable for your OS/arch):

  ```bash
  rtz go ls
  ```

- **Install a specific Go version**:

  ```bash
  rtz go install 1.22.0
  ```
  or
  ```bash
  rtz go i 1.22.0
  ```

- **Install latest stable Go**:

  ```bash
  rtz go install latest
  ```

- **Install by major/minor prefix** (resolves to highest matching patch):

  ```bash
  rtz go install 1.22   # resolves to latest 1.22.x available
  ```

- **Uninstall a Go version**:

  ```bash
  rtz go uninstall 1.21.3
  ```
  or
  ```bash
  rtz go u 1.21.3
  ```

- **Activate a Go version** (writes shims and updates PATH if needed):

  ```bash
  rtz go use 1.22.0
  ```

- **Activate using prefixes or latest**:

  ```bash
  rtz go use latest   # uses highest installed Go
  rtz go use 1.22     # uses highest installed 1.22.x
  rtz go use 1        # uses highest installed 1.x.y
  ```

- **List Node.js versions** available for your platform:

  ```bash
  rtz node ls
  ```

- **Install and activate Node 20.x**:

  ```bash
  rtz node install 20.10.0
  rtz node use 20.10.0
  ```

- Now the `node`, `npm`, and `npx` shims from `~/.runtimz/shims` should be first on PATH:

  ```bash
  node -v
  npm -v
  ```

---

## How it works (high level)

- **State file**: `~/.runtimz/state.json` holds which version is active per runtime.
- **Installations**:
  - Go: `~/.runtimz/installations/go/<version>`
  - Node: `~/.runtimz/installations/node/<version>`
  - Python (Windows only): `~/.runtimz/installations/python/<version>`
- **Shims**:
  - Created in `~/.runtimz/shims`
  - For Go: `go`, `gofmt` (`.cmd`/`.exe` on Windows, Bash shims for Git Bash)
  - For Node: `node`, `npm`, `npx`
  - For Python (Windows): `python`, `pip`
- **Remote sources**:
  - Go metadata and archives from `https://go.dev/dl/`
  - Node metadata and archives from `https://nodejs.org/download/release/`
  - Python versions and embeddable zips from `https://www.python.org/ftp/python/`

On Windows, a small embedded `.exe` shim is dropped for tools that expect `go.exe`, `node.exe`, etc., and delegates to the real version under `installations/.../bin`.

---

## Development

Run tests (if present in the repo):

```bash
go test ./...
```

---

## License

TBD – add license information here if/when you decide on one.

