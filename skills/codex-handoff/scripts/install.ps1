param([string]$Version = "v1.0.1", [string]$InstallDir = "$env:LOCALAPPDATA\Programs\codex-lan-handoff")
$ErrorActionPreference = "Stop"
$repo = "zhiaiyrb/codex-lan-handoff"
$arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) { "X64" { "amd64" } "Arm64" { "arm64" } default { throw "Unsupported architecture" } }
$asset = "codex-lan-handoff_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/$Version"
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ("codex-handoff-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $temp | Out-Null
try {
  Invoke-WebRequest "$base/$asset" -OutFile (Join-Path $temp $asset)
  Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $temp "checksums.txt")
  $checksumLine = (Get-Content (Join-Path $temp "checksums.txt")) | Where-Object { $_ -match "\s$([regex]::Escape($asset))$" } | Select-Object -First 1
  if (-not $checksumLine) { throw "Checksum not found for $asset" }
  $expected = ($checksumLine -split '\s+', 2)[0]
  $actual = (Get-FileHash (Join-Path $temp $asset) -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actual -ne $expected.ToLowerInvariant()) { throw "Checksum mismatch" }
  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Expand-Archive -LiteralPath (Join-Path $temp $asset) -DestinationPath $InstallDir -Force
  Write-Output "Installed to $InstallDir. Add this directory to PATH, then run: codex-lan-handoff setup"
} finally { Remove-Item -LiteralPath $temp -Recurse -Force -ErrorAction SilentlyContinue }
