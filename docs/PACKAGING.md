# 打包一览

| 产物 | 脚本 | 文档 |
|------|------|------|
| EXE thin | `scripts/build-exe-thin.ps1` | [EXE-BUILD.md](EXE-BUILD.md) |
| EXE unified | `scripts/build-exe-unified.ps1` | [EXE-UNIFIED.md](EXE-UNIFIED.md) |
| EXE portable zip | `scripts/build-portable-unified.ps1` | [EXE-UNIFIED.md](EXE-UNIFIED.md) |
| APK thin | `scripts/build-apk-thin.ps1` | [APK-THIN.md](APK-THIN.md) |
| Linux thin | `scripts/build-linux-thin.sh` | [LINUX-PACKAGING.md](LINUX-PACKAGING.md) |
| Linux unified | `scripts/build-linux-unified.sh` | [LINUX-PACKAGING.md](LINUX-PACKAGING.md) |
| Linux portable | `scripts/build-linux-portable.sh` | [LINUX-PACKAGING.md](LINUX-PACKAGING.md) |
| PostgreSQL | — | [PG-MIGRATION.md](PG-MIGRATION.md) |

产物目录：`dist-packages/`。

清理构建产物：

```powershell
powershell -File scripts/clean-build.ps1
```
