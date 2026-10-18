[CmdletBinding()]
param(
    [string]$WorkspaceRoot = (Split-Path -Parent $PSScriptRoot),
    [string]$PackageDir = 'dist/asset-packages',
    [switch]$Published
)

$ErrorActionPreference = 'Stop'
$WorkspaceRoot = (Resolve-Path -LiteralPath $WorkspaceRoot).Path
$catalogPath = Join-Path $WorkspaceRoot 'catalog/asset-packs.json'
if (-not (Test-Path -LiteralPath $catalogPath)) { throw "找不到资产目录：$catalogPath" }

& (Join-Path $PSScriptRoot 'pack-asset-packs.ps1') -WorkspaceRoot $WorkspaceRoot -OutputDir $PackageDir | Out-Host

$catalog = Get-Content -Raw -LiteralPath $catalogPath | ConvertFrom-Json
$known = @{}
foreach ($asset in $catalog.assets) { $known[$asset.id] = $true }
$styleRoot = Join-Path $WorkspaceRoot 'style-packs'
Get-ChildItem -LiteralPath $styleRoot -Directory | ForEach-Object {
    $manifestPath = Join-Path $_.FullName 'asset-pack.json'
    if (-not (Test-Path -LiteralPath $manifestPath)) { return }
    $manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
    if ($known.ContainsKey($manifest.id)) { return }
    $catalog.assets += [pscustomobject]@{
        id = $manifest.id
        name = $manifest.name
        version = $manifest.version
        file_name = ''
        file_size = 0
        sha256 = ''
        download_url = ''
        license = $manifest.license
        status = 'prepared'
        tags = @($manifest.tags)
    }
    $known[$manifest.id] = $true
}
$status = if ($Published) { 'published' } else { 'prepared' }
$releasePrefix = if ($catalog.release_prefix) { $catalog.release_prefix } else { 'asset-pack' }
$repository = [string]$catalog.repository
if (-not $repository) { throw '资产目录缺少 repository，无法生成 Release 地址' }

foreach ($asset in $catalog.assets) {
    $fileName = "$($asset.id)-$($asset.version).hmpack"
    $packagePath = Join-Path $WorkspaceRoot (Join-Path $PackageDir $fileName)
    if (-not (Test-Path -LiteralPath $packagePath)) { throw "缺少资产包：$packagePath" }
    $info = Get-Item -LiteralPath $packagePath
    $asset.file_name = $fileName
    $asset.file_size = $info.Length
    $asset.sha256 = (Get-FileHash -LiteralPath $packagePath -Algorithm SHA256).Hash.ToLowerInvariant()
    $asset.status = $status
    $tag = "$releasePrefix-$($asset.id)-v$($asset.version)"
    $asset.download_url = "$repository/releases/download/$tag/$fileName"
    Write-Host "gh release create $tag `"$packagePath`" --title `"$($asset.name) v$($asset.version)`" --notes `"HiMind Asset Pack $($asset.id) $($asset.version)`""
}

$catalog | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $catalogPath -Encoding utf8
Write-Host "catalog updated: $catalogPath (status=$status)" -ForegroundColor Green
