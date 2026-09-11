package ir

import (
	"encoding/binary"
	"math"
	"path/filepath"
	"testing"
	"time"
)

// FuzzGenerate throws arbitrary numeric options at synthesis. Invariants:
// no panic, no NaN/Inf, output bounded to [-1, 1], length matches request.
// The harness bounds the input ranges so fuzzing explores values, not
// multi-gigabyte allocations.
func FuzzGenerate(f *testing.F) {
	f.Add(44100.0, 0.3, int64(1))
	f.Add(8000.0, 0.001, int64(-7))
	f.Add(192000.0, 1.0, int64(0))
	f.Add(1.0, 0.000001, int64(42))
	f.Add(22050.0, 0.0, int64(99))

	f.Fuzz(func(t *testing.T, srF, durF float64, seed int64) {
		sr := srF
		if sr < 1 {
			sr = 1
		}
		if sr > 384000 {
			sr = 384000
		}
		dur := durF
		if dur < 0.000001 {
			dur = 0.000001
		}
		if dur > 2 {
			dur = 2
		}
		idx := int(seed) % len(Catalog)
		if idx < 0 {
			idx += len(Catalog)
		}
		cab := Catalog[idx]
		duration := time.Duration(dur * float64(time.Second))
		srI := int(sr)
		out, err := cab.Generate(Options{
			SampleRate: srI,
			Duration:   duration,
			Seed:       seed,
		})
		if err != nil {
			t.Fatalf("valid-range options rejected: %v", err)
		}
		want := int(float64(srI) * duration.Seconds())
		if len(out) != want {
			t.Fatalf("length = %d, want %d", len(out), want)
		}
		for i, v := range out {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("sample %d not finite: %v (sr=%v dur=%v seed=%d)", i, v, srF, durF, seed)
			}
			if v < -1.001 || v > 1.001 {
				t.Fatalf("sample %d out of bounds: %v", i, v)
			}
		}
	})
}

// FuzzWriteWAV checks the encoder on arbitrary sample data. []float64 is not
// a legal fuzz parameter, so samples arrive as packed little-endian float64.
func FuzzWriteWAV(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 240, 63, 0, 0, 0, 0, 0, 0, 240, 127}, 44100)
	f.Add([]byte{}, 44100)
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 8000)

	f.Fuzz(func(t *testing.T, raw []byte, sr int) {
		if sr <= 0 || sr > 384000 {
			t.Skip()
		}
		if len(raw) > 8*(1<<18) {
			t.Skip()
		}
		samples := make([]float64, len(raw)/8)
		for i := range samples {
			bits := binary.LittleEndian.Uint64(raw[i*8:])
			samples[i] = math.Float64frombits(bits)
		}
		path := filepath.Join(t.TempDir(), "fuzz.wav")
		if err := WriteWAV(path, samples, sr); err != nil {
			t.Fatalf("WriteWAV failed: %v", err)
		}
	})
}
