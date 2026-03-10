param(
    [ValidateSet("build", "release", "assets", "test", "clean")]
    [string]$Target = "build"
)

$AppName   = "gitdex"
$CmdPath   = "./cmd/gitdex"
$BinDir    = "bin"
$DistDir   = "dist"
$Version   = & git describe --tags --always --dirty 2>$null
if (-not $Version) { $Version = "dev" }

$GOOS   = & go env GOOS
$GOARCH = & go env GOARCH
$Ext    = if ($GOOS -eq "windows") { ".exe" } else { "" }

$DevBin     = "$BinDir/$AppName$Ext"
$ReleaseBin = "$BinDir/$AppName-$GOOS-$GOARCH$Ext"
$LdFlags    = "-s -w -X main.version=$Version"
$ReleaseTargets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Name = "gitdex-windows-amd64.exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; Name = "gitdex-windows-arm64.exe" },
    @{ GOOS = "linux"; GOARCH = "amd64"; Name = "gitdex-linux-amd64" },
    @{ GOOS = "linux"; GOARCH = "arm64"; Name = "gitdex-linux-arm64" },
    @{ GOOS = "darwin"; GOARCH = "amd64"; Name = "gitdex-macos-amd64" },
    @{ GOOS = "darwin"; GOARCH = "arm64"; Name = "gitdex-macos-arm64" }
)

switch ($Target) {
    "build" {
        if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }
        & go build -ldflags $LdFlags -o $DevBin $CmdPath
        if ($LASTEXITCODE -eq 0) { Write-Host "Built: $DevBin" -ForegroundColor Green }
        else { Write-Host "Build failed" -ForegroundColor Red; exit 1 }
    }
    "release" {
        if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }
        $env:CGO_ENABLED = "0"
        & go build -ldflags $LdFlags -o $ReleaseBin $CmdPath
        if ($LASTEXITCODE -eq 0) { Write-Host "Built: $ReleaseBin" -ForegroundColor Green }
        else { Write-Host "Build failed" -ForegroundColor Red; exit 1 }
    }
    "assets" {
        if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
        New-Item -ItemType Directory -Path $DistDir -Force | Out-Null
        $env:CGO_ENABLED = "0"

        foreach ($targetSpec in $ReleaseTargets) {
            $env:GOOS = $targetSpec.GOOS
            $env:GOARCH = $targetSpec.GOARCH
            $outPath = Join-Path $DistDir $targetSpec.Name
            & go build -trimpath -ldflags $LdFlags -o $outPath $CmdPath
            if ($LASTEXITCODE -ne 0) {
                Write-Host "Build failed for $($targetSpec.Name)" -ForegroundColor Red
                exit 1
            }
            Write-Host "Built: $outPath" -ForegroundColor Green
        }

        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

        & git rev-parse --is-inside-work-tree 2>$null | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $sourceZip = Join-Path $DistDir "$AppName-source.zip"
            & git archive --format=zip --output $sourceZip HEAD
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Built: $sourceZip" -ForegroundColor Green
            }
        }

        $checksums = Get-ChildItem $DistDir -File "$AppName*" | Sort-Object Name | Get-FileHash -Algorithm SHA256 |
            ForEach-Object { "{0}  {1}" -f $_.Hash.ToLowerInvariant(), $_.Path.Substring((Resolve-Path $DistDir).Path.Length + 1) }
        Set-Content -Path (Join-Path $DistDir "checksums.txt") -Value $checksums
        Write-Host "Built: $(Join-Path $DistDir 'checksums.txt')" -ForegroundColor Green
    }
    "test" {
        & go test ./...
    }
    "clean" {
        if (Test-Path $BinDir) { Remove-Item -Recurse -Force $BinDir }
        Write-Host "Cleaned $BinDir" -ForegroundColor Green
    }
}
