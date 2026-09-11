# Sonicake Matribox `.prst` Preset File Format — Research Notes

Compiled: 2026-09-11. Tiers probed: Tier 1 (Sonicake official pages/manuals), Tier 2 (GitHub code/repos), Tier 4 (artist/blog pages), Tier 5 (Facebook/YouTube/forums). Confidence labels: **Verified** (Tier 1–2), **Probable** (Tier 3–4), **Unverified** (Tier 5).

## 1. What is a `.prst` file and how is it created/imported?

- The `.prst` file is the preset file format produced and consumed by the **Matribox desktop editor software** (Windows/macOS), distributed on Sonicake's official download pages. The QME-50 (original Matribox) page lists "Matribox Setup V1.1.1 for Mac and for Windows", firmware V1.1.0, USB ASIO driver V5.570, and manuals; the Matribox II page similarly lists Mac/Windows software and firmware. The software connects over USB and supports effects editing and NAM/IR imports (import features are explicitly referenced in the Windows 11 issue notes). — Verified. Sources: [QME-50 download page](https://www.sonicake.com/pages/matribox-software-firmware-download), [Matribox II download page](https://www.sonicake.com/pages/matribox-ii-firmware-software-download), [Matribox II Pro download page](https://www.sonicake.com/pages/matribox-ii-pro-software-firmware-1)
- Users export presets from the desktop software as `.prst` files and share them; third-party artists distribute `.prst` bundles (e.g., Alex Price's Matribox preset packs contain files such as `Deluxe.prst`, `LoneDrive.prst`). — Verified for the page's existence; **Probable** for the export workflow details (workflow inferred from distribution of `.prst` files and community editor import flows). Sources: [alexpricemusician.com/matribox](https://www.alexpricemusician.com/matribox)
- The Matribox shares software-suite infrastructure with Valeton/Hotone pedals: a community driver-fix repo describes "Valeton / Hotone / Sonicake pedals" sharing the same Windows 11 driver/transfer problems — evidence the Matribox editor is a rebranded member of the Valeton/Hotone "family" software line. — Verified (code comments in a Tier 2 repo). Source: [cesardamien/hotone-family-firmware-fix](https://github.com/cesardamien/hotone-family-firmware-fix)

## 2. Is the internal format documented? Reverse engineering

The format is **not officially documented** (no spec on any Sonicake page — the official download/manual pages make no mention of `.prst`). — Verified (absence on Tier 1 pages checked). However, there is substantial **community reverse engineering** (Tier 2):

### 2a. `Moustov/Andromidi` — Matribox II Pro preset decoder (Verified, Tier 2)

- The Android MIDI app [Moustov/Andromidi](https://github.com/Moustov/Andromidi) ships ~dozens of Matribox II Pro `.prst` files in `app/src/main/assets/` (e.g. `Mazzy.prst`, `60's OD.prst`, `AC Bass Sim.prst`) and documents the format in [`app/src/main/assets/preset.md`](https://github.com/Moustov/Andromidi/blob/main/app/src/main/assets/preset.md): the `.prst` file is **Base64 text**; decoding yields a bracketed, comma-separated list of byte integers such as `[3, 2, 0, 0, 16, 11, ...]`. Effect name lives around byte offsets 31–47 (NUL-terminated), style name around offsets 107–116; supported styles: Rock, Pop, Metal, Blues, Country, Jazz, Fusion, Folk, Funk, Acoustic. The doc includes a Python decoder/differ used to compare files like `60-C Matribox II PRO.prst` vs a v1 variant. — Verified.
- **Sample inspection (performed 2026-09-11)**: I downloaded [`Mazzy.prst`](https://raw.githubusercontent.com/Moustov/Andromidi/main/app/src/main/assets/Mazzy.prst) (1,476 ASCII chars, no line terminators; `file` says "ASCII text, with very long lines, with no line terminators"). First bytes: `WzMsMiwwLDAsMTYsMTEsMCwxMjgsMCw1LDEsNCwzLDEyLDEsNSwxLDE1LDEwNSwyLDEwNSwxNjQsMiwwLDIsMSw1NCwyMTYsMTYyLDgwLDcs...`. Base64-decoding gives 1,106 bytes of ASCII text `[3,2,0,0,16,11,0,128,0,5,1,4,3,12,1,5,1,15,105,2,105,164,2,0,2,1,54,216,162,80,77,97,122,122,121,0,...]`. Parsing the integer list as bytes (391 values): ASCII string **"Mazzy"** begins at byte offset ~31 (NUL-terminated), and the string **"...ox II PRO"** (i.e., "Matribox II PRO") appears a few bytes later — a model identifier embedded in the preset. Bytes 26–29 (`54,216,162,80`) are plausibly a 32-bit value (per the editor below, offsets 26–29 are a little-endian Unix timestamp). No compression; no `TSRP` magic — this is a *different container* from Valeton GP-200 `.prst` (see §3). — Verified (direct hex/text analysis of a Tier 2 sample).

### 2b. `maikonH/matribox-ai-generator` — full `.prst` editor (Verified, Tier 2)

- [maikonH/matribox-ai-generator](https://github.com/maikonH/matribox-ai-generator) (Vite/TypeScript web app, Portuguese UI) contains `src/components/PrstEditor.tsx` and `src/lib/prstEditor.ts` plus an effect catalog `src/data/alg_data.json` (Modules → alg → fxid/fxtitle/widgets with min/max/step). Its decoder (`decodePrst` in `prstEditor.ts`) accepts three container variants: raw Base64, a **JSON wrapper `{"data": "<base64>"}`**, or a data-URI-prefixed Base64 string. Decoded payload is raw binary with: preset name (string), a **little-endian Unix timestamp at byte offsets 26–29** (rewritten on export), effect **blocks** identified by a 32-bit little-endian `fxid` (amp fxid prefixes 0x04/0x05/0x07/0x08; cab fxids 0x0A000000–0x0A000045), an amp block optionally followed by a linked cab header at `block.start + 5`, and parameter lists of **IEEE-754 little-endian floats** (4 bytes each). `encodePrst` writes the file back out as a `.prst` download. — Verified (source code read directly).
- Interpretation caveat: the Andromidi sample (§2a) decodes to *ASCII bracketed integers*, while `prstEditor.ts` treats the Base64 payload as *raw binary*. Both were described from real Matribox II Pro files; the exact relationship (two sub-format generations, vs. editor-vs-device export variants) is unresolved. — flagged as open question.

### 2c. Related but distinct: Valeton GP-200 `.prst` (Verified, Tier 2 — different format)

- [dax99993/libPRST](https://github.com/dax99993/libPRST) (fork of mikeliddle/PRSTDecoder) is a Python codec for **Valeton GP-200 v1.8** `.prst` files: binary format with magic bytes **"TSRP"** ("PRST" reversed), version, product_id "GP-2", firmware version, timestamp, size, metadata (BPM, volume, pan, category, name, author), chain/module ordering, per-module effect codes and float parameters, expression pedals, control switches, and a checksum; no compression; decodes to JSON. — Verified. Source: [libPRST README](https://github.com/dax99993/libPRST), [mikeliddle/PRSTDecoder](https://github.com/mikeliddle/PRSTDecoder) (sample `examples/V1.prst`).
- The Matribox `.prst` is **not** this format (no TSRP magic observed in the Matribox II Pro sample), despite the identical file extension and the Valeton/Hotone/Sonicake family connection. A Facebook Valeton Users thread about loading Matribox presets on Valeton GP units confirms cross-import attempts. — Verified (format difference) / Unverified (the FB thread). Source: [facebook.com/groups/ValetonUsers/posts/1243528853362810](https://www.facebook.com/groups/ValetonUsers/posts/1243528853362810/)

## 3. Cross-model compatibility (QME-50 ↔ Matribox II / II Pro QME-200)

- **Presets are generally NOT cross-compatible between Matribox generations.** Guitarist Alex Price's preset page states explicitly: "The presets on this page are only compatible with the original Matribox 1 and NOT the Matribox II or Matribox II Pro," with ports to Matribox II listed as in-progress. — Probable (Tier 4 artist page, explicit statement). Source: [alexpricemusician.com/matribox](https://www.alexpricemusician.com/matribox)
- Community reports of **"import failed / file is not compatible" errors** when importing mismatched `.prst` files on the Matribox 2 Pro, and discussion of whether Matribox I/II presets load on Valeton GP units, corroborate per-model binding (the model string embedded in the file — e.g., "Matribox II PRO" in `Mazzy.prst` — likely participates in validation). — Unverified (Tier 5). Sources: [facebook.com/groups/ligamusical/posts/25565967016363528](https://www.facebook.com/groups/ligamusical/posts/25565967016363528/), [facebook.com/groups/ValetonUsers/posts/1243528853362810](https://www.facebook.com/groups/ValetonUsers/posts/1243528853362810/)
- No official Sonicake statement on `.prst` cross-generation compatibility was found on any Tier 1 page. — Verified absence. [Official download pages](https://www.sonicake.com/pages/matribox-software-firmware-download)

## 4. Where users share `.prst` files

- **Facebook**: [Sonicake Matribox Worldwide](https://www.facebook.com/groups/1773605073058797/) group (preset sharing), Valeton Users group (cross-brand attempts), ligamusical group posts. — Unverified (Tier 5).
- **Artist pages**: Alex Price's free/paid Matribox preset packs ([alexpricemusician.com/matribox](https://www.alexpricemusician.com/matribox)); Jimmy Cooper's free Matribox II Pro presets and IRs (`jimmy-cooper.mykajabi.com/Matribox-2-Pro`, linked from his YouTube review). — Probable (Tier 4).
- **GitHub**: `Moustov/Andromidi` (factory Matribox II Pro `.prst` files in-repo), `maikonH/matribox-ai-generator` (editor/generator), `dax99993/libPRST` + `mikeliddle/PRSTDecoder` (Valeton only). — Verified (Tier 2).
- **YouTube**: preset demo videos with download links in descriptions (e.g., [Matribox QME-50 – All The Presets](https://www.youtube.com/watch?v=q8w-d5PZnN4), [16 Presets On The Sonicake Matribox II](https://www.youtube.com/watch?v=puE9jqjyBRk), [NAM profiles on Matribox II Pro](https://www.youtube.com/watch?v=CGMClYa_-LU)). — Unverified (Tier 5).
- **heyworshipleader.com**: no Matribox `.prst` content surfaced in searches. — noted gap.

## 5. Sample files and tools (locations)

| Item | Location | Notes |
|---|---|---|
| ~40 Matribox II Pro factory `.prst` samples | https://github.com/Moustov/Andromidi/tree/main/app/src/main/assets | Base64→bracketed-integer-ASCII format; inspected `Mazzy.prst` |
| `.prst` format notes + Python decoder | https://github.com/Moustov/Andromidi/blob/main/app/src/main/assets/preset.md | offsets for effect/style names, style list |
| Web `.prst` editor (amp/cab, fxid + float params) | https://github.com/maikonH/matribox-ai-generator (`src/lib/prstEditor.ts`, `src/data/alg_data.json`) | LE timestamp @ bytes 26–29; fxid-prefixed blocks; IEEE-754 LE floats |
| Effect/algorithm catalog (fxids, widgets) | https://github.com/maikonH/matribox-ai-generator/blob/main/src/data/alg_data.json | AMP/CAB modules with fxid, titles, parameter ranges |
| Valeton GP-200 codec (different format, same extension) | https://github.com/dax99993/libPRST | TSRP magic; product_id GP-2 |

## Addendum (2026-09-11): real QME-50 export sample analyzed

A genuine QME-50 preset bundle (`prsts.prst`, 411 KB, exported via Matribox Setup **V1.0.3 / firmware 1.1**, product string "Maxtribox" [sic], MAC platform, ms-epoch time 1789136041945 ≈ 2026-09-11) was analyzed and **resolves the container question for Matribox 1**: the software's multi-preset export is a **plain UTF-8 XML document** (CRLF), not Base64:

- `<preset_info software firmware product count platform time>` header, `count="99"`.
- `<ppIRInfo>` table of 15 user IR slots: `ppIRNum` (sequential device IDs 168820736+), `ppIRName`, signed `ppIRCRC`.
- 99 `<presets ppBank ppName ppVolume ppID ppBPM ppIRNum ppType ppAuthor ppNotes ppTypeName>` elements — factory dump is banks 0–?, ids 0–98, names padded to 12 chars ("MatriBox   ", "60's OD  "), `ppTypeName` ∈ Rock/Pop/Metal/Blues/Country/Jazz/Fusion/Folk/Funk/Acoustic.
- Each preset has exactly 9 `<Effect>` elements (fixed chain), always RVB→DLY→MOD→EQ→CAB→NR→AMP→FX2→FX1 with chain position `x="0..8"`, ~5.6 enabled (`effectState="1"`) per preset, and exactly 15 `params_N` attributes each.
- **`effectCode` = `(modulePrefix << 24) | algoID`**: RVB `0x0C`, DLY `0x0B`, CAB `0x0A0000xx` (matches maikonH's cab range), AMP `0x07xxxxxx`, FX1/wah `0x05`, MOD `0x04`, FX2/dist `0x03`, EQ `0x01`, and plain small ints (<0x20) for NR/gates and FX1 comp/boost (no prefix). This corroborates the II Pro fxid scheme; whether the algo tables are identical across generations is still open.
- `ppEXP1` (expTarget/expVolume/min/max) and `ppCtrl` (c11..c23 knob mappings) complete each preset.
- Params are mixed-scale raw device values (percents 0–100, packed ints like 65535/12800, floats like 400.4) — same style as the II Pro float/packed params.

So `.prst` files are not one format: the desktop software's **bundle export is XML**, while device/firmware-oriented presets (Andromidi II Pro samples) are Base64-wrapped binary. Mapping between the XML `effectCode` and the Base64 payload `fxid` is direct (same prefix scheme).

## Addendum 2 (2026-09-11): Go ecosystem precedent for an IR generator

- No dedicated Go IR-generation library exists (searched GitHub: reverb/IR generators are JS/C++/VST). The idiomatic Go audio stack is **gonum/dsp/fourier** (FFT) + **go-audio/wav** (PCM WAV encode/decode) glued with a small conversion layer. Sources: [go-audio/wav](https://pkg.go.dev/github.com/go-audio/wav), [gonum fourier](https://pkg.go.dev/gonum.org/v1/gonum/dsp/fourier) — Verified (Tier 2).
- Closest algorithmic reference is [adelespinasse/reverbGen](https://github.com/adelespinasse/reverbGen) (JS): simulated room IRs via exponentially decaying noise + early reflections — the same family the `qme50ir` Go implementation uses. — Verified (Tier 2). Others surveyed: KlangFalter, reevr, GeneticReverb (all C++/VST, convolution playback rather than generation).

## Open questions

- **Tiers attempted**: Tier 1 (official download pages, manual PDF — downloaded `Matribox_Online Manual_Firmware V1.0.4.pdf` but could not extract text: no `pdftotext` available, naive PDF stream extraction failed), Tier 2 (GitHub search: `matribox`, `prst`, code search for `extension:prst`), Tier 3 (N/A — no standards), Tier 4 (artist preset pages), Tier 5 (Facebook/YouTube/forum snippets via search summaries).
- Which container variant is canonical per model/firmware: Andromidi's samples decode to ASCII bracketed integer lists while `prstEditor.ts` parses raw binary — same format at different generations, or editor-vs-device exports? (Both repos target Matribox II Pro.)
- Whether the QME-50 (Matribox 1) format is byte-identical to Matribox II / II Pro beyond the embedded model string — no QME-50 sample `.prst` was obtained (Alex Price's free bundle requires cart checkout; not fetched).
- Exact checksum/validation scheme for Matribox `.prst` (libPRST documents one for Valeton GP-200 only; Andromidi's SysEx section is empty).
- Byte-level semantics of the Andromidi ASCII-integer payload beyond name/style offsets (only Andromidi's `preset.md` covers these).
- Whether Jimmy Cooper's Matribox II Pro preset bundle and the Valeton-Users cross-import thread reflect firmware-version-dependent compatibility (e.g., NAM-era V1.1.x firmware changing the schema).
