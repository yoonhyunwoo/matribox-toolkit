// Command qme50ir generates procedural cabinet IRs and builds Matribox
// (QME-50) preset bundles around them.
//
// Usage:
//
//	qme50ir list
//	qme50ir gen  -out DIR -n COUNT -type "4x12 V30" [-dur 300ms] [-sr 44100] [-seed N]
//	qme50ir preset -in BUNDLE.prst -out OUT.prst -template PPID -name NAME \
//	               -ir PP_IR_NUM [-bank N] [-vol N] [-bpm N] [-genre Rock]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yoonhyunwoo/matribox-toolkit/ir"
	"github.com/yoonhyunwoo/matribox-toolkit/prst"
)

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "qme50ir: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "list":
		cmdList()
	case "gen":
		cmdGen(os.Args[2:])
	case "preset":
		cmdPreset(os.Args[2:])
	case "-h", "-help", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "qme50ir: unknown command %q\n\n", os.Args[1])
		usage()
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `qme50ir — Matribox (QME-50) IR generator & preset builder

Commands:
  list    show the built-in cabinet catalog
  gen     synthesize IR WAV files
  preset  clone a preset in a .prst bundle onto a user IR
`)
	os.Exit(2)
}

func cmdList() {
	fmt.Println("built-in cabinet catalog:")
	for _, c := range ir.Catalog {
		fmt.Printf("  %-14s low=%4.0fHz(+%.1fdB) highcut=%5.0fHz tilt=%+.0fdB decay=%v\n",
			c.Name, c.LowBump, c.LowGain, c.HighCut, c.Brightness, c.Decay)
	}
}

func cmdGen(args []string) {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	outDir := fs.String("out", "irs", "output directory")
	n := fs.Int("n", 1, "number of IR variations to generate")
	typ := fs.String("type", "4x12 V30", "cabinet name (see `qme50ir list`)")
	dur := fs.Duration("dur", 300*time.Millisecond, "IR length (Go duration, e.g. 300ms)")
	sr := fs.Int("sr", 44100, "sample rate")
	seed := fs.Int64("seed", time.Now().UnixNano(), "random seed")
	fs.Parse(args)

	cab, err := ir.CabinetByName(*typ)
	if err != nil {
		die("%v", err)
	}
	if *n <= 0 {
		die("-n must be positive, got %d", *n)
	}
	for i := 0; i < *n; i++ {
		s := *seed + int64(i)*7919
		samples, err := cab.Generate(ir.Options{SampleRate: *sr, Duration: *dur, Seed: s})
		if err != nil {
			die("%v", err)
		}
		name := fmt.Sprintf("%s_%s_%02d.wav",
			strings.NewReplacer(" ", "").Replace(cab.Name),
			fmt.Sprint(s%100000), i+1)
		path := filepath.Join(*outDir, name)
		if err := ir.WriteWAV(path, samples, *sr); err != nil {
			die("%v", err)
		}
		fmt.Printf("wrote %s (%d samples, %v)\n", path, len(samples), *dur)
	}
}

func cmdPreset(args []string) {
	fs := flag.NewFlagSet("preset", flag.ExitOnError)
	in := fs.String("in", "", "input .prst bundle (required)")
	out := fs.String("out", "", "output .prst bundle (required)")
	tmplID := fs.Int("template", 0, "ppID of the preset to clone")
	name := fs.String("name", "My IR Tone", "new preset name (max 12 chars)")
	irNum := fs.String("ir", "", "ppIRNum of the user IR slot (required)")
	bank := fs.Int("bank", 1, "ppBank for the new preset")
	vol := fs.Int("vol", -1, "preset volume 0-99 (default: template value)")
	bpm := fs.Int("bpm", -1, "BPM (default: template value)")
	genre := fs.String("genre", "", "genre name (default: template value)")
	fs.Parse(args)

	if *in == "" || *out == "" || *irNum == "" {
		die("preset: -in, -out and -ir are required")
	}
	b, err := prst.Load(*in)
	if err != nil {
		die("%v", err)
	}
	slot, err := b.FindIR(*irNum)
	if err != nil {
		die("%v", err)
	}
	tmpl, err := b.PresetByID(*tmplID)
	if err != nil {
		die("%v", err)
	}
	newPreset := tmpl.CloneAs(b.NextID(), *name, slot.IRNum)
	newPreset.Bank = *bank
	if *vol >= 0 {
		newPreset.Volume = *vol
	}
	if *bpm >= 0 {
		newPreset.BPM = *bpm
	}
	if *genre != "" {
		newPreset.TypeName = *genre
	}
	b.Presets = append(b.Presets, *newPreset)
	b.Info.Count = len(b.Presets)
	if err := b.Save(*out); err != nil {
		die("save: %v", err)
	}
	fmt.Printf("added preset %q (ppID=%d, bank=%d) on user IR %s (%s)\n",
		strings.TrimSpace(newPreset.Name), newPreset.ID, newPreset.Bank, slot.IRNum, slot.Name)
	fmt.Printf("wrote %s (%d presets)\n", *out, len(b.Presets))
}
