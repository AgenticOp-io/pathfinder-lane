# Update-Install.ps1 — rebuild and reinstall PathfinderSSH MSP (preserves ~/.pathfinderssh data)
#
#   .\Update-Install.ps1
#   .\Update-Install.ps1 -Setup solo

param(
    [string]$Setup = ""
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

Write-Host ">> Rebuilding installer bundle..."
& "$PSScriptRoot\build-windows.ps1" -Targets "pathfinder,pfseed,pfinstall,pfenroll"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$installer = Join-Path $PSScriptRoot "dist\windows\pfinstall.exe"
$args = @("-install")
if ($Setup -ne "") {
    $args += "-setup"
    $args += $Setup
}
& $installer @args
exit $LASTEXITCODE
