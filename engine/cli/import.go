package cli

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/project"
	"goenginekenga/engine/scene"
)

func newImportCommand() *cobra.Command {
	var projectDir string
	var autoAssign bool
	var scenePath string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import project assets into derived cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := asset.Open(projectDir)
			if err != nil {
				return err
			}
			idx, err := db.ImportAll()
			if err != nil {
				return err
			}

			if autoAssign {
				return autoAssignImportedMeshes(projectDir, scenePath, idx)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "project", ".", "Project directory")
	cmd.Flags().BoolVar(&autoAssign, "auto-assign", true, "Auto-assign imported glTF asset IDs to matching entities with empty MeshAssetID")
	cmd.Flags().StringVar(&scenePath, "scene", "", "Scene path to auto-assign (relative to project). Default: first from project.kenga.json or scenes/main.scene.json")
	return cmd
}

func autoAssignImportedMeshes(projectDir, scenePath string, idx *asset.Index) error {
	if idx == nil {
		return nil
	}

	// Выбираем сцену по умолчанию.
	if scenePath == "" {
		if p, err := project.Load(projectDir); err == nil {
			if len(p.Scenes) > 0 {
				scenePath = p.Scenes[0]
			}
		}
		if scenePath == "" {
			// common default
			scenePath = filepath.ToSlash(filepath.Join("scenes", "main.scene.json"))
		}
	}

	sceneAbs := filepath.Join(projectDir, filepath.FromSlash(scenePath))
	sc, err := scene.Load(sceneAbs)
	if err != nil {
		// если сцена не существует — не считаем это фатальным для import
		return nil
	}

	changed := false
	for _, rec := range idx.Assets {
		if rec.Type != asset.TypeGLTF {
			continue
		}
		// Имя ассета (без расширения) используем для эвристического матчинга.
		base := filepath.Base(filepath.FromSlash(rec.SourcePath))
		name := strings.TrimSuffix(base, filepath.Ext(base))
		nameLower := strings.ToLower(name)

		// Собираем кандидатов.
		candidates := make([]*scene.SceneEntity, 0, 4)
		for i := range sc.Entities {
			e := &sc.Entities[i]
			if e.MeshRenderer == nil {
				continue
			}
			if e.MeshRenderer.MeshAssetID != "" {
				continue
			}
			if strings.Contains(strings.ToLower(e.Name), nameLower) {
				candidates = append(candidates, e)
			}
		}

		// Чтобы не делать неожиданных массовых изменений — назначаем только если кандидат ровно один.
		if len(candidates) == 1 {
			candidates[0].MeshRenderer.MeshAssetID = rec.ID
			changed = true
		}
	}

	if changed {
		return scene.Save(sceneAbs, sc)
	}
	return nil
}
