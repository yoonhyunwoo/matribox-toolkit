package ir

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCabinetByName(t *testing.T) {
	c, err := CabinetByName("4x12 V30")
	if err != nil {
		t.Fatalf("catalog lookup failed: %v", err)
	}
	if c.LowBump != 110 {
		t.Errorf("unexpected LowBump = %v", c.LowBump)
	}
	if _, err := CabinetByName("nope"); err == nil {
		t.Error("expected error for unknown cabinet name")
	}
}

func TestGenerateDeterministic(t *testing.T) {
	opts := Options{SampleRate: 44100, Duration: 100 * time.Millisecond, Seed: 42}
	a, err := Catalog[0].Generate(opts)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Catalog[0].Generate(opts)
	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("sample %d differs: same seed must be deterministic", i)
		}
	}
	c, _ := Catalog[0].Generate(Options{SampleRate: 44100, Duration: 100 * time.Millisecond, Seed: 43})
	same := true
	for i := range a {
		if a[i] != c[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different seeds produced identical output")
	}
}

func TestGenerateLengthAndLevel(t *testing.T) {
	opts := Options{SampleRate: 44100, Duration: 300 * time.Millisecond, Seed: 1}
	for _, c := range Catalog {
		out, err := c.Generate(opts)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		want := 44100 * 300 / 1000
		if len(out) != want {
			t.Errorf("%s: len = %d, want %d", c.Name, len(out), want)
		}
		for _, v := range out {
			if v > 1.0 || v < -1.0 {
				t.Fatalf("%s: sample out of range: %v", c.Name, v)
			}
		}
	}
}

func TestGenerateRejectsBadOptions(t *testing.T) {
	for _, o := range []Options{
		{SampleRate: -1, Duration: time.Second},
		{SampleRate: 44100, Duration: -1},
	} {
		if _, err := Catalog[0].Generate(o); err == nil {
			t.Errorf("expected error for %+v", o)
		}
	}
}

// Regression for a fuzz-found NaN: a cabinet tuned for 44.1kHz (HighCut
// 4200Hz) rendered at sr=8000 pushed the biquad past Nyquist. The exact
// input also lives in testdata/fuzz/FuzzGenerate.
func TestGenerateFiniteAtLowSampleRate(t *testing.T) {
	for _, name := range []string{"4x12 V30", "Bass 1x15", "Acoustic Mini"} {
		cab, err := CabinetByName(name)
		if err != nil {
			t.Fatal(err)
		}
		out, err := cab.Generate(Options{SampleRate: 8000, Duration: time.Second, Seed: -78})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for i, v := range out {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("%s: sample %d not finite at sr=8000", name, i)
			}
		}
	}
}

func TestWriteWAV(t *testing.T) {
	out, err := Catalog[2].Generate(Options{SampleRate: 44100, Duration: 50 * time.Millisecond, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.wav")
	if err := WriteWAV(path, out, 44100); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// 44-byte RIFF header plus 16-bit mono PCM payload.
	want := 44 + len(out)*2
	if fi.Size() != int64(want) {
		t.Errorf("wav size = %d, want %d", fi.Size(), want)
	}
}
