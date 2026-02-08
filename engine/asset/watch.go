package asset

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"goenginekenga/engine/project"
)

// Watcher следит за assets и сценами. При изменении файлов — триггерит re-import и reload.
// Используется для CAD-режима и IDE: правки в Blender/внешнем редакторе сразу видны в viewport.
type Watcher struct {
	projectDir string
	assetsDir  string
	indexPath  string
	scenePath  string

	watcher *fsnotify.Watcher
	mu     sync.Mutex

	// Флаги для главного потока (не thread-safe для записи иначе)
	assetsDirty bool // нужно переимпортировать assets
	indexDirty  bool // index изменился, resolver нужно обновить
	sceneDirty  bool // сцена изменилась, нужно перезагрузить world
}

// NewWatcher создаёт watcher для projectDir.
// scenePath — путь к текущей сцене (относительно projectDir или абсолютный).
func NewWatcher(projectDir, scenePath string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	p, err := loadProjectPaths(projectDir)
	if err != nil {
		w.Close()
		return nil, err
	}

	sw := &Watcher{
		projectDir: projectDir,
		assetsDir:  p.assetsDir,
		indexPath:  p.indexPath,
		scenePath:  scenePath,
		watcher:   w,
	}

	// Добавляем директории в watch (best-effort)
	_ = os.MkdirAll(sw.assetsDir, 0o755)
	_ = os.MkdirAll(filepath.Dir(sw.indexPath), 0o755)
	_ = w.Add(sw.assetsDir)
	_ = w.Add(filepath.Dir(sw.indexPath))
	if scenePath != "" {
		absScene := filepath.Join(projectDir, scenePath)
		if dir := filepath.Dir(absScene); dir != "." {
			_ = os.MkdirAll(dir, 0o755)
			if err := sw.watcher.Add(dir); err == nil {
				sw.scenePath = absScene
			}
		}
	}

	go sw.run()
	return sw, nil
}

type projectPaths struct {
	assetsDir string
	indexPath string
}

func loadProjectPaths(projectDir string) (*projectPaths, error) {
	assetsDir := filepath.Join(projectDir, "assets")
	indexPath := filepath.Join(projectDir, ".kenga", "assets", "index.json")
	if p, err := project.Load(projectDir); err == nil {
		assetsDir = filepath.Join(projectDir, filepath.FromSlash(p.AssetsDir))
	}
	return &projectPaths{assetsDir: assetsDir, indexPath: indexPath}, nil
}

func (w *Watcher) run() {
	debounce := time.NewTicker(200 * time.Millisecond)
	defer debounce.Stop()
	var pendingAssets, pendingIndex, pendingScene bool

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			base := filepath.Base(event.Name)
			if w.isAssetFile(event.Name) {
				pendingAssets = true
				pendingIndex = true
			} else if base == "index.json" {
				pendingIndex = true
			} else if filepath.Ext(event.Name) == ".json" && strings.Contains(strings.ToLower(event.Name), "scene") {
				pendingScene = true
			}

		case <-debounce.C:
			if pendingAssets || pendingIndex || pendingScene {
				w.mu.Lock()
				w.assetsDirty = w.assetsDirty || pendingAssets
				w.indexDirty = w.indexDirty || pendingIndex
				w.sceneDirty = w.sceneDirty || pendingScene
				w.mu.Unlock()
				pendingAssets = false
				pendingIndex = false
				pendingScene = false
			}

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) isAssetFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	rel, _ := filepath.Rel(w.assetsDir, path)
	if strings.Contains(rel, "..") {
		return false
	}
	return ext == ".gltf" || ext == ".glb" || ext == ".png" || ext == ".jpg" || ext == ".jpeg"
}

// ConsumeAssetsDirty снимает флаг assetsDirty и возвращает true, если он был.
func (w *Watcher) ConsumeAssetsDirty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	v := w.assetsDirty
	w.assetsDirty = false
	return v
}

// ConsumeIndexDirty снимает флаг indexDirty и возвращает true.
func (w *Watcher) ConsumeIndexDirty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	v := w.indexDirty
	w.indexDirty = false
	return v
}

// ConsumeSceneDirty снимает флаг sceneDirty и возвращает true.
func (w *Watcher) ConsumeSceneDirty() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	v := w.sceneDirty
	w.sceneDirty = false
	return v
}

// Close останавливает watcher.
func (w *Watcher) Close() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}
