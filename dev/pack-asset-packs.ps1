[CmdletBinding()]
param(
    [string]$WorkspaceRoot = (Split-Path -Parent $PSScriptRoot),
    [string]$Binary = '',
    [string]$OutputDir = 'dist/asset-packages',
    [string[]]$Id = @()
)

$ErrorActionPreference = 'Stop'
$WorkspaceRoot = (Resolve-Path -LiteralPath $WorkspaceRoot).Path
if (-not $Binary) {
    $Binary = Join-Path $WorkspaceRoot 'plugins/com.himind.media-workbench/media-workbench/bin/media-workbench.exe'
}
$Binary = (Resolve-Path -LiteralPath $Binary).Path
$packs = Get-ChildItem -LiteralPath (Join-Path $WorkspaceRoot 'style-packs') -Directory | ForEach-Object {
    $manifest = Join-Path $_.FullName 'asset-pack.json'
    if (-not (Test-Path -LiteralPath $manifest)) { return }
    $json = Get-Content -Raw -LiteralPath $manifest | ConvertFrom-Json
    if ($Id.Count -gt 0 -and $Id -notcontains $json.id) { return }
    [pscustomobject]@{ Id = $json.id; Version = $json.version }
}
if (-not $packs) { throw '没有找到可打包的 style-packs/<id>/asset-pack.json' }

$rows = foreach ($pack in $packs) {
    $target = Join-Path $OutputDir ($pack.Id + '-' + $pack.Version + '.hmpack')
    $params = @{
        workspace_root = $WorkspaceRoot
        pack_id = $pack.Id
        version = $pack.Version
        output = $target
    }
    $request = @{ jsonrpc = '2.0'; id = 1; method = 'media.workbench.asset.pack'; params = $params } | ConvertTo-Json -Compress -Depth 8
    $raw = $request | & $Binary
    $result = $raw | ConvertFrom-Json
    if ($result.error) { throw ($pack.Id + ': ' + $result.error.message) }
    [pscustomobject]@{
        Id = $result.result.pack_id
        Version = $result.result.version
        Files = $result.result.files
        Size = $result.result.size
        SHA256 = $result.result.package_sha256
        Path = $result.result.package_path
    }
}
$rows | Format-Table -AutoSize
