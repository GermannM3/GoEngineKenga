package audio

import (
	"math"
	"math/cmplx"
)

// AudioAnalyzer performs FFT and audio analysis for reactive effects
type AudioAnalyzer struct {
	SampleRate     int
	BufferSize     int
	audioBuffer    []float64
	fftBuffer      []complex128
	spectrum       []float64
	smoothSpectrum []float64

	// Frequency bands
	Bass    float64 // 20-250 Hz
	LowMid  float64 // 250-500 Hz
	Mid     float64 // 500-2000 Hz
	HighMid float64 // 2000-4000 Hz
	High    float64 // 4000+ Hz

	// Beat detection
	BeatDetected  bool
	BeatEnergy    float64
	beatHistory   []float64
	beatThreshold float64

	// Overall
	Volume float64
	Peak   float64
}

// NewAudioAnalyzer creates a new audio analyzer
func NewAudioAnalyzer(sampleRate, bufferSize int) *AudioAnalyzer {
	return &AudioAnalyzer{
		SampleRate:     sampleRate,
		BufferSize:     bufferSize,
		audioBuffer:    make([]float64, bufferSize),
		fftBuffer:      make([]complex128, bufferSize),
		spectrum:       make([]float64, bufferSize/2),
		smoothSpectrum: make([]float64, bufferSize/2),
		beatHistory:    make([]float64, 43), // ~1 second at 43fps
		beatThreshold:  1.5,
	}
}

// PushSamples adds audio samples for analysis
func (a *AudioAnalyzer) PushSamples(samples []float64) {
	// Shift buffer and add new samples
	copy(a.audioBuffer, a.audioBuffer[len(samples):])
	copy(a.audioBuffer[len(a.audioBuffer)-len(samples):], samples)
}

// Analyze performs FFT and updates all audio metrics
func (a *AudioAnalyzer) Analyze() {
	// Calculate volume (RMS)
	sum := 0.0
	peak := 0.0
	for _, s := range a.audioBuffer {
		sum += s * s
		if math.Abs(s) > peak {
			peak = math.Abs(s)
		}
	}
	a.Volume = math.Sqrt(sum / float64(len(a.audioBuffer)))
	a.Peak = peak

	// Apply window function (Hann)
	for i := range a.fftBuffer {
		window := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(a.fftBuffer))))
		a.fftBuffer[i] = complex(a.audioBuffer[i]*window, 0)
	}

	// Perform FFT
	fft(a.fftBuffer)

	// Calculate magnitude spectrum
	for i := range a.spectrum {
		mag := cmplx.Abs(a.fftBuffer[i]) / float64(a.BufferSize)
		// Smooth spectrum
		a.smoothSpectrum[i] = a.smoothSpectrum[i]*0.8 + mag*0.2
		a.spectrum[i] = mag
	}

	// Calculate frequency bands
	a.calculateBands()

	// Beat detection
	a.detectBeat()
}

func (a *AudioAnalyzer) calculateBands() {
	freqPerBin := float64(a.SampleRate) / float64(a.BufferSize)

	// Bass: 20-250 Hz
	a.Bass = a.averageBand(int(20/freqPerBin), int(250/freqPerBin))

	// LowMid: 250-500 Hz
	a.LowMid = a.averageBand(int(250/freqPerBin), int(500/freqPerBin))

	// Mid: 500-2000 Hz
	a.Mid = a.averageBand(int(500/freqPerBin), int(2000/freqPerBin))

	// HighMid: 2000-4000 Hz
	a.HighMid = a.averageBand(int(2000/freqPerBin), int(4000/freqPerBin))

	// High: 4000+ Hz
	a.High = a.averageBand(int(4000/freqPerBin), len(a.spectrum)-1)
}

func (a *AudioAnalyzer) averageBand(startBin, endBin int) float64 {
	if startBin < 0 {
		startBin = 0
	}
	if endBin >= len(a.smoothSpectrum) {
		endBin = len(a.smoothSpectrum) - 1
	}
	if startBin >= endBin {
		return 0
	}

	sum := 0.0
	for i := startBin; i <= endBin; i++ {
		sum += a.smoothSpectrum[i]
	}
	return sum / float64(endBin-startBin+1)
}

func (a *AudioAnalyzer) detectBeat() {
	// Use bass energy for beat detection
	energy := a.Bass

	// Calculate average energy from history
	avgEnergy := 0.0
	for _, e := range a.beatHistory {
		avgEnergy += e
	}
	avgEnergy /= float64(len(a.beatHistory))

	// Shift history
	copy(a.beatHistory, a.beatHistory[1:])
	a.beatHistory[len(a.beatHistory)-1] = energy

	// Detect beat if energy exceeds threshold * average
	a.BeatDetected = energy > avgEnergy*a.beatThreshold && avgEnergy > 0.001
	a.BeatEnergy = energy
}

// GetSpectrum returns the frequency spectrum
func (a *AudioAnalyzer) GetSpectrum() []float64 {
	return a.smoothSpectrum
}

// GetSpectrumNormalized returns normalized spectrum (0-1)
func (a *AudioAnalyzer) GetSpectrumNormalized() []float64 {
	result := make([]float64, len(a.smoothSpectrum))
	max := 0.0
	for _, v := range a.smoothSpectrum {
		if v > max {
			max = v
		}
	}
	if max > 0 {
		for i, v := range a.smoothSpectrum {
			result[i] = v / max
		}
	}
	return result
}

// FFT implementation (Cooley-Tukey radix-2)
func fft(x []complex128) {
	n := len(x)
	if n <= 1 {
		return
	}

	// Bit-reversal permutation
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}

	// Cooley-Tukey iterative FFT
	for size := 2; size <= n; size <<= 1 {
		halfSize := size >> 1
		tableStep := n / size
		for i := 0; i < n; i += size {
			for j, k := 0, 0; j < halfSize; j, k = j+1, k+tableStep {
				w := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n)))
				u := x[i+j]
				t := w * x[i+j+halfSize]
				x[i+j] = u + t
				x[i+j+halfSize] = u - t
			}
		}
	}
}

// AudioReactiveEffect is a base for audio-reactive visual effects
type AudioReactiveEffect struct {
	Analyzer    *AudioAnalyzer
	Sensitivity float64
}

// NewAudioReactiveEffect creates a new audio-reactive effect
func NewAudioReactiveEffect(analyzer *AudioAnalyzer) *AudioReactiveEffect {
	return &AudioReactiveEffect{
		Analyzer:    analyzer,
		Sensitivity: 1.0,
	}
}

// GetBassScale returns a scale factor based on bass
func (e *AudioReactiveEffect) GetBassScale() float64 {
	return 1.0 + e.Analyzer.Bass*e.Sensitivity*2
}

// GetColorShift returns RGB shift values based on frequency bands
func (e *AudioReactiveEffect) GetColorShift() (r, g, b float64) {
	return e.Analyzer.Bass * e.Sensitivity,
		e.Analyzer.Mid * e.Sensitivity,
		e.Analyzer.High * e.Sensitivity
}

// GetPulseIntensity returns pulse intensity from beat detection
func (e *AudioReactiveEffect) GetPulseIntensity() float64 {
	if e.Analyzer.BeatDetected {
		return 1.0
	}
	return 0.0
}

// WaveformVisualizer generates waveform visualization data
type WaveformVisualizer struct {
	Analyzer *AudioAnalyzer
	Width    int
	Height   int
}

// NewWaveformVisualizer creates a waveform visualizer
func NewWaveformVisualizer(analyzer *AudioAnalyzer, width, height int) *WaveformVisualizer {
	return &WaveformVisualizer{
		Analyzer: analyzer,
		Width:    width,
		Height:   height,
	}
}

// GetPoints returns waveform points for rendering
func (w *WaveformVisualizer) GetPoints() [][2]int {
	points := make([][2]int, w.Width)
	samplesPerPoint := len(w.Analyzer.audioBuffer) / w.Width

	centerY := w.Height / 2

	for i := 0; i < w.Width; i++ {
		// Average samples for this point
		sum := 0.0
		for j := 0; j < samplesPerPoint; j++ {
			idx := i*samplesPerPoint + j
			if idx < len(w.Analyzer.audioBuffer) {
				sum += w.Analyzer.audioBuffer[idx]
			}
		}
		avg := sum / float64(samplesPerPoint)

		// Convert to Y coordinate
		y := centerY + int(avg*float64(w.Height/2))
		if y < 0 {
			y = 0
		}
		if y >= w.Height {
			y = w.Height - 1
		}

		points[i] = [2]int{i, y}
	}

	return points
}

// SpectrumVisualizer generates spectrum visualization data
type SpectrumVisualizer struct {
	Analyzer *AudioAnalyzer
	Bars     int
	Height   int
}

// NewSpectrumVisualizer creates a spectrum visualizer
func NewSpectrumVisualizer(analyzer *AudioAnalyzer, bars, height int) *SpectrumVisualizer {
	return &SpectrumVisualizer{
		Analyzer: analyzer,
		Bars:     bars,
		Height:   height,
	}
}

// GetBarHeights returns heights for spectrum bars
func (s *SpectrumVisualizer) GetBarHeights() []int {
	heights := make([]int, s.Bars)
	spectrum := s.Analyzer.GetSpectrumNormalized()
	binsPerBar := len(spectrum) / s.Bars

	for i := 0; i < s.Bars; i++ {
		// Average bins for this bar
		sum := 0.0
		for j := 0; j < binsPerBar; j++ {
			idx := i*binsPerBar + j
			if idx < len(spectrum) {
				sum += spectrum[idx]
			}
		}
		avg := sum / float64(binsPerBar)

		// Convert to height
		heights[i] = int(avg * float64(s.Height))
	}

	return heights
}

// CircularVisualizer creates circular audio visualizations
type CircularVisualizer struct {
	Analyzer   *AudioAnalyzer
	Points     int
	BaseRadius float64
}

// NewCircularVisualizer creates a circular visualizer
func NewCircularVisualizer(analyzer *AudioAnalyzer, points int) *CircularVisualizer {
	return &CircularVisualizer{
		Analyzer:   analyzer,
		Points:     points,
		BaseRadius: 100,
	}
}

// GetPoints returns points for a circular visualization
func (c *CircularVisualizer) GetPoints(centerX, centerY float64) [][2]float64 {
	points := make([][2]float64, c.Points)
	spectrum := c.Analyzer.GetSpectrumNormalized()
	binsPerPoint := len(spectrum) / c.Points

	for i := 0; i < c.Points; i++ {
		angle := 2 * math.Pi * float64(i) / float64(c.Points)

		// Get spectrum value for this point
		sum := 0.0
		for j := 0; j < binsPerPoint; j++ {
			idx := i*binsPerPoint + j
			if idx < len(spectrum) {
				sum += spectrum[idx]
			}
		}
		avg := sum / float64(binsPerPoint)

		// Calculate radius with spectrum influence
		radius := c.BaseRadius * (1 + avg*0.5)

		// Add bass pulse
		radius *= 1 + c.Analyzer.Bass*0.3

		points[i] = [2]float64{
			centerX + math.Cos(angle)*radius,
			centerY + math.Sin(angle)*radius,
		}
	}

	return points
}
