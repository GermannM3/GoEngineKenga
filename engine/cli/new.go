package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"goenginekenga/engine/project"
)

func newNewCommand() *cobra.Command {
	var template string

	cmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Create a new GoEngineKenga project",
		Long: `Create a new GoEngineKenga project with the standard directory structure.

Examples:
  kenga new mygame
  kenga new mygame --template platformer`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			return createProject(projectName, template)
		},
	}

	cmd.Flags().StringVarP(&template, "template", "t", "default", "Project template (default, platformer, topdown, shooter, puzzle, rpg)")

	return cmd
}

func createProject(name, template string) error {
	// Check if directory already exists
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	fmt.Printf("Creating project %q with template %q...\n", name, template)

	// Create directory structure
	dirs := []string{
		name,
		filepath.Join(name, "scenes"),
		filepath.Join(name, "assets"),
		filepath.Join(name, "scripts", "game"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create project.kenga.json
	proj := &project.Project{
		Name:       name,
		AssetsDir:  "assets",
		DerivedDir: ".kenga/derived",
		Scenes:     []string{"scenes/main.scene.json"},
	}

	projPath := filepath.Join(name, "project.kenga.json")
	projData, err := json.MarshalIndent(proj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}
	if err := os.WriteFile(projPath, projData, 0644); err != nil {
		return fmt.Errorf("failed to write project file: %w", err)
	}

	// Create main scene based on template
	scene := createScene(template)
	scenePath := filepath.Join(name, "scenes", "main.scene.json")
	sceneData, err := json.MarshalIndent(scene, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scene: %w", err)
	}
	if err := os.WriteFile(scenePath, sceneData, 0644); err != nil {
		return fmt.Errorf("failed to write scene file: %w", err)
	}

	// Create WASM script template
	scriptPath := filepath.Join(name, "scripts", "game", "main.go")
	scriptContent := createScriptTemplate(template)
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	// Create README
	readmePath := filepath.Join(name, "README.md")
	readmeContent := createReadme(name)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	fmt.Printf("\nProject created successfully!\n\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", name)
	fmt.Printf("  kenga run --project . --scene scenes/main.scene.json --backend ebiten\n")
	fmt.Printf("\nTo build WASM scripts:\n")
	fmt.Printf("  kenga script build --project .\n")

	return nil
}

// SceneFile represents a scene file structure
type SceneFile struct {
	Name     string        `json:"name"`
	Entities []SceneEntity `json:"entities"`
}

// SceneEntity represents an entity in scene file
type SceneEntity struct {
	Name       string                 `json:"name"`
	Components map[string]interface{} `json:"components"`
}

func createScene(template string) *SceneFile {
	scene := &SceneFile{
		Name:     "Main Scene",
		Entities: []SceneEntity{},
	}

	// Camera entity (common to all templates)
	camera := SceneEntity{
		Name: "MainCamera",
		Components: map[string]interface{}{
			"transform": map[string]interface{}{
				"position": map[string]float32{"x": 0, "y": 2, "z": 10},
				"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
				"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
			},
			"camera": map[string]interface{}{
				"fovYDegrees": 60,
				"near":        0.1,
				"far":         1000,
			},
		},
	}
	scene.Entities = append(scene.Entities, camera)

	// Light entity
	light := SceneEntity{
		Name: "DirectionalLight",
		Components: map[string]interface{}{
			"transform": map[string]interface{}{
				"position": map[string]float32{"x": 5, "y": 10, "z": 5},
				"rotation": map[string]float32{"x": -45, "y": 45, "z": 0},
				"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
			},
			"light": map[string]interface{}{
				"kind":      "directional",
				"colorRGB":  map[string]float32{"x": 1, "y": 1, "z": 1},
				"intensity": 1.0,
			},
		},
	}
	scene.Entities = append(scene.Entities, light)

	switch template {
	case "platformer":
		// Player with rigidbody
		player := SceneEntity{
			Name: "Player",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": 2, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
				},
				"rigidbody": map[string]interface{}{
					"mass":       1.0,
					"useGravity": true,
					"drag":       0.1,
				},
				"collider": map[string]interface{}{
					"type":   "box",
					"size":   map[string]float32{"x": 1, "y": 2, "z": 1},
					"center": map[string]float32{"x": 0, "y": 1, "z": 0},
				},
			},
		}
		scene.Entities = append(scene.Entities, player)

		// Ground platform
		ground := SceneEntity{
			Name: "Ground",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": -0.5, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 20, "y": 1, "z": 20},
				},
				"collider": map[string]interface{}{
					"type":   "box",
					"size":   map[string]float32{"x": 20, "y": 1, "z": 20},
					"center": map[string]float32{"x": 0, "y": 0, "z": 0},
				},
			},
		}
		scene.Entities = append(scene.Entities, ground)

	case "shooter":
		player := SceneEntity{
			Name: "Player",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": 1, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
				},
				"rigidbody": map[string]interface{}{"mass": 1.0, "useGravity": false, "drag": 2.0},
				"collider":  map[string]interface{}{"type": "sphere", "radius": 0.5, "center": map[string]float32{"x": 0, "y": 0.5, "z": 0}},
			},
		}
		scene.Entities = append(scene.Entities, player)
		ground := SceneEntity{
			Name: "Ground",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": -0.5, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 50, "y": 1, "z": 50},
				},
				"collider": map[string]interface{}{"type": "box", "size": map[string]float32{"x": 50, "y": 1, "z": 50}, "center": map[string]float32{"x": 0, "y": 0, "z": 0}},
			},
		}
		scene.Entities = append(scene.Entities, ground)

	case "puzzle":
		player := SceneEntity{
			Name: "Player",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": 0, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
				},
				"rigidbody": map[string]interface{}{"mass": 1.0, "useGravity": false},
				"collider":  map[string]interface{}{"type": "box", "size": map[string]float32{"x": 1, "y": 1, "z": 1}, "center": map[string]float32{"x": 0, "y": 0, "z": 0}},
			},
		}
		scene.Entities = append(scene.Entities, player)
		ground := SceneEntity{
			Name: "Ground",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": -0.5, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 20, "y": 1, "z": 20},
				},
				"collider": map[string]interface{}{"type": "box", "size": map[string]float32{"x": 20, "y": 1, "z": 20}, "center": map[string]float32{"x": 0, "y": 0, "z": 0}},
			},
		}
		scene.Entities = append(scene.Entities, ground)

	case "rpg":
		player := SceneEntity{
			Name: "Player",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": 0, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
				},
				"rigidbody": map[string]interface{}{"mass": 1.0, "useGravity": false, "drag": 3.0},
				"collider":  map[string]interface{}{"type": "sphere", "radius": 0.5, "center": map[string]float32{"x": 0, "y": 0, "z": 0}},
			},
		}
		scene.Entities = append(scene.Entities, player)
		ground := SceneEntity{
			Name: "Ground",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": -0.5, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 100, "y": 1, "z": 100},
				},
				"collider": map[string]interface{}{"type": "box", "size": map[string]float32{"x": 100, "y": 1, "z": 100}, "center": map[string]float32{"x": 0, "y": 0, "z": 0}},
			},
		}
		scene.Entities = append(scene.Entities, ground)

	case "topdown":
		// Player without gravity
		player := SceneEntity{
			Name: "Player",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": 0, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
				},
				"rigidbody": map[string]interface{}{
					"mass":       1.0,
					"useGravity": false,
					"drag":       5.0,
				},
				"collider": map[string]interface{}{
					"type":   "sphere",
					"radius": 0.5,
					"center": map[string]float32{"x": 0, "y": 0, "z": 0},
				},
			},
		}
		scene.Entities = append(scene.Entities, player)

	default: // "default"
		// Simple cube entity
		cube := SceneEntity{
			Name: "Cube",
			Components: map[string]interface{}{
				"transform": map[string]interface{}{
					"position": map[string]float32{"x": 0, "y": 0, "z": 0},
					"rotation": map[string]float32{"x": 0, "y": 0, "z": 0},
					"scale":    map[string]float32{"x": 1, "y": 1, "z": 1},
				},
			},
		}
		scene.Entities = append(scene.Entities, cube)
	}

	return scene
}

func createScriptTemplate(template string) string {
	base := `//go:build wasm

package main

import "unsafe"

//go:wasmimport env debugLog
func debugLog(ptr uint32, l uint32)

//go:wasmimport env getInputKey
func getInputKey(key int32) int32

//go:wasmimport env getMouseX
func getMouseX() int32

//go:wasmimport env getMouseY
func getMouseY() int32

func log(msg string) {
	if len(msg) == 0 {
		return
	}
	b := []byte(msg)
	debugLog(uint32(uintptr(unsafe.Pointer(&b[0]))), uint32(len(b)))
}

// Input key constants
const (
	KeyW = 22
	KeyA = 0
	KeyS = 18
	KeyD = 3
	KeySpace = 36
)

`

	switch template {
	case "platformer":
		return base + `//export Update
func Update(dtMillis int32) {
	dt := float32(dtMillis) / 1000.0
	_ = dt

	// Movement
	if getInputKey(KeyA) != 0 {
		log("Moving left\n")
	}
	if getInputKey(KeyD) != 0 {
		log("Moving right\n")
	}
	if getInputKey(KeySpace) != 0 {
		log("Jump!\n")
	}
}

func main() {}
`

	case "topdown":
		return base + `//export Update
func Update(dtMillis int32) {
	dt := float32(dtMillis) / 1000.0
	_ = dt

	// Movement (WASD)
	if getInputKey(KeyW) != 0 {
		log("Moving up\n")
	}
	if getInputKey(KeyS) != 0 {
		log("Moving down\n")
	}
	if getInputKey(KeyA) != 0 {
		log("Moving left\n")
	}
	if getInputKey(KeyD) != 0 {
		log("Moving right\n")
	}
}

func main() {}
`

	case "shooter":
		return base + `//export Update
func Update(dtMillis int32) {
	dt := float32(dtMillis) / 1000.0
	_ = dt

	// WASD move, mouse look, click fire
	if getInputKey(KeyW) != 0 { log("Forward\n") }
	if getInputKey(KeyS) != 0 { log("Back\n") }
	if getInputKey(KeyA) != 0 { log("Strafe left\n") }
	if getInputKey(KeyD) != 0 { log("Strafe right\n") }
	// Use getMouseX/getMouseY for look
}

func main() {}
`

	case "puzzle":
		return base + `//export Update
func Update(dtMillis int32) {
	dt := float32(dtMillis) / 1000.0
	_ = dt

	// Grid move or click-to-select puzzle logic
	if getInputKey(KeyW) != 0 { log("Up\n") }
	if getInputKey(KeyS) != 0 { log("Down\n") }
	if getInputKey(KeyA) != 0 { log("Left\n") }
	if getInputKey(KeyD) != 0 { log("Right\n") }
	if getInputKey(KeySpace) != 0 { log("Interact\n") }
}

func main() {}
`

	case "rpg":
		return base + `//export Update
func Update(dtMillis int32) {
	dt := float32(dtMillis) / 1000.0
	_ = dt

	// Overworld move, dialogue, inventory
	if getInputKey(KeyW) != 0 { log("North\n") }
	if getInputKey(KeyS) != 0 { log("South\n") }
	if getInputKey(KeyA) != 0 { log("West\n") }
	if getInputKey(KeyD) != 0 { log("East\n") }
	if getInputKey(KeySpace) != 0 { log("Interact / Confirm\n") }
}

func main() {}
`

	default:
		return base + `//export Update
func Update(dtMillis int32) {
	_ = dtMillis
	// Your game logic here
}

func main() {}
`
	}
}

func createReadme(projectName string) string {
	return fmt.Sprintf(`# %s

Game created with GoEngineKenga.

## Quick Start

Run the game:
`+"```bash"+`
kenga run --project . --scene scenes/main.scene.json --backend ebiten
`+"```"+`

Build WASM scripts:
`+"```bash"+`
kenga script build --project .
`+"```"+`

## Project Structure

- `+"`scenes/`"+` - Scene files (JSON)
- `+"`assets/`"+` - Game assets (models, textures, audio)
- `+"`scripts/game/`"+` - WASM scripts (Go/TinyGo)
- `+"`project.kenga.json`"+` - Project configuration

## Documentation

See [GoEngineKenga documentation](https://github.com/GermannM3/GoEngineKenga)
`, projectName)
}
