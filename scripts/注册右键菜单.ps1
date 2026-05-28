param( $param1, $param2 )
# ��鲢�Թ���Ա��������PS�����ϲ���
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

# ���ȣ���ű�ͬĿ¼��Release ��ѹ��������Σ��ֿ� build ���
$exeCandidates = @(
    (Join-Path $scriptDir "kaf-cli.exe"),
    (Join-Path $repoRoot "build\windows-amd64\kaf-cli.exe")
)
$exe_path = $exeCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $exe_path) {
    Write-Error "δ�ҵ� kaf-cli.exe���뽫 exe ���ڽű�ͬĿ¼������ִ�� .\build.ps1"
    pause
    exit 1
}

$ico_path = $exe_path
$exe_cmd = $exe_path + ' "%1"'

New-Item -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\ʹ��kaf-cliת��
New-ItemProperty -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\ʹ��kaf-cliת�� -Name Icon -PropertyType String -Value $ico_path

New-Item -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\ʹ��kaf-cliת��\command
New-ItemProperty -Force -Path Registry::HKEY_CLASSES_ROOT\txtfile\shell\ʹ��kaf-cliת��\command -Name "(default)" -PropertyType String -Value $exe_cmd

echo "ע���Ҽ��˵��ɹ�! ʹ��: $exe_path"
pause
