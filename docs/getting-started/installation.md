# Installation & Setup

RepoCortex ships as a single, self-contained binary with **zero external runtime dependencies** and **zero CGO**. It runs locally on macOS, Linux, and Windows.

---

## 1. Automated Installation (Recommended)

### macOS & Linux
```bash
curl -fsSL https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/install.sh | sh
```
*Installs the universal binary to `/usr/local/bin/repocortex` (or `~/.local/bin/repocortex`), removes macOS quarantine flags, creates the `repocortex` alias, and verifies the SHA256 checksum against official release artifacts.*

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/satyendrakumarsingh/repocortex/main/install.ps1 | iex
```
*Installs `repocortex.exe` and `repocortex.exe` to `%LOCALAPPDATA%\Programs\RepoCortex` and adds it to your user `PATH`.*

---

## 2. Install via Go Toolchain

If you have Go installed (Go 1.23+):
```bash
go install github.com/satyendrakumarsingh/repocortex/cmd/repocortex@latest
```
*Compiles the pure-Go binary with embedded Tree-sitter parsers and local SQLite.*

---

## 3. Precompiled Release Binaries

Download precompiled, checksummed binaries from the [GitHub Releases page](https://github.com/satyendrakumarsingh/repocortex/releases):

| Target | Architecture | Archive |
|---|---|---|
| **macOS Universal** | Apple Silicon (M1-M4) & Intel x86_64 | `repocortex-mac.zip` |
| **Linux AMD64** | x86_64 servers & containers | `repocortex_v1.0.0_linux_amd64.tar.gz` |
| **Linux ARM64** | AWS Graviton, Raspberry Pi, ARM servers | `repocortex_v1.0.0_linux_arm64.tar.gz` |
| **Windows AMD64** | Windows 10 / 11 x86_64 | `repocortex_v1.0.0_windows_amd64.zip` / `repocortex.exe` |

Verify integrity using `SHA256SUMS`:
```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
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

The bundle serves an index; it does not build one. The repository you pick must already have been indexed with `repocortex init`, `repocortex index` and `repocortex resolve`, which needs the command-line tool from one of the methods above. Pointed at a repository with no index, the server exits with `no index at … run 'repocortex init' and 'repocortex index' first`.

---

## 7. Verification

Confirm installation:
```bash
repocortex version
```
Expected output:
```text
repocortex v1.0.0 (darwin/arm64)
```

---

## Next Steps
- Follow the [60-Second Quickstart](quickstart.md) to index your first repository.
- Wire your coding environment with the [Agent Integration Guide](agent-integration.md).
