# Pre-push check: runs tests and linter.
# Usage: .\scripts\pre-push.ps1
# Or configure as git hook (see below).

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "=== pre-push: go test ===" -ForegroundColor Cyan
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "Tests failed. Push aborted." -ForegroundColor Red
    exit 1
}

Write-Host "=== pre-push: golangci-lint ===" -ForegroundColor Cyan
golangci-lint run
if ($LASTEXITCODE -ne 0) {
    Write-Host "Lint failed. Push aborted." -ForegroundColor Red
    exit 1
}

Write-Host "=== pre-push: all checks passed ===" -ForegroundColor Green
