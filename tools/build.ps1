param(
    [string]$Version = "1.2.32",
    [string]$SigningPfx = "",
    [string]$SigningPassword = "",
    [switch]$AllowUnsignedDev
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Dist = Join-Path $Root "dist"

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
    if (-not ($SigningPfx -and $SigningPassword) -and -not $AllowUnsignedDev) {
        throw "Build de distribuicao bloqueado sem assinatura. Use um PFX confiavel ou -AllowUnsignedDev somente para teste local. As Releases oficiais usam Microsoft Artifact Signing no GitHub Actions."
    }

    if (Test-Path $Dist) { Remove-Item $Dist -Recurse -Force }
    New-Item -ItemType Directory -Path $Dist | Out-Null

    Write-Host "[1/7] gofmt..."
    $badFormat = Invoke-NativeCommand -Command "gofmt" -Arguments @("-l", ".") -What "gofmt"
    if ($badFormat) {
        Write-Host $badFormat
        throw "Existem arquivos Go fora do padrao gofmt."
    }

    Write-Host "[2/7] Recursos Windows..."
    & (Join-Path $PSScriptRoot "prepare-resources.ps1") -Version $Version

    Write-Host "[3/7] go vet (Windows x64)..."
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    Invoke-NativeCommand -Command "go" -Arguments @("vet", "./...") -What "go vet"

    Write-Host "[4/7] Compilando testes Windows x64/x86..."
    Invoke-NativeCommand -Command "go" -Arguments @("test", "-c", "-o", (Join-Path $Dist "FirebirdToolbox_tests_x64.exe"), ".") -What "go test -c x64"
    $env:GOARCH = "386"
    Invoke-NativeCommand -Command "go" -Arguments @("test", "-c", "-o", (Join-Path $Dist "FirebirdToolbox_tests_x86.exe"), ".") -What "go test -c x86"

    Write-Host "[5/7] Compilando aplicativo unico Windows x86/x64 com metadados..."
    $ld = "-H=windowsgui -X main.appVersion=$Version"
    $suffix = ""
    if (-not ($SigningPfx -and $SigningPassword)) {
        $suffix = "_UNSIGNED_DEV"
    }

    # Um unico executavel 386 atende Windows x86 e x64 (WOW64).
    $env:GOARCH = "386"
    $universalOut = Join-Path $Dist "FirebirdToolbox_v${Version}_x86_x64${suffix}.exe"
    $goArgs = @("build", "-trimpath", "-buildvcs=false", "-ldflags", $ld, "-o", $universalOut, ".")
    Invoke-NativeCommand -Command "go" -Arguments $goArgs -What "go build x86/x64"

    $apps = @($universalOut)
    foreach ($app in $apps) {
        & (Join-Path $PSScriptRoot "verify-windows-build.ps1") -Path $app -Version $Version
    }

    Write-Host "[6/7] Assinatura Authenticode..."
    if ($SigningPfx -and $SigningPassword) {
        & (Join-Path $PSScriptRoot "sign-windows.ps1") -Path $apps -PfxPath $SigningPfx -Password $SigningPassword
        foreach ($app in $apps) {
            & (Join-Path $PSScriptRoot "verify-windows-build.ps1") -Path $app -Version $Version -RequireSignature -RequireTimestamp
        }
        "SIGNED - Authenticode local verificado." | Set-Content -Encoding ASCII (Join-Path $Dist "SIGNATURE_STATUS.txt")
    } else {
        "UNSIGNED DEV BUILD - NAO DISTRIBUIR. Use a Release oficial assinada pelo Microsoft Artifact Signing." | Set-Content -Encoding ASCII (Join-Path $Dist "SIGNATURE_STATUS.txt")
        Write-Warning "Build local sem assinatura criado somente para desenvolvimento. NAO DISTRIBUIR."
    }

    Write-Host "[7/7] SHA-256..."
    Get-ChildItem $Dist -Filter "FirebirdToolbox_v*.exe" |
        Sort-Object Name |
        ForEach-Object {
            $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
            "$hash  $($_.Name)"
        } | Set-Content -Encoding ASCII (Join-Path $Dist "SHA256.txt")

    Write-Host "Build concluido em: $Dist"
}
finally {
    Get-ChildItem -Path $Root -Filter "rsrc_windows_*.syso" -ErrorAction SilentlyContinue | Remove-Item -Force
    Pop-Location
}
