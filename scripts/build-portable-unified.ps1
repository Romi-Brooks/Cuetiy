$ErrorActionPreference = 'Stop'
# Cuetiy unified 便携 zip — 需先完成 unified 构建
$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$frontend = Join-Path $repo 'frontend'
$backend = Join-Path $repo 'backend'
$rel = Join-Path $frontend 'src-tauri\target\release'
$triple = 'x86_64-pc-windows-msvc'
$out = Join-Path $repo 'dist-packages'
$stage = Join-Path $out 'Cuetiy-unified-portable'
$zip = Join-Path $out 'Cuetiy-unified-portable.zip'

if (-not (Test-Path (Join-Path $rel 'cuetiy.exe'))) {
  throw "Missing $rel\cuetiy.exe — run scripts/build-exe-unified.ps1 first"
}

if (Test-Path $stage) { Remove-Item $stage -Recurse -Force }
New-Item -ItemType Directory -Force -Path (Join-Path $stage 'data\files'), (Join-Path $stage 'data\archives'), (Join-Path $stage 'skills') | Out-Null

Copy-Item (Join-Path $rel 'cuetiy.exe') (Join-Path $stage 'Cuetiy.exe') -Force
$sidecar = Join-Path $frontend "src-tauri\binaries\cuetiy-backend-$triple.exe"
if (-not (Test-Path $sidecar)) { $sidecar = Join-Path $frontend 'src-tauri\binaries\cuetiy-backend.exe' }
if (-not (Test-Path $sidecar)) {
  $env:CGO_ENABLED = '0'
  Set-Location $backend
  go build -ldflags '-s -w' -o (Join-Path $stage 'cuetiy-backend.exe') ./cmd/main.go
} else {
  Copy-Item $sidecar (Join-Path $stage 'cuetiy-backend.exe') -Force
}

$skillsSrc = Join-Path $backend 'skill\default_skills'
if (Test-Path $skillsSrc) { Copy-Item (Join-Path $skillsSrc '*') (Join-Path $stage 'skills') -Recurse -Force }

$envTpl = Join-Path $frontend 'src-tauri\resources\env\.env.unified'
if (Test-Path $envTpl) {
  Copy-Item $envTpl (Join-Path $stage '.env') -Force
} else {
  @'
DB_DRIVER=sqlite
SQLITE_PATH=./data/cuetiy.db
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
JWT_SECRET=change-me-local-cuetiy
STORAGE_DIR=./data/files
ARCHIVE_DIR=./data/archives
SKILLS_DIR=./skills
RUNTIME_DIR=.
REDIS_HOST=
'@ | Set-Content (Join-Path $stage '.env') -Encoding UTF8
}

@'
Cuetiy portable (unified)

1. Run Cuetiy.exe
2. Local backend starts at http://127.0.0.1:8080 (SQLite)
3. Register a local account
4. Set DeepSeek/MiMo keys in Me -> Data & Server (never shown)
5. Data lives in data/ — copy the folder to backup
'@ | Set-Content (Join-Path $stage 'README-PORTABLE.txt') -Encoding UTF8

New-Item -ItemType Directory -Force -Path $out | Out-Null
if (Test-Path $zip) { Remove-Item $zip -Force }
Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip -Force
Get-Item $zip | Select-Object FullName, Length
