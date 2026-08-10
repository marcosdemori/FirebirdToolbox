param(
    [string]$Version = "1.2.12"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Push-Location $Root
try {
    $tool = Get-Command go-winres -ErrorAction SilentlyContinue
    if (-not $tool) {
        Write-Host "Instalando go-winres v0.3.3..."
        go install github.com/tc-hib/go-winres@v0.3.3
        if ($LASTEXITCODE -ne 0) { throw "Falha ao instalar go-winres v0.3.3." }
        $goBin = Join-Path (go env GOPATH) "bin"
        $env:PATH = "$goBin;$env:PATH"
    }

    Get-ChildItem -Path $Root -Filter "rsrc_windows_*.syso" -ErrorAction SilentlyContinue | Remove-Item -Force

    Write-Host "Gerando recursos Windows (icone, manifesto e VersionInfo) para v$Version..."
    go-winres make --file-version "$Version.0" --product-version "$Version.0"
    if ($LASTEXITCODE -ne 0) { throw "go-winres make falhou." }

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
