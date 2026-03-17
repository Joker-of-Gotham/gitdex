param(
    [switch]$SkipNetwork
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "== GitDex V3 Cutover Preflight =="

$status = & git status --porcelain
if ($LASTEXITCODE -ne 0) {
    throw "git status failed"
}
if ($status) {
    throw "Working tree is not clean. Commit or stash changes before cutover."
}

if (-not $SkipNetwork) {
    Write-Host "[network] git fetch --all --prune"
    & git fetch --all --prune
    if ($LASTEXITCODE -ne 0) {
        throw "git fetch --all --prune failed"
    }
}

Write-Host "[1/6] go vet ./..."
& go vet ./...
if ($LASTEXITCODE -ne 0) {
    throw "go vet failed"
}

Write-Host "[2/6] go test ./... -count=1"
& go test ./... -count=1
if ($LASTEXITCODE -ne 0) {
    throw "go test ./... failed"
}

Write-Host "[3/6] go test regression packages"
& go test ./internal/executor ./internal/llmfactory -run Regression -count=1
if ($LASTEXITCODE -ne 0) {
    throw "regression tests failed"
}

Write-Host "[4/6] go build ./..."
& go build ./...
if ($LASTEXITCODE -ne 0) {
    throw "go build ./... failed"
}

Write-Host "[5/6] config diagnostics smoke"
& go run ./cmd/gitdex config lint | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "gitdex config lint failed"
}
& go run ./cmd/gitdex config explain | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "gitdex config explain failed"
}
& go run ./cmd/gitdex config source | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "gitdex config source failed"
}

Write-Host "[6/6] preflight checks passed" -ForegroundColor Green
