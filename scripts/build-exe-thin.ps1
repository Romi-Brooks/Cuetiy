$ErrorActionPreference = 'Stop'
# Cuetiy EXE thin — 依赖: Go, pnpm, Rust (cargo 在 PATH)
$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$frontend = Join-Path $repo 'frontend'
$backend = Join-Path $repo 'backend'
$binDir = Join-Path $frontend 'src-tauri\binaries'
$triple = 'x86_64-pc-windows-msvc'

# 自动补齐 PATH（cargo / node / system32）
$sys = Join-Path $env:SystemRoot 'System32'
if ($sys -and (Test-Path $sys)) { $env:PATH = "$sys;$env:PATH" }
$cargoDir = Join-Path $env:USERPROFILE '.cargo\bin'
if (Test-Path $cargoDir) { $env:PATH = "$cargoDir;$env:PATH" }
foreach ($n in @("$env:ProgramFiles\nodejs")) {
  if ($n -and (Test-Path $n)) { $env:PATH = "$n;$env:PATH" }
}
if (-not (Get-Command cargo -ErrorAction SilentlyContinue)) {
  throw 'cargo not found in PATH (install Rust / set PATH)'
}
if (-not (Get-Command pnpm -ErrorAction SilentlyContinue)) {
  throw 'pnpm not found in PATH'
}

$env:CGO_ENABLED = '0'
$env:CUETIY_PACKAGE = 'thin'

Write-Host '==> Go backend sidecar'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
Set-Location $backend
go build -ldflags '-s -w' -o (Join-Path $binDir 'cuetiy-backend.exe') ./cmd/main.go
if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
Copy-Item (Join-Path $binDir 'cuetiy-backend.exe') (Join-Path $binDir "cuetiy-backend-$triple.exe") -Force

Write-Host '==> Frontend + tauri build (thin)'
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
    $name = "Cuetiy-EXE-thin-$($_.Name)"
    Copy-Item $_.FullName (Join-Path $out $name) -Force
    Write-Host "  $name"
  }
Write-Host "==> $out"
