// Command creep builds "Creep Chunk" (Radiohead "Creep" rhythm crunch,
// research-grounded) as a single-preset Matribox patch file. Patch files are
// always single-preset; multi-preset output is not supported.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yoonhyunwoo/matribox-toolkit/prst"
)

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "creep: "+f+"\n", a...)
	os.Exit(1)
}

func main() {
	in := flag.String("in", "prsts.prst", "factory bundle")
	out := flag.String("out", "creep.prst", "output patch file")
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

	creeper := tmpl.CloneAs(0, "Creep Chunk", "")
	creeper.Bank = 0
	creeper.Volume = 65
	creeper.BPM = 92
	on := map[string]prst.Effect{
		"AMP": findBlock("AMP", "B-Man N"),
		"FX2": findBlock("FX2", "JP Dist"),
		"CAB": findBlock("CAB", "Viblux 1x12"),
		"NR":  findBlock("NR", "Gate 2"),
		"RVB": findBlock("RVB", "Spring"),
	}
	var cabCode uint32
	for i := range creeper.Effects {
		e := &creeper.Effects[i]
		if blk, ok := on[e.Module]; ok {
			*e = blk
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
	b.Info.Time = strconv.FormatInt(time.Now().UnixMilli(), 10) // like the app's exporter
	if err := b.Save(*out); err != nil {
		die("save: %v", err)
	}
	fmt.Printf("wrote %s — %q (ppID=%d, bank=%d)\n", *out, creeper.Name, creeper.ID, creeper.Bank)
	for _, e := range creeper.Effects {
		state := "off"
		if e.State == 1 {
			state = "ON "
		}
		fmt.Printf("  x=%d %s %-4s %-14s 0x%08X\n", e.X, state, e.Module, e.Name, e.Code)
	}
}
