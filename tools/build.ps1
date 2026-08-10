$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Dist = Join-Path $Root "dist"

Push-Location $Root
try {
    if (Test-Path $Dist) { Remove-Item $Dist -Recurse -Force }
    New-Item -ItemType Directory -Path $Dist | Out-Null

    Write-Host "[1/5] gofmt..."
    $badFormat = gofmt -l .
    if ($badFormat) {
        Write-Host $badFormat
        throw "Existem arquivos Go fora do padrao gofmt."
    }

    Write-Host "[2/5] go vet (Windows x64)..."
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go vet ./...

    Write-Host "[3/5] Compilando testes Windows x64/x86..."
    go test -c -o (Join-Path $Dist "FirebirdToolbox_tests_x64.exe") .
    $env:GOARCH = "386"
    go test -c -o (Join-Path $Dist "FirebirdToolbox_tests_x86.exe") .

    Write-Host "[4/5] Compilando aplicativo Windows x64/x86..."
    $env:GOARCH = "amd64"
    go build -trimpath -ldflags="-H=windowsgui -s -w" -o (Join-Path $Dist "FirebirdToolbox_x64.exe") .
    $env:GOARCH = "386"
    go build -trimpath -ldflags="-H=windowsgui -s -w" -o (Join-Path $Dist "FirebirdToolbox_x86.exe") .

    Write-Host "[5/5] SHA-256..."
    Get-ChildItem $Dist -Filter "FirebirdToolbox_*.exe" |
        Sort-Object Name |
        ForEach-Object {
            $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
            "$hash  $($_.Name)"
        } | Set-Content -Encoding ASCII (Join-Path $Dist "SHA256.txt")

    Write-Host "Build concluido em: $Dist"
}
finally {
    Pop-Location
}
