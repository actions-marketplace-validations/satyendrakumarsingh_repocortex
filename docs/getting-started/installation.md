# Installation & Setup

RepoCortex ships as a single, self-contained binary with **zero external runtime dependencies** and **zero CGO**. It runs locally on macOS, Linux, and Windows.

---

## 1. Automated Installation (Recommended)

### macOS & Linux
```bash
curl -fsSL https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/install.sh | sh
```
*Downloads the release archive for your OS and architecture, verifies its SHA-256 against the release's `SHA256SUMS` (there is no flag to skip this), installs `repocortex` to `~/.local/bin`, and on macOS removes the quarantine flag. If `~/.local/bin` is not on your `PATH` the script prints the line to add.*

To choose the directory or a specific release, pass the script its options:

```bash
curl -fsSL https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/install.sh | sh -s -- --dir /usr/local/bin --version v1.0.1
```

It needs `curl`, `tar` and `awk`, and writing to `/usr/local/bin` needs permission to do so.

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/install.ps1 | iex
```
*Downloads `repocortex_<tag>_windows_amd64.zip`, verifies its SHA-256 against the release's `SHA256SUMS`, installs `repocortex.exe` to `%LOCALAPPDATA%\Programs\repocortex` and adds that directory to your user `PATH`. Open a new terminal afterwards.*

To choose the directory or a specific release, run the script as a block so it can take parameters:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/install.ps1))) -Dir C:\Tools\repocortex -Version v1.0.1
```

---

## 2. Install via Go Toolchain

If you have Go installed (Go 1.26 or newer, which is what `go.mod` requires):
```bash
go install github.com/satyendrakumarsingh/repocortex/cmd/repocortex@latest
```
*Compiles the pure-Go binary with embedded Tree-sitter parsers and local SQLite, and puts it in `$(go env GOPATH)/bin`. A binary built this way reports its version as `dev`: the release version is stamped in by the release build, not by `go install`.*

---

## 3. Precompiled Release Binaries

Download precompiled, checksummed binaries from the [GitHub Releases page](https://github.com/satyendrakumarsingh/repocortex/releases):

The archive names carry the release tag. For v1.0.1, the current release:

| Target | Architecture | Archive |
|---|---|---|
| **macOS Universal** | Apple Silicon & Intel x86_64 in one binary | `repocortex-mac.zip` |
| **macOS ARM64** | Apple Silicon | `repocortex_v1.0.1_darwin_arm64.tar.gz` |
| **macOS AMD64** | Intel x86_64 | `repocortex_v1.0.1_darwin_amd64.tar.gz` |
| **Linux AMD64** | x86_64 servers & containers | `repocortex_v1.0.1_linux_amd64.tar.gz` |
| **Linux ARM64** | AWS Graviton, Raspberry Pi, ARM servers | `repocortex_v1.0.1_linux_arm64.tar.gz` |
| **Windows AMD64** | Windows 10 / 11 x86_64 | `repocortex_v1.0.1_windows_amd64.zip`, or the bare `repocortex.exe` |

Every archive holds the binary, `LICENSE`, `NOTICE` and `README.md` at its top level, with no enclosing directory. Unpack it and move `repocortex` (or `repocortex.exe`) to a directory on your `PATH`. A binary downloaded with a browser on macOS is quarantined; `xattr -d com.apple.quarantine repocortex` clears it.

Verify integrity using `SHA256SUMS`, downloaded from the same release into the same directory:
```bash
shasum -a 256 -c SHA256SUMS --ignore-missing     # macOS
sha256sum -c SHA256SUMS --ignore-missing         # Linux
```

A checksum shows the download is intact, not who built it. From the next release (the first after v1.0.1), `SHA256SUMS` is signed with Sigstore, every archive has a build provenance attestation, and each has a CycloneDX SBOM. [Verifying a release](../operations/verifying-a-release.md) has the exact commands.

---

## 4. Container Image

> **Available from the next release.** The image is first published with the release after v1.0.1; `docker pull` fails until then. To use it today, build it from a checkout: `docker build -t repocortex .`

```bash
docker pull ghcr.io/satyendrakumarsingh/repocortex:latest
```

The image is built for `linux/amd64` and `linux/arm64`, holds the static binary and nothing else (no shell, no package manager), and runs as a non-root user (uid 65532). Tags carry no `v`: release `v1.0.2` is `:1.0.2`, `:1.0` and `:latest`.

RepoCortex keeps its index inside the repository, in `.repocortex/`. Mount the repository at `/workspace`, which is the working directory:

```bash
# Index, then serve over stdio to an MCP client
docker run --rm -v "$PWD:/workspace" ghcr.io/satyendrakumarsingh/repocortex init
docker run --rm -v "$PWD:/workspace" ghcr.io/satyendrakumarsingh/repocortex index
docker run --rm -v "$PWD:/workspace" ghcr.io/satyendrakumarsingh/repocortex resolve
docker run --rm -i -v "$PWD:/workspace" ghcr.io/satyendrakumarsingh/repocortex serve
```

On Linux the container user must be able to write `.repocortex/` in your checkout, so run it as yourself by adding `--user "$(id -u):$(id -g)"` to each command. Docker Desktop on macOS and Windows maps ownership for you.

To keep the source tree read-only and the index out of it, mount the repository `:ro` and give `.repocortex` a volume of its own. The directory has to exist first, because Docker cannot create a mount point inside a read-only mount:

```bash
mkdir -p .repocortex
docker run --rm -v "$PWD:/workspace:ro" -v repocortex-index:/workspace/.repocortex \
  ghcr.io/satyendrakumarsingh/repocortex index
```

Inside the container the project is named `workspace`, after the mount point, unless `.repocortex.yml` sets `project.name`.

---

## 5. Windows Package Managers

### Scoop

> **Available from the next release.** The manifest is `packaging/scoop/repocortex.json` in this repository. It is not in a Scoop bucket yet, so it is installed by URL.

```powershell
scoop install https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/packaging/scoop/repocortex.json
```

Scoop downloads the release archive, checks it against the SHA-256 in the manifest and puts `repocortex` on your `PATH`.

### winget

> **Not available yet.** The manifests are in `packaging/winget/` and are submitted to the community repository ([microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)) after the next release. The command below works once that submission is accepted, which is a review this project does not control.

```powershell
winget install SatyendraKumarSingh.RepoCortex
```

---

## 6. Claude Desktop Extension (MCP Bundle)

> **Available from the next release.** Bundles are attached to the release after v1.0.1 as `repocortex_<tag>_<platform>.mcpb`.

An `.mcpb` file is a one-file extension for Claude Desktop and other clients that read the [MCPB format](https://github.com/modelcontextprotocol/mcpb). Download the one for your platform from the [GitHub Releases page](https://github.com/satyendrakumarsingh/repocortex/releases) and open it; the client asks which repository to serve and then runs `repocortex serve` against it.

| Platform | Bundle |
|---|---|
| macOS (Apple Silicon & Intel) | `repocortex_<tag>_darwin_universal.mcpb` |
| Windows x86_64 | `repocortex_<tag>_windows_amd64.mcpb` |
| Linux x86_64 | `repocortex_<tag>_linux_amd64.mcpb` |
| Linux ARM64 | `repocortex_<tag>_linux_arm64.mcpb` |

The bundle serves an index; it does not build one. The repository you pick must already have been indexed with `repocortex init`, `repocortex index` and `repocortex resolve`, which needs the command-line tool from one of the methods above. Pointed at a repository with no index, the server still starts: it lists its tools and answers every call with the commands to run and the directory to run them in, and it begins answering from the index as soon as one exists, with no restart. It never starts indexing on its own. (Releases up to v1.0.1 exit instead, with `no index at … run 'repocortex init' and 'repocortex index' first`, which a client shows only as a server that disconnected.)

---

## 7. Verification

Confirm installation:
```bash
repocortex version
```
Expected output, with your release, commit, build time and platform:
```text
repocortex v1.0.1 (commit 1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b, built 2026-11-12T09:30:00Z, darwin/arm64)
```
A build from source or `go install` prints `repocortex dev (commit none, built unknown, darwin/arm64)`.

---

## Next Steps
- Follow the [60-Second Quickstart](quickstart.md) to index your first repository.
- Wire your coding environment with the [Agent Integration Guide](agent-integration.md).
