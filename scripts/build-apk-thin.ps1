$ErrorActionPreference = 'Stop'
# RainYi APK thin — 需要 ANDROID_HOME + JDK 17-21
$sys = Join-Path $env:SystemRoot 'System32'
if ($sys -and (Test-Path -LiteralPath $sys)) {
  $env:PATH = "$sys;$env:PATH"
}
$cmd = Join-Path $sys 'cmd.exe'
if (!(Test-Path -LiteralPath $cmd)) { $cmd = 'cmd.exe' }

$repo = Resolve-Path (Join-Path $PSScriptRoot '..')
$frontend = Join-Path $repo 'frontend'
$android = Join-Path $frontend 'android'

function Get-JavaMajor {
  param([Parameter(Mandatory=$true)][string]$JdkHome)
  $java = Join-Path $JdkHome 'bin\java.exe'
  if (!(Test-Path -LiteralPath $java)) { return 0 }
  # java -version 写 stderr；用 cmd 合并输出，避免 ErrorActionPreference=Stop 误伤
  $txt = & $cmd /c "`"$java`" -version 2>&1"
  $txt = ($txt | Out-String)
  if ($txt -match 'version "(\d+)') { return [int]$Matches[1] }
  return 0
}

function Find-JdkHome {
  $list = @()
  if ($env:RAINYI_JDK) { $list += $env:RAINYI_JDK }
  if ($env:JAVA_HOME) { $list += $env:JAVA_HOME }

  $roots = @($env:ProgramFiles, ${env:ProgramFiles(x86)}, $env:LOCALAPPDATA, 'D:\Program', 'C:\Program Files', 'C:\Program Files (x86)')
  foreach ($r in $roots) {
    if (-not $r -or -not (Test-Path -LiteralPath $r)) { continue }
    $list += (Join-Path $r 'Android\Android Studio\jbr')
    $list += (Join-Path $r 'AndroidStudio\jbr')
    $list += (Join-Path $r 'Programs\Android Studio\jbr')
    $list += (Join-Path $r 'Java')
  }

  $picked = $null
  $pickedMajor = 0
  foreach ($cand in $list) {
    if (-not $cand) { continue }
    if (!(Test-Path -LiteralPath (Join-Path $cand 'bin\java.exe'))) { continue }
    $m = Get-JavaMajor -JdkHome $cand
    if ($m -ge 17 -and $m -le 21) { return @{ Home = $cand; Major = $m } }
    if ($m -gt $pickedMajor) { $picked = $cand; $pickedMajor = $m }
  }
  if ($picked) { return @{ Home = $picked; Major = $pickedMajor } }
  return $null
}

# Android SDK
if (-not $env:ANDROID_HOME) { $env:ANDROID_HOME = $env:ANDROID_SDK_ROOT }
if (-not $env:ANDROID_HOME) {
  $guess = Join-Path $env:LOCALAPPDATA 'Android\Sdk'
  if (Test-Path -LiteralPath $guess) { $env:ANDROID_HOME = $guess }
}
if (-not $env:ANDROID_HOME -or -not (Test-Path -LiteralPath $env:ANDROID_HOME)) {
  throw "ANDROID_HOME not found. Set ANDROID_HOME to Android SDK."
}

$jdkInfo = Find-JdkHome
if (-not $jdkInfo) { throw 'JDK not found. Set RAINYI_JDK or JAVA_HOME to JDK 17-21.' }
if ($jdkInfo.Major -lt 17 -or $jdkInfo.Major -gt 21) {
  Write-Warning ("JDK major={0} may fail Android Gradle Plugin; prefer 17-21." -f $jdkInfo.Major)
}

$env:JAVA_HOME = $jdkInfo.Home
$env:PATH = "$($jdkInfo.Home)\bin;$env:ANDROID_HOME\platform-tools;$env:PATH"

# local.properties（gitignore）
if (-not (Test-Path -LiteralPath $android)) { throw "android project missing: $android" }
$lp = Join-Path $android 'local.properties'
$sdkDir = $env:ANDROID_HOME -replace '\\', '\\'
$lpBody = "sdk.dir=$sdkDir"
if (!(Test-Path -LiteralPath $lp) -or ((Get-Content -LiteralPath $lp -Raw) -notmatch 'sdk\.dir=')) {
  Set-Content -LiteralPath $lp -Value $lpBody -Encoding ASCII
}

Write-Host ("JAVA_HOME={0} major={1}" -f $env:JAVA_HOME, $jdkInfo.Major)
Write-Host ("ANDROID_HOME={0}" -f $env:ANDROID_HOME)

Write-Host '==> Frontend build'
Set-Location $frontend
$env:VITE_API_URL = ''
pnpm type-check
if ($LASTEXITCODE -ne 0) { throw 'tsc failed' }
pnpm exec vite build
if ($LASTEXITCODE -ne 0) { throw 'vite build failed' }

Write-Host '==> cap sync android'
pnpm exec cap sync android
if ($LASTEXITCODE -ne 0) { throw 'cap sync failed' }

Write-Host '==> gradlew assembleDebug'
Set-Location $android
$jhomeArg = ($jdkInfo.Home -replace '\\', '/')
& .\gradlew.bat assembleDebug --no-daemon "-Dorg.gradle.java.home=$jhomeArg"
if ($LASTEXITCODE -ne 0) { throw 'gradlew failed' }

$apk = Join-Path $android 'app\build\outputs\apk\debug\app-debug.apk'
if (!(Test-Path -LiteralPath $apk)) { throw "APK not found: $apk" }

$out = Join-Path $repo 'dist-packages'
New-Item -ItemType Directory -Force -Path $out | Out-Null
Copy-Item -LiteralPath $apk -Destination (Join-Path $out 'RainYi-APK-thin-debug.apk') -Force
Get-Item (Join-Path $out 'RainYi-APK-thin-debug.apk') | Format-List FullName, Length
Write-Host '==> APK done'
