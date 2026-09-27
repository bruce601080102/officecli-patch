param(
  [string]$OfficeCLIVersion = 'latest',
  [switch]$SkipFetch
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$toolsDir = Join-Path $root 'tools'
$releaseDir = Join-Path $root 'release'

# Each entry uses the official binary name that must be placed in tools/.
# Add all official releases once; missing files are reported and skipped, so the
# current Windows-only checkout remains buildable until Linux/macOS binaries arrive.
$targets = @(
  @{ os = 'windows'; arch = 'amd64'; tool = 'officecli-win-x64.exe';          output = 'officecli-patch-win-x64.exe';          tags = '' },
  @{ os = 'windows'; arch = 'arm64'; tool = 'officecli-win-arm64.exe';        output = 'officecli-patch-win-arm64.exe';        tags = '' },
  @{ os = 'linux';   arch = 'amd64'; tool = 'officecli-linux-x64';            output = 'officecli-patch-linux-x64';            tags = '' },
  @{ os = 'linux';   arch = 'arm64'; tool = 'officecli-linux-arm64';          output = 'officecli-patch-linux-arm64';          tags = '' },
  @{ os = 'linux';   arch = 'amd64'; tool = 'officecli-linux-alpine-x64';     output = 'officecli-patch-linux-alpine-x64';     tags = 'alpine' },
  @{ os = 'linux';   arch = 'arm64'; tool = 'officecli-linux-alpine-arm64';   output = 'officecli-patch-linux-alpine-arm64';   tags = 'alpine' },
  @{ os = 'darwin';  arch = 'amd64'; tool = 'officecli-mac-x64';              output = 'officecli-patch-mac-x64';              tags = '' },
  @{ os = 'darwin';  arch = 'arm64'; tool = 'officecli-mac-arm64';            output = 'officecli-patch-mac-arm64';            tags = '' }
)

New-Item -ItemType Directory -Force $releaseDir | Out-Null
Push-Location $root
try {
  if (-not $SkipFetch) {
    & python "$root/scripts/fetch_officecli.py" --version $OfficeCLIVersion --tools-dir $toolsDir
    if ($LASTEXITCODE -ne 0) { throw '下載官方 OfficeCLI 失敗。' }
  }
  $env:CGO_ENABLED = '0'
  $env:GOCACHE = Join-Path $root '.gocache'
  foreach ($target in $targets) {
    $official = Join-Path $toolsDir $target.tool
    if (-not (Test-Path -LiteralPath $official -PathType Leaf)) {
      Write-Warning "略過 $($target.output)：tools/$($target.tool) 尚未放入。"
      continue
    }
    $env:GOOS = $target.os
    $env:GOARCH = $target.arch
    $output = Join-Path $releaseDir $target.output
    $buildArgs = @('build', '-buildvcs=false', '-trimpath', '-ldflags=-s -w', '-o', $output, '.')
    if ($target.tags) {
      $buildArgs = @('build', '-buildvcs=false', '-trimpath', '-tags', $target.tags, '-ldflags=-s -w', '-o', $output, '.')
    }
    Write-Host "建置 $($target.output)"
    & go @buildArgs
    if ($LASTEXITCODE -ne 0) { throw "建置失敗：$($target.output)" }
  }
} finally {
  Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
  Remove-Item Env:GOCACHE -ErrorAction SilentlyContinue
  Remove-Item Env:GOOS -ErrorAction SilentlyContinue
  Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
  Pop-Location
}
