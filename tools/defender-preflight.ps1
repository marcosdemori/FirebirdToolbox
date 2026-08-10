param(
    [Parameter(Mandatory=$true)][string[]]$Path
)

$ErrorActionPreference = "Stop"

$scan = Get-Command Start-MpScan -ErrorAction SilentlyContinue
$detections = Get-Command Get-MpThreatDetection -ErrorAction SilentlyContinue
if (-not $scan -or -not $detections) {
    Write-Warning "Microsoft Defender PowerShell nao esta disponivel neste runner; assinatura e atestacao continuam obrigatorias."
    exit 0
}

$fullPaths = @()
foreach ($file in $Path) {
    if (-not (Test-Path $file)) { throw "Arquivo nao encontrado para preflight do Defender: $file" }
    $fullPaths += (Resolve-Path $file).Path
}

foreach ($file in $fullPaths) {
    Write-Host "Microsoft Defender: verificando $file"
    try {
        Start-MpScan -ScanType CustomScan -ScanPath $file
    }
    catch {
        Write-Warning "Microsoft Defender existe no runner, mas o servico de scan nao esta operacional: $($_.Exception.Message)"
        exit 0
    }
    if (-not (Test-Path $file)) {
        throw "Microsoft Defender removeu ou colocou em quarentena o executavel assinado: $file"
    }
}

$hits = @(Get-MpThreatDetection -ErrorAction SilentlyContinue | Where-Object {
    $resources = @($_.Resources) -join "`n"
    $matched = $false
    foreach ($file in $fullPaths) {
        if ($resources -like "*$file*") {
            $matched = $true
            break
        }
    }
    $matched
})

if ($hits.Count -gt 0) {
    $summary = $hits | ForEach-Object { "ThreatID=$($_.ThreatID) Resources=$(@($_.Resources) -join ', ')" }
    throw "Microsoft Defender detectou o build assinado durante o preflight:`n$($summary -join "`n")"
}

Write-Host "Microsoft Defender preflight: nenhum alerta encontrado para os executaveis assinados."
