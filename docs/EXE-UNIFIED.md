# Cuetiy EXE unified + 便携包

## 安装包构建

```powershell
powershell -File scripts/build-exe-unified.ps1
```

产物：`frontend/src-tauri/target/release/bundle/nsis|R msi`。

## 便携 zip

```powershell
powershell -File scripts/build-portable-unified.ps1
```

产物：`dist-packages/Cuetiy-unified-portable.zip`（需先完成 unified 构建）。

便携目录：

```text
Cuetiy/
  Cuetiy.exe
  cuetiy-backend.exe
  data/cuetiy.db      # 首次启动生成
  data/files/
  skills/
  .env
  README-PORTABLE.txt
```

## 使用

1. 双击 `Cuetiy.exe`  
2. 自动拉起本机后端 `http://127.0.0.1:8080` + SQLite  
3. 注册本地账号  
4. 「我的 → 数据与服务器」填 DeepSeek / MiMo Key 

聊天/TTS 仍需联网调模型；记录在本机 `data/`。

## .env（可选）

便携包已带默认 `.env`。一般不用改；Key 用 App 设置即可。

```env
DB_DRIVER=sqlite
SQLITE_PATH=./data/cuetiy.db
SERVER_HOST=127.0.0.1
SERVER_PORT=8080
REDIS_HOST=
```
