# install.ps1 — install cloudres from GitHub releases.
#
# Usage (PowerShell):
#   Invoke-WebRequest -Uri https://raw.githubusercontent.com/mars-base/cloudres/main/scripts/install.ps1 -OutFile install.ps1
#   .\install.ps1
#
#   # Install a specific version:
#   .\install.ps1 -Tag v1.0.0
#
#   # Install to a custom directory:
#   .\install.ps1 -InstallDir C:\tools

param (
    [string]$Tag = "latest",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

$Repo = "mars-base/cloudres"

# --- detect architecture ----------------------------------------------
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default {
        Write-Error "unsupported architecture: $Arch"
        exit 1
    }
}

$Binary = "cloudres-windows-${Arch}.exe"

# --- resolve version --------------------------------------------------
if ($Tag -eq "latest") {
    $ReleaseUrl = "https://api.github.com/repos/${Repo}/releases/latest"
    $Response = Invoke-RestMethod -Uri $ReleaseUrl -Method Get
    $DownloadUrl = ($Response.assets | Where-Object name -eq $Binary).browser_download_url
} else {
    $DownloadUrl = "https://github.com/${Repo}/releases/download/${Tag}/${Binary}"
}

if (-not $DownloadUrl) {
    Write-Error "could not find download URL for ${Binary} (tag: ${Tag})"
    exit 1
}

# --- install ----------------------------------------------------------
if ($InstallDir -eq "") {
    $InstallDir = Join-Path $env:LOCALAPPDATA "cloudres"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$Dest = Join-Path $InstallDir "cloudres.exe"

Write-Host "→ downloading ${Binary} ..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $Dest

# Add to PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*${InstallDir}*") {
    [Environment]::SetEnvironmentVariable("Path", "${UserPath};${InstallDir}", "User")
    $env:Path += ";${InstallDir}"
    Write-Host "→ added ${InstallDir} to user PATH"
}

Write-Host "✓ cloudres installed to ${Dest}"
cloudres version
