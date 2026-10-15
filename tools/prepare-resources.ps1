param(
    [string]$Version = "1.2.32"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

# O Windows PowerShell 5.1 transforma a saida de stderr de comandos nativos em
# NativeCommandError terminante quando $ErrorActionPreference e "Stop", mesmo
# quando o comando termina com exit code 0 (por exemplo "go: downloading ...").
# Executa o comando nativo com o preference relaxado, preserva o "Stop" para o
# restante do script e decide o sucesso apenas pelo $LASTEXITCODE.
function Invoke-NativeCommand {
    param(
        [Parameter(Mandatory = $true)][string]$Command,
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$What
    )
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        & $Command @Arguments
    }
    finally {
        $ErrorActionPreference = $previous
    }
    if ($LASTEXITCODE -ne 0) { throw "$What falhou com codigo $LASTEXITCODE." }
}

Push-Location $Root
try {
    $tool = Get-Command go-winres -ErrorAction SilentlyContinue
    if (-not $tool) {
        Write-Host "Instalando go-winres v0.3.3..."
        Invoke-NativeCommand -Command "go" -Arguments @("install", "github.com/tc-hib/go-winres@v0.3.3") -What "go install go-winres v0.3.3"
        $goBin = Join-Path (Invoke-NativeCommand -Command "go" -Arguments @("env", "GOPATH") -What "go env GOPATH") "bin"
        $env:PATH = "$goBin;$env:PATH"
    }

    Get-ChildItem -Path $Root -Filter "rsrc_windows_*.syso" -ErrorAction SilentlyContinue | Remove-Item -Force

    Write-Host "Gerando recursos Windows (icone, manifesto e VersionInfo) para v$Version..."
    Invoke-NativeCommand -Command "go-winres" -Arguments @("make", "--file-version", "$Version.0", "--product-version", "$Version.0") -What "go-winres make"

    $expected = @("rsrc_windows_amd64.syso", "rsrc_windows_386.syso")
    foreach ($name in $expected) {
        if (-not (Test-Path (Join-Path $Root $name))) {
            throw "Recurso Windows nao gerado: $name"
        }
    }
}
finally {
    Pop-Location
}
