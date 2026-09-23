# Cuetiy APK thin（连服务器）

前置：

- Node / pnpm
- Android SDK（compileSdk 35）
- **JDK 17–21**（不要用 22+，Gradle 会挂）

```powershell
$env:ANDROID_HOME = '<Android SDK 路径>'
$env:JAVA_HOME = '<JDK 17-21 路径>'   # 或 CUETIY_JDK=
```

脚本会优先用 `CUETIY_JDK` / 17–21 的 `JAVA_HOME`，并尝试 Android Studio 自带 JBR。

## 构建

```powershell
powershell -File scripts/build-apk-thin.ps1
```

产物：`dist-packages/Cuetiy-APK-thin-debug.apk`

## 使用

1. 安装 APK  
2. 登录页 →「服务器设置」→ `http://<服务器IP>:8080`  
3. 保存并测试 → 登录  

密钥在 App「数据与服务器」填写。

## 说明

- debug 包可直接装；发布需签名 `assembleRelease`  
- 局域网可用 HTTP；公网建议 HTTPS  
