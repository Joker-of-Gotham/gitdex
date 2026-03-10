param(
    [Parameter(Mandatory = $true)]
    [string]$ModulePath
)

$ErrorActionPreference = "Stop"

$Placeholder = "github.com/Joker-of-Gotham/gitdex"
$Utf8NoBom = [System.Text.UTF8Encoding]::new($false)
$AllowedExtensions = @(".go", ".mod", ".md", ".ps1", ".sh", ".txt", ".yaml", ".yml")

if ($ModulePath -notmatch '^github\.com/[^/]+/[^/]+$') {
    throw "Expected a full module path such as github.com/<user-or-org>/gitdex"
}

$targets = @(
    "go.mod",
    "README.md",
    "cmd",
    "internal",
    "test",
    "docs",
    ".github",
    "scripts"
)

$files = New-Object System.Collections.Generic.List[string]

foreach ($target in $targets) {
    if (-not (Test-Path $target)) {
        continue
    }

    $item = Get-Item $target
    if ($item.PSIsContainer) {
        Get-ChildItem $target -Recurse -File | ForEach-Object {
            if ($_.FullName -match '\\(bin|_bmad-output|\.git)\\') {
                return
            }
            if ($AllowedExtensions -notcontains $_.Extension.ToLowerInvariant()) {
                return
            }
            $files.Add($_.FullName)
        }
        continue
    }

    $files.Add($item.FullName)
}

$updated = 0
foreach ($file in $files | Sort-Object -Unique) {
    $content = [System.IO.File]::ReadAllText($file)
    if (-not $content.Contains($Placeholder)) {
        continue
    }

    $next = $content.Replace($Placeholder, $ModulePath)
    [System.IO.File]::WriteAllText($file, $next, $Utf8NoBom)
    Write-Host "Updated $file"
    $updated++
}

if ($updated -eq 0) {
    Write-Host "No placeholder module path found."
    exit 0
}

& gofmt -w cmd internal test
if ($LASTEXITCODE -ne 0) {
    throw "gofmt failed after updating the module path"
}

Write-Host ""
Write-Host "Replaced $Placeholder -> $ModulePath" -ForegroundColor Green
Write-Host "Next: go test ./..." -ForegroundColor Yellow
