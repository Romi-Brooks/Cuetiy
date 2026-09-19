// Cuetiy desktop shell
// thin: 仅 WebView，登录页配置 API
// unified: sidecar 启动 cuetiy-backend（SQLite），默认 http://127.0.0.1:8080

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::path::PathBuf;
use tauri::{Manager, RunEvent};
use tauri_plugin_shell::process::CommandEvent;
use tauri_plugin_shell::ShellExt;

fn package_mode() -> string_helper::Mode {
    string_helper::Mode::detect()
}

mod string_helper {
    #[derive(Clone, Copy, PartialEq, Eq)]
    pub enum Mode {
        Thin,
        Unified,
    }

    impl Mode {
        pub fn detect() -> Mode {
            if let Ok(m) = std::env::var("CUETIY_PACKAGE") {
                let m = m.trim().to_lowercase();
                if m == "unified" {
                    return Mode::Unified;
                }
                if m == "thin" {
                    return Mode::Thin;
                }
            }
            // 编译期写入（tauri build 时环境变量 CUETIY_PACKAGE）
            match option_env!("CUETIY_PACKAGE") {
                Some(m) if m.trim().eq_ignore_ascii_case("unified") => Mode::Unified,
                _ => Mode::Thin,
            }
        }

        pub fn is_unified(self) -> bool {
            matches!(self, Mode::Unified)
        }

        #[allow(dead_code)]
        pub fn as_str(self) -> &'static str {
            match self {
                Mode::Unified => "unified",
                Mode::Thin => "thin",
            }
        }
    }
}

fn backend_work_dir(app: &tauri::AppHandle) -> PathBuf {
    // 优先：安装目录 / exe 旁（便于 data/、skills/、.env 同级）
    if let Ok(exe) = std::env::current_exe() {
        if let Some(dir) = exe.parent() {
            return dir.to_path_buf();
        }
    }
    app.path()
        .resource_dir()
        .map(|p| p.to_path_buf())
        .unwrap_or_else(|_| PathBuf::from("."))
}

fn start_unified_backend(app: &tauri::AppHandle) {
    let work_dir = backend_work_dir(app);
    let data_dir = work_dir.join("data");
    let _ = std::fs::create_dir_all(data_dir.join("files"));
    let _ = std::fs::create_dir_all(data_dir.join("archives"));
    let _ = std::fs::create_dir_all(work_dir.join("skills"));

    // 写入默认 .env（不覆盖已有）
    let env_path = work_dir.join(".env");
    if !env_path.exists() {
        let default_env = format!(
            "DB_DRIVER=sqlite\nSQLITE_PATH=./data/cuetiy.db\nSERVER_HOST=127.0.0.1\nSERVER_PORT=8080\nJWT_SECRET=cuetiy-local-desktop\nSTORAGE_DIR=./data/files\nARCHIVE_DIR=./data/archives\nSKILLS_DIR=./skills\nRUNTIME_DIR=.\nREDIS_HOST=\n"
        );
        let _ = std::fs::write(&env_path, default_env);
    }

    let sidecar = app.shell().sidecar("cuetiy-backend");
    match sidecar {
        Ok(cmd) => {
            let cmd = cmd
                .current_dir(work_dir.clone())
                .env("DB_DRIVER", "sqlite")
                .env(
                    "SQLITE_PATH",
                    work_dir.join("data").join("cuetiy.db").to_string_lossy().to_string(),
                )
                .env("SERVER_HOST", "127.0.0.1")
                .env(
                    "SERVER_PORT",
                    std::env::var("CUETIY_BACKEND_PORT").unwrap_or_else(|_| "8080".into()),
                )
                .env(
                    "STORAGE_DIR",
                    work_dir.join("data").join("files").to_string_lossy().to_string(),
                )
                .env(
                    "ARCHIVE_DIR",
                    work_dir.join("data").join("archives").to_string_lossy().to_string(),
                )
                .env(
                    "SKILLS_DIR",
                    work_dir.join("skills").to_string_lossy().to_string(),
                )
                .env("RUNTIME_DIR", work_dir.to_string_lossy().to_string())
                .env("REDIS_HOST", "");
            match cmd.spawn() {
                Ok((mut rx, _child)) => {
                    eprintln!("[cuetiy] unified backend starting in {:?}", work_dir);
                    std::thread::spawn(move || {
                        while let Some(event) = rx.blocking_recv() {
                            match event {
                                CommandEvent::Stdout(line) => {
                                    eprintln!("[backend] {}", String::from_utf8_lossy(&line))
                                }
                                CommandEvent::Stderr(line) => {
                                    eprintln!("[backend] {}", String::from_utf8_lossy(&line))
                                }
                                CommandEvent::Terminated(payload) => {
                                    eprintln!("[backend] terminated: {:?}", payload.code);
                                    break;
                                }
                                _ => {}
                            }
                        }
                    });
                }
                Err(e) => eprintln!("[cuetiy] failed to spawn backend: {e}"),
            }
        }
        Err(e) => {
            eprintln!("[cuetiy] unified sidecar missing (ok for thin builds): {e}");
        }
    }
}

fn inject_unified_defaults(win: &tauri::WebviewWindow) {
    // 仅在用户未手写 API 时写入本机默认，避免覆盖 thin 用户配置
    let js = r#"try{
      if(!localStorage.getItem('cuetiy:api_base')){
        localStorage.setItem('cuetiy:api_base','http://127.0.0.1:8080');
      }
      localStorage.setItem('cuetiy:package','unified');
    }catch(e){}"#;
    let _ = win.eval(js);
}

fn main() {
    let mode = package_mode();
    let mode_for_setup = mode;

    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .setup(move |app| {
            let mode = mode_for_setup;
            if mode.is_unified() {
                let handle = app.handle().clone();
                std::thread::spawn(move || {
                    std::thread::sleep(std::time::Duration::from_millis(200));
                    start_unified_backend(&handle);
                });
            }
            if let Some(win) = app.get_webview_window("main") {
                let title = if mode.is_unified() {
                    "Cuetiy · 本地数据"
                } else {
                    "Cuetiy"
                };
                let _ = win.set_title(title);
                if mode.is_unified() {
                    std::thread::sleep(std::time::Duration::from_millis(400));
                    inject_unified_defaults(&win);
                }
            }
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while running Cuetiy tauri application")
        .run(|_app, event| {
            if let RunEvent::Exit = event {
                // sidecar 随进程退出
            }
        });
}
