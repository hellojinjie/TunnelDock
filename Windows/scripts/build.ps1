$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

go test ./...
go vet ./...

$dist = Join-Path $projectRoot "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null
go build -trimpath -ldflags="-H=windowsgui -s -w" -o (Join-Path $dist "TunnelDock.exe") ./cmd/tunneldock
