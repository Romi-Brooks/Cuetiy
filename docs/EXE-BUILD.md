# RainYi EXE thin（连服务器）

前置：Go、Node/pnpm、Rust（MSVC）、WebView2。详见本机 Rust/VS 安装说明。

## 构建

在仓库根目录：

```powershell
powershell -File scripts/build-exe-thin.ps1
```

或手动：

```powershell
cd backend
$env:CGO_ENABLED = '0'
go build -ldflags '-s -w' -o ../frontend/src-tauri/binaries/rainyi-backend.exe ./cmd/main.go
# Tauri sidecar 需要 target 名
Copy-Item ../frontend/src-tauri/binaries/rainyi-backend.exe `
  ../frontend/src-tauri/binaries/rainyi-backend-x86_64-pc-windows-msvc.exe

cd ../frontend
pnpm install
pnpm type-check
$env:RAIN_YI_PACKAGE = 'thin'
pnpm exec tauri build
```

产物：

```text
frontend/src-tauri/target/release/bundle/nsis/*-setup.exe
frontend/src-tauri/target/release/bundle/msi/*.msi
```

复制到 `dist-packages/` 并改名，例如 `RainYi-EXE-thin-setup.exe`。

## 使用

1. 安装后打开 RainYi  
2. 登录页 →「服务器设置」→ 填 `http://<服务器IP>:8080`  
3. 保存并测试 → 登录  

密钥在「我的 → 数据与服务器」填写（不回显）。

## 开发调试

```powershell
cd frontend
$env:RAIN_YI_PACKAGE = 'thin'
pnpm exec tauri dev
```

网络若需代理，自行设置 `HTTP_PROXY` / `HTTPS_PROXY`（不要写进仓库）。
