---
name: Bug report
about: Create a report to help us improve
title: 'Bug: '
labels: bug
assignees: ''

---

## Describe the bug / 问题描述
A clear and concise description of what the bug is.
简洁清晰地描述出现的程序故障问题。

## Environment / 环境
- Package form / 打包形态: [Web / EXE thin / EXE unified / EXE portable / APK thin / from source]
- OS and version / 操作系统与版本: [e.g. Windows 11 23H2, Ubuntu 24.04, Android 14]
- Backend DB / 后端数据库 (`DB_DRIVER`): [sqlite / postgres / mysql]
- Redis enabled / Redis 是否启用: [yes / no]
- Frontend run mode / 前端启动方式: [`pnpm dev` / prebuilt bundle / Capacitor / Tauri]
- Versions / 版本: Go [e.g. 1.26], Node [e.g. 22], app version [e.g. 0.1.0]

## To Reproduce / 复现步骤
Steps to reproduce the behavior:
复现问题的操作步骤：
1. Go to '...'
   进入指定页面/功能「...」
2. Click on '....'
   点击「....」按钮/选项
3. Scroll down to '....'
   下滑至「....」位置
4. See error
   出现异常报错

## Expected behavior / 预期表现
A clear and concise description of what you expected to happen.
简洁清晰地说明程序本该正常运行的效果。

## Actual behavior / 实际表现
What actually happened instead. Include the relevant log or error output, desensitized.
实际发生了什么。请附上相关日志或报错输出（须脱敏）。

## Screenshots / 截图
If applicable, add screenshots to help explain your problem.
如有需要，附上问题截图辅助说明故障。

## Additional context / 补充说明
Add any other context about the problem here, such as whether it also reproduces on another package form.
填写和故障相关的其他补充信息，例如是否在其他打包形态下同样复现。

## Security check / 安全检查
Do **not** paste API keys, JWTs, or unredacted `.env` contents. Security issues (key leakage, cross-user access, path traversal) should be reported privately first — see [CONTRIBUTING.md](https://github.com/Romi-Brooks/Cuetiy/blob/main/CONTRIBUTING.md).
**不要**粘贴 API Key、JWT 或未脱敏的 `.env` 内容。安全类问题（密钥泄漏、越权读取他人会话、路径穿越）请先私下说明影响面，见 [CONTRIBUTING.md](https://github.com/Romi-Brooks/Cuetiy/blob/main/CONTRIBUTING.md)。
