// Command creep builds Radiohead "Creep" tone presets as single-preset
// Matribox patch files (the only patch format the desktop app accepts).
//
//	-tone crunch  pre-chorus rhythm crunch (default)
//	-tone clean   verse clean
//
// Every effect block is spliced from a factory preset that already uses the
// same effectCode, so all params are factory-attested values.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yoonhyunwoo/matribox-toolkit/prst"
)

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "creep: "+f+"\n", a...)
	os.Exit(1)
}

// toneSpec describes one researched tone.
type toneSpec struct {
	name   string
	file   string
	blocks map[string]string // module -> factory effect name to enable
}

var tones = map[string]toneSpec{
	"crunch": {
		name: "Creep Chunk",
		file: "creep.prst",
		blocks: map[string]string{
			"AMP": "B-Man N",     // Fender Bassman normal: pedal-friendly clean platform
			"FX2": "JP Dist",     // op-amp high-gain dist (Marshall Shredmaster stand-in)
			"CAB": "Viblux 1x12", // small Fender-style combo cab, tight lows
			"NR":  "Gate 2",      // single-coil hiss control under gain
			"RVB": "Spring",      // Fender spring reverb
		},
	},
	"clean": {
		name: "Creep Clean",
		file: "creep-clean.prst",
		blocks: map[string]string{
			"FX1": "COMP",        // light compression under the arpeggio
			"AMP": "B-Man N",     // sparkling clean Fender platform
			"CAB": "Viblux 1x12", // small combo cab keeps the low end tidy
			"MOD": "Tremolo",     // subtle pulsing (Kuassa's verse chain)
			"RVB": "Plate",       // light plate spread
		},
	},
}

func main() {
	in := flag.String("in", "prsts.prst", "factory bundle")
	out := flag.String("out", "", "output patch file (default: per-tone name)")
	tone := flag.String("tone", "crunch", "tone to build: crunch|clean")
	flag.Parse()

	spec, ok := tones[*tone]
	if !ok {
		die("unknown tone %q (crunch|clean)", *tone)
	}
	b, err := prst.Load(*in)
	if err != nil {
		die("%v", err)
	}
	tmpl, err := b.PresetByID(0)
	if err != nil {
		die("%v", err)
	}
	findBlock := func(module, name string) prst.Effect {
		var any prst.Effect
		found := false
		for _, p := range b.Presets {
			for _, e := range p.Effects {
				if e.Module == module && strings.TrimSpace(e.Name) == name {
					if e.State == 1 {
						return e // prefer an enabled factory usage
					}
					if !found {
						any, found = e, true
					}
				}
			}
		}
		if found {
			return any
		}
		die("no factory block %s/%s in bundle", module, name)
		return prst.Effect{}
	}

	creeper := tmpl.CloneAs(0, spec.name, "")
	creeper.Bank = 0
	creeper.Volume = 65
	creeper.BPM = 92 // Creep tempo

	var cabCode uint32
	for i := range creeper.Effects {
		e := &creeper.Effects[i]
		if name, on := spec.blocks[e.Module]; on {
			*e = findBlock(e.Module, name)
			if e.Module == "CAB" {
				cabCode = e.Code
			}
		} else {
			e.State = 0
		}
	}
	// The desktop exporter resolves ppIRNum to the factory IR index of the
	// active CAB model (its low byte): it wrote 40 = 0x28 = "Sol 4x12" when
	// re-exporting a preset whose bundle value was 0. Mirror that here, or
	// the patch loads with the wrong cabinet IR.
	creeper.IRNum = fmt.Sprint(cabCode & 0xFF)

	b.Presets = []prst.Preset{*creeper} // patch files are single-preset only
	b.IRInfo.IRs = nil                  // ... and carry no ppIRInfo section
	b.Info.Count = 1
	b.Info.Time = fmt.Sprint(time.Now().UnixMilli()) // like the app's exporter

	path := *out
	if path == "" {
		path = spec.file
	}
	if err := b.Save(path); err != nil {
		die("save: %v", err)
	}
	fmt.Printf("wrote %s — %q (ppID=%d, bank=%d, ppIRNum=%s)\n", path, creeper.Name, creeper.ID, creeper.Bank, creeper.IRNum)
	for _, e := range creeper.Effects {
		state := "off"
		if e.State == 1 {
			state = "ON "
		}
		fmt.Printf("  x=%d %s %-4s %-14s 0x%08X\n", e.X, state, e.Module, e.Name, e.Code)
	}
}
