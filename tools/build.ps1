param(
    [string]$Version = "1.2.10",
    [string]$SigningPfx = "",
    [string]$SigningPassword = ""
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Dist = Join-Path $Root "dist"

Push-Location $Root
try {
    if (Test-Path $Dist) { Remove-Item $Dist -Recurse -Force }
    New-Item -ItemType Directory -Path $Dist | Out-Null

    Write-Host "[1/7] gofmt..."
    $badFormat = gofmt -l .
    if ($badFormat) {
        Write-Host $badFormat
        throw "Existem arquivos Go fora do padrao gofmt."
    }

    Write-Host "[2/7] Recursos Windows..."
    & (Join-Path $PSScriptRoot "prepare-resources.ps1") -Version $Version

    Write-Host "[3/7] go vet (Windows x64)..."
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go vet ./...

    Write-Host "[4/7] Compilando testes Windows x64/x86..."
    go test -c -o (Join-Path $Dist "FirebirdToolbox_tests_x64.exe") .
    $env:GOARCH = "386"
    go test -c -o (Join-Path $Dist "FirebirdToolbox_tests_x86.exe") .

    Write-Host "[5/7] Compilando aplicativo Windows x64/x86 com metadados..."
    $ld = "-H=windowsgui -X main.appVersion=$Version"
    $env:GOARCH = "amd64"
    go build -trimpath -buildvcs=false -ldflags=$ld -o (Join-Path $Dist "FirebirdToolbox_v$Version.exe") .
    $env:GOARCH = "386"
    go build -trimpath -buildvcs=false -ldflags=$ld -o (Join-Path $Dist "FirebirdToolbox_v${Version}_x86.exe") .

    $apps = @(
        (Join-Path $Dist "FirebirdToolbox_v$Version.exe"),
        (Join-Path $Dist "FirebirdToolbox_v${Version}_x86.exe")
    )
    foreach ($app in $apps) {
        & (Join-Path $PSScriptRoot "verify-windows-build.ps1") -Path $app -Version $Version
    }

    Write-Host "[6/7] Assinatura Authenticode..."
    if ($SigningPfx -and $SigningPassword) {
        & (Join-Path $PSScriptRoot "sign-windows.ps1") -Path $apps -PfxPath $SigningPfx -Password $SigningPassword
        foreach ($app in $apps) {
            & (Join-Path $PSScriptRoot "verify-windows-build.ps1") -Path $app -Version $Version -RequireSignature
        }
    } else {
        Write-Host "Assinatura nao configurada. Os EXEs terao VersionInfo/manifesto, mas permanecerao sem Authenticode."
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
