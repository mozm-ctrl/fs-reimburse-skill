# Installs the fsr CLI on Windows to %LOCALAPPDATA%\fsr and adds it to the user PATH.
# Messages are ASCII so the script also parses under Windows PowerShell 5.1 when run as a file.
# Usage: irm https://raw.githubusercontent.com/mozm-ctrl/fs-reimburse-skill/main/install.ps1 | iex
$ErrorActionPreference = "Stop"
$Repo = "mozm-ctrl/fs-reimburse-skill"
$BinDir = if ($env:FSR_BIN_DIR) { $env:FSR_BIN_DIR } else { "$env:LOCALAPPDATA\fsr" }
$Base = if ($env:FSR_VERSION) { "https://github.com/$Repo/releases/download/$env:FSR_VERSION" } else { "https://github.com/$Repo/releases/latest/download" }
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
$Name = "fsr-windows-$Arch.exe"

$Tmp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid())
New-Item -ItemType Directory -Force -Path $Tmp, $BinDir | Out-Null
try {
  Invoke-WebRequest -UseBasicParsing -Uri "$Base/$Name" -OutFile "$Tmp\$Name"
  Invoke-WebRequest -UseBasicParsing -Uri "$Base/sha256sums.txt" -OutFile "$Tmp\sha256sums.txt"
  $Want = (Get-Content "$Tmp\sha256sums.txt" | Where-Object { ($_ -split "\s+")[1] -eq $Name } | ForEach-Object { ($_ -split "\s+")[0] })
  $Got = (Get-FileHash -Algorithm SHA256 "$Tmp\$Name").Hash.ToLower()
  if (-not $Want -or $Want -ne $Got) { throw "Checksum mismatch for $Name, please retry" }
  Move-Item -Force "$Tmp\$Name" "$BinDir\fsr.exe"
} finally {
  Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}

$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($UserPath -split ";") -notcontains $BinDir) {
  [Environment]::SetEnvironmentVariable("Path", "$UserPath;$BinDir", "User")
  Write-Host "Added $BinDir to your user PATH; open a new terminal for it to take effect"
}
$env:Path = "$env:Path;$BinDir"
Write-Host "Installed $(& "$BinDir\fsr.exe" version) to $BinDir\fsr.exe"
