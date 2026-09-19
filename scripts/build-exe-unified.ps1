$ErrorActionPreference = 'Stop'
# RainYi EXE unified — 依赖: Go, pnpm, Rust
$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$frontend = Join-Path $repo 'frontend'
$backend = Join-Path $repo 'backend'
$binDir = Join-Path $frontend 'src-tauri\binaries'
$triple = 'x86_64-pc-windows-msvc'

$env:CGO_ENABLED = '0'
$env:RAIN_YI_PACKAGE = 'unified'

Write-Host '==> Go backend sidecar'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
Set-Location $backend
go build -ldflags '-s -w' -o (Join-Path $binDir 'rainyi-backend.exe') ./cmd/main.go
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
Copy-Item (Join-Path $binDir 'rainyi-backend.exe') (Join-Path $binDir "rainyi-backend-$triple.exe") -Force

$skillsDst = Join-Path $frontend 'src-tauri\resources\skills'
$skillsSrc = Join-Path $backend 'skill\default_skills'
if (Test-Path $skillsSrc) {
  New-Item -ItemType Directory -Force -Path $skillsDst | Out-Null
  Copy-Item (Join-Path $skillsSrc '*') $skillsDst -Recurse -Force
}

Write-Host '==> Frontend + tauri build (unified)'
Set-Location $frontend
pnpm install --prefer-offline
pnpm type-check
if ($LASTEXITCODE -ne 0) { throw 'tsc failed' }
& (Join-Path $frontend 'node_modules\.bin\tauri.cmd') build
if ($LASTEXITCODE -ne 0) { throw 'tauri build failed' }

$out = Join-Path $repo 'dist-packages'
New-Item -ItemType Directory -Force -Path $out | Out-Null
Get-ChildItem (Join-Path $frontend 'src-tauri\target\release\bundle') -Recurse -Include *.exe,*.msi -ErrorAction SilentlyContinue |
  ForEach-Object {
    $name = "RainYi-EXE-unified-$($_.Name)"
    Copy-Item $_.FullName (Join-Path $out $name) -Force
    Write-Host "  $name"
  }
Write-Host "==> $out"
