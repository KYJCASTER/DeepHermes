param(
    [string]$Output = "build\bin\DeepHermes.exe"
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Frontend = Join-Path $Root "frontend"
$Cache = Join-Path $Root ".gocache-codex"

if (-not (Test-Path $Cache)) {
    New-Item -ItemType Directory -Path $Cache | Out-Null
}

Push-Location $Frontend
try {
    node ".\node_modules\typescript\bin\tsc"
    node ".\node_modules\vite\bin\vite.js" build
}
finally {
    Pop-Location
}

$env:GOCACHE = $Cache
go build -buildvcs=false -tags "desktop,production" -ldflags "-w -s -H windowsgui" -o $Output .

Write-Host "Built $Output"
