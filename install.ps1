# RepoCortex installer for Windows.
#
# Checksum verification is mandatory and has no bypass flag, matching
# install.sh. Invoke-WebRequest marks downloads with the Mark-of-the-Web, so
# the archive is unblocked after verification, not before.
[CmdletBinding()]
param(
    [string]$Dir = "$env:LOCALAPPDATA\Programs\repocortex",
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"
$repo = "satyendrakumarsingh/repocortex"

if ($Version -eq "latest") {
    $rel = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $Version = $rel.tag_name
}
if (-not $Version) { throw "could not resolve the latest release tag" }

$archive = "repocortex_${Version}_windows_amd64.zip"
$base = "https://github.com/$repo/releases/download/$Version"
$tmp = Join-Path $env:TEMP ("repocortex-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    Write-Host "downloading $archive ($Version)"
    Invoke-WebRequest -Uri "$base/$archive" -OutFile (Join-Path $tmp $archive)
    Invoke-WebRequest -Uri "$base/SHA256SUMS" -OutFile (Join-Path $tmp "SHA256SUMS")

    Write-Host "verifying checksum"
    $actual = (Get-FileHash -Algorithm SHA256 -Path (Join-Path $tmp $archive)).Hash.ToLower()
    $line = Select-String -Path (Join-Path $tmp "SHA256SUMS") -Pattern ([regex]::Escape($archive)) | Select-Object -First 1
    if (-not $line) { throw "$archive is absent from SHA256SUMS" }
    $expected = ($line.Line -split '\s+')[0].ToLower()
    if ($actual -ne $expected) {
        throw "checksum mismatch for ${archive}: expected $expected, actual $actual"
    }

    Unblock-File -Path (Join-Path $tmp $archive)
    Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath $tmp -Force

    # The release workflow zips repocortex.exe, LICENSE, NOTICE and README.md
    # at the ROOT of the archive (.github/workflows/release.yml), so after
    # Expand-Archive the binary is directly under $tmp. This used to look in a
    # repocortex_<tag>_windows_amd64\ subdirectory that no release has ever
    # contained, and failed at Copy-Item on every install.
    # TestInstallersMatchTheReleaseWorkflow keeps the two in step.
    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    $src = Join-Path $tmp "repocortex.exe"
    if (-not (Test-Path $src)) {
        throw "repocortex.exe is not at the root of $archive"
    }
    Copy-Item -Path $src -Destination (Join-Path $Dir "repocortex.exe") -Force

    Write-Host "installed $Dir\repocortex.exe"
    & (Join-Path $Dir "repocortex.exe") version

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$Dir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$Dir", "User")
        # ASCII only, here and everywhere in this file. Windows PowerShell 5.1 reads
        # a script with no byte-order mark as the ANSI code page, where the bytes
        # of a UTF-8 dash include 0x94, a closing smart quote, which PowerShell
        # accepts as a string delimiter: the string ends early and the script
        # does not parse.
        Write-Host "added $Dir to your user PATH. Open a new terminal to pick it up."
    }

    Write-Host ""
    Write-Host "Next: cd into a repository and run"
    Write-Host "  repocortex init; repocortex index; repocortex resolve"
    Write-Host "  repocortex install     # register with your coding agent"
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
