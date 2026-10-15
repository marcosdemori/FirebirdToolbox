param(
    [Parameter(Mandatory=$true)][string]$Path,
    [string]$Version = "1.2.32",
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
    FileDescription = "Firebird Toolbox - manutenção, conversão e recuperação Firebird"
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

# Confere que o binario realmente carrega os recursos que o usuario ve no
# Explorer/atalhos. Em especial, RT_GROUP_ICON (14) evita distribuir novamente
# um EXE com o icone generico do Windows. RT_MANIFEST (24) preserva Common
# Controls v6 e a declaracao de DPI usada pela interface Win32.
$numberOfSections = [BitConverter]::ToUInt16($bytes, $peOffset + 6)
$sizeOfOptionalHeader = [BitConverter]::ToUInt16($bytes, $peOffset + 20)
$magic = [BitConverter]::ToUInt16($bytes, $optionalHeader)
if ($magic -eq 0x10b) {
    $dataDirectory = $optionalHeader + 96
} elseif ($magic -eq 0x20b) {
    $dataDirectory = $optionalHeader + 112
} else {
    throw "Optional Header PE desconhecido: 0x$($magic.ToString('x'))"
}
$resourceRva = [BitConverter]::ToUInt32($bytes, $dataDirectory + 16)
$resourceSize = [BitConverter]::ToUInt32($bytes, $dataDirectory + 20)
if ($resourceRva -eq 0 -or $resourceSize -eq 0) {
    throw "Diretorio de recursos Windows ausente. O EXE ficaria sem icone/manifesto incorporado."
}

$sectionTable = $optionalHeader + $sizeOfOptionalHeader
$resourceOffset = $null
for ($i = 0; $i -lt $numberOfSections; $i++) {
    $section = $sectionTable + ($i * 40)
    $virtualSize = [BitConverter]::ToUInt32($bytes, $section + 8)
    $virtualAddress = [BitConverter]::ToUInt32($bytes, $section + 12)
    $rawSize = [BitConverter]::ToUInt32($bytes, $section + 16)
    $rawPointer = [BitConverter]::ToUInt32($bytes, $section + 20)
    $span = [Math]::Max([uint64]$virtualSize, [uint64]$rawSize)
    if ([uint64]$resourceRva -ge [uint64]$virtualAddress -and [uint64]$resourceRva -lt ([uint64]$virtualAddress + $span)) {
        $resourceOffset = [int]([uint64]$rawPointer + ([uint64]$resourceRva - [uint64]$virtualAddress))
        break
    }
}
if ($null -eq $resourceOffset -or ($resourceOffset + 16) -gt $bytes.Length) {
    throw "Nao foi possivel localizar a raiz do diretorio de recursos Windows."
}

$namedTypes = [BitConverter]::ToUInt16($bytes, $resourceOffset + 12)
$idTypes = [BitConverter]::ToUInt16($bytes, $resourceOffset + 14)
$typeCount = [int]$namedTypes + [int]$idTypes
$resourceTypes = @()
for ($i = 0; $i -lt $typeCount; $i++) {
    $entryOffset = $resourceOffset + 16 + ($i * 8)
    if (($entryOffset + 8) -gt $bytes.Length) { throw "Diretorio de recursos truncado." }
    $nameOrId = [BitConverter]::ToUInt32($bytes, $entryOffset)
    if (($nameOrId -band 0x80000000) -eq 0) {
        $resourceTypes += [int]($nameOrId -band 0xffff)
    }
}
if ($resourceTypes -notcontains 14) {
    throw "RT_GROUP_ICON (14) ausente. O executavel seria exibido com icone generico no Windows."
}
if ($resourceTypes -notcontains 24) {
    throw "RT_MANIFEST (24) ausente. O manifesto Windows nao foi incorporado ao executavel."
}
Write-Host "Recursos Windows: RT_GROUP_ICON e RT_MANIFEST OK"

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
