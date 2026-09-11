// Command creep composes the "Creep Chunk" preset: a research-grounded
// rebuild of Radiohead's "Creep" rhythm-crunch tone on the Matribox QME-50.
// One-off tool; run from the module root.
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
	flag.Parse()

	b, err := prst.Load(*in)
	if err != nil {
		die("%v", err)
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

	creeper := tmpl.CloneAs(b.NextID(), "Creep Chunk", "")
	creeper.Bank = 1
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
	b.Presets = append(b.Presets, *creeper)
	b.Info.Count = len(b.Presets)
	if err := b.Save(*out); err != nil {
		die("save: %v", err)
	}
	fmt.Printf("wrote %s — preset %q ppID=%d\n", *out, creeper.Name, creeper.ID)
	for _, e := range creeper.Effects {
		state := "off"
		if e.State == 1 {
			state = "ON "
		}
		fmt.Printf("  x=%d %s %-4s %-14s 0x%08X\n", e.X, state, e.Module, e.Name, e.Code)
	}
}
