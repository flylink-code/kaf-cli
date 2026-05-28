$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$env:GOPROXY = "https://goproxy.cn,direct"
$ldflags = "-s -w -X main.version=dev"

$built = @()
$failed = @()

function Build-Bin {
    param(
        [string]$Name,
        [string]$Package,
        [string]$ExtraLdflags = ""
    )

    $flags = if ($ExtraLdflags) { "$ldflags $ExtraLdflags" } else { $ldflags }
    Write-Host ">> building $Name ..."

    & go build -ldflags $flags -o $Name $Package
    if ($LASTEXITCODE -ne 0) {
        Write-Host "   FAILED (exit $LASTEXITCODE)" -ForegroundColor Red
        $script:failed += $Name
        return
    }

    Write-Host "   ok" -ForegroundColor Green
    $script:built += $Name
}

Build-Bin "kaf-cli.exe" "./cmd"
Build-Bin "kaf-cli-gui.exe" "./cmd/gui" "-H windowsgui"

$env:GOARCH = "386"
try {
    Build-Bin "kaf-cli_32.exe" "./cmd"
} finally {
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
}

# windigo 不支持 32 位，GUI 仅提供 amd64 版本

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
Write-Host "  $($built -join ', ')"
