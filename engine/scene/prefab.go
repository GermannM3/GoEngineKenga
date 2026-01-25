package scene

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"goenginekenga/engine/ecs"
	emath "goenginekenga/engine/math"
)

// Prefab представляет переиспользуемый шаблон объекта
type Prefab struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Entities []PrefabEntity `json:"entities"`
}

// PrefabEntity представляет сущность в префабе
type PrefabEntity struct {
	Name string `json:"name"`
	SceneEntity
}

// LoadPrefab загружает префаб из файла
func LoadPrefab(path string) (*Prefab, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var prefab Prefab
	if err := json.Unmarshal(data, &prefab); err != nil {
		return nil, err
	}

	return &prefab, nil
}

// SavePrefab сохраняет префаб в файл
func SavePrefab(path string, prefab *Prefab) error {
	data, err := json.MarshalIndent(prefab, "", "  ")
	if err != nil {
		return err
	}

	// Создаем директорию если не существует
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Instantiate создает инстанс префаба в мире
func (p *Prefab) Instantiate(world *ecs.World, position emath.Vec3, overrides map[string]interface{}) ecs.EntityID {
	rootEntity := ecs.EntityID(0)

	for i, prefabEntity := range p.Entities {
		entityID := world.CreateEntity(fmt.Sprintf("%s_%s_%d", p.Name, prefabEntity.Name, i))

		// Копируем все компоненты
		if prefabEntity.Transform != nil {
			transform := *prefabEntity.Transform
			// Применяем смещение позиции для root entity
			if i == 0 {
				rootEntity = entityID
				transform.Position.X += position.X
				transform.Position.Y += position.Y
				transform.Position.Z += position.Z
			}
			world.SetTransform(entityID, transform)
		}

		if prefabEntity.Camera != nil {
			world.SetCamera(entityID, *prefabEntity.Camera)
		}

		if prefabEntity.MeshRenderer != nil {
			mr := *prefabEntity.MeshRenderer
			// Применяем overrides если есть
			if override, ok := overrides["meshAssetId"]; ok {
				if assetID, ok := override.(string); ok {
					mr.MeshAssetID = assetID
				}
			}
			if override, ok := overrides["materialAssetId"]; ok {
				if assetID, ok := override.(string); ok {
					mr.MaterialAssetID = assetID
				}
			}
			world.SetMeshRenderer(entityID, mr)
		}

		if prefabEntity.Light != nil {
			world.SetLight(entityID, *prefabEntity.Light)
		}

		if prefabEntity.Rigidbody != nil {
			rb := *prefabEntity.Rigidbody
			// Применяем overrides
			if override, ok := overrides["mass"]; ok {
				if mass, ok := override.(float64); ok {
					rb.Mass = float32(mass)
				}
			}
			if override, ok := overrides["useGravity"]; ok {
				if gravity, ok := override.(bool); ok {
					rb.UseGravity = gravity
				}
			}
			world.SetRigidbody(entityID, rb)
		}

		if prefabEntity.Collider != nil {
			collider := *prefabEntity.Collider
			// Применяем overrides для размера коллайдера
			if override, ok := overrides["colliderSize"]; ok {
				if sizeData, ok := override.(map[string]interface{}); ok {
					if x, ok := sizeData["x"].(float64); ok {
						collider.Size.X = float32(x)
					}
					if y, ok := sizeData["y"].(float64); ok {
						collider.Size.Y = float32(y)
					}
					if z, ok := sizeData["z"].(float64); ok {
						collider.Size.Z = float32(z)
					}
				}
			}
			world.SetCollider(entityID, collider)
		}

		if prefabEntity.AudioSource != nil {
			as := *prefabEntity.AudioSource
			// Применяем overrides
			if override, ok := overrides["volume"]; ok {
				if volume, ok := override.(float64); ok {
					as.Volume = float32(volume)
				}
			}
			if override, ok := overrides["loop"]; ok {
				if loop, ok := override.(bool); ok {
					as.Loop = loop
				}
			}
			if override, ok := overrides["spatial"]; ok {
				if spatial, ok := override.(bool); ok {
					as.Spatial = spatial
				}
			}
			world.SetAudioSource(entityID, as)
		}
	}

	return rootEntity
}

// CreatePrefabFromEntities создает префаб из списка сущностей
func CreatePrefabFromEntities(world *ecs.World, entityIDs []ecs.EntityID, name string) *Prefab {
	prefab := &Prefab{
		ID:       generatePrefabID(),
		Name:     name,
		Entities: make([]PrefabEntity, 0, len(entityIDs)),
	}

	for _, entityID := range entityIDs {
		prefabEntity := PrefabEntity{
			Name: world.Name(entityID),
		}

		if transform, ok := world.GetTransform(entityID); ok {
			prefabEntity.Transform = &transform
		}

		if camera, ok := world.GetCamera(entityID); ok {
			prefabEntity.Camera = &camera
		}

		if meshRenderer, ok := world.GetMeshRenderer(entityID); ok {
			prefabEntity.MeshRenderer = &meshRenderer
		}

		if light, ok := world.GetLight(entityID); ok {
			prefabEntity.Light = &light
		}

		if rigidbody, ok := world.GetRigidbody(entityID); ok {
			prefabEntity.Rigidbody = &rigidbody
		}

		if collider, ok := world.GetCollider(entityID); ok {
			prefabEntity.Collider = &collider
		}

		if audioSource, ok := world.GetAudioSource(entityID); ok {
			prefabEntity.AudioSource = &audioSource
		}

		prefab.Entities = append(prefab.Entities, prefabEntity)
	}

	return prefab
}

// generatePrefabID генерирует уникальный ID для префаба
func generatePrefabID() string {
	// В реальности здесь должна быть более надежная генерация ID
	// Пока используем простой счетчик или UUID
	return fmt.Sprintf("prefab_%d", len(prefabRegistry))
}

// prefabRegistry - глобальный реестр префабов (для простоты)
var prefabRegistry = make(map[string]*Prefab)

// RegisterPrefab регистрирует префаб в реестре
func RegisterPrefab(prefab *Prefab) {
	prefabRegistry[prefab.ID] = prefab
}

// GetPrefab получает префаб из реестра
func GetPrefab(id string) (*Prefab, bool) {
	prefab, ok := prefabRegistry[id]
	return prefab, ok
}
