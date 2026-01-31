package editor

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goenginekenga/engine/asset"
	"goenginekenga/engine/project"
	"goenginekenga/engine/runtime"
	"goenginekenga/engine/scene"
)

// Цвета панелей в стиле Unity/Unreal — читаемые, без чисто чёрного экрана.
var (
	viewportBgColor   = color.NRGBA{R: 0x2d, G: 0x2d, B: 0x30, A: 255} // тёмно-серый
	panelHeaderColor  = color.NRGBA{R: 0x3c, G: 0x3c, B: 0x3c, A: 255}
	sceneTitleColor   = color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 255}
)

// Run запускает редактор (Hierarchy / Inspector / Content / Console / Viewport + Play/Stop).
func Run() {
	a := app.New()
	// Светлая тема по умолчанию — избегаем чисто чёрного экрана.
	a.Settings().SetTheme(theme.LightTheme())
	w := a.NewWindow("GoEngineKenga Editor")
	w.Resize(fyne.NewSize(1280, 820))

	ed := newEditor()

	// ---------- Строка статуса ----------
	status := widget.NewLabel("Project: " + filepath.Base(ed.projectDir))
	status.TextStyle = fyne.TextStyle{Bold: true}

	// ---------- Viewport: не чёрный экран, а панель с фоном и подписью ----------
	viewportBg := canvas.NewRectangle(viewportBgColor)
	viewportBg.SetMinSize(fyne.NewSize(400, 300))
	sceneTitle := canvas.NewText("Scene", sceneTitleColor)
	sceneTitle.TextSize = 18
	sceneTitle.TextStyle = fyne.TextStyle{Bold: true}
	sceneSubtitle := canvas.NewText("3D view — embed coming soon", color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 255})
	sceneSubtitle.TextSize = 12
	viewportPlaceholder := container.NewStack(
		viewportBg,
		container.NewCenter(container.NewVBox(sceneTitle, sceneSubtitle)),
	)
	// Информация о выбранном объекте/режиме — в отдельном читаемом поле под viewport.
	viewportInfo := widget.NewMultiLineEntry()
	viewportInfo.SetPlaceHolder("Select an object in Hierarchy or an asset in Content…")
	viewportInfo.Disable()
	viewportInfo.Wrapping = fyne.TextWrapWord
	ed.updateViewport = func() {
		viewportInfo.SetText(ed.viewportInfoText())
	}
	ed.updateViewport()
	viewportStack := container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), viewportInfo), nil, nil, viewportPlaceholder)

	gizmoBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() { ed.nudgeSelected(0, 0.1, 0) }),
		widget.NewToolbarAction(theme.ContentRemoveIcon(), func() { ed.nudgeSelected(0, -0.1, 0) }),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.NavigateNextIcon(), func() { ed.nudgeSelected(0.1, 0, 0) }),
		widget.NewToolbarAction(theme.NavigateBackIcon(), func() { ed.nudgeSelected(-0.1, 0, 0) }),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MoveUpIcon(), func() { ed.nudgeSelected(0, 0, 0.1) }),
		widget.NewToolbarAction(theme.MoveDownIcon(), func() { ed.nudgeSelected(0, 0, -0.1) }),
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
		ed.updateViewport()
	}

	for _, entry := range []*widget.Entry{posX, posY, posZ, rotX, rotY, rotZ, sclX, sclY, sclZ} {
		entry.OnSubmitted = func(string) { applyTransform() }
	}

	inspectorTitle.TextStyle = fyne.TextStyle{Bold: true}
	inspector := container.NewVBox(
		container.NewPadded(inspectorTitle),
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
		ed.updateViewport()
	}

	// Заголовки панелей в стиле Unity (жирный текст).
	hierarchyTitle := widget.NewLabel("Hierarchy")
	hierarchyTitle.TextStyle = fyne.TextStyle{Bold: true}
	contentTitle := widget.NewLabel("Content")
	contentTitle.TextStyle = fyne.TextStyle{Bold: true}
	leftPanel := container.NewVSplit(
		container.NewBorder(container.NewPadded(hierarchyTitle), nil, nil, nil, hierarchy),
		container.NewBorder(container.NewPadded(contentTitle), nil, nil, nil, assetsList),
	)
	leftPanel.Offset = 0.6

	// ---------- Toolbar ----------
	play := func() {
		ed.rt.StartPlay()
		ed.updateViewport()
		ed.log("runtime: Play\n")
	}
	stop := func() {
		ed.rt.StopPlay()
		ed.updateViewport()
		ed.log("runtime: Stop\n")
	}
	runExternal := func() {
		if ed.runtimeProc != nil {
			ed.log("external runtime: already running (pid=%d)\n", ed.runtimeProc.Process.Pid)
			return
		}
		sceneRel, err := filepath.Rel(ed.projectDir, ed.scenePathAbs)
		if err != nil {
			sceneRel = ed.scenePathAbs
		}
		// Запускаем установленный CLI-рантайм.
		// Путь к бинарнику можно задать через переменную окружения KENGA_CLI,
		// иначе используется просто "kenga" (из PATH).
		kengaBin := os.Getenv("KENGA_CLI")
		if kengaBin == "" {
			kengaBin = "kenga"
		}
		cmd := exec.Command(kengaBin,
			"run",
			"--project", ed.projectDir,
			"--scene", filepath.ToSlash(sceneRel),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			ed.log("external runtime error: %v\n", err)
			return
		}
		ed.runtimeProc = cmd
		ed.log("external runtime: started (pid=%d)\n", cmd.Process.Pid)
		go func() {
			_ = cmd.Wait()
			ed.log("external runtime: exited\n")
			ed.runtimeProc = nil
		}()
	}
	stopExternal := func() {
		if ed.runtimeProc == nil || ed.runtimeProc.Process == nil {
			ed.log("external runtime: not running\n")
			return
		}
		if err := ed.runtimeProc.Process.Kill(); err != nil {
			ed.log("external runtime kill error: %v\n", err)
			return
		}
		ed.log("external runtime: killed\n")
		ed.runtimeProc = nil
	}
	doImport := func() {
		if err := ed.importAssets(); err != nil {
			ed.log("import error: %v\n", err)
		} else {
			assetsList.Refresh()
			ed.log("import: ok (%d assets)\n", len(ed.assets))
		}
		ed.updateViewport()
	}
	saveScene := func() {
		if err := scene.Save(ed.scenePathAbs, ed.sc.Scene); err != nil {
			ed.log("save error: %v\n", err)
		} else {
			ed.log("scene saved: %s\n", filepath.ToSlash(ed.scenePathAbs))
		}
	}

	// ---------- Меню (как в Unity/Unreal) ----------
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open Project…", func() { ed.log("File > Open Project\n") }),
		fyne.NewMenuItem("Save Scene", saveScene),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() { w.Close() }),
	)
	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Import Assets", doImport),
		fyne.NewMenuItem("Preferences…", func() { ed.log("Edit > Preferences\n") }),
	)
	windowMenu := fyne.NewMenu("Window",
		fyne.NewMenuItem("Hierarchy", func() {}),
		fyne.NewMenuItem("Inspector", func() {}),
		fyne.NewMenuItem("Console", func() {}),
		fyne.NewMenuItem("Scene", func() {}),
	)
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("Documentation", func() { ed.log("Help > Documentation\n") }),
		fyne.NewMenuItem("About GoEngineKenga", func() { ed.log("GoEngineKenga Editor\n") }),
	)
	mainMenu := fyne.NewMainMenu(fileMenu, editMenu, windowMenu, helpMenu)
	w.SetMainMenu(mainMenu)
	if desk, ok := a.(desktop.App); ok {
		desk.SetSystemTrayMenu(fyne.NewMenu("Editor", fyne.NewMenuItem("Quit", func() { w.Close() })))
	}

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DownloadIcon(), doImport),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MediaPlayIcon(), play),
		widget.NewToolbarAction(theme.MediaStopIcon(), stop),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MediaFastForwardIcon(), runExternal),
		widget.NewToolbarAction(theme.CancelIcon(), stopExternal),
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
			ed.updateViewport()
		}
	}

	center := container.NewBorder(status, gizmoBar, nil, nil, viewportStack)

	hs := container.NewHSplit(leftPanel, center)
	hs.Offset = 0.25
	main := container.NewHSplit(hs, inspector)
	main.Offset = 0.8

	consoleTitle := widget.NewLabel("Console")
	consoleTitle.TextStyle = fyne.TextStyle{Bold: true}
	w.SetContent(container.NewBorder(toolbar, container.NewVSplit(container.NewPadded(consoleTitle), console), nil, nil, main))
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

	log          func(string, ...any)
	updateViewport func() // обновляет текст в панели под viewport
	runtimeProc  *exec.Cmd
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

// viewportInfoText возвращает текст для панели под viewport (выбранный объект/ассет, режим).
func (ed *editorState) viewportInfoText() string {
	var b strings.Builder
	b.WriteString("Mode: ")
	if ed.rt.Mode == runtime.ModePlay {
		b.WriteString("Play")
	} else {
		b.WriteString("Edit")
	}
	b.WriteString("  |  ")

	if ed.selectedEntityIndex >= 0 && ed.selectedEntityIndex < len(ed.sc.Scene.Entities) {
		e := ed.sc.Scene.Entities[ed.selectedEntityIndex]
		b.WriteString("Entity: " + e.Name)
		if e.MeshRenderer != nil && e.MeshRenderer.MeshAssetID != "" {
			b.WriteString("  [Mesh: " + e.MeshRenderer.MeshAssetID + "]")
		}
		b.WriteString("\n")
	} else if ed.selectedAssetIndex >= 0 && ed.selectedAssetIndex < len(ed.assets) {
		a := ed.assets[ed.selectedAssetIndex]
		b.WriteString("Asset: " + a.SourcePath + "  (" + string(a.Type) + ")\n")
	} else {
		b.WriteString("No selection\n")
	}
	return b.String()
}

// nudgeSelected — простой «gizmo»: подвинуть Transform выбранной сущности.
func (ed *editorState) nudgeSelected(dx, dy, dz float32) {
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
	ed.updateViewport()
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
