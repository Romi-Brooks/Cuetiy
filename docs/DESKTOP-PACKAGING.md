# Desktop / package matrix

| Package | Backend | Data | User setup |
|---------|---------|------|------------|
| EXE thin | none | server | API URL in app |
| EXE unified | embedded Go | local SQLite | default `http://127.0.0.1:8080` |
| EXE portable zip | same as unified | `./data/` | unzip and run |
| APK thin | none | server | API URL in app |
| Linux thin/unified/portable | see LINUX-PACKAGING.md | same idea | see docs |

Details:

- [EXE-BUILD.md](EXE-BUILD.md)
- [EXE-UNIFIED.md](EXE-UNIFIED.md)
- [APK-THIN.md](APK-THIN.md)
- [LINUX-PACKAGING.md](LINUX-PACKAGING.md)

Runtime API URL is stored in the app (`localStorage`), not baked into thin builds.

AI keys: Me → Data & Server. Never echoed.
