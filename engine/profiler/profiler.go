package profiler

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Profiler управляет сбором метрик производительности
type Profiler struct {
	mu sync.RWMutex

	// Метрики производительности
	frameTime   time.Duration
	updateTime  time.Duration
	renderTime  time.Duration
	physicsTime time.Duration
	audioTime   time.Duration
	scriptTime  time.Duration

	// Счетчики
	entityCount    int
	drawCalls      int
	trianglesDrawn int
	memoryUsage    uint64

	// История для графиков
	frameTimes []time.Duration

	// Флаги включения профилирования
	enabled      bool
	detailedMode bool

	// Callback для вывода результатов
	outputCallback func(string)
}

// NewProfiler создает новый профилировщик
func NewProfiler() *Profiler {
	return &Profiler{
		frameTimes:     make([]time.Duration, 0, 60), // Храним последние 60 кадров
		enabled:        true,
		detailedMode:   false,
		outputCallback: func(s string) { fmt.Println(s) },
	}
}

// StartFrame начинает измерение кадра
func (p *Profiler) StartFrame() time.Time {
	if !p.enabled {
		return time.Time{}
	}
	return time.Now()
}

// EndFrame завершает измерение кадра
func (p *Profiler) EndFrame(startTime time.Time) {
	if !p.enabled || startTime.IsZero() {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.frameTime = time.Since(startTime)

	// Добавляем в историю
	p.frameTimes = append(p.frameTimes, p.frameTime)
	if len(p.frameTimes) > 60 {
		p.frameTimes = p.frameTimes[1:] // Удаляем старые значения
	}
}

// StartSection начинает измерение секции
func (p *Profiler) StartSection(section string) *SectionTimer {
	if !p.enabled {
		return &SectionTimer{}
	}
	return &SectionTimer{
		name:      section,
		startTime: time.Now(),
		profiler:  p,
	}
}

// EndSection завершает измерение секции
func (p *Profiler) EndSection(timer *SectionTimer) {
	if !p.enabled || timer.startTime.IsZero() {
		return
	}

	duration := time.Since(timer.startTime)

	p.mu.Lock()
	switch timer.name {
	case "update":
		p.updateTime = duration
	case "render":
		p.renderTime = duration
	case "physics":
		p.physicsTime = duration
	case "audio":
		p.audioTime = duration
	case "script":
		p.scriptTime = duration
	}
	p.mu.Unlock()
}

// SectionTimer таймер для измерения секций
type SectionTimer struct {
	name      string
	startTime time.Time
	profiler  *Profiler
}

// End завершает таймер
func (st *SectionTimer) End() {
	st.profiler.EndSection(st)
}

// SetEntityCount устанавливает количество сущностей
func (p *Profiler) SetEntityCount(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entityCount = count
}

// SetDrawCalls устанавливает количество вызовов рисования
func (p *Profiler) SetDrawCalls(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.drawCalls = count
}

// SetTrianglesDrawn устанавливает количество нарисованных треугольников
func (p *Profiler) SetTrianglesDrawn(count int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.trianglesDrawn = count
}

// UpdateMemoryUsage обновляет информацию о памяти
func (p *Profiler) UpdateMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	p.mu.Lock()
	p.memoryUsage = m.Alloc
	p.mu.Unlock()
}

// GetReport возвращает отчет о производительности
func (p *Profiler) GetReport() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	report := "=== Performance Report ===\n"

	// FPS и время кадра
	if len(p.frameTimes) > 0 {
		avgFrameTime := time.Duration(0)
		for _, ft := range p.frameTimes {
			avgFrameTime += ft
		}
		avgFrameTime /= time.Duration(len(p.frameTimes))
		fps := float64(time.Second) / float64(avgFrameTime)
		report += fmt.Sprintf("FPS: %.1f (avg frame time: %v)\n", fps, avgFrameTime)
	}

	report += fmt.Sprintf("Entities: %d\n", p.entityCount)
	report += fmt.Sprintf("Draw calls: %d\n", p.drawCalls)
	report += fmt.Sprintf("Triangles: %d\n", p.trianglesDrawn)
	report += fmt.Sprintf("Memory: %.2f MB\n", float64(p.memoryUsage)/(1024*1024))

	if p.detailedMode {
		report += "\nDetailed timing:\n"
		report += fmt.Sprintf("  Update: %v\n", p.updateTime)
		report += fmt.Sprintf("  Render: %v\n", p.renderTime)
		report += fmt.Sprintf("  Physics: %v\n", p.physicsTime)
		report += fmt.Sprintf("  Audio: %v\n", p.audioTime)
		report += fmt.Sprintf("  Script: %v\n", p.scriptTime)
	}

	return report
}

// PrintReport выводит отчет
func (p *Profiler) PrintReport() {
	if !p.enabled {
		return
	}

	report := p.GetReport()
	p.outputCallback(report)
}

// SetEnabled включает/выключает профилирование
func (p *Profiler) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// SetDetailedMode включает подробный режим
func (p *Profiler) SetDetailedMode(detailed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.detailedMode = detailed
}

// SetOutputCallback устанавливает callback для вывода
func (p *Profiler) SetOutputCallback(callback func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outputCallback = callback
}

// GetFrameTime возвращает среднее время кадра
func (p *Profiler) GetFrameTime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.frameTimes) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, ft := range p.frameTimes {
		total += ft
	}
	return total / time.Duration(len(p.frameTimes))
}

// GetFPS возвращает средний FPS
func (p *Profiler) GetFPS() float64 {
	frameTime := p.GetFrameTime()
	if frameTime == 0 {
		return 0
	}
	return float64(time.Second) / float64(frameTime)
}

// GetMemoryUsage возвращает использование памяти
func (p *Profiler) GetMemoryUsage() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.memoryUsage
}

// GetEntityCount возвращает количество сущностей
func (p *Profiler) GetEntityCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.entityCount
}
