package convert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConvertIPTIAMToGLTF конвертирует .ipt или .iam в .gltf/.glb через Autodesk Forge.
// Требует FORGE_CLIENT_ID и FORGE_CLIENT_SECRET.
// Для финального шага (SVF→glTF) использует forge-convert-utils (npx), если доступен.
func ConvertIPTIAMToGLTF(inputPath, outputPath string) error {
	ext := strings.ToLower(filepath.Ext(inputPath))
	if ext != ".ipt" && ext != ".iam" {
		return fmt.Errorf("поддерживаются только .ipt и .iam, получено: %s", ext)
	}
	if outputPath == "" {
		outputPath = strings.TrimSuffix(inputPath, ext) + ".glb"
	}
	outExt := strings.ToLower(filepath.Ext(outputPath))
	if outExt != ".gltf" && outExt != ".glb" {
		outputPath = outputPath + ".glb"
	}

	client := NewForgeClient()
	if !client.Configured() {
		return fmt.Errorf("задайте FORGE_CLIENT_ID и FORGE_CLIENT_SECRET (см. docs/CAD_FORMATS.md)")
	}

	urn, err := client.UploadFile(inputPath)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	if err := client.StartConversionJob(urn); err != nil {
		return fmt.Errorf("start job: %w", err)
	}

	if err := client.WaitForJob(urn, defaultJobTimeout); err != nil {
		return fmt.Errorf("wait job: %w", err)
	}

	// forge-convert-utils: forge-convert <urn> --output-folder <dir>
	outDir := filepath.Dir(outputPath)
	cmd := exec.Command("npx", "-y", "forge-convert-utils", urn, "--output-folder", outDir)
	cmd.Env = append(os.Environ(), "FORGE_CLIENT_ID="+client.ClientID, "FORGE_CLIENT_SECRET="+client.ClientSecret)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge-convert-utils (нужен Node.js): %w", err)
	}
	// forge-convert выводит в папку (возможно в подпапку); ищем .gltf/.glb
	var foundPath string
	_ = filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || foundPath != "" {
			return nil
		}
		name := strings.ToLower(info.Name())
		if strings.HasSuffix(name, ".glb") || strings.HasSuffix(name, ".gltf") {
			foundPath = path
		}
		return nil
	})
	if foundPath != "" && filepath.Clean(foundPath) != filepath.Clean(outputPath) {
		data, err := os.ReadFile(foundPath)
		if err == nil {
			_ = os.WriteFile(outputPath, data, 0o644)
			_ = os.Remove(foundPath)
		}
	}
	return nil
}
