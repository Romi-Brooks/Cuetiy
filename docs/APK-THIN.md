# RainYi APK thin（连服务器）

前置：JDK、Android SDK（compileSdk 35）、Node/pnpm。环境变量：

```powershell
$env:JAVA_HOME = '<JDK路径>'
$env:ANDROID_HOME = '<Android SDK路径>'
```

## 构建

```powershell
powershell -File scripts/build-apk-thin.ps1
```

或手动：

```powershell
cd frontend
$env:VITE_API_URL = ''
pnpm exec vite build
pnpm exec cap sync android
cd android
./gradlew.bat assembleDebug
```

产物：

```text
frontend/android/app/build/outputs/apk/debug/app-debug.apk
```

复制到 `dist-packages/`，例如 `RainYi-APK-thin-debug.apk`。

## 使用

1. 安装 APK  
2. 登录页 →「服务器设置」→ `http://<服务器IP>:8080`  
3. 保存并测试 → 登录  

服务器需放行 8080；密钥在 App「数据与服务器」里填。

## 说明

- 构建不写死 API，用户运行时配置  
- 当前为 debug 包；发布请配置签名后 `assembleRelease`  
- 局域网可用 HTTP；公网建议 HTTPS  
