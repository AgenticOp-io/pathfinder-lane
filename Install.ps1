# Install.ps1 — PathfinderSSH MSP installer (graphical or command-line)
#
#   .\Install.ps1              # graphical wizard (builds if needed)
#   .\Install.ps1 -Gui         # explicit GUI
#   .\Install.ps1 -Setup solo  # silent install + solo sign-in
#   .\Install.ps1 -Setup o365  # silent install (finish sign-in in app)
#   .\Install.ps1 -Uninstall
#   .\Install.ps1 -Build       # only build dist\windows\*.exe

param(
    [switch]$Gui,
    [string]$Setup = "",
    [switch]$Uninstall,
    [switch]$Build,
    [switch]$Enroll
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$Dist = Join-Path $PSScriptRoot "dist\windows"
$Installer = Join-Path $Dist "pfinstall.exe"
$Pathfinder = Join-Path $Dist "pathfinder.exe"

function Ensure-Built {
    if (-not (Test-Path $Installer) -or -not (Test-Path $Pathfinder)) {
        Write-Host ">> Building installers (dist\windows)..."
        & "$PSScriptRoot\build-windows.ps1" -Targets "pathfinder,pfseed,pfinstall,pfenroll"
        if ($LASTEXITCODE -ne 0) { throw "build failed" }
    }
}

if ($Build) {
    & "$PSScriptRoot\build-windows.ps1" -Targets "pathfinder,pfseed,pfinstall,pfenroll"
    exit $LASTEXITCODE
}

if ($Uninstall) {
    Ensure-Built
    & $Installer -uninstall
    exit $LASTEXITCODE
}

Ensure-Built

if ($Gui -or ($Setup -eq "" -and -not $Enroll)) {
    if ($Setup -ne "") {
        & $Installer -install-gui -setup $Setup
    } else {
        & $Installer -install-gui
    }
    exit $LASTEXITCODE
}

$args = @("-install")
if ($Setup -ne "") { $args += "-setup"; $args += $Setup }
if ($Enroll) { $args += "-enroll" }
& $Installer @args
exit $LASTEXITCODE
