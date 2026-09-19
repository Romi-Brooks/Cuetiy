$ErrorActionPreference = 'Stop'
# RainYi APK thin — 需要 JAVA_HOME、ANDROID_HOME 已在环境中
# 可选: HTTP_PROXY / HTTPS_PROXY（不要写入仓库）

$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$frontend = Join-Path $repo 'frontend'
$android = Join-Path $frontend 'android'

if (-not $env:JAVA_HOME) { throw 'Set JAVA_HOME to your JDK path' }
if (-not $env:ANDROID_HOME -and -not $env:ANDROID_SDK_ROOT) { throw 'Set ANDROID_HOME to your Android SDK path' }
if ($env:ANDROID_SDK_ROOT) { $env:ANDROID_HOME = $env:ANDROID_SDK_ROOT }

$env:PATH = "$env:JAVA_HOME\bin;$env:ANDROID_HOME\platform-tools;$env:PATH"

Write-Host "JAVA_HOME=$env:JAVA_HOME"
Write-Host "ANDROID_HOME=$env:ANDROID_HOME"

Write-Host '==> Frontend build (no baked API URL)'
Set-Location $frontend
$env:VITE_API_URL = ''
pnpm type-check
pnpm exec vite build
if ($LASTEXITCODE -ne 0) { throw 'vite build failed' }

Write-Host '==> cap sync'
pnpm exec cap sync android
if ($LASTEXITCODE -ne 0) { throw 'cap sync failed' }

Write-Host '==> gradlew assembleDebug'
Set-Location $android
& .\gradlew.bat assembleDebug --no-daemon
if ($LASTEXITCODE -ne 0) { throw 'gradlew failed' }

$apk = Join-Path $android 'app\build\outputs\apk\debug\app-debug.apk'
if (-not (Test-Path $apk)) { throw "APK not found: $apk" }
$out = Join-Path $repo 'dist-packages'
New-Item -ItemType Directory -Force -Path $out | Out-Null
Copy-Item $apk (Join-Path $out 'RainYi-APK-thin-debug.apk') -Force
Get-Item (Join-Path $out 'RainYi-APK-thin-debug.apk') | Select-Object FullName, Length
