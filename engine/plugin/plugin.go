package plugin

import (
	"fmt"
	"plugin"
	"sync"

	"goenginekenga/engine/ecs"
	"goenginekenga/engine/runtime"
)

// Plugin интерфейс плагина
type Plugin interface {
	// Инициализация плагина
	Init(manager *Manager) error

	// Обновление плагина (вызывается каждый кадр)
	Update(deltaTime float32) error

	// Завершение работы плагина
	Shutdown() error

	// Получить имя плагина
	Name() string

	// Получить версию плагина
	Version() string

	// Получить описание плагина
	Description() string
}

// SystemPlugin интерфейс для плагинов, которые добавляют системы ECS
type SystemPlugin interface {
	Plugin

	// Создать систему
	CreateSystem(world *ecs.World) System
}

// RenderPlugin интерфейс для плагинов рендеринга
type RenderPlugin interface {
	Plugin

	// Рендеринг (вызывается после основного рендеринга)
	Render(world *ecs.World) error
}

// System интерфейс системы ECS
type System interface {
	// Инициализация системы
	Init(world *ecs.World) error

	// Обновление системы
	Update(world *ecs.World, deltaTime float32) error

	// Завершение системы
	Shutdown() error
}

// Manager управляет плагинами
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	systems map[string]System

	runtime *runtime.Runtime
}

// NewManager создает новый менеджер плагинов
func NewManager(runtime *runtime.Runtime) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		systems: make(map[string]System),
		runtime: runtime,
	}
}

// LoadPlugin загружает плагин из .so файла
func (pm *Manager) LoadPlugin(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Получаем символ плагина
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export Plugin symbol: %w", path, err)
	}

	// Проверяем, что это Plugin
	pluginInstance, ok := sym.(Plugin)
	if !ok {
		return fmt.Errorf("plugin %s does not implement Plugin interface", path)
	}

	// Инициализируем плагин
	if err := pluginInstance.Init(pm); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", path, err)
	}

	// Регистрируем плагин
	pm.mu.Lock()
	pm.plugins[pluginInstance.Name()] = pluginInstance
	pm.mu.Unlock()

	// Если это SystemPlugin, создаем систему
	if systemPlugin, ok := pluginInstance.(SystemPlugin); ok {
		if world, err := pm.runtime.ActiveWorld(); err == nil {
			system := systemPlugin.CreateSystem(world)
			if err := system.Init(world); err != nil {
				return fmt.Errorf("failed to initialize system for plugin %s: %w", pluginInstance.Name(), err)
			}

			pm.mu.Lock()
			pm.systems[pluginInstance.Name()] = system
			pm.mu.Unlock()
		}
	}

	return nil
}

// UnloadPlugin выгружает плагин
func (pm *Manager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Выгружаем систему, если есть
	if system, hasSystem := pm.systems[name]; hasSystem {
		if err := system.Shutdown(); err != nil {
			// Логируем ошибку, но продолжаем
			fmt.Printf("Warning: failed to shutdown system for plugin %s: %v\n", name, err)
		}
		delete(pm.systems, name)
	}

	// Выгружаем плагин
	if err := plugin.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %w", name, err)
	}

	delete(pm.plugins, name)
	return nil
}

// Update обновляет все плагины
func (pm *Manager) Update(deltaTime float32) {
	pm.mu.RLock()
	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		plugins = append(plugins, p)
	}
	pm.mu.RUnlock()

	for _, p := range plugins {
		if err := p.Update(deltaTime); err != nil {
			fmt.Printf("Plugin %s update error: %v\n", p.Name(), err)
		}
	}

	// Обновляем системы
	pm.mu.RLock()
	systems := make([]System, 0, len(pm.systems))
	for _, s := range pm.systems {
		systems = append(systems, s)
	}
	pm.mu.RUnlock()

	if world, err := pm.runtime.ActiveWorld(); err == nil {
		for _, s := range systems {
			if err := s.Update(world, deltaTime); err != nil {
				fmt.Printf("System update error: %v\n", err)
			}
		}
	}
}

// Render вызывает рендеринг плагинов
func (pm *Manager) Render(world *ecs.World) {
	pm.mu.RLock()
	plugins := make([]Plugin, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		plugins = append(plugins, p)
	}
	pm.mu.RUnlock()

	for _, p := range plugins {
		if renderPlugin, ok := p.(RenderPlugin); ok {
			if err := renderPlugin.Render(world); err != nil {
				fmt.Printf("Plugin %s render error: %v\n", p.Name(), err)
			}
		}
	}
}

// GetPlugin получает плагин по имени
func (pm *Manager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins возвращает список загруженных плагинов
func (pm *Manager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

// Shutdown завершает работу всех плагинов
func (pm *Manager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []error

	// Завершаем системы
	for name, system := range pm.systems {
		if err := system.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("system %s shutdown error: %w", name, err))
		}
	}

	// Завершаем плагины
	for name, plugin := range pm.plugins {
		if err := plugin.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s shutdown error: %w", name, err))
		}
	}

	pm.systems = make(map[string]System)
	pm.plugins = make(map[string]Plugin)

	if len(errors) > 0 {
		return fmt.Errorf("multiple shutdown errors: %v", errors)
	}

	return nil
}
