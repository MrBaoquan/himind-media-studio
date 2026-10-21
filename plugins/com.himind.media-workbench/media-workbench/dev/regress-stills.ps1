#requires -Version 7
<#
.SYNOPSIS
  镜头静帧回归：渲染指定帧并与基线做 SSIM 比对。

.DESCRIPTION
  每次改 Shot 后运行本脚本，确认画面没有意外漂移。
  基线存放在 catalog/fixtures/<id>/frame-XXXX.png。

  - 默认：渲染 → 缩放 → SSIM 比对，低于 manifest.ssimFloor 即失败
  - -Update：重新生成基线（改完 Shot 且确认新画面正确后使用）

.PARAMETER Runtime
  Remotion 运行时目录（含 node_modules/@remotion/cli）。通常是
  <workspace>/.himind-media/runtime/remotion-4.0.521

.EXAMPLE
  pwsh -File dev/regress-stills.ps1 -Runtime "$env:TEMP\ws\.himind-media\runtime\remotion-4.0.521"

.EXAMPLE
  pwsh -File dev/regress-stills.ps1 -Runtime <...> -Update
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string] $Runtime,
    # 可选：从插件源码同步渲染器到运行时。不传则假定运行时已是最新。
    [string] $SyncFrom = '',
    # 可选：只跑指定 fixture（按 id），用于新增镜头时定向重建基线。
    [string[]] $FixtureId = @(),
    [switch] $Update
)

$ErrorActionPreference = 'Stop'
$pluginRoot = Split-Path -Parent $PSScriptRoot
$fixtures = Join-Path $pluginRoot 'catalog\fixtures'
$manifestPath = Join-Path $fixtures 'manifest.json'

if (-not (Test-Path -LiteralPath $manifestPath)) { throw "找不到 fixture 清单：$manifestPath" }
if (-not (Test-Path -LiteralPath $Runtime)) { throw "找不到 Remotion 运行时：$Runtime" }

if ($SyncFrom) {
    $rendererSource = Join-Path $SyncFrom 'renderer\remotion'
    if (-not (Test-Path -LiteralPath $rendererSource)) { throw "SyncFrom 下找不到 renderer/remotion：$rendererSource" }
    foreach ($name in @('index.jsx', 'player.jsx', 'package.json')) {
        Copy-Item -LiteralPath (Join-Path $rendererSource $name) -Destination (Join-Path $Runtime $name) -Force
    }
    foreach ($sub in @('foundation', 'shots', 'techniques', 'compositions', 'public')) {
        $sourceDir = Join-Path $rendererSource $sub
        if (-not (Test-Path -LiteralPath $sourceDir)) { continue }
        $targetDir = Join-Path $Runtime $sub
        New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
        Copy-Item -Path (Join-Path $sourceDir '*') -Destination $targetDir -Force -Recurse
    }
    Write-Host "已从 $rendererSource 同步渲染器源码" -ForegroundColor Cyan
}

$manifest = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json
$cli = Join-Path $Runtime 'node_modules\@remotion\cli\remotion-cli.js'
$entry = Join-Path $Runtime 'index.jsx'
if (-not (Test-Path -LiteralPath $cli)) { throw "运行时缺少 Remotion CLI：$cli" }

$ffmpeg = (Get-Command ffmpeg -ErrorAction SilentlyContinue).Source
if (-not $ffmpeg) {
    foreach ($candidate in @('C:\ProgramData\chocolatey\bin\ffmpeg.exe', 'C:\ffmpeg\bin\ffmpeg.exe')) {
        if (Test-Path -LiteralPath $candidate) { $ffmpeg = $candidate; break }
    }
}
if (-not $ffmpeg) { throw '找不到 ffmpeg（SSIM 比对需要）' }

$chrome = ''
foreach ($candidate in @(
    (Join-Path $env:ProgramFiles 'Google\Chrome\Application\chrome.exe'),
    (Join-Path ${env:ProgramFiles(x86)} 'Microsoft\Edge\Application\msedge.exe'),
    (Join-Path $env:ProgramFiles 'Microsoft\Edge\Application\msedge.exe')
)) {
    if ($candidate -and (Test-Path -LiteralPath $candidate)) { $chrome = $candidate; break }
}

$width = [int]$manifest.size.width
$height = [int]$manifest.size.height
$floor = [double]$manifest.ssimFloor
$diffYAvgMax = [double]$manifest.diffYAvgMax
$diffYMaxMax = [double]$manifest.diffYMaxMax
$tempRoot = Join-Path $env:TEMP ("himind-regress-" + (Get-Date -Format 'yyyyMMdd-HHmmss'))
New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null

$results = [System.Collections.Generic.List[object]]::new()

foreach ($fixture in $manifest.fixtures) {
    if ($FixtureId.Count -gt 0 -and $FixtureId -notcontains $fixture.id) { continue }
    $recipe = Join-Path $fixtures ($fixture.recipe -replace '/', '\')
    if (-not (Test-Path -LiteralPath $recipe)) { throw "fixture $($fixture.id) 缺少 recipe：$recipe" }
    $outDir = Join-Path $fixtures $fixture.id
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null

    foreach ($frame in $fixture.frames) {
        $label = '{0}#{1}' -f $fixture.id, $frame
        $full = Join-Path $tempRoot ("{0}-{1}-full.png" -f $fixture.id, $frame)
        $small = Join-Path $tempRoot ("{0}-{1}.png" -f $fixture.id, $frame)
        $baseline = Join-Path $outDir ('frame-{0:D4}.png' -f $frame)

        $arguments = @($cli, 'still', $entry, $fixture.composition, $full, "--props=$recipe", "--frame=$frame", '--log=error')
        if ($chrome) { $arguments += "--browser-executable=$chrome" }
        & node @arguments
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $full)) {
            $results.Add([pscustomobject]@{ Fixture = $label; Status = 'FAIL'; SSIM = ''; Diff = ''; Note = '渲染失败' })
            continue
        }

        & $ffmpeg -v error -y -i $full -vf "scale=${width}:${height}" -pix_fmt rgb24 $small
        if ($LASTEXITCODE -ne 0) {
            $results.Add([pscustomobject]@{ Fixture = $label; Status = 'FAIL'; SSIM = ''; Diff = ''; Note = '缩放失败' })
            continue
        }

        if ($Update) {
            Copy-Item -LiteralPath $small -Destination $baseline -Force
            $results.Add([pscustomobject]@{ Fixture = $label; Status = 'BASELINE'; SSIM = '1.0000'; Diff = '0.00/0'; Note = '已更新基线' })
            continue
        }

        if (-not (Test-Path -LiteralPath $baseline)) {
            $results.Add([pscustomobject]@{ Fixture = $label; Status = 'FAIL'; SSIM = ''; Diff = ''; Note = '缺少基线，先运行 -Update' })
            continue
        }

        $ssimText = (& $ffmpeg -hide_banner -i $baseline -i $small -lavfi ssim -f null - 2>&1 | Out-String)
        $match = [regex]::Match($ssimText, 'All:([0-9.]+)')
        if (-not $match.Success) {
            $results.Add([pscustomobject]@{ Fixture = $label; Status = 'FAIL'; SSIM = ''; Diff = ''; Note = 'SSIM 解析失败' })
            continue
        }
        $ssim = [double]::Parse($match.Groups[1].Value, [System.Globalization.CultureInfo]::InvariantCulture)

        # SSIM 是全局平均，对小面积但明显的改动不敏感
        # （实测：120x120 的红块只让 SSIM 掉到 0.9757）。
        # 因此再加一层"差分统计"：YAVG 抓整体漂移，YMAX 抓局部突变。
        $diffText = (& $ffmpeg -hide_banner -i $baseline -i $small -lavfi "blend=all_mode=difference,format=gray,signalstats,metadata=print:file=-" -f null - 2>&1 | Out-String)
        $yavgMatch = [regex]::Match($diffText, 'YAVG=([0-9.]+)')
        $ymaxMatch = [regex]::Match($diffText, 'YMAX=([0-9.]+)')
        $yavg = if ($yavgMatch.Success) { [double]::Parse($yavgMatch.Groups[1].Value, [System.Globalization.CultureInfo]::InvariantCulture) } else { 999 }
        $ymax = if ($ymaxMatch.Success) { [double]::Parse($ymaxMatch.Groups[1].Value, [System.Globalization.CultureInfo]::InvariantCulture) } else { 999 }

        $ok = ($ssim -ge $floor) -and ($yavg -le $diffYAvgMax) -and ($ymax -le $diffYMaxMax)
        $results.Add([pscustomobject]@{
            Fixture = $label
            Status  = $(if ($ok) { 'PASS' } else { 'FAIL' })
            SSIM    = '{0:F4}' -f $ssim
            Diff    = ('{0:F2}/{1}' -f $yavg, [math]::Round($ymax))
            Note    = $(if ($ok) { '' } else { "疑似漂移：ssim=$('{0:F4}' -f $ssim) yavg=$('{0:F2}' -f $yavg) ymax=$([math]::Round($ymax))" })
        })
    }
}

$results | Format-Table -AutoSize | Out-String | Write-Host
$failed = @($results | Where-Object { $_.Status -eq 'FAIL' }).Count
if ($Update) {
    Write-Host "已更新 $(@($results | Where-Object { $_.Status -eq 'BASELINE' }).Count) 个基线。" -ForegroundColor Yellow
    exit 0
}
if ($failed -gt 0) {
    Write-Host "$failed 项回归失败：画面与基线不一致。" -ForegroundColor Red
    exit 1
}
Write-Host "全部通过（$($results.Count) 项，阈值 $floor）。" -ForegroundColor Green
exit 0
