# Song Tone Guide — how to add a researched song tone

How-to guide for AI agents. Read this end to end before writing code; it
encodes every hard-won constraint of the Matribox patch format. Companion
docs: [AGENTS.md](../AGENTS.md) (repo-wide agent contract) and
[prst-format-research.md](prst-format-research.md) (format analysis).

Goal: given a song name, produce a single-preset `.prst` patch file the
Matribox desktop app accepts, with effect settings traceable to evidence.

## Workflow overview

1. **Research** the original recording chain. Write a cited artifact.
2. **Map** the researched gear onto the factory effect catalog.
3. **Register** the tone in `cmd/songtone/main.go`.
4. **Generate** the patch file and **verify** it against the checklist.
5. **Commit** code + research artifact together.

Each step has hard rules. The two that have already caused user-visible
failures: non-factory parameter values produce wrong tones, and format drift
crashes the desktop app. Do not skip the checklists.

## Step 1 — Research the original chain

Collect, with sources: guitar(s) and pickups, drive pedals, amp model,
cabinet/speakers, and any time/space effects on the rhythm or lead part in
question. Then write `docs/research/<artist>-<song>-tone.md` with:

- One bullet per gear fact, each carrying a source URL.
- Confidence labels: **Verified** (official docs/manuals), **Probable**
  (reputable gear databases, artist interviews, rig rundowns), **Unverified**
  (forums, social posts, tab-site notes). Korean artists often have only
  Tier 5 sources — that is acceptable when labeled, never presented as fact.
- An "Open questions" section for anything unconfirmed (most studio settings
  are undocumented; say so).

Do not invent gear details to fill gaps. A tone built from thin research is
still shippable if the gaps are labeled.

## Step 2 — Map gear onto the factory catalog

The factory bundle (`prsts.prst`, 99 presets) is the catalog of truth. Every
effect that exists on the device appears there with an `effectCode`:

```
effectCode = (modulePrefix << 24) | algoID
```

| Module slot (chain x) | Prefix | Example names from the factory bundle |
|---|---|---|
| FX1 (x=0) | `0x00`–`0x05` | COMP, COMP2, Boost, AC Sim, Cry-W (wah) |
| FX2 (x=1) | `0x00`–`0x03` | Skreamer (TS-clone), Blues OD, JP Dist, Dark Mouse (RAT-style), Dist Plus |
| AMP (x=2) | `0x07` | TWD Deluxe, B-Man N (Bassman), Voks 30N (AC30), Brit 45, Brit 50JP (JMP/Plexi), Brit 800 (JCM800), Sol 100 OD/LD (Soldano), Eng 120, Calif DualV (Mesa), Superb CL (Fender Super) |
| NR (x=3) | none | Gate 1, Gate 2 |
| CAB (x=4) | `0x0A` | TWD 1x8, Viblux 1x12, Jazz 2x12, Brit75 4x12, Sol 4x12, … (32 total) |
| EQ (x=5) | `0x01` | Guitar EQ, Bass EQ |
| MOD (x=6) | `0x04` | Chorus A/B, Tremolo, Phaser, Flanger, Vibrato, Vibe |
| DLY (x=7) | `0x0B` | Pure, Warm, Tape, Ping Pong, Mag, 999 Echo, … |
| RVB (x=8) | `0x0C` | Room, Hall, Church, Plate, Spring, Sky, Sea, Mod RVB |

Mapping tips backed by prior tones:

- "RAT" family → `Dark Mouse`. "Tube Screamer"/TS9/TS808 boost → `Skreamer`
  (drive low, level high). "JCM800" → `Brit 800`. Plexi/JMP → `Brit 50JP`.
  "AC30" → `Voks 30N`. Fender Bassman/Twin-ish → `B-Man N`, `Superb CL`,
  `TWD Deluxe`. Bassman-as-pedal-platform → `B-Man N`.
- Check what the factory already pairs: a preset using your target AMP often
  carries a sensible CAB. Reuse that pairing when it fits the research.
- One note per module slot. MOD holds tremolo **or** chorus, not both.

## Step 3 — Register the tone

Add an entry to the `tones` map in `cmd/songtone/main.go` (git history has
worked examples). Rules enforced by the registry pattern:

- **Every enabled block must be spliced from a factory preset that uses the
  same effectCode.** Never hand-write `params_N` values — their device
  semantics are only partially known, and invented values ship wrong tones
  (the Dark Mouse block is bass-voiced for this exact reason; note such
  caveats in the entry comment).
- `findBlock` prefers a state=1 factory usage, so spliced blocks arrive
  switched on. Do not bypass it.
- `name` is truncated/padded to 12 characters by `CloneAs`. Pick a name that
  survives truncation ("Creep Chunk", "IDLY Chunk", "Beatle Clean").
- `file` follows `<song>-<part>.prst`, lowercase, ASCII (e.g.
  `idly-chunk.prst`). `bpm` is the song's tempo.
- CAB model choice determines the tone's IR: the code sets `ppIRNum`
  automatically (see Step 4), so pick the CAB deliberately.
- Multi-preset output is not supported — one tone, one patch file. This is
  user-mandated, and the app's fixed slot count makes appending dangerous.

## Step 4 — Generate and verify

```bash
go vet ./... && go test ./...                 # must be clean first
go run ./cmd/songtone -song <your-song> \
  -in /path/to/prsts.prst -out /path/to/<file>.prst
```

The factory bundle is user-local (not committed); ask for its path if the
default `prsts.prst` is absent — the test fixtures in `prst_test.go` cover
format checks without it.

Then verify the generated file. Every assertion below has bitten us before:

```python
import re
x = open("<file>.prst", "rb").read().decode()
assert x.count("<presets ") == 1 and 'count="1"' in x        # single preset
assert "ppIRInfo" not in x                                    # app: "wrong patch file" otherwise
assert x.count("></") == 0                                    # self-closing elements only
assert "\r\n" in x and (x.count("\n") - x.count("\r\n")) == 0  # CRLF endings
assert max(len(l) for l in x.split("\r\n")) <= 93             # line-wrap limit
m = re.search(r"<presets .*?</presets>", x, re.S)
slots = [d["x"] for d in (dict(re.findall(r'(\w+)="([^"]*)"', e.group(1)))
         for e in re.finditer(r"<Effect ([^>]+?)/>", m.group(0)))]
assert sorted(slots) == list("012345678"[:9])                 # one effect per chain slot
```

Also eyeball the tool's stdout: enabled blocks should be exactly the tone's
researched set, and `ppIRNum` must equal the CAB block's `effectCode & 0xFF`
(the exporter resolves the patch's IR reference to the factory cab index —
getting this wrong loads the preset with the wrong cabinet).

Known pitfalls, all previously observed:

- A spliced block carries its **source preset's chain slot**; the registry
  loop re-slots to canonical positions, so keep that loop intact.
- Appending a 100th preset crashed the app (fixed slot count). Patches are
  single-preset, full stop.
- `encoding/xml`-style output (LF, `<Effect ...></Effect>`) crashed the app.
  `prst.Save` hand-writes the factory style — leave it alone.
- `findBlock` fallback can return a state=0 block only when no enabled usage
  exists; if a block you expected ON arrives OFF, the block may simply have
  no enabled factory usage — splice from a different preset or note it.

## Step 5 — Commit

One commit per tone, message format:

```
feat(cmd/songtone): add <artist> '<song>' <part> patch (<song-key>)

<original chain in one line> — blocks spliced from factory preset
<name> which pairs exactly that set.
```

Push to `origin/main`. The research artifact from Step 1 goes in the same
commit if it lives in this repository.

## Source tone registry (as of 2026-09-13)

| Song key | Artist — song | Part | Chain |
|---|---|---|---|
| `creep-chunk` | Radiohead — Creep | pre-chorus rhythm | JP Dist → B-Man N → Gate 2 → Viblux 1x12 → Spring |
| `creep-clean` | Radiohead — Creep | verse clean | COMP → B-Man N → Viblux 1x12 → Tremolo → Plate |
| `idly-chunk` | My Chemical Romance — I Don't Love You | chorus rhythm | Skreamer → Brit 800 → Gate 2 → Brit75 4x12 → Room |
| `suspect-lead` | 한로로 — 용의자 | lead | Boost → Dark Mouse → Voks 30N → Gate 2 → Jazz 2x12 → Pure → Hall |
| `beatles-clean` | The Beatles | rhythm clean | COMP → Voks 30N → Jazz 2x12 → Plate |
| `am-idiot` | Green Day — American Idiot | rhythm | Brit 50JP → Guitar EQ → Brit75 4x12 → Spring |
