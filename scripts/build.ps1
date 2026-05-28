$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$env:GOPROXY = "https://goproxy.cn,direct"
$ldflags = "-s -w -X main.version=dev"

$outAmd64 = Join-Path $Root "build\windows-amd64"
$out386 = Join-Path $Root "build\windows-386"
New-Item -ItemType Directory -Force -Path $outAmd64, $out386 | Out-Null

$built = @()
$failed = @()

function Build-Bin {
    param(
        [string]$OutPath,
        [string]$Package,
        [string]$ExtraLdflags = ""
    )

    $flags = if ($ExtraLdflags) { "$ldflags $ExtraLdflags" } else { $ldflags }
    Write-Host ">> building $OutPath ..."

    & go build -ldflags $flags -o $OutPath $Package
    if ($LASTEXITCODE -ne 0) {
        Write-Host "   FAILED (exit $LASTEXITCODE)" -ForegroundColor Red
        $script:failed += $OutPath
        return
    }

    Write-Host "   ok" -ForegroundColor Green
    $script:built += $OutPath
}

Build-Bin (Join-Path $outAmd64 "kaf-cli.exe") "./cmd"
Build-Bin (Join-Path $outAmd64 "kaf-cli-gui.exe") "./cmd/gui" "-H windowsgui"

$env:GOARCH = "386"
try {
    Build-Bin (Join-Path $out386 "kaf-cli.exe") "./cmd"
} finally {
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
}

Write-Host ""
if ($failed.Count -gt 0) {
    Write-Host "build finished with errors:" -ForegroundColor Red
    Write-Host "  failed: $($failed -join ', ')"
    if ($built.Count -gt 0) {
        Write-Host "  built:  $($built -join ', ')"
    }
    exit 1
}

Write-Host "build done!" -ForegroundColor Green
Write-Host "  $($built -join "`n  ")"
