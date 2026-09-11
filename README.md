# matribox-toolkit

IR generator &amp; preset toolkit for the **Sonicake Matribox (QME-50)** guitar
multi-effects pedal, written in Go.

It does two things:

1. **Procedural cabinet IRs** — synthesizes guitar-cabinet impulse responses
   (exponentially decaying noise tail shaped by peaking-EQ biquads, plus early
   reflections) and writes them as 16-bit mono PCM WAV files for the pedal's
   15 user IR slots.
2. **`.prst` bundle editing** — parses and rewrites the preset bundle XML
   exported by the Matribox Setup desktop app, losslessly.

## Why hand-rolled parsing

The `.prst` bundle format uses dynamic element names (`ppIRInfo0`,
`ppEXP1_2`) and interleaved `params_N` attributes that `encoding/xml` struct
tags cannot express, so the parser uses custom marshalers. The guarantee is
simple: **anything that parses survives a `Load → Save → Load` cycle with its
structure intact**, and this invariant is enforced by a fuzz target
(`FuzzRoundTrip`), not just unit tests.

## Install

```bash
go install github.com/yoonhyunwoo/matribox-toolkit/cmd/qme50ir@latest
```

Or build from source:

```bash
git clone https://github.com/yoonhyunwoo/matribox-toolkit
cd matribox-toolkit
go build -o qme50ir ./cmd/qme50ir
```

## CLI usage

```bash
qme50ir list                                   # built-in cabinet catalog

# synthesize 3 variations of a 4x12 cabinet IR
qme50ir gen -out irs -n 3 -type "4x12 V30" -seed 42

# clone factory preset #5 onto user IR slot 168820742 ("User IR 7")
qme50ir preset -in prsts.prst -out new.prst \
  -template 5 -name "V30 Proced" -ir 168820742 -vol 75
```

`gen` flags: `-type` (cabinet name, see `list`), `-n` (variations), `-dur`
(IR length, default `300ms`), `-sr` (sample rate, default 44100), `-seed`
(reproducible variations). `preset` requires `-in`, `-out`, and `-ir`; the
`-ir` value must be a `ppIRNum` present in the bundle's IR table — invalid
numbers are rejected.

Import the generated WAVs through the Matribox desktop app's user IR import,
then load the edited bundle.

## Library usage

```go
import (
    "time"

    "github.com/yoonhyunwoo/matribox-toolkit/ir"
    "github.com/yoonhyunwoo/matribox-toolkit/prst"
)

// Synthesis is deterministic: same cabinet, options, and seed → same samples.
cab, _ := ir.CabinetByName("4x12 V30")
samples, err := cab.Generate(ir.Options{
    SampleRate: 44100,
    Duration:   300 * time.Millisecond,
    Seed:       42,
})
err = ir.WriteWAV("out.wav", samples, 44100)

// Bundle editing is lossless.
b, err := prst.Load("prsts.prst")
slot, err := b.FindIR("168820742")
tmpl, _ := b.PresetByID(5)
clone := tmpl.CloneAs(b.NextID(), "V30 Proced", slot.IRNum) // deep copy
b.Presets = append(b.Presets, *clone)
b.Info.Count = len(b.Presets)
err = b.Save("new.prst")
```

API contracts worth knowing:

- Synthesis output is bounded to [-1, 1], and
  `len(samples) == int(SampleRate × Duration.Seconds())`. Filter frequencies
  are clamped below Nyquist, so 44.1kHz-tuned cabinets render safely at any
  sample rate.
- Every preset carries a fixed 9-slot effect chain (RVB, DLY, MOD, EQ, CAB,
  NR, AMP, FX2, FX1) with 15 `params_N` attributes each — build new presets
  by cloning (`CloneAs`), not from scratch. Preset names are 12 characters,
  space-padded.

## Development

```bash
go test ./...                        # unit tests + fuzz crash corpus regressions

# fuzzing (one target per package at a time; crashes land in testdata/)
go test -fuzz FuzzLoad       -fuzztime 60s ./prst
go test -fuzz FuzzRoundTrip  -fuzztime 60s ./prst
go test -fuzz FuzzGenerate   -fuzztime 90s ./ir
go test -fuzz FuzzWriteWAV   -fuzztime 60s ./ir
```

The `.prst` format was reverse-engineered from a real factory export; the
full analysis (structure, `effectCode = module << 24 | algoID` table, IR
slots) lives in
[docs/prst-format-research.md](docs/prst-format-research.md).
Agent-facing engineering docs are in [AGENTS.md](AGENTS.md).

The synthesis approach follows the simulated-room-IR family used by
convolution reverb generators (e.g.
[adelespinasse/reverbGen](https://github.com/adelespinasse/reverbGen));
WAV I/O uses [go-audio/wav](https://github.com/go-audio/wav).

## Known limitations

- The synthesized IRs are procedural approximations, not measured cabinets.
- Whether a preset's `ppIRNum` field alone makes the pedal play the
  referenced user IR slot is **unverified on hardware**.
- Matribox generation compatibility is not guaranteed: community reports say
  Matribox 1 presets do not load on Matribox II / II Pro. Valeton GP-200
  `.prst` files share the extension but are a completely different format.
