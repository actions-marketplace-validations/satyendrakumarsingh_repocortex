# Verifying a release

How to check that a RepoCortex download is the file the project built, from the
source it says it was built from, before you run it.

This applies to releases **after v1.0.1**. v1.0.1 and earlier publish
`SHA256SUMS` only, so for them only the [checksum step](#2-check-the-archive-against-sha256sums)
applies, and it proves the download is intact, not who made it.

## Contents

- [What a release publishes](#what-a-release-publishes)
- [What you need](#what-you-need)
- [1. Verify the signature on SHA256SUMS](#1-verify-the-signature-on-sha256sums)
- [2. Check the archive against SHA256SUMS](#2-check-the-archive-against-sha256sums)
- [3. Verify the build provenance](#3-verify-the-build-provenance)
- [The SBOM](#the-sbom)
- [The container image](#the-container-image)
- [The MCP bundles](#the-mcp-bundles)
- [Which identity signed what](#which-identity-signed-what)
- [What this does and does not prove](#what-this-does-and-does-not-prove)

## What a release publishes

Every file is attached to the GitHub release for the tag. `<tag>` is the tag
with its `v`, for example `v1.0.2`.

| File | What it is |
|---|---|
| `repocortex_<tag>_<os>_<arch>.tar.gz`, `repocortex_<tag>_windows_amd64.zip`, `repocortex-mac.zip`, `repocortex.exe` | The archives and binaries |
| `SHA256SUMS` | The SHA-256 of each file above |
| `SHA256SUMS.sigstore.json` | A Sigstore bundle: the signature over `SHA256SUMS`, the signing certificate and the transparency-log proof |
| `repocortex_<tag>_<os>_<arch>.cdx.json` | A CycloneDX SBOM of the binary inside the matching archive |
| `repocortex_<tag>_<platform>.mcpb`, `MCPB_SHA256SUMS` | The MCP bundles and their checksums |

Not attached, because GitHub stores them itself: a **build provenance
attestation** for every archive, binary, SBOM and bundle, and for the
container image.

Nothing is signed with a long-lived key. The signature is *keyless*: the
release workflow proves its identity to Sigstore with a short-lived GitHub
token, receives a certificate that names the workflow file and the tag, signs,
and the certificate expires minutes later. What you verify is that identity.

## What you need

- [`cosign`](https://docs.sigstore.dev/cosign/system_config/installation/) 3.0
  or later, for the signature. The commands below use the `--bundle` format
  that cosign 3 writes.
- [`gh`](https://cli.github.com/), the GitHub CLI, for the provenance, signed
  in with `gh auth login`. `gh attestation verify` refuses to run without a
  login, even against a public repository.
- `sha256sum` (Linux) or `shasum` (macOS).

Set the tag once and download what you want to check:

```bash
TAG=v1.0.2
ARCHIVE="repocortex_${TAG}_linux_amd64.tar.gz"

gh release download "$TAG" --repo satyendrakumarsingh/repocortex \
  --pattern "$ARCHIVE" --pattern SHA256SUMS --pattern SHA256SUMS.sigstore.json
```

Without `gh`, the same three files are at
`https://github.com/satyendrakumarsingh/repocortex/releases/download/<tag>/<file>`.

## 1. Verify the signature on SHA256SUMS

```bash
cosign verify-blob \
  --bundle SHA256SUMS.sigstore.json \
  --certificate-identity "https://github.com/satyendrakumarsingh/repocortex/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  SHA256SUMS
```

It prints `Verified OK`. That means `SHA256SUMS` is the file signed by
`.github/workflows/release.yml` in this repository, running for that tag.

Both flags matter. `--certificate-oidc-issuer` says the identity was vouched
for by GitHub Actions; `--certificate-identity` says which workflow, in which
repository, on which ref. A signature from a fork, from another workflow in
this repository, or from a different tag fails the check. Do not replace the
identity with a loose `--certificate-identity-regexp`: a pattern that matches
any repository proves nothing.

## 2. Check the archive against SHA256SUMS

Linux:

```bash
sha256sum -c SHA256SUMS --ignore-missing
```

macOS:

```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
```

Windows (PowerShell), comparing by eye or in a script:

```powershell
(Get-FileHash -Algorithm SHA256 repocortex_v1.0.2_windows_amd64.zip).Hash.ToLower()
Select-String -Path SHA256SUMS -Pattern "repocortex_v1.0.2_windows_amd64.zip"
```

`--ignore-missing` checks the files you downloaded and skips the lines for the
ones you did not. Each checked file prints `OK`.

Step 1 made `SHA256SUMS` trustworthy; this step extends that to the archive.
Run on its own, a checksum only shows the download was not corrupted: anyone
who could replace the archive could replace `SHA256SUMS` beside it.

## 3. Verify the build provenance

```bash
gh attestation verify "$ARCHIVE" \
  --repo satyendrakumarsingh/repocortex \
  --signer-workflow satyendrakumarsingh/repocortex/.github/workflows/release.yml \
  --source-ref "refs/tags/${TAG}"
```

This checks a signed [SLSA provenance](https://slsa.dev/spec/v1.0/provenance)
statement that GitHub recorded when the archive was built. It ties the exact
bytes of the file to the repository, the workflow, the tag and the commit, and
it does so independently of `SHA256SUMS`: it is a second route to the same
conclusion, and it works for any single file without downloading the
checksums.

Add `--format json` to read the statement, including the commit the build ran
from.

## The SBOM

Each archive has a software bill of materials in
[CycloneDX](https://cyclonedx.org/) 1.6 JSON, named after it:
`repocortex_<tag>_linux_amd64.cdx.json` for
`repocortex_<tag>_linux_amd64.tar.gz`. There is one per platform because the
dependency set differs: the Windows binary links modules the Linux one does
not.

It is read out of the built binary, not out of `go.mod`, so it lists the
modules and the Go standard library version that were linked in. The property
`cdx:gomod:binary:hash:SHA-256` in its metadata is the hash of the binary
inside the archive:

```bash
gh release download "$TAG" --repo satyendrakumarsingh/repocortex \
  --pattern "repocortex_${TAG}_linux_amd64.cdx.json"

jq -r '.metadata.properties[] | select(.name == "cdx:gomod:binary:hash:SHA-256") | .value' \
  "repocortex_${TAG}_linux_amd64.cdx.json"
tar -xzf "$ARCHIVE" repocortex && sha256sum repocortex
```

The two hashes are the same. The SBOM has its own provenance attestation:

```bash
gh attestation verify "repocortex_${TAG}_linux_amd64.cdx.json" \
  --repo satyendrakumarsingh/repocortex
```

## The container image

The image is `ghcr.io/satyendrakumarsingh/repocortex`, for `linux/amd64` and
`linux/arm64`. Its tags carry no `v`: the release `v1.0.2` is `:1.0.2`, and
also `:1.0` and `:latest`.

```bash
IMAGE=ghcr.io/satyendrakumarsingh/repocortex:${TAG#v}

cosign verify "$IMAGE" \
  --certificate-identity "https://github.com/satyendrakumarsingh/repocortex/.github/workflows/release.yml@refs/tags/${TAG}" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

gh attestation verify "oci://${IMAGE}" \
  --repo satyendrakumarsingh/repocortex \
  --signer-workflow satyendrakumarsingh/repocortex/.github/workflows/release.yml
```

Both resolve the tag to a digest and verify the digest. A tag can be moved, so
once verified, pin what you run to the digest `cosign verify` printed:
`ghcr.io/satyendrakumarsingh/repocortex@sha256:…`.

`gh attestation verify` reads the image from the registry, and its help states
that you must already be signed in to that registry; run `docker login ghcr.io`
first if it reports an authentication error.

## The MCP bundles

```bash
gh release download "$TAG" --repo satyendrakumarsingh/repocortex \
  --pattern "repocortex_${TAG}_darwin_universal.mcpb" --pattern MCPB_SHA256SUMS

sha256sum -c MCPB_SHA256SUMS --ignore-missing
gh attestation verify "repocortex_${TAG}_darwin_universal.mcpb" \
  --repo satyendrakumarsingh/repocortex
```

`MCPB_SHA256SUMS` is not signed with cosign; the attestation is what
authenticates a bundle. The binary inside a bundle is the same file, byte for
byte, as the one in the archive for that platform.

## Which identity signed what

The certificate identity is the workflow file plus the ref the run started
from, so it depends on how the release was made.

| How the run started | `--certificate-identity` ends with | `--source-ref` |
|---|---|---|
| A `v*` tag was pushed (every normal release) | `release.yml@refs/tags/<tag>` | `refs/tags/<tag>` |
| The `canary` pre-release, built on each push to `main` | `release.yml@refs/heads/main` | `refs/heads/main` |
| A maintainer ran the workflow by hand from `main` with a tag name | `release.yml@refs/heads/main` | `refs/heads/main` |

In all three the issuer is `https://token.actions.githubusercontent.com` and
the identity starts with
`https://github.com/satyendrakumarsingh/repocortex/.github/workflows/`.

If a release fails verification with the tag identity, do not reach for a
looser one. Open the run that produced it (the repository's **Actions** tab,
workflow **Release**) and read which ref it started from. The canary publishes
a signature and archive attestations but no SBOMs, bundles or image.

## What this does and does not prove

It proves that the file was produced by this repository's release workflow,
for the tag you named, from a commit you can look up, and that it has not
changed since.

It does not prove the source is free of defects, or that a maintainer's
account was not misused to push the tag: provenance tells you where a binary
came from, not whether what it came from is good. It also does not cover
`go install`, which builds from source on your machine and is checked by the
Go checksum database instead.

A release is still published if signing fails. The archives and `SHA256SUMS`
are built and uploaded first, and the signature, attestations, SBOMs, bundles
and image are added by later jobs, so that an outage at Sigstore cannot hold
back a fix. If `SHA256SUMS.sigstore.json` is missing from a release, that is
what happened: the **Release** workflow run for that tag shows the failed job,
and the checksum step is all that can be verified until the job is re-run.
