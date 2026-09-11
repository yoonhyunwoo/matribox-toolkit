// Command songtone builds song-specific guitar tones as single-preset
// Matribox patch files (the only patch format the desktop app accepts).
//
//	-song creep-chunk   Radiohead "Creep" pre-chorus rhythm crunch
//	-song creep-clean   Radiohead "Creep" verse clean
//	-song idly-chunk    My Chemical Romance "I Don't Love You" rhythm crunch
//	-song suspect-lead  한로로 "용의자" lead
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
	fmt.Fprintf(os.Stderr, "songtone: "+f+"\n", a...)
	os.Exit(1)
}

// toneSpec describes one researched tone.
type toneSpec struct {
	name   string // 12 chars max, space-padded like factory presets
	file   string
	bpm    int
	blocks map[string]string // module -> factory effect name to enable
}

var tones = map[string]toneSpec{
	// Radiohead "Creep" — Telecaster Plus -> Marshall Shredmaster -> clean
	// Fender combo. Research notes in the workspace research directory.
	"creep-chunk": {
		name: "Creep Chunk", file: "creep.prst", bpm: 92,
		blocks: map[string]string{
			"AMP": "B-Man N",     // Fender Bassman normal: pedal-friendly clean platform
			"FX2": "JP Dist",     // op-amp high-gain dist (Shredmaster stand-in)
			"CAB": "Viblux 1x12", // small Fender-style combo cab, tight lows
			"NR":  "Gate 2",      // single-coil hiss control under gain
			"RVB": "Spring",      // Fender spring reverb
		},
	},
	"creep-clean": {
		name: "Creep Clean", file: "creep-clean.prst", bpm: 92,
		blocks: map[string]string{
			"FX1": "COMP",        // light compression under the arpeggio
			"AMP": "B-Man N",     // sparkling clean Fender platform
			"CAB": "Viblux 1x12", // small combo cab keeps the low end tidy
			"MOD": "Tremolo",     // subtle pulsing (Kuassa's verse chain)
			"RVB": "Plate",       // light plate spread
		},
	},
	// My Chemical Romance "I Don't Love You" — Les Paul/Tele -> TS9 boost ->
	// borrowed Marshall JCM800 head (Rob Cavallo's) -> Marshall 4x12. The
	// verse is "clean with a little crunch, not overdrive"; the chorus is
	// the full Marshall crunch this patch targets.
	"idly-chunk": {
		name: "IDLY Chunk", file: "idly-chunk.prst", bpm: 88,
		blocks: map[string]string{
			"AMP": "Brit 800",    // Marshall JCM800 head
			"FX2": "Skreamer",    // TS9 Tube Screamer-style boost into the amp
			"CAB": "Brit75 4x12", // Marshall-style 4x12
			"NR":  "Gate 2",      // high-gain hiss control
			"RVB": "Room",        // light room; the track is fairly dry
		},
	},
	// 한로로 "용의자" — community tone recipes: RAT-style fuzz/distortion,
	// an TS808-family boost on top, delay+reverb mixed low. Lead goes
	// through a Vox-style clean platform (the RAT supplies all the gain).
	"suspect-lead": {
		name: "Suspect Lead", file: "suspect-lead.prst", bpm: 107,
		blocks: map[string]string{
			"FX1": "Boost",      // TS808-family boost for lead cut
			"FX2": "Dark Mouse", // RAT-style distortion/fuzz
			"AMP": "Voks 30N",   // Vox AC30-normal clean platform
			"NR":  "Gate 2",     // fuzz hiss control between phrases
			"CAB": "Jazz 2x12",  // attested pairing with Voks 30N ("Hang Over")
			"DLY": "Pure",       // plain delay, mix kept low
			"RVB": "Hall",       // light hall
		},
	},
}

func chainSlot(module string) int {
	order := []string{"FX1", "FX2", "AMP", "NR", "CAB", "EQ", "MOD", "DLY", "RVB"}
	for i, m := range order {
		if m == module {
			return i
		}
	}
	return 0
}

func main() {
	in := flag.String("in", "prsts.prst", "factory bundle")
	out := flag.String("out", "", "output patch file (default: per-song name)")
	song := flag.String("song", "creep-chunk", "tone to build: creep-chunk|creep-clean|idly-chunk|suspect-lead")
	flag.Parse()

	spec, ok := tones[*song]
	if !ok {
		die("unknown song %q (creep-chunk|creep-clean|idly-chunk|suspect-lead)", *song)
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

	patch := tmpl.CloneAs(0, spec.name, "")
	patch.Bank = 0
	patch.Volume = 65
	patch.BPM = spec.bpm

	var cabCode uint32
	for i := range patch.Effects {
		e := &patch.Effects[i]
		if name, on := spec.blocks[e.Module]; on {
			*e = findBlock(e.Module, name)
			if e.Module == "CAB" {
				cabCode = e.Code
			}
		} else {
			e.State = 0
		}
		e.X = chainSlot(e.Module) // factory blocks carry their source preset's slot
	}
	// The desktop exporter resolves ppIRNum to the factory IR index of the
	// active CAB model (its low byte): it wrote 40 = 0x28 = "Sol 4x12" when
	// re-exporting a preset whose bundle value was 0. Mirror that here, or
	// the patch loads with the wrong cabinet IR.
	patch.IRNum = fmt.Sprint(cabCode & 0xFF)

	b.Presets = []prst.Preset{*patch} // patch files are single-preset only
	b.IRInfo.IRs = nil                // ... and carry no ppIRInfo section
	b.Info.Count = 1
	b.Info.Time = fmt.Sprint(time.Now().UnixMilli()) // like the app's exporter

	path := *out
	if path == "" {
		path = spec.file
	}
	if err := b.Save(path); err != nil {
		die("save: %v", err)
	}
	fmt.Printf("wrote %s — %q (ppID=%d, bank=%d, ppIRNum=%s)\n", path, patch.Name, patch.ID, patch.Bank, patch.IRNum)
	for _, e := range patch.Effects {
		state := "off"
		if e.State == 1 {
			state = "ON "
		}
		fmt.Printf("  x=%d %s %-4s %-14s 0x%08X\n", e.X, state, e.Module, e.Name, e.Code)
	}
}
