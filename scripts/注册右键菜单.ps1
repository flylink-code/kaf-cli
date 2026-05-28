param( $param1, $param2 )
# 检查并以管理员身份运行PS并带上参数
$currentWi = [Security.Principal.WindowsIdentity]::GetCurrent()
$currentWp = [Security.Principal.WindowsPrincipal]$currentWi
if( -not $currentWp.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))
{
    $boundPara = ($MyInvocation.BoundParameters.Keys | foreach{'-{0} {1}' -f  $_ ,$MyInvocation.BoundParameters[$_]} ) -join ' '
    $currentFile = $MyInvocation.MyCommand.Definition
    $fullPara = $boundPara + ' ' + $args -join ' '
    Start-Process "$psHome\powershell.exe"   -ArgumentList "$currentFile $fullPara"   -verb runas
    return
}

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$repoRoot = Split-Path -Parent $scriptDir

# 优先：与脚本同目录（Release 解压包）；其次：仓库 build 输出
$exeCandidates = @(
    (Join-Path $scriptDir "kaf-cli.exe"),
    (Join-Path $repoRoot "build\windows-amd64\kaf-cli.exe")
)
$exe_path = $exeCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $exe_path) {
    Write-Error "未找到 kaf-cli.exe，请将 exe 放在脚本同目录，或先执行 .\build.ps1"
    pause
    exit 1
}

$ico_path = $exe_path
$exe_cmd = $exe_path + ' "%1"'

New-Item -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\使用kaf-cli转换
New-ItemProperty -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\使用kaf-cli转换 -Name Icon -PropertyType String -Value $ico_path

New-Item -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\使用kaf-cli转换\command
New-ItemProperty -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\使用kaf-cli转换\command -Name "(default)" -PropertyType String -Value $exe_cmd

echo "注册右键菜单成功! 使用: $exe_path"
pause
