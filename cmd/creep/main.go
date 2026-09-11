// Command creep builds the "Creep Chunk" preset (Radiohead "Creep" rhythm
// crunch, research-grounded) into a Matribox .prst bundle.
//
// Modes:
//
//	-single    output a bundle containing only the Creep preset (default)
//	-replace   overwrite factory preset -slot inside a full bundle
//	-identity  load and re-save the bundle with no edits (writer validation)
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/yoonhyunwoo/matribox-toolkit/prst"
)

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "creep: "+f+"\n", a...)
	os.Exit(1)
}

func main() {
	in := flag.String("in", "prsts.prst", "factory bundle")
	out := flag.String("out", "creep-rhythm.prst", "output bundle")
	mode := flag.String("mode", "single", "single|replace|identity")
	slot := flag.Int("slot", 98, "ppID to overwrite in replace mode")
	flag.Parse()

	b, err := prst.Load(*in)
	if err != nil {
		die("%v", err)
	}
	if *mode == "single" {
		*slot = 0
	}
	if *mode == "identity" {
		if err := b.Save(*out); err != nil {
			die("save: %v", err)
		}
		fmt.Printf("wrote %s (identity re-save, %d presets)\n", *out, len(b.Presets))
		return
	}

	tmpl, err := b.PresetByID(0)
	if err != nil {
		die("%v", err)
	}
	findBlock := func(module, name string) prst.Effect {
		for _, p := range b.Presets {
			for _, e := range p.Effects {
				if e.Module == module && strings.TrimSpace(e.Name) == name {
					return e
				}
			}
		}
		die("no factory block %s/%s in bundle", module, name)
		return prst.Effect{}
	}

	creeper := tmpl.CloneAs(*slot, "Creep Chunk", "")
	creeper.Bank = 0
	if *mode == "replace" {
		creeper.Bank = b.Presets[*slot].Bank
	}
	creeper.Volume = 65
	creeper.BPM = 92
	on := map[string]prst.Effect{
		"AMP": findBlock("AMP", "B-Man N"),
		"FX2": findBlock("FX2", "JP Dist"),
		"CAB": findBlock("CAB", "Viblux 1x12"),
		"NR":  findBlock("NR", "Gate 2"),
		"RVB": findBlock("RVB", "Spring"),
	}
	for i := range creeper.Effects {
		e := &creeper.Effects[i]
		if blk, ok := on[e.Module]; ok {
			*e = blk
		} else {
			e.State = 0
		}
	}
	if *mode == "replace" {
		b.Presets[*slot] = *creeper
	} else { // single
		b.Presets = []prst.Preset{*creeper}
	}
	b.Info.Count = len(b.Presets)
	if err := b.Save(*out); err != nil {
		die("save: %v", err)
	}
	fmt.Printf("wrote %s — %s mode: slot %d = %q (%d presets)\n", *out, *mode, *slot, creeper.Name, len(b.Presets))
	for _, e := range creeper.Effects {
		state := "off"
		if e.State == 1 {
			state = "ON "
		}
		fmt.Printf("  x=%d %s %-4s %-14s 0x%08X\n", e.X, state, e.Module, e.Name, e.Code)
	}
}
