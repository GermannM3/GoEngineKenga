// GoEngineKenga installer — один exe, без NSIS.
// Сборка: сначала положить в embed/ файлы (kenga-windows-amd64.exe, README.md, LICENSE, samples/hello), затем go build.
// Или положить готовые файлы в одну папку с установщиком и запустить — установщик возьмёт их оттуда.
// Запуск с /uninstall или -uninstall — удаление.

package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

//go:embed embed/*
var embedFS embed.FS

const (
	appName    = "GoEngineKenga"
	version    = "1.0.0"
	defaultDir = "C:\\Program Files\\GoEngineKenga"
)

func main() {
	if isUninstall() {
		doUninstall()
		return
	}
	doInstall()
}

func isUninstall() bool {
	for _, a := range os.Args[1:] {
		if a == "/uninstall" || a == "-uninstall" || a == "/S" || a == "-S" {
			return true
		}
	}
	return false
}

func selfDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

// srcDir возвращает каталог с файлами: либо распакованный embed во временную папку, либо папка с exe.
func getSrcDir() (string, func()) {
	// Проверяем embed (файлы встроены в exe)
	entries, err := fs.ReadDir(embedFS, "embed")
	if err == nil && len(entries) > 0 {
		tmp, err := os.MkdirTemp("", "kenga-setup-")
		if err != nil {
			fmt.Println("Warning: could not create temp dir:", err)
		} else {
			extractEmbed("embed", tmp)
			return tmp, func() { os.RemoveAll(tmp) }
		}
	}
	return selfDir(), func() {}
}

func extractEmbed(prefix, dest string) {
	entries, _ := fs.ReadDir(embedFS, prefix)
	for _, e := range entries {
		fullPath := filepath.Join(prefix, e.Name())
		destPath := filepath.Join(dest, e.Name())
		if e.IsDir() {
			os.MkdirAll(destPath, 0o755)
			extractEmbed(fullPath, destPath)
		} else {
			data, _ := fs.ReadFile(embedFS, fullPath)
			os.WriteFile(destPath, data, 0755)
		}
	}
}

func doInstall() {
	fmt.Println("=== GoEngineKenga Setup ===")
	fmt.Println()

	srcDir, cleanup := getSrcDir()
	defer cleanup()
	destDir := defaultDir

	cliExe := filepath.Join(srcDir, "kenga-windows-amd64.exe")
	if _, err := os.Stat(cliExe); err != nil {
		fmt.Println("Error: kenga-windows-amd64.exe not found.")
		fmt.Println("Build with: put kenga-windows-amd64.exe in cmd/kenga-installer/embed/ then go build.")
		os.Exit(1)
	}

	fmt.Printf("Install to [%s]: ", destDir)
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input != "" {
		destDir = input
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		fmt.Println("Error creating directory:", err)
		os.Exit(1)
	}

	copyFile(cliExe, filepath.Join(destDir, "kenga-windows-amd64.exe"))
	if ed := filepath.Join(srcDir, "kenga-editor-windows-amd64.exe"); fileExists(ed) {
		copyFile(ed, filepath.Join(destDir, "kenga-editor-windows-amd64.exe"))
	}
	for _, name := range []string{"README.md", "LICENSE"} {
		if src := filepath.Join(srcDir, name); fileExists(src) {
			copyFile(src, filepath.Join(destDir, name))
		}
	}
	helloSrc := filepath.Join(srcDir, "samples", "hello")
	helloDst := filepath.Join(destDir, "examples", "hello")
	if dirExists(helloSrc) {
		os.MkdirAll(helloDst, 0o755)
		copyDir(helloSrc, helloDst)
	}

	// Ярлыки через PowerShell
	createShortcut(filepath.Join(destDir, "kenga-windows-amd64.exe"), "GoEngineKenga CLI", os.Getenv("USERPROFILE")+"\\Desktop\\GoEngineKenga CLI.lnk")
	if fileExists(filepath.Join(destDir, "kenga-editor-windows-amd64.exe")) {
		createShortcut(filepath.Join(destDir, "kenga-editor-windows-amd64.exe"), "GoEngineKenga", os.Getenv("USERPROFILE")+"\\Desktop\\GoEngineKenga.lnk")
	}

	// Меню Пуск
	startMenu := filepath.Join(os.Getenv("PROGRAMDATA"), "Microsoft", "Windows", "Start Menu", "Programs", appName)
	os.MkdirAll(startMenu, 0o755)
	createShortcut(filepath.Join(destDir, "kenga-windows-amd64.exe"), "GoEngineKenga CLI", filepath.Join(startMenu, "GoEngineKenga CLI.lnk"))
	if fileExists(filepath.Join(destDir, "kenga-editor-windows-amd64.exe")) {
		createShortcut(filepath.Join(destDir, "kenga-editor-windows-amd64.exe"), "GoEngineKenga Editor", filepath.Join(startMenu, "GoEngineKenga Editor.lnk"))
	}
	uninstallExe := filepath.Join(destDir, "uninstall.exe")
	copyFile(os.Args[0], uninstallExe)
	createShortcut(uninstallExe, "Uninstall GoEngineKenga", filepath.Join(startMenu, "Uninstall.lnk"))

	// Реестр: Установка и удаление программ
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+appName, registry.SET_VALUE)
	if err != nil {
		fmt.Println("Warning: could not write registry (run as Administrator?):", err)
	} else {
		defer k.Close()
		k.SetStringValue("DisplayName", appName)
		k.SetStringValue("UninstallString", `"`+uninstallExe+`" /uninstall`)
		k.SetStringValue("QuietUninstallString", `"`+uninstallExe+`" /S`)
		k.SetStringValue("InstallLocation", destDir)
		k.SetStringValue("DisplayVersion", version)
		k.SetDWordValue("NoModify", 1)
		k.SetDWordValue("NoRepair", 1)
	}

	fmt.Println()
	fmt.Println("Installation complete.")
	fmt.Println("  CLI:   ", filepath.Join(destDir, "kenga-windows-amd64.exe"))
	fmt.Println("  Editor:", filepath.Join(destDir, "kenga-editor-windows-amd64.exe"))
}

func doUninstall() {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+appName, registry.QUERY_VALUE)
	if err != nil {
		fmt.Println("GoEngineKenga is not installed or already uninstalled.")
		os.Exit(0)
	}
	dir, _, _ := k.GetStringValue("InstallLocation")
	k.Close()
	if dir == "" {
		fmt.Println("Install location not found.")
		os.Exit(1)
	}

	os.Remove(filepath.Join(dir, "kenga-windows-amd64.exe"))
	os.Remove(filepath.Join(dir, "kenga-editor-windows-amd64.exe"))
	os.Remove(filepath.Join(dir, "README.md"))
	os.Remove(filepath.Join(dir, "LICENSE"))
	os.Remove(filepath.Join(dir, "uninstall.exe"))
	os.RemoveAll(filepath.Join(dir, "examples"))
	registry.DeleteKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+appName)
	startMenu := filepath.Join(os.Getenv("PROGRAMDATA"), "Microsoft", "Windows", "Start Menu", "Programs", appName)
	os.RemoveAll(startMenu)
	os.Remove(os.Getenv("USERPROFILE") + "\\Desktop\\GoEngineKenga.lnk")
	os.Remove(os.Getenv("USERPROFILE") + "\\Desktop\\GoEngineKenga CLI.lnk")
	os.RemoveAll(dir)
	fmt.Println("GoEngineKenga uninstalled.")
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
func dirExists(p string) bool  { fi, err := os.Stat(p); return err == nil && fi.IsDir() }

func copyFile(src, dst string) {
	r, _ := os.Open(src)
	defer r.Close()
	w, _ := os.Create(dst)
	io.Copy(w, r)
	w.Close()
	os.Chmod(dst, 0755)
}

func copyDir(src, dst string) {
	os.MkdirAll(dst, 0o755)
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		sp := filepath.Join(src, e.Name())
		dp := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDir(sp, dp)
		} else {
			copyFile(sp, dp)
		}
	}
}

func createShortcut(target, desc, lnkPath string) {
	script := fmt.Sprintf(`$s=New-Object -ComObject WScript.Shell; $l=$s.CreateShortcut('%s'); $l.TargetPath='%s'; $l.Description='%s'; $l.Save()`,
		strings.ReplaceAll(lnkPath, "'", "''"),
		strings.ReplaceAll(target, "'", "''"),
		strings.ReplaceAll(desc, "'", "''"))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
}
