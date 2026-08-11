param(
    [Parameter(Mandatory=$true)][string]$Path,
    [string]$Version = "1.2.20",
    [switch]$RequireSignature,
    [switch]$RequireTimestamp
)

$ErrorActionPreference = "Stop"
if (-not (Test-Path $Path)) { throw "Executavel nao encontrado: $Path" }

$item = Get-Item $Path
$vi = $item.VersionInfo
$checks = @{
    ProductName = "Firebird Toolbox"
    CompanyName = "Marcos De Mori Laiola"
    FileDescription = "Firebird Toolbox - manutenção e recuperação Firebird"
}
foreach ($entry in $checks.GetEnumerator()) {
    if ($vi.($entry.Key) -ne $entry.Value) {
        throw "VersionInfo invalido em $($entry.Key): '$($vi.($entry.Key))'"
    }
}
if (-not $vi.FileVersion.StartsWith($Version)) {
    throw "FileVersion inesperada: $($vi.FileVersion)"
}
if (-not $vi.ProductVersion.StartsWith($Version)) {
    throw "ProductVersion inesperada: $($vi.ProductVersion)"
}

# Confere diretamente o cabeçalho PE. O Toolbox é aplicativo GUI e nunca deve
# ser aceito como IMAGE_SUBSYSTEM_WINDOWS_CUI (3), que abre uma janela de console.
$bytes = [System.IO.File]::ReadAllBytes((Resolve-Path $Path))
if ($bytes.Length -lt 0x100) { throw "Executavel PE pequeno/invalido: $Path" }
$peOffset = [BitConverter]::ToInt32($bytes, 0x3c)
if ($peOffset -lt 0 -or ($peOffset + 24 + 70) -gt $bytes.Length) {
    throw "Cabecalho PE invalido: $Path"
}
$signature = [Text.Encoding]::ASCII.GetString($bytes, $peOffset, 4)
if ($signature -ne "PE`0`0") { throw "Assinatura PE invalida: $Path" }
$optionalHeader = $peOffset + 24
$subsystem = [BitConverter]::ToUInt16($bytes, $optionalHeader + 68)
if ($subsystem -ne 2) {
    throw "Subsystem Windows incorreto: $subsystem. Esperado 2 (Windows GUI); build console nao pode ser distribuido."
}
Write-Host "PE Subsystem: Windows GUI (2)"

$sig = Get-AuthenticodeSignature -FilePath $Path
if ($RequireSignature -and $sig.Status -ne 'Valid') {
    throw "Assinatura Authenticode obrigatoria, status atual: $($sig.Status)"
}
if ($RequireSignature -and -not $sig.SignerCertificate) {
    throw "Assinatura Authenticode sem certificado de signatario."
}
if ($RequireTimestamp -and -not $sig.TimeStamperCertificate) {
    throw "Timestamp RFC3161 obrigatorio, mas nao foi encontrado."
}

Write-Host "VersionInfo OK: $($vi.ProductName) $($vi.ProductVersion) | $($vi.CompanyName)"
Write-Host "Authenticode: $($sig.Status)"
if ($sig.SignerCertificate) {
    Write-Host "Signer: $($sig.SignerCertificate.Subject)"
}
if ($sig.TimeStamperCertificate) {
    Write-Host "Timestamp: $($sig.TimeStamperCertificate.Subject)"
}
