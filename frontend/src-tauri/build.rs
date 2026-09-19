fn main() {
    // 将打包类型在编译期注入（thin / unified）
    if std::env::var("RAIN_YI_PACKAGE").is_err() {
        // tauri/cli 构建时若未设置，默认 thin；unified 脚本会设置
        std::env::set_var("RAIN_YI_PACKAGE", "thin");
    }
    println!(
        "cargo:rustc-env=RAIN_YI_PACKAGE={}",
        std::env::var("RAIN_YI_PACKAGE").unwrap_or_else(|_| "thin".into())
    );
    println!("cargo:rerun-if-env-changed=RAIN_YI_PACKAGE");
    tauri_build::build()
}

