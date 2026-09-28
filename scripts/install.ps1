[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "DoctorCode\bin"),
    [string]$SourceDir = "",
    [string]$BaseUrl = "https://github.com/maxqstudio/DoctorCode/releases/download"
)

$ErrorActionPreference = "Stop"

$architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
switch ($architecture) {
    "X64" { $arch = "amd64" }
    "Arm64" { $arch = "arm64" }
    default { throw "Unsupported architecture: $architecture" }
}

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("doctorcode-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempRoot | Out-Null

try {
    if ($SourceDir) {
        $resolvedSource = (Resolve-Path -LiteralPath $SourceDir).Path
        $matches = @(Get-ChildItem -LiteralPath $resolvedSource -Filter "doctorcode_*_windows_$arch.zip" -File)
        if ($matches.Count -ne 1) {
            throw "Expected exactly one native archive in $resolvedSource, found $($matches.Count)"
        }
        $archivePath = $matches[0].FullName
        $filename = $matches[0].Name
        $checksumPath = Join-Path $resolvedSource "checksums.txt"
        if (-not (Test-Path -LiteralPath $checksumPath -PathType Leaf)) {
            throw "Missing checksums.txt in $resolvedSource"
        }
        if (-not $Version) {
            $match = [regex]::Match($filename, "^doctorcode_(.+)_windows_$arch\.zip$")
            if (-not $match.Success) {
                throw "Cannot derive version from $filename"
            }
            $Version = $match.Groups[1].Value
        }
    }
    else {
        if (-not $Version) {
            throw "-Version is required for network installs"
        }
        $filename = "doctorcode_${Version}_windows_${arch}.zip"
        $archivePath = Join-Path $tempRoot $filename
        $checksumPath = Join-Path $tempRoot "checksums.txt"
        Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/v$Version/$filename" -OutFile $archivePath
        Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/v$Version/checksums.txt" -OutFile $checksumPath
    }

    $escaped = [regex]::Escape($filename)
    $lines = @(Get-Content -LiteralPath $checksumPath | Where-Object { $_ -match "^[0-9a-fA-F]{64}\s+$escaped$" })
    if ($lines.Count -ne 1) {
        throw "Checksum entry missing or ambiguous for $filename"
    }
    $expected = ($lines[0] -split "\s+", 2)[0].ToLowerInvariant()
    $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
    if ($actual -ne $expected) {
        throw "SHA-256 mismatch for $filename"
    }

    $extractDir = Join-Path $tempRoot "extract"
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
    $packageDir = Join-Path $extractDir "doctorcode_${Version}_windows_${arch}"
    $doctorcode = Join-Path $packageDir "doctorcode.exe"
    $mcp = Join-Path $packageDir "doctorcode-mcp.exe"
    if (-not (Test-Path -LiteralPath $doctorcode -PathType Leaf)) {
        throw "Release archive missing doctorcode.exe"
    }
    if (-not (Test-Path -LiteralPath $mcp -PathType Leaf)) {
        throw "Release archive missing doctorcode-mcp.exe"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -LiteralPath $doctorcode -Destination (Join-Path $InstallDir "doctorcode.exe") -Force
    Copy-Item -LiteralPath $mcp -Destination (Join-Path $InstallDir "doctorcode-mcp.exe") -Force
    Write-Host "DoctorCode $Version installed to $InstallDir"
    Write-Host "Add $InstallDir to PATH if it is not already present."
}
finally {
    Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
}
