package debug

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"goenginekenga/engine/ecs"
)

// LogLevel уровень логирования
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// Logger интерфейс для логирования
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

// DefaultLogger стандартная реализация логгера
type DefaultLogger struct {
	level  LogLevel
	logger *log.Logger
	mu     sync.Mutex
}

// NewDefaultLogger создает новый логгер
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *DefaultLogger) log(level LogLevel, prefix string, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] %s", prefix, message)

	if level == LogLevelFatal {
		os.Exit(1)
	}
}

func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, "DEBUG", format, args...)
}

func (l *DefaultLogger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, "INFO", format, args...)
}

func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	l.log(LogLevelWarn, "WARN", format, args...)
}

func (l *DefaultLogger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, "ERROR", format, args...)
}

func (l *DefaultLogger) Fatal(format string, args ...interface{}) {
	l.log(LogLevelFatal, "FATAL", format, args...)
}

// Debugger предоставляет инструменты отладки
type Debugger struct {
	logger      Logger
	breakpoints map[string]bool
	watchVars   map[string]interface{}
	enabled     bool
	mu          sync.RWMutex
}

// NewDebugger создает новый отладчик
func NewDebugger(logger Logger) *Debugger {
	return &Debugger{
		logger:      logger,
		breakpoints: make(map[string]bool),
		watchVars:   make(map[string]interface{}),
		enabled:     true,
	}
}

// SetBreakpoint устанавливает точку останова
func (d *Debugger) SetBreakpoint(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.breakpoints[name] = true
	d.logger.Debug("Breakpoint set: %s", name)
}

// ClearBreakpoint снимает точку останова
func (d *Debugger) ClearBreakpoint(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.breakpoints, name)
	d.logger.Debug("Breakpoint cleared: %s", name)
}

// IsBreakpoint проверяет точку останова
func (d *Debugger) IsBreakpoint(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.breakpoints[name]
}

// WatchVariable добавляет переменную для отслеживания
func (d *Debugger) WatchVariable(name string, value interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.watchVars[name] = value
}

// GetWatchVariable получает значение отслеживаемой переменной
func (d *Debugger) GetWatchVariable(name string) (interface{}, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	value, exists := d.watchVars[name]
	return value, exists
}

// InspectWorld инспектирует состояние мира
func (d *Debugger) InspectWorld(world *ecs.World, detailed bool) {
	if !d.enabled {
		return
	}

	d.logger.Info("=== World Inspection ===")
	d.logger.Info("Entities: %d", len(world.Entities()))

	if detailed {
		for _, entityID := range world.Entities() {
			d.inspectEntity(world, entityID)
		}
	}
}

func (d *Debugger) inspectEntity(world *ecs.World, entityID ecs.EntityID) {
	name := world.Name(entityID)
	d.logger.Info("Entity %d: %s", entityID, name)

	if transform, ok := world.GetTransform(entityID); ok {
		d.logger.Debug("  Transform: pos=(%.2f,%.2f,%.2f) rot=(%.2f,%.2f,%.2f) scale=(%.2f,%.2f,%.2f)",
			transform.Position.X, transform.Position.Y, transform.Position.Z,
			transform.Rotation.X, transform.Rotation.Y, transform.Rotation.Z,
			transform.Scale.X, transform.Scale.Y, transform.Scale.Z)
	}

	if camera, ok := world.GetCamera(entityID); ok {
		d.logger.Debug("  Camera: fov=%.1f, near=%.2f, far=%.1f",
			camera.FovYDegrees, camera.Near, camera.Far)
	}

	if meshRenderer, ok := world.GetMeshRenderer(entityID); ok {
		d.logger.Debug("  MeshRenderer: mesh=%s, material=%s",
			meshRenderer.MeshAssetID, meshRenderer.MaterialAssetID)
	}

	if rigidbody, ok := world.GetRigidbody(entityID); ok {
		d.logger.Debug("  Rigidbody: mass=%.2f, velocity=(%.2f,%.2f,%.2f), kinematic=%v, gravity=%v",
			rigidbody.Mass,
			rigidbody.Velocity.X, rigidbody.Velocity.Y, rigidbody.Velocity.Z,
			rigidbody.IsKinematic, rigidbody.UseGravity)
	}

	if collider, ok := world.GetCollider(entityID); ok {
		d.logger.Debug("  Collider: type=%s, size=(%.2f,%.2f,%.2f), trigger=%v",
			collider.Type,
			collider.Size.X, collider.Size.Y, collider.Size.Z,
			collider.IsTrigger)
	}

	if audioSource, ok := world.GetAudioSource(entityID); ok {
		d.logger.Debug("  AudioSource: clip=%s, volume=%.2f, spatial=%v, loop=%v",
			audioSource.Clip, audioSource.Volume, audioSource.Spatial, audioSource.Loop)
	}
}

// PerformanceMonitor мониторит производительность
type PerformanceMonitor struct {
	sampleRate time.Duration
	lastSample time.Time
	samples    []PerformanceSample
	maxSamples int
	mu         sync.Mutex
}

type PerformanceSample struct {
	Timestamp   time.Time
	FrameTime   time.Duration
	MemoryUsage uint64
	EntityCount int
	FPS         float64
}

// NewPerformanceMonitor создает монитор производительности
func NewPerformanceMonitor(sampleRate time.Duration, maxSamples int) *PerformanceMonitor {
	return &PerformanceMonitor{
		sampleRate: sampleRate,
		lastSample: time.Now(),
		samples:    make([]PerformanceSample, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// Update обновляет монитор (вызывать каждый кадр)
func (pm *PerformanceMonitor) Update(frameTime time.Duration, entityCount int, fps float64) {
	now := time.Now()
	if now.Sub(pm.lastSample) < pm.sampleRate {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sample := PerformanceSample{
		Timestamp:   now,
		FrameTime:   frameTime,
		MemoryUsage: m.Alloc,
		EntityCount: entityCount,
		FPS:         fps,
	}

	pm.mu.Lock()
	pm.samples = append(pm.samples, sample)
	if len(pm.samples) > pm.maxSamples {
		pm.samples = pm.samples[1:]
	}
	pm.lastSample = now
	pm.mu.Unlock()
}

// GetSamples возвращает собранные сэмплы
func (pm *PerformanceMonitor) GetSamples() []PerformanceSample {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	result := make([]PerformanceSample, len(pm.samples))
	copy(result, pm.samples)
	return result
}

// GetAverageFrameTime возвращает среднее время кадра
func (pm *PerformanceMonitor) GetAverageFrameTime() time.Duration {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.samples) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, sample := range pm.samples {
		total += sample.FrameTime
	}
	return total / time.Duration(len(pm.samples))
}

// GetAverageFPS возвращает средний FPS
func (pm *PerformanceMonitor) GetAverageFPS() float64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.samples) == 0 {
		return 0
	}

	totalFPS := 0.0
	for _, sample := range pm.samples {
		totalFPS += sample.FPS
	}
	return totalFPS / float64(len(pm.samples))
}
