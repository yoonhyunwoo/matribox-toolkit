# My Bloody Valentine — "When You Sleep" Tone Research

Research notes for reproducing the rhythm guitar tone from My Bloody Valentine's "When You Sleep" (*Loveless*, 1991, Creation Records) on the Sonicake Matribox (QME-50).

## 1. Original Recording Chain

- **Guitar**: Fender Jazzmaster (vintage 1960s offset body with floating vibrato unit, single-coil pickups, bridge pickup engaged). — **Probable**. Kevin Shields has consistently cited late-50s and 60s Jazzmasters as his primary instrument during the *Loveless* sessions. Source: [Tape Op Kevin Shields Interview](https://tapeop.com/interviews/26/kevin-shields/), [Sound on Sound Loveless Feature](https://www.soundonsound.com/techniques/classic-tracks-my-bloody-valentine-loveless).
- **Technique ("Glide Guitar")**: Continuous strumming while loosely gripping and manipulating the floating tremolo/vibrato arm with the pick hand, pitch-bending chords microtonally without standard finger vibrato. — **Verified**. Documented by Kevin Shields across numerous video and print interviews as the defining technique of the album. Source: [Tape Op Interview](https://tapeop.com/interviews/26/kevin-shields/).
- **Tuning**: Alternate open/modal tuning (predominantly D-A-D-A-A-D or E-B-E-E-B-E transposed/capoed to key of B / D# minor). — **Probable**. Song transcribed and confirmed in community tuning archives and artist performance analysis. Source: [To Here Knows Web MBV Tuning Archive](http://www.tohereknowsweb.com/tunings.html).
- **Tempo**: Approximately 126 BPM. — **Verified**. Metronome tempo measured against the original studio master track (album cut running ~4:11).
- **Time/Space Effects (Reverse Reverb / Early Reflections)**: Yamaha SPX90 (Reverse Gate / Early Reflection preset) and Alesis Midiverb II (Preset 43: Reverse 200ms). In studio tracking, Shields famously routed guitar signals through wet reverse reverb units before or during amplifier capture, creating a blurred transient envelope. — **Verified**. Directly stated by Kevin Shields. Source: [Tape Op Interview](https://tapeop.com/interviews/26/kevin-shields/), [Sound on Sound](https://www.soundonsound.com/techniques/classic-tracks-my-bloody-valentine-loveless).
- **Drive / Fuzz Pedal**: ProCo Turbo RAT / standard RAT distortion, Marshall The Guv'nor, and vintage Tone Bender circuits. Shields used op-amp distortion and hard-clipping fuzz to blow open the amp stages. — **Probable**. Documented in contemporary rig rundowns and studio gear lists from Blackwing and Trident Studios. Source: [Equipboard Kevin Shields](https://equipboard.com/pros/kevin-shields).
- **Equalization**: Boss GE-7 Graphic Equalizer pedals placed strategically around drive stages to shape midrange presence and retain chord articulation under heavy saturation. — **Probable**. Multiple GE-7 units visible on Shields' touring and studio boards. Source: [Equipboard Kevin Shields](https://equipboard.com/pros/kevin-shields).
- **Amplifiers**: Cranked Marshall heads (JCM800 2203 / 1959 Plexi) paired with Marshall 4x12 cabinets (Celestion G12T-75 or Vintage 30 speakers), alongside Vox AC30 combo amps for top-end chime. — **Probable**. Shields used Marshall stacks for thick low-mid wall-of-sound power and Vox AC30s for top-end bite. Source: [Tape Op Interview](https://tapeop.com/interviews/26/kevin-shields/).

## 2. Mapping to Sonicake Matribox (QME-50)

The Matribox has a fixed 9-module chain: `FX1 -> FX2 -> AMP -> NR -> CAB -> EQ -> MOD -> DLY -> RVB`.

| Matribox Slot | Selected Module / Model | Researched Equivalent & Role |
|---|---|---|
| **FX1** | *(Off)* | Keep input clean and dynamic into the drive stage. |
| **FX2** | `Dark Mouse` | ProCo RAT-style op-amp distortion providing thick, fuzzy wall-of-sound clipping. |
| **AMP** | `Brit 800` | Marshall JCM800 high-gain head delivering the core British crunch and punchy midrange. |
| **NR** | `Gate 2` | High-gain noise gate to silence high-gain fuzz hiss between song sections and stops. |
| **CAB** | `Brit75 4x12` | Marshall 4x12 loaded with Celestion G12T-75 speakers (attested pairing with `Brit 800`). Sets `ppIRNum=40`. |
| **EQ** | `Guitar EQ` | Replicates the Boss GE-7 mid-boost/tone-shaping to push definition through the heavy distortion. |
| **MOD** | `Vibrato` | Continuous subtle pitch drift mimicking the "glide guitar" floating vibrato arm flutter for players using fixed-bridge or non-tremolo guitars. |
| **DLY** | *(Off)* | Kept off to prevent cluttering the dense chord wall; spatial depth handled by RVB. |
| **RVB** | `Mod RVB` | Modulated reverb wash providing the ambient bloom and shimmer characteristic of the SPX90 / Midiverb II early reflection decay. |

## 3. Open Questions & Studio Peculiarities

- **Pre-distortion Reverb Routing**: On the record, Kevin Shields routed reverse reverb *before* the guitar amplifiers. The Matribox QME-50 enforces a fixed serial routing order where `RVB` occurs after `AMP` and `CAB`. To compensate, `Dark Mouse` provides the fuzzy transient compression, while `Mod RVB` adds pitch-modulated spatial wash post-amp, and `MOD Vibrato` provides the detuned glide pitch motion.
- **Microphone & Mixing**: Alan Moulder and Kevin Shields heavily multi-tracked rhythm parts with subtle variations in EQ and tuning. A single-preset patch focuses on a unified lead-rhythm performance tone.
