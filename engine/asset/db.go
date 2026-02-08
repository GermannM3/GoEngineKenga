package asset

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"goenginekenga/engine/asset/gltf"
	"goenginekenga/engine/convert"
	"goenginekenga/engine/project"
)

type Record struct {
	ID         string    `json:"id"`
	Type       Type      `json:"type"`
	SourcePath string    `json:"sourcePath"`
	Derived    []string  `json:"derived"`
	ImportedAt time.Time `json:"importedAt"`
}

type Index struct {
	Assets []Record `json:"assets"`
}

type Database struct {
	ProjectDir string
	Project    *project.Project
}

func Open(projectDir string) (*Database, error) {
	p, err := project.Load(projectDir)
	if err != nil {
		return nil, err
	}
	return &Database{ProjectDir: projectDir, Project: p}, nil
}

func (db *Database) assetsDirAbs() string {
	return filepath.Join(db.ProjectDir, filepath.FromSlash(db.Project.AssetsDir))
}

func (db *Database) derivedDirAbs() string {
	return filepath.Join(db.ProjectDir, filepath.FromSlash(db.Project.DerivedDir))
}

func (db *Database) indexPathAbs() string {
	return filepath.Join(db.ProjectDir, ".kenga", "assets", "index.json")
}

func (db *Database) EnsureDirs() error {
	if err := os.MkdirAll(db.derivedDirAbs(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(db.indexPathAbs()), 0o755); err != nil {
		return err
	}
	return nil
}

func (db *Database) ImportAll() (*Index, error) {
	if err := db.EnsureDirs(); err != nil {
		return nil, err
	}

	root := db.assetsDirAbs()
	var records []Record

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".meta") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".gltf", ".glb":
			rec, err := db.importGLTF(path)
			if err != nil {
				return err
			}
			records = append(records, *rec)
		case ".ipt", ".iam":
			rec, err := db.importInventor(path)
			if err != nil {
				return err
			}
			if rec != nil {
				records = append(records, *rec)
			}
		default:
			// v0: игнорируем остальные типы
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	idx := &Index{Assets: records}
	if err := db.saveIndex(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (db *Database) importInventor(sourceAbs string) (*Record, error) {
	client := convert.NewForgeClient()
	if !client.Configured() {
		log.Printf("kenga: пропуск %s (задайте FORGE_CLIENT_ID и FORGE_CLIENT_SECRET для конвертации IPT/IAM; см. docs/CAD_FORMATS.md)", sourceAbs)
		return nil, nil
	}
	if err := db.EnsureDirs(); err != nil {
		return nil, err
	}
	meta, err := LoadOrCreateMeta(sourceAbs, TypeGLTF)
	if err != nil {
		return nil, err
	}
	convertedPath := filepath.Join(db.derivedDirAbs(), meta.ID+".glb")
	if err := convert.ConvertIPTIAMToGLTF(sourceAbs, convertedPath); err != nil {
		return nil, fmt.Errorf("convert %s: %w", sourceAbs, err)
	}
	rec, err := db.importGLTF(convertedPath)
	if err != nil {
		return nil, err
	}
	// Сохраняем путь к исходному IPT/IAM для прослеживаемости
	if rel, err := filepath.Rel(db.ProjectDir, sourceAbs); err == nil {
		rec.SourcePath = filepath.ToSlash(rel)
	}
	return rec, nil
}

func (db *Database) importGLTF(sourceAbs string) (*Record, error) {
	meta, err := LoadOrCreateMeta(sourceAbs, TypeGLTF)
	if err != nil {
		return nil, err
	}

	res, err := gltf.ImportFile(sourceAbs)
	if err != nil {
		return nil, err
	}
	if len(res.Meshes) == 0 {
		return nil, fmt.Errorf("no meshes imported from %s", sourceAbs)
	}

	derived := make([]string, 0, len(res.Meshes)+len(res.Materials)+len(res.Textures)+len(res.Skins)+len(res.Animations))

	// Сохраняем скины
	skinRelPaths := make([]string, len(res.Skins))
	for i, s := range res.Skins {
		name := fmt.Sprintf("%s_skin_%d.skeleton.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		sk := Skeleton{
			Name:                s.Name,
			JointNames:          s.JointNames,
			ParentIndices:       s.ParentIndices,
			InverseBindMatrices: s.InverseBindMatrices,
		}
		if err := writeJSONFile(outAbs, sk); err != nil {
			return nil, err
		}
		rel := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name))
		skinRelPaths[i] = rel
		derived = append(derived, rel)
	}

	// Сохраняем анимации
	for i, a := range res.Animations {
		name := fmt.Sprintf("%s_anim_%d.clip.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		clip := gltfAnimationToClip(a, res.NodeNames)
		if err := writeJSONFile(outAbs, clip); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

	// Сохраняем меши
	for i, m := range res.Meshes {
		name := fmt.Sprintf("%s_%d.mesh.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		matIdx := m.MaterialIndex
		if matIdx < 0 || matIdx >= len(res.Materials) {
			matIdx = 0
		}
		matRelPath := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, fmt.Sprintf("%s_%d.material.json", meta.ID, matIdx)))
		mesh := Mesh{
			Name:       m.Name,
			Positions:  m.Positions,
			Normals:    m.Normals,
			UV0:        m.UV0,
			Indices:    m.Indices,
			MaterialID: matRelPath,
			Joints:     m.Joints,
			Weights:    m.Weights,
		}
		if m.SkinIndex >= 0 && m.SkinIndex < len(skinRelPaths) {
			mesh.SkinID = skinRelPaths[m.SkinIndex]
		}
		if err := writeJSONFile(outAbs, mesh); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

		// Сохраняем материалы
	for i, mat := range res.Materials {
		name := fmt.Sprintf("%s_%d.material.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		matCopy := mat
		if i < len(res.BaseColorTexIndex) && res.BaseColorTexIndex[i] >= 0 && res.BaseColorTexIndex[i] < len(res.Textures) {
			texRelPath := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, fmt.Sprintf("%s_%d.texture.json", meta.ID, res.BaseColorTexIndex[i])))
			matCopy.BaseColorTex = texRelPath
		}
		if i < len(res.NormalTexIndex) && res.NormalTexIndex[i] >= 0 && res.NormalTexIndex[i] < len(res.Textures) {
			normRelPath := filepath.ToSlash(filepath.Join(db.Project.DerivedDir, fmt.Sprintf("%s_%d.texture.json", meta.ID, res.NormalTexIndex[i])))
			matCopy.NormalTex = normRelPath
		}
		if err := writeJSONFile(outAbs, matCopy); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

	// Сохраняем текстуры
	for i, tex := range res.Textures {
		name := fmt.Sprintf("%s_%d.texture.json", meta.ID, i)
		outAbs := filepath.Join(db.derivedDirAbs(), name)
		// Конвертируем gltf.Texture в asset.Texture
		assetTex := Texture{
			Name:   tex.Name,
			Width:  tex.Width,
			Height: tex.Height,
			Data:   tex.Data,
			Format: tex.Format,
		}
		if err := writeJSONFile(outAbs, assetTex); err != nil {
			return nil, err
		}
		derived = append(derived, filepath.ToSlash(filepath.Join(db.Project.DerivedDir, name)))
	}

	meta.ImportedAt = time.Now()
	if err := SaveMeta(sourceAbs, meta); err != nil {
		return nil, err
	}

	rel, _ := filepath.Rel(db.ProjectDir, sourceAbs)
	return &Record{
		ID:         meta.ID,
		Type:       meta.Type,
		SourcePath: filepath.ToSlash(rel),
		Derived:    derived,
		ImportedAt: meta.ImportedAt,
	}, nil
}

func (db *Database) saveIndex(idx *Index) error {
	return writeJSONFile(db.indexPathAbs(), idx)
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// gltfAnimationToClip конвертирует gltf.AnimationData в asset.AnimationClip.
func gltfAnimationToClip(a gltf.AnimationData, nodeNames []string) AnimationClip {
	// Группируем каналы по node
	type chanKey struct {
		node int
		path string
	}
	chByNode := make(map[int][]gltf.AnimationChannelData)
	for _, ch := range a.Channels {
		chByNode[ch.NodeIndex] = append(chByNode[ch.NodeIndex], ch)
	}

	// Собираем все уникальные времена
	timeSet := make(map[float32]struct{})
	for _, chs := range chByNode {
		for _, ch := range chs {
			for _, t := range ch.Times {
				timeSet[t] = struct{}{}
			}
		}
	}
	times := make([]float32, 0, len(timeSet))
	for t := range timeSet {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })

	var tracks []AnimationTrack
	for nodeIdx, chs := range chByNode {
		nodeName := "Node"
		if nodeIdx >= 0 && nodeIdx < len(nodeNames) {
			nodeName = nodeNames[nodeIdx]
		}
		kfs := make([]Keyframe, 0, len(times))
		for _, t := range times {
			kf := Keyframe{
				Time:     t,
				Position: [3]float32{0, 0, 0},
				Rotation: [4]float32{0, 0, 0, 1},
				Scale:    [3]float32{1, 1, 1},
			}
			for _, ch := range chs {
				switch ch.Path {
				case "translation":
					if v := sampleVec3(ch.Times, ch.Values, t); v != nil {
						kf.Position = *v
					}
				case "scale":
					if v := sampleVec3(ch.Times, ch.Values, t); v != nil {
						kf.Scale = *v
					}
				case "rotation":
					if v := sampleVec4(ch.Times, ch.Values, t); v != nil {
						kf.Rotation = *v
					}
				}
			}
			kfs = append(kfs, kf)
		}
		tracks = append(tracks, AnimationTrack{NodeName: nodeName, Keyframes: kfs})
	}
	return AnimationClip{
		Name:     a.Name,
		Duration: a.Duration,
		Tracks:   tracks,
		Loop:     true,
	}
}

func sampleVec3(times []float32, values []float32, t float32) *[3]float32 {
	if len(times) == 0 || len(values) < 3 {
		return nil
	}
	if t <= times[0] {
		return &[3]float32{values[0], values[1], values[2]}
	}
	if t >= times[len(times)-1] {
		n := len(times) - 1
		return &[3]float32{values[n*3+0], values[n*3+1], values[n*3+2]}
	}
	for i := 0; i < len(times)-1; i++ {
		if t >= times[i] && t <= times[i+1] {
			dt := times[i+1] - times[i]
			if dt <= 0 {
				return &[3]float32{values[i*3+0], values[i*3+1], values[i*3+2]}
			}
			u := (t - times[i]) / dt
			return &[3]float32{
				values[i*3+0] + u*(values[(i+1)*3+0]-values[i*3+0]),
				values[i*3+1] + u*(values[(i+1)*3+1]-values[i*3+1]),
				values[i*3+2] + u*(values[(i+1)*3+2]-values[i*3+2]),
			}
		}
	}
	return nil
}

func sampleVec4(times []float32, values []float32, t float32) *[4]float32 {
	if len(times) == 0 || len(values) < 4 {
		return nil
	}
	if t <= times[0] {
		return &[4]float32{values[0], values[1], values[2], values[3]}
	}
	if t >= times[len(times)-1] {
		n := len(times) - 1
		return &[4]float32{values[n*4+0], values[n*4+1], values[n*4+2], values[n*4+3]}
	}
	for i := 0; i < len(times)-1; i++ {
		if t >= times[i] && t <= times[i+1] {
			dt := times[i+1] - times[i]
			if dt <= 0 {
				return &[4]float32{values[i*4+0], values[i*4+1], values[i*4+2], values[i*4+3]}
			}
			u := (t - times[i]) / dt
			// Slerp для quaternion
			return &[4]float32{
				values[i*4+0] + u*(values[(i+1)*4+0]-values[i*4+0]),
				values[i*4+1] + u*(values[(i+1)*4+1]-values[i*4+1]),
				values[i*4+2] + u*(values[(i+1)*4+2]-values[i*4+2]),
				values[i*4+3] + u*(values[(i+1)*4+3]-values[i*4+3]),
			}
		}
	}
	return nil
}
