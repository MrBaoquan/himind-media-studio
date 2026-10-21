param(
  [string]$Version = '',
  [string]$HostUrl = 'http://127.0.0.1:18082'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$pluginPath = Join-Path $root 'plugins\com.himind.media-workbench\media-workbench'
$manifestPath = Join-Path $pluginPath 'plugin.json'
if (-not $Version) {
  $Version = (Get-Content $manifestPath -Raw | ConvertFrom-Json).version
}
$output = Join-Path $root "dist\plugins\com.himind.media-workbench-$Version.hmpkg"
New-Item -ItemType Directory -Force -Path (Split-Path $output) | Out-Null

function Invoke-Capability([string]$Id, $InputObject) {
  $body = @{ capability_id = $Id; input = $InputObject } | ConvertTo-Json -Depth 30 -Compress
  $response = Invoke-RestMethod -Method Post -Uri "$HostUrl/capabilities/invoke" -ContentType 'application/json' -Body $body
  if (-not $response.ok) { throw "$Id failed: $($response.error)" }
  return $response.result
}

Invoke-Capability 'extension.plugin.validate' @{ workspace_root = $root; path = $pluginPath } | Out-Null
$package = Invoke-Capability 'extension.plugin.package' @{ workspace_root = $root; path = $pluginPath; output = $output }
$saved = Invoke-Capability 'extension.plugin.candidate.save' @{ package_path = $output }
$tested = Invoke-Capability 'extension.plugin.candidate.test' @{ id = 'com.himind.media-workbench'; version = $Version }
$confirmed = Invoke-Capability 'extension.plugin.candidate.confirm' @{ id = 'com.himind.media-workbench'; version = $Version }

[pscustomobject]@{
  version = $Version
  package = $package.output
  sha256 = $saved.candidate_sha256
  tested_at = $tested.tested_at
  confirmed_at = $confirmed.confirmed_at
  development_path = $confirmed.development_path
} | ConvertTo-Json -Depth 6
