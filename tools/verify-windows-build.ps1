param(
    [Parameter(Mandatory=$true)][string]$Path,
    [string]$Version = "1.2.14",
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
