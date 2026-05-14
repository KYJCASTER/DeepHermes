param(
    [string]$Version = "1.1.0",
    [string]$Platform = "windows/amd64",
    [switch]$SkipFrontend
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Cache = Join-Path $Root ".gocache-codex"

if (-not (Test-Path $Cache)) {
    New-Item -ItemType Directory -Path $Cache | Out-Null
}

$Commit = "dev"
try {
    $Commit = (git rev-parse --short HEAD).Trim()
}
catch {
    $Commit = "dev"
}

$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LdFlags = "-w -s -H windowsgui -X github.com/ad201/deephermes/app.Version=$Version -X github.com/ad201/deephermes/app.BuildCommit=$Commit -X github.com/ad201/deephermes/app.BuildDate=$BuildDate"

$env:GOCACHE = $Cache

Push-Location $Root
try {
    $Args = @(
        "build",
        "-platform", $Platform,
        "-clean",
        "-nsis",
        "-tags", "desktop,production",
        "-ldflags", $LdFlags
    )
    if ($SkipFrontend) {
        $Args += "-s"
    }
    & wails @Args
}
finally {
    Pop-Location
}

Write-Host "Installer build finished. Check build\bin for the exe and installer output."
