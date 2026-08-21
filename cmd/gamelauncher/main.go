// Лаунчер игры для Windows: запускает движок kenga.exe с проектом без
// консольного окна (GUI-подсистема). При ошибке показывает MessageBox
// и пишет лог last_run.log рядом с exe. Для других ОС — обычный запуск.
// Сборка релиза: go build -ldflags "-H windowsgui" -o gamelauncher.exe ./cmd/gamelauncher
package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

func main() {
	exe, err := os.Executable()
	if err != nil {
		exe, _ = os.Getwd()
	}
	dir := filepath.Dir(exe)
	_ = os.Chdir(dir) // kenga run --project game резолвится относительно cwd

	project := "game"
	backend := "ebiten"
	if p := os.Getenv("KENG_GAME_PROJECT"); p != "" {
		project = p
	}
	if b := os.Getenv("KENG_GAME_BACKEND"); b != "" {
		backend = b
	}

	engine := filepath.Join(dir, "kenga.exe")
	logf, err := os.Create(filepath.Join(dir, "last_run.log"))
	if err == nil {
		defer logf.Close()
	}
	writeLog := func(s string) {
		if logf != nil {
			logf.WriteString(s + "\r\n")
			logf.Sync()
		}
	}

	log.Printf("starting: %s run --project %s --backend %s", engine, project, backend)
	cmd := exec.Command(engine, "run", "--project", project, "--backend", backend)
	cmd.Dir = dir
	var b strings.Builder
	cmd.Stdout = &b
	cmd.Stderr = &b
	runErr := cmd.Run()
	if runErr != nil {
		writeLog("ERROR: " + runErr.Error() + "\n" + b.String())
		msgBox("GoEngineKenga: ошибка запуска игры\n" + b.String() + "\nПодробности: last_run.log")
	} else {
		writeLog("exit: ok\n" + b.String())
	}
}

// msgBox показывает системное окно с сообщением (Windows), на других ОС — печать.
func msgBox(text string) {
	if runtime.GOOS == "windows" {
		user32, err := syscall.LoadDLL("user32.dll")
		if err == nil {
			if m, err := user32.FindProc("MessageBoxW"); err == nil {
				t, _ := syscall.UTF16PtrFromString(text)
				c, _ := syscall.UTF16PtrFromString("GoEngineKenga")
				m.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), 0x10) // MB_ICONERROR
			}
		}
	} else {
		os.Stderr.WriteString(text + "\n")
	}
}