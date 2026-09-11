// Package ir procedurally synthesizes guitar-cabinet impulse responses and
// encodes them as 16-bit PCM WAV files suitable for the Sonicake Matribox
// (QME-50) user IR slots.
//
// The synthesis follows the simulated-room-IR approach used by convolution
// reverb generators (e.g. adelespinasse/reverbGen): an exponentially decaying
// noise tail shaped by an EQ curve toward a cabinet spectral target, plus a
// sparse early-reflection pattern. WAV I/O uses github.com/go-audio/wav, the
// de-facto Go standard for PCM wave files.
package ir

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// Cabinet describes a spectral target for procedural IR synthesis.
type Cabinet struct {
	Name string
	// LowBump is the center frequency in Hz of the cabinet's low resonance.
	LowBump Hz
	// LowGain is the resonance bump gain in dB (0 = flat).
	LowGain dB
	// HighCut is the frequency in Hz where the response starts rolling off.
	HighCut Hz
	// Brightness is the tilt above HighCut in dB (negative = darker).
	Brightness dB
	// Decay is the noise-tail decay time constant.
	Decay time.Duration
	// Reflections is the sparse early-reflection pattern.
	Reflections []Reflection
}

// Hz and dB make cabinet parameters self-documenting at call sites.
type (
	// Hz is a frequency in hertz.
	Hz float64
	// dB is a gain in decibels.
	dB float64
)

// Reflection is a single early-reflection tap.
type Reflection struct {
	Delay time.Duration
	Gain  float64
}

// Catalog holds the built-in cabinet presets.
var Catalog = []Cabinet{
	{
		Name: "4x12 V30", LowBump: 110, LowGain: 5.5, HighCut: 4200, Brightness: -9,
		Decay:       110 * time.Millisecond,
		Reflections: []Reflection{{1200 * time.Microsecond, 0.35}, {3100 * time.Microsecond, 0.22}, {6400 * time.Microsecond, 0.12}},
	},
	{
		Name: "4x12 Vintage", LowBump: 95, LowGain: 6.5, HighCut: 3400, Brightness: -11,
		Decay:       130 * time.Millisecond,
		Reflections: []Reflection{{1000 * time.Microsecond, 0.30}, {2800 * time.Microsecond, 0.25}, {5900 * time.Microsecond, 0.15}},
	},
	{
		Name: "2x12 Open", LowBump: 130, LowGain: 4.0, HighCut: 5200, Brightness: -7,
		Decay:       80 * time.Millisecond,
		Reflections: []Reflection{{800 * time.Microsecond, 0.40}, {2200 * time.Microsecond, 0.20}},
	},
	{
		Name: "1x12 Blues", LowBump: 150, LowGain: 3.0, HighCut: 6200, Brightness: -5,
		Decay:       60 * time.Millisecond,
		Reflections: []Reflection{{600 * time.Microsecond, 0.45}, {1800 * time.Microsecond, 0.18}},
	},
	{
		Name: "Bass 1x15", LowBump: 70, LowGain: 7.0, HighCut: 2800, Brightness: -13,
		Decay:       160 * time.Millisecond,
		Reflections: []Reflection{{1500 * time.Microsecond, 0.30}, {4000 * time.Microsecond, 0.18}, {7500 * time.Microsecond, 0.10}},
	},
	{
		Name: "Acoustic Mini", LowBump: 180, LowGain: 2.0, HighCut: 9000, Brightness: -3,
		Decay:       45 * time.Millisecond,
		Reflections: []Reflection{{400 * time.Microsecond, 0.50}, {1100 * time.Microsecond, 0.20}},
	},
}

// CabinetByName returns the catalog cabinet with the given name.
func CabinetByName(name string) (Cabinet, error) {
	for _, c := range Catalog {
		if c.Name == name {
			return c, nil
		}
	}
	names := make([]string, len(Catalog))
	for i, c := range Catalog {
		names[i] = c.Name
	}
	return Cabinet{}, fmt.Errorf("unknown cabinet %q (available: %s)", name, strings.Join(names, ", "))
}

// Options controls synthesis.
type Options struct {
	// SampleRate must be positive; 44100 is what Matribox expects.
	SampleRate int
	// Duration is the IR length; 300ms is typical for a guitar cab.
	Duration time.Duration
	// Seed makes generation reproducible; identical options and seeds
	// produce identical output.
	Seed int64
}

func (o Options) validate() error {
	if o.SampleRate <= 0 {
		return fmt.Errorf("ir: SampleRate must be positive, got %d", o.SampleRate)
	}
	if o.Duration <= 0 {
		return fmt.Errorf("ir: Duration must be positive, got %v", o.Duration)
	}
	return nil
}

// Defaults fills zero fields with sane values and returns the result.
func (o Options) Defaults() Options {
	if o.SampleRate == 0 {
		o.SampleRate = 44100
	}
	if o.Duration == 0 {
		o.Duration = 300 * time.Millisecond
	}
	return o
}

// Generate synthesizes a mono IR with values in [-1, 1]. The output is
// deterministic for identical cabinet, options, and seed.
func (c Cabinet) Generate(opts Options) ([]float64, error) {
	opts = opts.Defaults()
	if err := opts.validate(); err != nil {
		return nil, err
	}
	n := int(float64(opts.SampleRate) * opts.Duration.Seconds())
	out := make([]float64, n)

	c.noiseTail(out, opts)
	c.earlyReflections(out, opts)
	c.shapeSpectrum(out, opts)
	normalize(out, 0.9)
	fadeEdges(out, opts.SampleRate/1000) // ~1ms click guard
	return out, nil
}

// noiseTail fills out with low-passed white noise decaying at c.Decay.
func (c Cabinet) noiseTail(out []float64, opts Options) {
	rng := rand.New(rand.NewSource(opts.Seed))
	coef := lowpassCoef(float64(opts.SampleRate), nyquistSafe(float64(opts.SampleRate), float64(c.HighCut)))
	state := 0.0
	decaySec := c.Decay.Seconds()
	for i := range out {
		state += coef * (rng.NormFloat64() - state)
		out[i] = state * math.Exp(-float64(i)/float64(opts.SampleRate)/decaySec)
	}
}

// earlyReflections adds the direct impulse and the cabinet's reflection taps.
func (c Cabinet) earlyReflections(out []float64, opts Options) {
	for _, r := range c.Reflections {
		idx := int(r.Delay.Seconds() * float64(opts.SampleRate))
		if idx >= 0 && idx < len(out) {
			out[idx] += r.Gain
		}
	}
	if len(out) > 0 {
		out[0] += 0.8
	}
}

// shapeSpectrum applies the low resonance bump and the high-frequency tilt.
// Filter frequencies are clamped below Nyquist: a cabinet tuned for 44.1kHz
// rendered at a low sample rate (e.g. HighCut 4200Hz at sr=8000) would push
// the biquad past its stability region and emit NaNs.
func (c Cabinet) shapeSpectrum(out []float64, opts Options) {
	sr := float64(opts.SampleRate)
	bump := newPeaking(sr, nyquistSafe(sr, float64(c.LowBump)), float64(c.LowGain), 1.1)
	lowSide := newPeaking(sr, nyquistSafe(sr, float64(c.HighCut)/2), -float64(c.Brightness)*0.4, 0.7)
	highSide := newPeaking(sr, nyquistSafe(sr, float64(c.HighCut)), float64(c.Brightness), 0.7)
	for i := range out {
		y := bump.process(out[i])
		y = lowSide.process(y)
		y = highSide.process(y)
		out[i] = y
	}
}

// nyquistSafe clamps a filter frequency into (0, sr/2).
func nyquistSafe(sr, fc float64) float64 {
	max := sr / 2 * 0.999
	switch {
	case fc > max:
		return max
	case fc < 1:
		return 1
	}
	return fc
}

// biquad is an RBJ-cookbook direct-form-1 peaking-EQ filter.
type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func (f *biquad) process(x float64) float64 {
	y := f.b0*x + f.b1*f.x1 + f.b2*f.x2 - f.a1*f.y1 - f.a2*f.y2
	f.x2, f.x1, f.y1, f.y2 = f.x1, x, y, f.y1
	return y
}

func newPeaking(sr, fc, gainDb, q float64) *biquad {
	A := math.Pow(10, gainDb/40)
	w := 2 * math.Pi * fc / sr
	alpha := math.Sin(w) / (2 * q)
	c := math.Cos(w)
	a0 := 1 + alpha/A
	return &biquad{
		b0: (1 + alpha*A) / a0,
		b1: (-2 * c) / a0,
		b2: (1 - alpha*A) / a0,
		a1: (-2 * c) / a0,
		a2: (1 - alpha/A) / a0,
	}
}

// lowpassCoef returns the one-pole smoothing coefficient for a cutoff in Hz.
func lowpassCoef(sr, fc float64) float64 {
	return 1 - math.Exp(-2*math.Pi*fc/sr)
}

// normalize scales x so its peak reaches target.
func normalize(x []float64, target float64) {
	maxAbs := 0.0
	for _, v := range x {
		if a := math.Abs(v); a > maxAbs {
			maxAbs = a
		}
	}
	if maxAbs == 0 {
		return
	}
	g := target / maxAbs
	for i := range x {
		x[i] *= g
	}
}

// fadeEdges applies linear fades of about 1ms at both ends to avoid clicks.
func fadeEdges(x []float64, fadeSamples int) {
	if fadeSamples <= 0 {
		return
	}
	if fadeSamples > len(x)/2 {
		fadeSamples = len(x) / 2
	}
	for i := 0; i < fadeSamples; i++ {
		g := float64(i) / float64(fadeSamples)
		x[i] *= g
		x[len(x)-1-i] *= g
	}
}

// WriteWAV encodes samples as a mono 16-bit PCM WAV file.
func WriteWAV(path string, samples []float64, sampleRate int) error {
	if sampleRate <= 0 {
		return fmt.Errorf("ir: sample rate must be positive, got %d", sampleRate)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ir: create output dir: %w", err)
	}
	pcm := make([]int, len(samples))
	for i, s := range samples {
		pcm[i] = int(math.Round(clamp(s, -1, 1) * 32767))
	}
	buf := &audio.IntBuffer{
		Data: pcm, Format: &audio.Format{SampleRate: sampleRate, NumChannels: 1},
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("ir: create %s: %w", path, err)
	}
	defer f.Close()
	// PCM wave format code = 1 (go-audio/wav takes a plain int here).
	const pcmFormat = 1
	enc := wav.NewEncoder(f, sampleRate, 16, 1, pcmFormat)
	if err := enc.Write(buf); err != nil {
		return fmt.Errorf("ir: encode %s: %w", path, err)
	}
	return enc.Close()
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
