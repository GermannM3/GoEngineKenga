package scene

import (
	"encoding/json"
	"os"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

type Scene struct {
	Name     string        `json:"name"`
	Entities []SceneEntity `json:"entities"`
}

type SceneEntity struct {
	Name string `json:"name"`

	Transform    *ecs.Transform    `json:"transform,omitempty"`
	Camera       *ecs.Camera       `json:"camera,omitempty"`
	MeshRenderer *ecs.MeshRenderer `json:"meshRenderer,omitempty"`
	Light        *ecs.Light        `json:"light,omitempty"`
	Rigidbody    *ecs.Rigidbody    `json:"rigidbody,omitempty"`
	Collider     *ecs.Collider     `json:"collider,omitempty"`
	AudioSource  *ecs.AudioSource  `json:"audioSource,omitempty"`
	UICanvas     *ecs.UICanvas     `json:"uiCanvas,omitempty"`

	// Для префабов
	PrefabID     string `json:"prefabId,omitempty"`    // ID префаба, если это инстанс
	Overrides    map[string]interface{} `json:"overrides,omitempty"` // Переопределения полей
}

func DefaultScene() *Scene {
	return &Scene{
		Name: "Main",
		Entities: []SceneEntity{
			{
				Name: "Camera",
				Transform: &ecs.Transform{
					Position: emath.V3(0, 0, 5),
					Rotation: emath.V3(0, 0, 0),
					Scale:    emath.V3(1, 1, 1),
				},
				Camera: &ecs.Camera{FovYDegrees: 60, Near: 0.1, Far: 1000},
			},
			{
				Name: "Triangle",
				Transform: &ecs.Transform{
					Position: emath.V3(0, 0, 0),
					Rotation: emath.V3(0, 0, 0),
					Scale:    emath.V3(1, 1, 1),
				},
				MeshRenderer: &ecs.MeshRenderer{MeshAssetID: ""}, // v0: procedural triangle
			},
		},
	}
}

func Load(path string) (*Scene, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Scene
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func Save(path string, s *Scene) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (s *Scene) ToWorld() *ecs.World {
	w := ecs.NewWorld()
	for _, se := range s.Entities {
		id := w.CreateEntity(se.Name)
		if se.Transform != nil {
			w.SetTransform(id, *se.Transform)
		}
		if se.Camera != nil {
			w.SetCamera(id, *se.Camera)
		}
		if se.MeshRenderer != nil {
			w.SetMeshRenderer(id, *se.MeshRenderer)
		}
		if se.Light != nil {
			w.SetLight(id, *se.Light)
		}
		if se.Rigidbody != nil {
			w.SetRigidbody(id, *se.Rigidbody)
		}
		if se.Collider != nil {
			w.SetCollider(id, *se.Collider)
		}
		if se.AudioSource != nil {
			w.SetAudioSource(id, *se.AudioSource)
		}
	}
	return w
}

