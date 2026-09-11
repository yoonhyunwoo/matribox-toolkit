package prst

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoad feeds arbitrary bytes to the parser. It must never panic; any
// malformed input may only produce an error.
func FuzzLoad(f *testing.F) {
	f.Add([]byte(fixture))
	f.Add([]byte("<Matribox></Matribox>"))
	f.Add([]byte("<?xml version=\"1.0\"?><Matribox><presets ppBank=\"x\"/></Matribox>"))
	f.Add([]byte("<Matribox><ppIRInfo><ppIRInfoNOPE ppIRNum=\"1\"/></ppIRInfo></Matribox>"))
	f.Add([]byte("<Matribox><presets><Weird/></presets></Matribox>"))
	f.Add([]byte(""))
	f.Add([]byte{0x00, 0x01, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "fuzz.prst")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		b, err := Load(path)
		if err != nil {
			return // parse errors are fine; panics are not
		}
		// If it parses, basic structural access must be safe.
		_ = b.NextID()
		for i := range b.Presets {
			_ = b.Presets[i].CloneAs(0, "x", "")
		}
		_, _ = b.FindIR("1") // must not panic on any parsed input
	})
}

// FuzzRoundTrip checks the invariant: any bundle that parses must survive
// Save → Load with its structure intact and be re-parseable forever.
func FuzzRoundTrip(f *testing.F) {
	f.Add([]byte(fixture))

	f.Fuzz(func(t *testing.T, data []byte) {
		in := filepath.Join(t.TempDir(), "in.prst")
		if err := os.WriteFile(in, data, 0o644); err != nil {
			t.Fatal(err)
		}
		b, err := Load(in)
		if err != nil {
			t.Skip()
		}
		out := filepath.Join(t.TempDir(), "out.prst")
		if err := b.Save(out); err != nil {
			t.Fatalf("Save failed on a bundle that parsed: %v", err)
		}
		b2, err := Load(out)
		if err != nil {
			t.Fatalf("re-Load failed after Save: %v", err)
		}
		if len(b2.Presets) != len(b.Presets) {
			t.Fatalf("preset count changed: %d → %d", len(b.Presets), len(b2.Presets))
		}
		if len(b2.IRInfo.IRs) != len(b.IRInfo.IRs) {
			t.Fatalf("IR slot count changed: %d → %d", len(b.IRInfo.IRs), len(b2.IRInfo.IRs))
		}
		for i := range b.Presets {
			if len(b2.Presets[i].Effects) != len(b.Presets[i].Effects) {
				t.Fatalf("preset %d effect count changed: %d → %d",
					i, len(b.Presets[i].Effects), len(b2.Presets[i].Effects))
			}
			for j := range b.Presets[i].Effects {
				if b.Presets[i].Effects[j].P != b2.Presets[i].Effects[j].P {
					t.Fatalf("preset %d effect %d params changed on round trip", i, j)
				}
				if b.Presets[i].Effects[j].Code != b2.Presets[i].Effects[j].Code {
					t.Fatalf("preset %d effect %d effectCode changed on round trip", i, j)
				}
			}
			if (b.Presets[i].EXP == nil) != (b2.Presets[i].EXP == nil) {
				t.Fatalf("preset %d EXP presence changed", i)
			}
			if b.Presets[i].EXP != nil && len(b.Presets[i].EXP.Kids) != len(b2.Presets[i].EXP.Kids) {
				t.Fatalf("preset %d EXP child count changed", i)
			}
		}
	})
}
