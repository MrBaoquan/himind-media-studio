param(
  [Parameter(Mandatory=$true)][string]$Id,
  [Parameter(Mandatory=$true)][string]$Name,
  [string]$Base = 'data-kinetic',
  [string]$Root = (Split-Path -Parent $PSScriptRoot)
)
$ErrorActionPreference = 'Stop'
if ($Id -notmatch '^[a-z0-9][a-z0-9-]{1,63}$') { throw "invalid id: $Id" }
$baseTemplate = Join-Path $Root "style-packs\$Base\template"
$target = Join-Path $Root "style-packs\$Id"
if (-not (Test-Path -LiteralPath $baseTemplate)) { throw "base template not found: $baseTemplate" }
if (Test-Path -LiteralPath $target) { throw "style pack already exists: $target" }
New-Item -ItemType Directory -Force -Path (Join-Path $target 'template'), (Join-Path $target 'preview') | Out-Null
Copy-Item -Path (Join-Path $baseTemplate '*') -Destination (Join-Path $target 'template') -Recurse -Force
$manifestPath = Join-Path $target 'template\style-pack.json'
$manifest = Get-Content -Raw -LiteralPath $manifestPath | ConvertFrom-Json
$manifest.id = $Id
$manifest.name = $Name
$manifest.version = '0.1.0'
$manifest | ConvertTo-Json -Depth 12 | Set-Content -Encoding utf8 -LiteralPath $manifestPath
$assetPack = [ordered]@{
  schema='himind.asset-pack.v1'; id=$Id; version='0.1.0'; name=$Name
  description="Draft style pack: $Name"; license='MIT'; attribution='Local HiMind style pack draft.'
  resources=@(
    [ordered]@{id='template';type='template';path='template';role='render-template';runtime='browser-code';entry='render.mjs';format='directory'},
    [ordered]@{id='preview';type='image';path='preview/sheet.jpg';role='preview';mime='image/jpeg'}
  )
  entrypoints=@([ordered]@{id='render';resource='template';runtime='browser-code';entry='render.mjs';inputs=@('brief.json','data.json');outputs=@('video.mp4','video-web.mp4')})
  provenance=[ordered]@{source='local';commit='draft';license='MIT';adapted_files=@();changes=@('initial scaffold')}
  permissions=[ordered]@{filesystem='workspace';process=@('node','chrome','ffmpeg');network=[ordered]@{mode='none';allow=@()}}
  quality=[ordered]@{smoke='preview/sheet.jpg';determinism=$true}
  consumers=@('com.himind.media-workbench'); tags=@('video','style','draft')
}
$assetPack | ConvertTo-Json -Depth 12 | Set-Content -Encoding utf8 -LiteralPath (Join-Path $target 'asset-pack.json')
"# Attribution`n`nDraft style pack $Id. Replace with upstream source, license, adapted files and changes." | Set-Content -Encoding utf8 -LiteralPath (Join-Path $target 'ATTRIBUTION.md')
Write-Host "created $target" -ForegroundColor Green
Write-Host "next: implement template, preview.review, copy preview/sheet.jpg, asset.validate" -ForegroundColor Cyan
