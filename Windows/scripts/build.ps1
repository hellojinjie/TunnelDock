$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

go test ./...
go vet ./...

go run github.com/tc-hib/go-winres@v0.3.3 make `
  --in winres/winres.json `
  --arch amd64 `
  --out cmd/tunneldock/rsrc

$dist = Join-Path $projectRoot "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null
go build -trimpath -ldflags="-H=windowsgui -s -w" -o (Join-Path $dist "TunnelDock.exe") ./cmd/tunneldock
