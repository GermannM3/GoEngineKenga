package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/project"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
)

// Run запускает редактор v0.1 (Hierarchy/Inspector/Content/Console/Viewport + Play/Stop).
func Run() {
	a := app.New()
	w := a.NewWindow("GoEngineKenga Editor")
	w.Resize(fyne.NewSize(1200, 800))

	ed := newEditor()

	status := widget.NewLabel("Project: " + ed.projectDir)
	viewport := widget.NewMultiLineEntry()
	viewport.SetText("SceneViewport (v0): 3D preview позже.\nСейчас тут показываем информацию о выбранном объекте/ассете.")
	viewport.Disable()
	gizmoBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() { ed.nudgeSelected(0, 0.1, 0, viewport) }),
		widget.NewToolbarAction(theme.ContentRemoveIcon(), func() { ed.nudgeSelected(0, -0.1, 0, viewport) }),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.NavigateNextIcon(), func() { ed.nudgeSelected(0.1, 0, 0, viewport) }),
		widget.NewToolbarAction(theme.NavigateBackIcon(), func() { ed.nudgeSelected(-0.1, 0, 0, viewport) }),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MoveUpIcon(), func() { ed.nudgeSelected(0, 0, 0.1, viewport) }),
		widget.NewToolbarAction(theme.MoveDownIcon(), func() { ed.nudgeSelected(0, 0, -0.1, viewport) }),
	)

	console := widget.NewMultiLineEntry()
	console.Disable()
	ed.log = func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		console.SetText(console.Text + msg)
	}

	// ---------- Hierarchy ----------
	hierarchy := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "root" {
				out := make([]widget.TreeNodeID, 0, len(ed.sc.Scene.Entities))
				for i := range ed.sc.Scene.Entities {
					out = append(out, fmt.Sprintf("e:%d", i))
				}
				return out
			}
			return nil
		},
		func(uid widget.TreeNodeID) bool {
			return uid == "root"
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if uid == "root" {
				lbl.SetText("Scene")
				return
			}
			if strings.HasPrefix(uid, "e:") {
				i := mustAtoi(strings.TrimPrefix(uid, "e:"))
				if i >= 0 && i < len(ed.sc.Scene.Entities) {
					lbl.SetText(ed.sc.Scene.Entities[i].Name)
				} else {
					lbl.SetText(uid)
				}
				return
			}
			lbl.SetText(uid)
		},
	)
	hierarchy.Select("root")

	// ---------- Inspector ----------
	inspectorTitle := widget.NewLabel("Inspector")

	posX := widget.NewEntry()
	posY := widget.NewEntry()
	posZ := widget.NewEntry()
	rotX := widget.NewEntry()
	rotY := widget.NewEntry()
	rotZ := widget.NewEntry()
	sclX := widget.NewEntry()
	sclY := widget.NewEntry()
	sclZ := widget.NewEntry()

	applyTransform := func() {
		if ed.selectedEntityIndex < 0 || ed.selectedEntityIndex >= len(ed.sc.Scene.Entities) {
			return
		}
		e := &ed.sc.Scene.Entities[ed.selectedEntityIndex]
		if e.Transform == nil {
			return
		}
		e.Transform.Position.X = parseFloat32(posX.Text)
		e.Transform.Position.Y = parseFloat32(posY.Text)
		e.Transform.Position.Z = parseFloat32(posZ.Text)
		e.Transform.Rotation.X = parseFloat32(rotX.Text)
		e.Transform.Rotation.Y = parseFloat32(rotY.Text)
		e.Transform.Rotation.Z = parseFloat32(rotZ.Text)
		e.Transform.Scale.X = parseFloat32(sclX.Text)
		e.Transform.Scale.Y = parseFloat32(sclY.Text)
		e.Transform.Scale.Z = parseFloat32(sclZ.Text)

		ed.rebuildRuntimeFromScene()
		ed.updateViewport(viewport)
	}

	for _, entry := range []*widget.Entry{posX, posY, posZ, rotX, rotY, rotZ, sclX, sclY, sclZ} {
		entry.OnSubmitted = func(string) { applyTransform() }
	}

	inspector := container.NewVBox(
		inspectorTitle,
		widget.NewForm(
			widget.NewFormItem("Pos X", posX),
			widget.NewFormItem("Pos Y", posY),
			widget.NewFormItem("Pos Z", posZ),
			widget.NewFormItem("Rot X", rotX),
			widget.NewFormItem("Rot Y", rotY),
			widget.NewFormItem("Rot Z", rotZ),
			widget.NewFormItem("Scl X", sclX),
			widget.NewFormItem("Scl Y", sclY),
			widget.NewFormItem("Scl Z", sclZ),
		),
		widget.NewButton("Apply Transform", applyTransform),
	)

	// ---------- Content Browser ----------
	assetsTitle := widget.NewLabel("Content")
	assetsList := widget.NewList(
		func() int { return len(ed.assets) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(ed.assets) {
				return
			}
			obj.(*widget.Label).SetText(ed.assets[i].SourcePath)
		},
	)
	assetsList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(ed.assets) {
			return
		}
		ed.selectedAssetIndex = id
		ed.updateViewport(viewport)
	}

	leftPanel := container.NewVSplit(
		container.NewBorder(nil, nil, nil, nil, hierarchy),
		container.NewBorder(assetsTitle, nil, nil, nil, assetsList),
	)
	leftPanel.Offset = 0.6

	// ---------- Toolbar ----------
	play := func() {
		ed.rt.StartPlay()
		ed.updateViewport(viewport)
		ed.log("runtime: Play\n")
	}
	stop := func() {
		ed.rt.StopPlay()
		ed.updateViewport(viewport)
		ed.log("runtime: Stop\n")
	}
	doImport := func() {
		if err := ed.importAssets(); err != nil {
			ed.log("import error: %v\n", err)
		} else {
			assetsList.Refresh()
			ed.log("import: ok (%d assets)\n", len(ed.assets))
		}
		ed.updateViewport(viewport)
	}
	saveScene := func() {
		if err := scene.Save(ed.scenePathAbs, ed.sc.Scene); err != nil {
			ed.log("save error: %v\n", err)
		} else {
			ed.log("scene saved: %s\n", filepath.ToSlash(ed.scenePathAbs))
		}
	}

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DownloadIcon(), doImport),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MediaPlayIcon(), play),
		widget.NewToolbarAction(theme.MediaStopIcon(), stop),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), saveScene),
	)

	// ---------- Selection wiring ----------
	hierarchy.OnSelected = func(uid widget.TreeNodeID) {
		if strings.HasPrefix(uid, "e:") {
			ed.selectedEntityIndex = mustAtoi(strings.TrimPrefix(uid, "e:"))
			ed.selectedAssetIndex = -1
			// заполнить инспектор
			if ed.selectedEntityIndex >= 0 && ed.selectedEntityIndex < len(ed.sc.Scene.Entities) {
				e := ed.sc.Scene.Entities[ed.selectedEntityIndex]
				inspectorTitle.SetText("Inspector: " + e.Name)
				if e.Transform != nil {
					posX.SetText(fmt.Sprintf("%g", e.Transform.Position.X))
					posY.SetText(fmt.Sprintf("%g", e.Transform.Position.Y))
					posZ.SetText(fmt.Sprintf("%g", e.Transform.Position.Z))
					rotX.SetText(fmt.Sprintf("%g", e.Transform.Rotation.X))
					rotY.SetText(fmt.Sprintf("%g", e.Transform.Rotation.Y))
					rotZ.SetText(fmt.Sprintf("%g", e.Transform.Rotation.Z))
					sclX.SetText(fmt.Sprintf("%g", e.Transform.Scale.X))
					sclY.SetText(fmt.Sprintf("%g", e.Transform.Scale.Y))
					sclZ.SetText(fmt.Sprintf("%g", e.Transform.Scale.Z))
				}
			}
			ed.updateViewport(viewport)
		}
	}

	center := container.NewBorder(status, gizmoBar, nil, nil, viewport)

	hs := container.NewHSplit(leftPanel, center)
	hs.Offset = 0.25
	main := container.NewHSplit(hs, inspector)
	main.Offset = 0.8

	w.SetContent(container.NewBorder(toolbar, container.NewVSplit(widget.NewLabel("Console"), console), nil, nil, main))
	w.ShowAndRun()
}

type editorState struct {
	projectDir   string
	project      *project.Project
	scenePathAbs string

	sc *sceneContainer
	rt *runtime.Runtime

	assets              []asset.Record
	selectedAssetIndex  int
	selectedEntityIndex int

	log func(string, ...any)
}

type sceneContainer struct {
	Scene *scene.Scene
}

func newEditor() *editorState {
	pdir := defaultProjectDir()
	p, _ := project.Load(pdir)

	sceneAbs := filepath.Join(pdir, "scenes", "main.scene.json")
	s, err := scene.Load(sceneAbs)
	if err != nil {
		s = scene.DefaultScene()
		_ = os.MkdirAll(filepath.Dir(sceneAbs), 0o755)
		_ = scene.Save(sceneAbs, s)
	}

	ed := &editorState{
		projectDir:          pdir,
		project:             p,
		scenePathAbs:        sceneAbs,
		sc:                  &sceneContainer{Scene: s},
		rt:                  runtime.NewFromScene(s),
		selectedAssetIndex:  -1,
		selectedEntityIndex: -1,
	}
	_ = ed.importAssets() // best-effort
	return ed
}

func (ed *editorState) rebuildRuntimeFromScene() {
	// v0: просто пересобираем runtime из Scene, чтобы изменения из инспектора отражались.
	ed.rt = runtime.NewFromScene(ed.sc.Scene)
}

func (ed *editorState) importAssets() error {
	db, err := asset.Open(ed.projectDir)
	if err != nil {
		return err
	}
	idx, err := db.ImportAll()
	if err != nil {
		return err
	}
	ed.assets = idx.Assets
	return nil
}

func (ed *editorState) updateViewport(viewport *widget.Entry) {
	var b strings.Builder
	b.WriteString("SceneViewport (v0)\n")
	b.WriteString("Mode: ")
	if ed.rt.Mode == runtime.ModePlay {
		b.WriteString("Play\n\n")
	} else {
		b.WriteString("Edit\n\n")
	}

	if ed.selectedEntityIndex >= 0 && ed.selectedEntityIndex < len(ed.sc.Scene.Entities) {
		e := ed.sc.Scene.Entities[ed.selectedEntityIndex]
		b.WriteString("Selected Entity:\n")
		b.WriteString("  Name: " + e.Name + "\n")
		if e.MeshRenderer != nil && e.MeshRenderer.MeshAssetID != "" {
			b.WriteString("  MeshAssetID: " + e.MeshRenderer.MeshAssetID + "\n")
		}
	}

	if ed.selectedAssetIndex >= 0 && ed.selectedAssetIndex < len(ed.assets) {
		a := ed.assets[ed.selectedAssetIndex]
		b.WriteString("\nSelected Asset:\n")
		b.WriteString("  Source: " + a.SourcePath + "\n")
		b.WriteString("  ID: " + a.ID + "\n")
		b.WriteString("  Type: " + string(a.Type) + "\n")
		for i, d := range a.Derived {
			b.WriteString(fmt.Sprintf("  Derived[%d]: %s\n", i, d))
			if strings.HasSuffix(d, ".mesh.json") {
				m, err := asset.LoadMesh(filepath.Join(ed.projectDir, filepath.FromSlash(d)))
				if err == nil {
					b.WriteString(fmt.Sprintf("    Mesh: %s\n", m.Name))
					b.WriteString(fmt.Sprintf("    Vertices: %d\n", len(m.Positions)/3))
					b.WriteString(fmt.Sprintf("    Indices: %d\n", len(m.Indices)))
				}
			}
		}
	}

	viewport.SetText(b.String())
}

// nudgeSelected — простой «gizmo» v0: подвинуть Transform на шаг.
func (ed *editorState) nudgeSelected(dx, dy, dz float32, viewport *widget.Entry) {
	if ed.selectedEntityIndex < 0 || ed.selectedEntityIndex >= len(ed.sc.Scene.Entities) {
		return
	}
	e := &ed.sc.Scene.Entities[ed.selectedEntityIndex]
	if e.Transform == nil {
		return
	}
	e.Transform.Position.X += dx
	e.Transform.Position.Y += dy
	e.Transform.Position.Z += dz
	ed.rebuildRuntimeFromScene()
	ed.updateViewport(viewport)
	if ed.log != nil {
		ed.log("gizmo: nudge (%.2f, %.2f, %.2f)\n", dx, dy, dz)
	}
}

func defaultProjectDir() string {
	// Если есть sample — открываем его.
	p := filepath.Join(".", "samples", "hello", "project.kenga.json")
	if _, err := os.Stat(p); err == nil {
		return filepath.Dir(p)
	}
	return "."
}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseFloat32(s string) float32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0
	}
	return float32(f)
}
