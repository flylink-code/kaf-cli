param(
    [string]$Version = "dev"
)
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$env:GOPROXY = "https://goproxy.cn,direct"
$ldflags = "-s -w -X main.version=$Version"

$outAmd64 = Join-Path $Root "build\windows-amd64"
$out386 = Join-Path $Root "build\windows-386"
New-Item -ItemType Directory -Force -Path $outAmd64, $out386 | Out-Null

$built = @()
$failed = @()
$warnings = @()

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

function Warn-Build {
    param([string]$Message)
    Write-Host "   skipped: $Message" -ForegroundColor Yellow
    $script:warnings += $Message
}

function Copy-BuildArtifact {
    param(
        [string]$SourcePath,
        [string]$TargetPath
    )

    try {
        Copy-Item $SourcePath $TargetPath -Force
        return $TargetPath
    } catch {
        $message = $_.Exception.Message
        if ($message -match "because it is being used by another process") {
            $fallbackPath = [System.IO.Path]::ChangeExtension($TargetPath, ".new.exe")
            Copy-Item $SourcePath $fallbackPath -Force
            Warn-Build "$TargetPath is in use, wrote updated artifact to $fallbackPath"
            return $fallbackPath
        }
        throw
    }
}

Build-Bin (Join-Path $outAmd64 "kaf-cli.exe") "./cmd"

$wailsProject = Join-Path $Root "cmd\gui-wails"
$wailsOut = Join-Path $outAmd64 "kaf-cli-wails.exe"
if (Test-Path $wailsProject) {
    $wails = Get-Command wails -ErrorAction SilentlyContinue
    if (-not $wails) {
        Write-Host ">> building $wailsOut ..."
        Warn-Build "Wails CLI not found, skip kaf-cli-wails.exe"
    } else {
        Write-Host ">> building $wailsOut ..."
        Push-Location $wailsProject
        try {
            & $wails.Source build -m -tags wailsgui
            $srcExe = Join-Path $wailsProject "build\bin\kaf-cli-wails.exe"
            if ($LASTEXITCODE -ne 0 -or -not (Test-Path $srcExe)) {
                Write-Host "   FAILED (exit $LASTEXITCODE)" -ForegroundColor Red
                $script:failed += $wailsOut
            } else {
                $copiedTo = Copy-BuildArtifact $srcExe $wailsOut
                Write-Host "   ok -> $copiedTo" -ForegroundColor Green
                $script:built += $copiedTo
            }
        } finally {
            Pop-Location
        }
    }
}

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
    if ($warnings.Count -gt 0) {
        Write-Host "  warnings:"
        foreach ($warning in $warnings) {
            Write-Host "    - $warning"
        }
    }
    exit 1
}

Write-Host "build done!" -ForegroundColor Green
Write-Host "  $($built -join "`n  ")"
if ($warnings.Count -gt 0) {
    Write-Host ""
    Write-Host "warnings:" -ForegroundColor Yellow
    foreach ($warning in $warnings) {
        Write-Host "  - $warning"
    }
}
