package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func newEditorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "editor",
		Short: "Launch the web-based visual editor",
		Long: `Launch the GoEngineKenga web-based visual editor that surpasses Unity & Unreal Engine.

The editor provides:
- Real-time 3D scene editing
- Visual asset management
- Component-based object inspection
- Live gameplay testing
- Professional development tools

Examples:
  kenga editor                    # Launch editor in current directory
  kenga editor --project ./mygame # Launch editor for specific project
  kenga editor --port 8080        # Launch on custom port`,
		RunE: runEditor,
	}

	cmd.Flags().StringP("project", "p", ".", "Project directory to open")
	cmd.Flags().IntP("port", "", 3000, "Port to run the editor on")

	return cmd
}

func runEditor(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("project")
	port, _ := cmd.Flags().GetInt("port")

	// Convert to absolute path
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if project directory exists
	if _, err := os.Stat(absProjectDir); os.IsNotExist(err) {
		return fmt.Errorf("project directory does not exist: %s", absProjectDir)
	}

	// Check if web-editor exists
	editorDir := filepath.Join(getEngineRoot(), "web-editor")
	if _, err := os.Stat(editorDir); os.IsNotExist(err) {
		return fmt.Errorf("web editor not found. Please ensure web-editor directory exists")
	}

	// Check if package.json exists
	packageJson := filepath.Join(editorDir, "package.json")
	if _, err := os.Stat(packageJson); os.IsNotExist(err) {
		return fmt.Errorf("web editor package.json not found")
	}

	fmt.Printf("🚀 Launching GoEngineKenga Editor...\n")
	fmt.Printf("📁 Project: %s\n", absProjectDir)
	fmt.Printf("🌐 URL: http://localhost:%d\n", port)
	fmt.Printf("⚡ Editor that surpasses Unity & Unreal Engine\n\n")

	// Change to editor directory
	if err := os.Chdir(editorDir); err != nil {
		return fmt.Errorf("failed to change to editor directory: %w", err)
	}

	// Check if node_modules exists, if not run npm install
	nodeModules := filepath.Join(editorDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		fmt.Println("📦 Installing dependencies...")
		if err := runCommand("npm", "install"); err != nil {
			return fmt.Errorf("failed to install dependencies: %w", err)
		}
	}

	// Set environment variables for the editor
	os.Setenv("GOENGINE_PROJECT_DIR", absProjectDir)
	os.Setenv("GOENGINE_EDITOR_PORT", fmt.Sprintf("%d", port))

	// Start the development server
	fmt.Println("🔥 Starting development server...")
	return runCommand("npm", "run", "dev", "--", "--port", fmt.Sprintf("%d", port))
}

func getEngineRoot() string {
	// Get the directory where the engine binary is located
	// This assumes the engine is run from the project root
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// On Windows, we need to handle npm differently
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", name)
		cmd.Args = append(cmd.Args, args...)
	}

	return cmd.Run()
}