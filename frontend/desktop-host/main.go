package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Cuetiy Desktop 宿主（thin / unified 共用）
// thin：只开 WebView，API 由用户在登录页配置
// unified：先拉起本机 Go 后端（SQLite），默认指向 127.0.0.1

func main() {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("CUETIY_PACKAGE")))
	if mode == "" {
		mode = "thin"
	}

	exe, err := os.Executable()
	if err != nil {
		log.Fatalln(err)
	}
	exeDir := filepath.Dir(exe)

	var cancel context.CancelFunc
	if mode == "unified" {
		cancel = startUnifiedBackend(exeDir)
		defer cancel()
		// 写入默认本地 API，便于首次打开即用
		_ = os.WriteFile(filepath.Join(exeDir, "package-mode.txt"), []byte("unified\n"+time.Now().Format(time.RFC3339)), 0o644)
	} else {
		_ = os.WriteFile(filepath.Join(exeDir, "package-mode.txt"), []byte("thin\n"+time.Now().Format(time.RFC3339)), 0o644)
	}

	runTauri(exeDir, mode)
}

func startUnifiedBackend(exeDir string) context.CancelFunc {
	names := []string{"cuetiy-backend.exe", "cuetiy-backend"}
	if runtime.GOOS != "windows" {
		names = []string{"cuetiy-backend"}
	}
	var binPath string
	candidates := []string{
		filepath.Join(exeDir, "binaries"),
		exeDir,
		filepath.Join(exeDir, ".."),
		filepath.Join(exeDir, "resources"),
	}
	for _, dir := range candidates {
		for _, n := range names {
			p := filepath.Join(dir, n)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				binPath = p
				break
			}
		}
		if binPath != "" {
			break
		}
	}
	if binPath == "" {
		log.Println("unified: 未找到 cuetiy-backend，跳过内嵌后端（可手动启动）")
		return func() {}
	}

	// 工作目录：优先 exe 旁 data/ 布局
	workDir := exeDir
	env := append(os.Environ(),
		"DB_DRIVER=sqlite",
		"SQLITE_PATH="+filepath.Join(workDir, "data", "cuetiy.db"),
		"SERVER_HOST=127.0.0.1",
		"SERVER_PORT="+envOr("CUETIY_BACKEND_PORT", "8080"),
		"STORAGE_DIR="+filepath.Join(workDir, "data", "files"),
		"ARCHIVE_DIR="+filepath.Join(workDir, "data", "archives"),
		"SKILLS_DIR="+filepath.Join(workDir, "skills"),
		"RUNTIME_DIR="+workDir,
	)
	// 若存在 env 文件则让后端自己加载；这里不强制覆盖用户 JWT/Key
	_ = godotenv.Load(filepath.Join(workDir, ".env"))

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, binPath)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("unified: 启动后端失败: %v", err)
		return cancel
	}
	log.Printf("unified: backend pid=%d dir=%s", cmd.Process.Pid, workDir)

	go func() {
		_ = cmd.Wait()
	}()
	return cancel
}

func runTauri(exeDir, mode string) {
	// Prefer invoking `tauri dev` / binary already built next to this host.
	// This binary is a thin orchestrator used in dev; production Tauri bundles
	// embed the frontend window directly (see frontend/src-tauri).
	frontendDist := filepath.Join(exeDir, "frontend", "dist")
	if _, err := os.Stat(frontendDist); err == nil {
		log.Printf("package=%s frontend=%s", mode, frontendDist)
	}
	fmt.Println("Cuetiy desktop host ready. Use `pnpm tauri dev` / `pnpm tauri build` for the real shell.")
	fmt.Printf("Package mode: %s\n", mode)
	if mode == "unified" {
		fmt.Println("API should be: http://127.0.0.1:8080")
	} else {
		fmt.Println("Configure API URL in the app login screen.")
	}
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
