param(
    [string]$Output = "build\bin\DeepHermes.exe",
    [string]$Version = "1.1.0"
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
$Commit = "dev"
try {
    $Commit = (git rev-parse --short HEAD).Trim()
}
catch {
    $Commit = "dev"
}
$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LdFlags = "-w -s -H windowsgui -X github.com/ad201/deephermes/app.Version=$Version -X github.com/ad201/deephermes/app.BuildCommit=$Commit -X github.com/ad201/deephermes/app.BuildDate=$BuildDate"
go build -buildvcs=false -tags "desktop,production" -ldflags $LdFlags -o $Output .

Write-Host "Built $Output"
