# AGENTS.md — qme50ir

Guidance for AI coding agents working in this repository. Every claim below is
verified against the code as of 2026-09-12; if the code contradicts this file,
trust the code and flag the mismatch to the user.

## What this repository is

`qme50ir` is a Go CLI and library for the Sonicake Matribox (QME-50) guitar
multi-effects pedal. It does two things:

1. Synthesizes procedural cabinet impulse responses (IRs) as 16-bit mono PCM
   WAV files for the pedal's 15 user IR slots.
2. Reads and edits the pedal's `.prst` preset bundle XML (the format exported
   by the Matribox Setup desktop app, e.g. `prsts.prst`).

Zero runtime dependencies except `github.com/go-audio/wav` (and its
`go-audio/audio`, `go-audio/riff` transitive deps) for WAV encoding. The
`.prst` parser is hand-rolled on `encoding/xml` because the format uses
dynamic element names (`ppIRInfo0`, `ppEXP1_2`) and interleaved attributes
(`params_N`) that struct tags cannot express.

## Commands

Run everything from the repository root (`qme50ir/`).

```bash
go build ./...                       # compile
go test ./...                        # unit + regression (fuzz crash corpus) tests
go vet ./... && gofmt -l .           # must be clean before you report done

go build -o /tmp/qme50ir ./cmd/qme50ir   # build the CLI binary

go test -fuzz FuzzLoad -fuzztime 60s -run '^$' ./prst        # parser fuzzing
go test -fuzz FuzzRoundTrip -fuzztime 60s -run '^$' ./prst   # round-trip invariant fuzzing
go test -fuzz FuzzGenerate -fuzztime 90s -run '^$' ./ir      # synthesis fuzzing
go test -fuzz FuzzWriteWAV -fuzztime 60s -run '^$' ./ir      # WAV encoder fuzzing
```

Fuzzing rules: run **one fuzz target per package at a time**. Two concurrent
`go test -fuzz` invocations on the same package interfere and drop throughput
from ~37k execs/min to ~11. Crashing inputs land in `ir/testdata/fuzz/` or
`prst/testdata/fuzz/` and run as unit tests on every `go test`; keep them.

CLI smoke test (end to end, no hardware needed):

```bash
/tmp/qme50ir list
/tmp/qme50ir gen -out irs -n 2 -type "4x12 V30"
/tmp/qme50ir preset -in prsts.prst -out new.prst -template 5 \
  -name "V30 Proced" -ir 168820742
```

`prsts.prst` is a real factory bundle (99 presets) used for e2e checks; it is
not committed here. If it is absent, generate a fixture via the tests
(`prst_test.go` embeds a minimal valid bundle).

## Repository map

| Path | Responsibility |
|---|---|
| `ir/ir.go` | IR synthesis: decaying noise tail, RBJ peaking biquads, early reflections, WAV encoding via `go-audio/wav` |
| `ir/ir_test.go` | Determinism, bounds, length, Nyquist regression tests |
| `ir/fuzz_test.go` | `FuzzGenerate`, `FuzzWriteWAV` harnesses |
| `prst/prst.go` | `.prst` bundle parser/writer with custom marshalers for dynamic element names |
| `prst/prst_test.go` | Parse, round-trip, clone, name-padding tests |
| `prst/fuzz_test.go` | `FuzzLoad`, `FuzzRoundTrip` harnesses |
| `cmd/qme50ir/main.go` | CLI: `list`, `gen`, `preset` subcommands |
| `README.md` | Human-facing overview (Korean) |

## API contracts

### `ir` package — synthesis

```go
cab, err := ir.CabinetByName("4x12 V30")            // error on unknown name
samples, err := cab.Generate(ir.Options{            // deterministic output
    SampleRate: 44100,
    Duration:   300 * time.Millisecond,
    Seed:       42,
})
err := ir.WriteWAV(path, samples, 44100)            // 16-bit mono PCM
```

Invariants you must not break:

- **Determinism**: identical cabinet, options, and seed produce identical
  samples. `Options.Seed` is the only randomness source.
- **Bounds**: output samples are in [-1, 1]; peak-normalized to 0.9.
- **Length**: `len(samples) == int(float64(SampleRate) * Duration.Seconds())`.
  `Duration` is a `time.Duration`, so sub-microsecond requests truncate.
- **Nyquist clamp**: filter frequencies go through `nyquistSafe`. A cabinet
  tuned for 44.1kHz rendered at a low sample rate (e.g. HighCut 4200Hz at
  sr=8000) previously emitted NaNs — the fuzz corpus in
  `ir/testdata/fuzz/FuzzGenerate/` locks this regression in.
- Zero-value `Options` falls back to 44100Hz / 300ms via `Defaults()`.
  Non-positive values after defaults fail `validate()` with an error.

### `prst` package — bundle editing

```go
b, err := prst.Load("prsts.prst")
slot, err := b.FindIR("168820742")       // ppIRNum must exist in the IR table
tmpl, err := b.PresetByID(5)
clone := tmpl.CloneAs(b.NextID(), "V30 Proced", slot.IRNum)  // deep copy
b.Presets = append(b.Presets, *clone)
b.Info.Count = len(b.Presets)
err = b.Save("new.prst")
```

Invariants you must not break:

- **Round-trip fidelity**: any bundle that parses must survive
  `Load → Save → Load` with identical preset count, effect count per preset,
  `effectCode`, all 15 `params_N` values, EXP presence and child count, and
  IR slot count. `FuzzRoundTrip` enforces this.
- **Dynamic names**: children are emitted as `ppIRInfoN` and `ppEXP1_N`
  (slot indices from parse), never as Go type names (`UserIR`, `EXPChild`).
  `prst_test.go` asserts this; do not "simplify" the custom marshalers into
  struct tags.
- **Effect ordering**: each preset holds exactly 9 `<Effect>` children in
  fixed chain order RVB, DLY, MOD, EQ, CAB, NR, AMP, FX2, FX1, with chain
  position `x="0..8"` and 15 `params_N` attributes each. New presets must
  clone an existing one (`CloneAs`) rather than be built from scratch.
- **Names are 12 chars, space-padded** (`padName`): longer names truncate.
  `CloneAs` applies this; keep it that way so device display matches factory
  presets.

## `.prst` format quick reference

Verified by hex/XML analysis of a real factory export (software 1.0.3,
firmware 1.1) — full notes in `docs/prst-format-research.md`.

- Root `<Matribox>` → `<preset_info>` header, `<ppIRInfo>` table (15 user IR
  slots: `ppIRNum` device ID, `ppIRName`, `ppIRCRC`), then N `<presets>`.
- `effectCode = (modulePrefix << 24) | algoID`:

  | Module | Prefix | Example |
  |---|---|---|
  | RVB (reverb) | `0x0C` | Hall = `0x0C000001` |
  | DLY (delay) | `0x0B` | Warm = `0x0B00000D` |
  | CAB (cabinet) | `0x0A` | `0x0A000001`…`0x0A000045` |
  | AMP | `0x07` | Calif DualV = `0x07000068` |
  | FX1 (wah) | `0x05` | Cry-W = `0x05000008` |
  | MOD | `0x04` | Chorus A = `0x04000000` |
  | FX2 (dist) | `0x03` | JP Dist = `0x0300002A` |
  | EQ | `0x01` | Guitar EQ = `0x01000015` |
  | NR / FX1 comp-boost | none | Gate 2 = `29`, COMP = `0`, Boost = `26` |

- Per-preset tail: `<ppEXP1>` (expression pedal, wraps 0–3 `ppEXP1_N` children
  mapping pedal → effect parameter) and `<ppCtrl c11..c23>` (3 knob mappings).
- This format is **not** the Valeton GP-200 `.prst` (TSRP magic, `libPRST`) —
  same extension, incompatible container. Do not cross-import.
- Generation compatibility: Matribox 1 presets reportedly do not load on
  Matribox II/II Pro (community-reported, unverified mechanism; the file
  embeds a model string that likely participates in validation).

## Known open questions

- Whether the CAB effect block's user-IR reference field is exactly
  `presets.ppIRNum` is **unverified on hardware** — presets set `ppIRNum` but
  no physical test has confirmed the pedal plays the referenced slot. Treat
  hardware verification as the next milestone and ask the user for test
  results before changing CAB-related code.
- The synthesized IRs are procedural approximations, not measured cabinets;
  they have never been auditioned on the pedal.

## Working conventions

- Language: Go 1.25, standard library style. Error strings start lowercase
  and wrap with `%w`. Public API returns errors; nothing panics on bad input
  (the fuzz harnesses enforce this — a panic in a fuzz target is a bug).
- Types carry units: `ir.Hz`, `ir.dB`, `time.Duration`. Keep it that way.
- Every new bug fixed from fuzzing: keep the crashing input in `testdata/`,
  add a named regression test if the input is non-obvious.
- Docs for humans live in `README.md` (Korean); this file is for agents
  (English). Update both when behavior changes.
