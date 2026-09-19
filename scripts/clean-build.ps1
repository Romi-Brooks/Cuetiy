$ErrorActionPreference = 'Continue'
$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
Write-Host "Cleaning build artifacts under $repo"

$targets = @(
  'dist-packages',
  'frontend/dist',
  'frontend/src-tauri/target',
  'frontend/src-tauri/binaries',
  'frontend/src-tauri/gen',
  'frontend/android/app/build',
  'frontend/android/.gradle',
  'frontend/android/build',
  'frontend/android/capacitor-cordova-android-plugins',
  'frontend/android/local.properties',
  'backend/output',
  'frontend/desktop-host/desktop-host.exe'
)
foreach ($rel in $targets) {
  $p = Join-Path $repo $rel
  if (Test-Path $p) {
    Write-Host "  remove $rel"
    Remove-Item $p -Recurse -Force -ErrorAction SilentlyContinue
  }
}
Get-ChildItem $repo -Recurse -Include *.log,*.exe,*.msi,*.apk,*.AppImage,*.deb -File -ErrorAction SilentlyContinue |
  Where-Object { $_.FullName -notmatch 'node_modules|\.git\\' } |
  Remove-Item -Force -ErrorAction SilentlyContinue
Write-Host 'Done.'
