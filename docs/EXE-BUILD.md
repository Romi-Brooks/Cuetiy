# Cuetiy EXE thin

前置：Go、Node/pnpm、Rust（MSVC）、WebView2。

## 构建

在仓库根目录：

```powershell
powershell -File scripts/build-exe-thin.ps1
```

或手动：

```powershell
cd backend
$env:CGO_ENABLED = '0'
go build -ldflags '-s -w' -o ../frontend/src-tauri/binaries/cuetiy-backend.exe ./cmd/main.go
# Tauri sidecar 需要 target 名
Copy-Item ../frontend/src-tauri/binaries/cuetiy-backend.exe `
  ../frontend/src-tauri/binaries/cuetiy-backend-x86_64-pc-windows-msvc.exe

cd ../frontend
pnpm install
pnpm type-check
$env:CUETIY_PACKAGE = 'thin'
pnpm exec tauri build
```

产物：

```text
frontend/src-tauri/target/release/bundle/nsis/*-setup.exe
frontend/src-tauri/target/release/bundle/msi/*.msi
```

复制到 `dist-packages/` 并改名，例如 `Cuetiy-EXE-thin-setup.exe`。

## 使用

1. 安装后打开 Cuetiy  
2. 登录页 →「服务器设置」→ 填 `http://<服务器IP>:8080`  
3. 保存并测试 → 登录  

密钥在「我的 → 数据与服务器」填写。

## 开发调试

```powershell
cd frontend
$env:CUETIY_PACKAGE = 'thin'
pnpm exec tauri dev
```
