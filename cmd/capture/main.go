// Команда capture — отрисовывает сцену игры как её рисует Ebiten-окно
// (render.Renderer3D → software rasterizer) и сохраняет PNG. Для визуальной
// диагностики без открытия окна. Не часть релиза.
package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"goenginekenga/engine/asset"
	ebitenrender "goenginekenga/engine/render/ebiten"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
)

func main() {
	project := flag.String("project", "", "путь к проекту игры (samples/cyber_ninja)")
	scenePath := flag.String("scene", "scenes/main.scene.json", "сцена относительно проекта")
	outPath := flag.String("out", "shot.png", "выходной PNG")
	width := flag.Int("w", 960, "ширина")
	height := flag.Int("h", 540, "высота")
	flag.Parse()

	if *project == "" {
		log.Fatal("-project обязателен")
	}

	full := filepath.Join(*project, *scenePath)
	sc, err := scene.Load(full)
	if err != nil {
		log.Fatalf("scene.Load(%s): %v", full, err)
	}

	rt := runtime.NewFromScene(sc)
	rt.StartPlay()
	for i := 0; i < 5; i++ {
		rt.Step()
	}
	world, err := rt.ActiveWorld()
	if err != nil {
		log.Fatalf("ActiveWorld: %v", err)
	}

	resolver, err := asset.NewResolver(*project)
	if err != nil {
		log.Fatalf("NewResolver: %v", err)
	}

	// --- Диагностика: что видим и что резолвится ---
	log.Printf("entities=%d", len(world.Entities()))
	shown := 0
	for _, id := range world.Entities() {
		mr, hasMR := world.GetMeshRenderer(id)
		if !hasMR {
			continue
		}
		if shown >= 8 {
			break
		}
		tr, _ := world.GetTransform(id)
		name := "?"
		if mr.MeshAssetID != "" {
			if m, err := resolver.ResolveMeshByAssetID(mr.MeshAssetID); err == nil && m != nil {
				name = fmt.Sprintf("%s(v=%d)", m.Name, len(m.Positions)/3)
			} else if err != nil {
				name = "RESOLVE-ERR:" + err.Error()
			} else {
				name = "nil-mesh"
			}
		}
		log.Printf("  mesh id=%v pos=(%.1f,%.1f,%.1f) scale=(%.1f,%.1f,%.1f) asset=%s %s",
			id, tr.Position.X, tr.Position.Y, tr.Position.Z,
			tr.Scale.X, tr.Scale.Y, tr.Scale.Z, mr.MeshAssetID, name)
		shown++
	}
	for _, id := range world.Entities() {
		c, ok := world.GetCamera(id)
		if !ok {
			continue
		}
		tr, _ := world.GetTransform(id)
		log.Printf("CAMERA id=%v pos=(%.1f,%.1f,%.1f) fov=%.0f far=%.0f",
			id, tr.Position.X, tr.Position.Y, tr.Position.Z, c.FovYDegrees, c.Far)
	}

	r3d := ebitenrender.NewRenderer3D(*width, *height)
	r3d.RenderWorld(world, resolver, color.RGBA{R: 14, G: 18, B: 26, A: 255})
	img := r3d.RenderToImage()
	if img == nil {
		log.Fatal("RenderToImage вернул nil")
	}

	f, err := os.Create(*outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Printf("OK %dx%d → %s", img.Bounds().Dx(), img.Bounds().Dy(), *outPath)
}