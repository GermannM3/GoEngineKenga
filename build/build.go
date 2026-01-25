package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildConfig конфигурация сборки
type BuildConfig struct {
	Name        string
	Version     string
	OutputDir   string
	Targets     []Target
	IncludeDeps bool
}

// Target целевая платформа для сборки
type Target struct {
	GOOS   string
	GOARCH string
	Name   string
}

func main() {
	config := BuildConfig{
		Name:      "GoEngineKenga",
		Version:   "1.0.0",
		OutputDir: "dist",
		Targets: []Target{
			{"windows", "amd64", "kenga-editor-windows-amd64.exe"},
			{"windows", "386", "kenga-editor-windows-386.exe"},
			{"linux", "amd64", "kenga-editor-linux-amd64"},
			{"linux", "386", "kenga-editor-linux-386"},
			{"darwin", "amd64", "kenga-editor-darwin-amd64"},
			{"darwin", "arm64", "kenga-editor-darwin-arm64"},
		},
		IncludeDeps: true,
	}

	if err := buildRelease(config); err != nil {
		fmt.Printf("Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Build completed successfully!")
}

func buildRelease(config BuildConfig) error {
	// Создаем выходную директорию
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	// Собираем для каждой целевой платформы
	for _, target := range config.Targets {
		if err := buildTarget(config, target); err != nil {
			return fmt.Errorf("failed to build for %s/%s: %w", target.GOOS, target.GOARCH, err)
		}
	}

	// Создаем архивы релиза
	if err := createArchives(config); err != nil {
		return fmt.Errorf("failed to create archives: %w", err)
	}

	// Создаем установщики (если возможно)
	if err := createInstallers(config); err != nil {
		fmt.Printf("Warning: failed to create installers: %v\n", err)
	}

	return nil
}

func buildTarget(config BuildConfig, target Target) error {
	fmt.Printf("Building for %s/%s...\n", target.GOOS, target.GOARCH)

	outputPath := filepath.Join(config.OutputDir, target.Name)

	// Устанавливаем переменные окружения для кросс-компиляции
	env := append(os.Environ(),
		"GOOS="+target.GOOS,
		"GOARCH="+target.GOARCH,
		"CGO_ENABLED=0", // Отключаем CGO для кросс-платформенности
	)

	// Собираем редактор
	cmd := exec.Command("go", "build",
		"-ldflags", fmt.Sprintf("-X main.version=%s -X main.buildTime=%s", config.Version, time.Now().Format(time.RFC3339)),
		"-o", outputPath,
		"./cmd/kenga-editor",
	)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Собираем CLI инструмент
	cliName := strings.Replace(target.Name, "kenga-editor", "kenga", 1)
	cliPath := filepath.Join(config.OutputDir, cliName)

	cmd = exec.Command("go", "build",
		"-ldflags", fmt.Sprintf("-X main.version=%s -X main.buildTime=%s", config.Version, time.Now().Format(time.RFC3339)),
		"-o", cliPath,
		"./cmd/kenga",
	)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build CLI failed: %w", err)
	}

	fmt.Printf("Built %s\n", target.Name)
	return nil
}

func createArchives(config BuildConfig) error {
	fmt.Println("Creating release archives...")

	// Создаем ZIP архивы для каждой платформы
	for _, target := range config.Targets {
		archiveName := fmt.Sprintf("%s-%s-%s.zip", config.Name, config.Version, strings.TrimSuffix(target.Name, filepath.Ext(target.Name)))

		// Собираем файлы для архива
		files := []string{
			filepath.Join(config.OutputDir, target.Name),
			filepath.Join(config.OutputDir, strings.Replace(target.Name, "kenga-editor", "kenga", 1)),
		}

		if config.IncludeDeps {
			files = append(files, "README.md", "LICENSE")
		}

		if err := createZip(filepath.Join(config.OutputDir, archiveName), files); err != nil {
			return fmt.Errorf("failed to create zip for %s: %w", target.Name, err)
		}

		fmt.Printf("Created %s\n", archiveName)
	}

	return nil
}

func createZip(zipPath string, files []string) error {
	// Для простоты используем простой подход - создаем tar.gz на Unix или zip на Windows
	if runtime.GOOS == "windows" {
		return createZipWindows(zipPath, files)
	}
	return createTarGz(zipPath, files)
}

func createZipWindows(zipPath string, files []string) error {
	// Используем PowerShell для создания ZIP
	args := []string{"-Command", "Compress-Archive", "-Path"}
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			args = append(args, file)
		}
	}
	args = append(args, "-DestinationPath", zipPath, "-Force")

	cmd := exec.Command("powershell", args...)
	return cmd.Run()
}

func createTarGz(tarPath string, files []string) error {
	// Используем tar для создания архива
	args := []string{"czf", tarPath}
	args = append(args, files...)

	cmd := exec.Command("tar", args...)
	return cmd.Run()
}

func createInstallers(config BuildConfig) error {
	fmt.Println("Creating installers...")

	// Создаем MSI для Windows
	if runtime.GOOS == "windows" {
		if err := createWindowsInstaller(config); err != nil {
			return fmt.Errorf("failed to create Windows installer: %w", err)
		}
	}

	// Создаем DEB для Linux
	if runtime.GOOS == "linux" {
		if err := createLinuxInstaller(config); err != nil {
			return fmt.Errorf("failed to create Linux installer: %w", err)
		}
	}

	return nil
}

func createWindowsInstaller(config BuildConfig) error {
	// Ищем WiX Toolset
	wixPath := findWiXToolset()
	if wixPath == "" {
		return fmt.Errorf("WiX Toolset not found. Install from https://wixtoolset.org/")
	}

	// Создаем WiX файл
	wxsContent := generateWixFile(config)
	wxsPath := filepath.Join(config.OutputDir, "installer.wxs")

	if err := os.WriteFile(wxsPath, []byte(wxsContent), 0644); err != nil {
		return fmt.Errorf("failed to write WXS file: %w", err)
	}

	// Компилируем MSI
	candlePath := filepath.Join(wixPath, "candle.exe")
	lightPath := filepath.Join(wixPath, "light.exe")

	// Запускаем candle
	cmd := exec.Command(candlePath, wxsPath)
	cmd.Dir = config.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("candle failed: %w", err)
	}

	// Запускаем light
	msiName := fmt.Sprintf("%s-%s.msi", config.Name, config.Version)
	wixobjPath := filepath.Join(config.OutputDir, "installer.wixobj")

	cmd = exec.Command(lightPath, wixobjPath, "-o", msiName)
	cmd.Dir = config.OutputDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("light failed: %w", err)
	}

	fmt.Printf("Created Windows installer: %s\n", msiName)
	return nil
}

func createLinuxInstaller(config BuildConfig) error {
	// Создаем простую структуру для DEB пакета
	debDir := filepath.Join(config.OutputDir, "deb")
	if err := os.MkdirAll(debDir, 0755); err != nil {
		return err
	}

	// Создаем структуру DEB
	debianDir := filepath.Join(debDir, "DEBIAN")
	if err := os.MkdirAll(debianDir, 0755); err != nil {
		return err
	}

	usrBinDir := filepath.Join(debDir, "usr", "bin")
	if err := os.MkdirAll(usrBinDir, 0755); err != nil {
		return err
	}

	// Копируем бинарные файлы
	// (здесь должна быть логика копирования файлов)

	// Создаем control файл
	controlContent := fmt.Sprintf(`Package: goenginekenga
Version: %s
Section: games
Priority: optional
Architecture: amd64
Depends:
Maintainer: GoEngineKenga Team <team@goenginekenga.org>
Description: Modern game engine written in Go
`, config.Version)

	if err := os.WriteFile(filepath.Join(debianDir, "control"), []byte(controlContent), 0644); err != nil {
		return err
	}

	// Создаем DEB архив
	debName := fmt.Sprintf("%s-%s.deb", config.Name, config.Version)
	cmd := exec.Command("dpkg-deb", "--build", debDir, filepath.Join(config.OutputDir, debName))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dpkg-deb failed: %w", err)
	}

	fmt.Printf("Created Debian package: %s\n", debName)
	return nil
}

func findWiXToolset() string {
	// Ищем WiX в стандартных местах установки
	paths := []string{
		`C:\Program Files (x86)\WiX Toolset v3.11\bin`,
		`C:\Program Files\WiX Toolset v3.11\bin`,
		`C:\Program Files (x86)\WiX Toolset v4.0\bin`,
		`C:\Program Files\WiX Toolset v4.0\bin`,
	}

	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(path, "candle.exe")); err == nil {
			return path
		}
	}

	return ""
}

func generateWixFile(config BuildConfig) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Wix xmlns="http://schemas.microsoft.com/wix/2006/wi">
  <Product Id="*" Name="%s" Language="1033" Version="%s" Manufacturer="GoEngineKenga Team" UpgradeCode="PUT-GUID-HERE">
    <Package InstallerVersion="200" Compressed="yes" InstallScope="perMachine" />

    <MajorUpgrade DowngradeErrorMessage="A newer version of [ProductName] is already installed." />
    <MediaTemplate />

    <Feature Id="ProductFeature" Title="%s" Level="1">
      <ComponentGroupRef Id="ProductComponents" />
    </Feature>
  </Product>

  <Fragment>
    <Directory Id="TARGETDIR" Name="SourceDir">
      <Directory Id="ProgramFilesFolder">
        <Directory Id="INSTALLFOLDER" Name="%s" />
      </Directory>
    </Directory>
  </Fragment>

  <Fragment>
    <ComponentGroup Id="ProductComponents" Directory="INSTALLFOLDER">
      <Component Id="MainExecutable" Guid="*">
        <File Id="KengaEditorExe" Source="kenga-editor-windows-amd64.exe" KeyPath="yes" />
      </Component>
      <Component Id="CliExecutable" Guid="*">
        <File Id="KengaCliExe" Source="kenga-windows-amd64.exe" />
      </Component>
    </ComponentGroup>
  </Fragment>
</Wix>`, config.Name, config.Version, config.Name, config.Name)
}
