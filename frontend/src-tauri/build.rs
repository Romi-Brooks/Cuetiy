fn main() {
    // 将打包类型在编译期注入（thin / unified）
    if std::env::var("CUETIY_PACKAGE").is_err() {
        // tauri/cli 构建时若未设置，默认 thin；unified 脚本会设置
        std::env::set_var("CUETIY_PACKAGE", "thin");
    }
    println!(
        "cargo:rustc-env=CUETIY_PACKAGE={}",
        std::env::var("CUETIY_PACKAGE").unwrap_or_else(|_| "thin".into())
    );
    println!("cargo:rerun-if-env-changed=CUETIY_PACKAGE");
    tauri_build::build()
}

