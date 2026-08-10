param(
    [Parameter(Mandatory=$true)][string[]]$Path,
    [Parameter(Mandatory=$true)][string]$PfxPath,
    [Parameter(Mandatory=$true)][string]$Password,
    [string]$TimestampUrl = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"

$kits = Join-Path ${env:ProgramFiles(x86)} "Windows Kits\10\bin"
$signtool = Get-ChildItem $kits -Filter signtool.exe -Recurse -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match '\\x64\\signtool\.exe$' } |
    Sort-Object FullName -Descending |
    Select-Object -First 1
if (-not $signtool) { throw "signtool.exe nao encontrado no Windows SDK." }
if (-not (Test-Path $PfxPath)) { throw "Certificado PFX nao encontrado: $PfxPath" }

foreach ($file in $Path) {
    if (-not (Test-Path $file)) { throw "Arquivo para assinatura nao encontrado: $file" }
    Write-Host "Assinando $file..."
    & $signtool.FullName sign /fd SHA256 /f $PfxPath /p $Password /tr $TimestampUrl /td SHA256 $file
    if ($LASTEXITCODE -ne 0) { throw "Falha ao assinar $file" }
    & $signtool.FullName verify /pa /v $file
    if ($LASTEXITCODE -ne 0) { throw "Falha ao verificar assinatura de $file" }
}
