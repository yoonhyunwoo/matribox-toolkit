// Package prst reads and edits the Sonicake Matribox (QME-50) preset bundle
// XML format (exported by Matribox Setup, e.g. prsts.prst).
package prst

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Bundle is the root <Matribox> document.
type Bundle struct {
	XMLName xml.Name  `xml:"Matribox"`
	Info    Info      `xml:"preset_info"`
	IRInfo  IRInfoSet `xml:"ppIRInfo"`
	Presets []Preset  `xml:"presets"`
}

type Info struct {
	Software string `xml:"software,attr"`
	Firmware string `xml:"firmware,attr"`
	Product  string `xml:"product,attr"`
	Count    int    `xml:"count,attr"`
	Platform string `xml:"platform,attr"`
	Time     string `xml:"time,attr"`
}

type IRInfoSet struct {
	IRs []UserIR // element names are ppIRInfo0..ppIRInfoN (custom marshal)
}

type UserIR struct {
	Slot  int    // recovered from the element name suffix
	IRNum string `xml:"ppIRNum,attr"`
	Name  string `xml:"ppIRName,attr"`
	CRC   string `xml:"ppIRCRC,attr"`
}

// UnmarshalXML is not used directly; IRInfoSet.UnmarshalXML captures slots.
func (s *IRInfoSet) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if !strings.HasPrefix(t.Name.Local, "ppIRInfo") {
				return fmt.Errorf("unexpected element <%s> in ppIRInfo", t.Name.Local)
			}
			var u struct {
				IRNum string `xml:"ppIRNum,attr"`
				Name  string `xml:"ppIRName,attr"`
				CRC   string `xml:"ppIRCRC,attr"`
			}
			if err := d.DecodeElement(&u, &t); err != nil {
				return err
			}
			suffix := strings.TrimPrefix(t.Name.Local, "ppIRInfo")
			slot, err := strconv.Atoi(suffix)
			if err != nil {
				return fmt.Errorf("bad IR slot element %q: %w", t.Name.Local, err)
			}
			s.IRs = append(s.IRs, UserIR{Slot: slot, IRNum: u.IRNum, Name: u.Name, CRC: u.CRC})
		case xml.EndElement:
			return nil
		}
	}
}

type Effect struct {
	Module string     `xml:"effectModuleName,attr"`
	Name   string     `xml:"effectName,attr"`
	State  int        `xml:"effectState,attr"`
	Code   uint32     `xml:"effectCode,attr"`
	X      int        `xml:"x,attr"`
	Y      int        `xml:"y,attr"`
	P      [15]string `xml:"-"`
}

// UnmarshalXML reads all attributes; unknown params_N land in P[i].
func (f *Effect) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "effectModuleName":
			f.Module = a.Value
		case "effectName":
			f.Name = a.Value
		case "effectState":
			f.State, _ = strconv.Atoi(a.Value)
		case "effectCode":
			v, _ := strconv.ParseUint(a.Value, 10, 64)
			f.Code = uint32(v)
		case "x":
			f.X, _ = strconv.Atoi(a.Value)
		case "y":
			f.Y, _ = strconv.Atoi(a.Value)
		default:
			if strings.HasPrefix(a.Name.Local, "params_") {
				if i, err := strconv.Atoi(strings.TrimPrefix(a.Name.Local, "params_")); err == nil && i >= 0 && i < 15 {
					f.P[i] = a.Value
				}
			}
		}
	}
	// consume to the end element
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		if _, ok := tok.(xml.EndElement); ok {
			return nil
		}
	}
}

type Preset struct {
	Bank     int      `xml:"ppBank,attr"`
	Name     string   `xml:"ppName,attr"`
	Volume   int      `xml:"ppVolume,attr"`
	ID       int      `xml:"ppID,attr"`
	BPM      int      `xml:"ppBPM,attr"`
	IRNum    string   `xml:"ppIRNum,attr"`
	Type     int      `xml:"ppType,attr"`
	Author   string   `xml:"ppAuthor,attr"`
	Notes    string   `xml:"ppNotes,attr"`
	TypeName string   `xml:"ppTypeName,attr"`
	Effects  []Effect `xml:"-"`
	EXP      *EXP     `xml:"ppEXP1"`
	Ctrl     *Ctrl    `xml:"ppCtrl"`
}

// UnmarshalXML preserves Effect child order while parsing.
func (p *Preset) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "ppBank":
			p.Bank, _ = strconv.Atoi(a.Value)
		case "ppName":
			p.Name = a.Value
		case "ppVolume":
			p.Volume, _ = strconv.Atoi(a.Value)
		case "ppID":
			p.ID, _ = strconv.Atoi(a.Value)
		case "ppBPM":
			p.BPM, _ = strconv.Atoi(a.Value)
		case "ppIRNum":
			p.IRNum = a.Value
		case "ppType":
			p.Type, _ = strconv.Atoi(a.Value)
		case "ppAuthor":
			p.Author = a.Value
		case "ppNotes":
			p.Notes = a.Value
		case "ppTypeName":
			p.TypeName = a.Value
		}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "Effect":
				var f Effect
				if err := f.UnmarshalXML(d, t); err != nil {
					return err
				}
				p.Effects = append(p.Effects, f)
			case "ppEXP1":
				var x EXP
				if err := d.DecodeElement(&x, &t); err != nil {
					return err
				}
				p.EXP = &x
			case "ppCtrl":
				var c Ctrl
				if err := d.DecodeElement(&c, &t); err != nil {
					return err
				}
				p.Ctrl = &c
			default:
				return fmt.Errorf("unexpected element <%s> in preset", t.Name.Local)
			}
		case xml.EndElement:
			return nil
		}
	}
}

type EXPChild struct {
	Slot  int    // recovered from element name suffix ppEXP1_N
	MId   string `xml:"expMId,attr"`
	Code  uint64 `xml:"expCode,attr"`
	Index int    `xml:"expIndex,attr"`
	Min   int    `xml:"expMin,attr"`
	Max   int    `xml:"expMax,attr"`
}

type EXP struct {
	XMLName   xml.Name   `xml:"ppEXP1"`
	Target    int        `xml:"expTarget,attr"`
	Volume    int        `xml:"expVolume,attr"`
	VolumeMin int        `xml:"expVolumeMin,attr"`
	VolumeMax int        `xml:"expVolumeMax,attr"`
	Kids      []EXPChild `xml:"-"`
}

// UnmarshalXML captures attributes and the ppEXP1_N children.
func (x *EXP) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "expTarget":
			x.Target, _ = strconv.Atoi(a.Value)
		case "expVolume":
			x.Volume, _ = strconv.Atoi(a.Value)
		case "expVolumeMin":
			x.VolumeMin, _ = strconv.Atoi(a.Value)
		case "expVolumeMax":
			x.VolumeMax, _ = strconv.Atoi(a.Value)
		}
	}
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if !strings.HasPrefix(t.Name.Local, "ppEXP1_") {
				return fmt.Errorf("unexpected element <%s> in ppEXP1", t.Name.Local)
			}
			var k struct {
				MId   string `xml:"expMId,attr"`
				Code  uint64 `xml:"expCode,attr"`
				Index int    `xml:"expIndex,attr"`
				Min   int    `xml:"expMin,attr"`
				Max   int    `xml:"expMax,attr"`
			}
			if err := d.DecodeElement(&k, &t); err != nil {
				return err
			}
			slot, err := strconv.Atoi(strings.TrimPrefix(t.Name.Local, "ppEXP1_"))
			if err != nil {
				return fmt.Errorf("bad exp child %q: %w", t.Name.Local, err)
			}
			x.Kids = append(x.Kids, EXPChild{Slot: slot, MId: k.MId, Code: k.Code, Index: k.Index, Min: k.Min, Max: k.Max})
		case xml.EndElement:
			return nil
		}
	}
}

type Ctrl struct {
	XMLName xml.Name `xml:"ppCtrl"`
	C11     int      `xml:"c11,attr"`
	C12     int      `xml:"c12,attr"`
	C13     int      `xml:"c13,attr"`
	C21     int      `xml:"c21,attr"`
	C22     int      `xml:"c22,attr"`
	C23     int      `xml:"c23,attr"`
}

// Load parses a .prst bundle XML file.
func Load(path string) (*Bundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b Bundle
	if err := xml.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &b, nil
}

// Save writes the bundle in the device-compatible serialization: CRLF line
// endings, self-closing empty elements, attributes greedily wrapped at 93
// columns (the factory exporter's maximum observed line length is 94).
//
// The Matribox desktop app crashes on equivalent-but-different XML — observed
// with `<Effect ...></Effect>` pairs and LF endings (2026-09-12) — so this
// writer mirrors the factory exporter's byte style instead of canonical XML.
// The format's interleaved attribute order (params_0, x, y, params_1, ...)
// strongly suggests a hand-rolled parser; do not replace this with
// encoding/xml marshaling.
func (b *Bundle) Save(path string) error {
	w := &bundleWriter{}
	w.raw("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n\r\n<Matribox>\r\n")

	w.elem("  ", "preset_info", []kv{
		{"software", b.Info.Software},
		{"firmware", b.Info.Firmware},
		{"product", b.Info.Product},
		{"count", strconv.Itoa(b.Info.Count)},
		{"platform", b.Info.Platform},
		{"time", b.Info.Time},
	}, true)

	// Single-preset patch files (count=1) carry no ppIRInfo section — the
	// desktop app rejects patch files that include one ("wrong patch file").
	// Full-bundle re-saves keep it.
	if len(b.IRInfo.IRs) > 0 {
		w.open("  ", "ppIRInfo")
		w.endTag()
		for _, ir := range b.IRInfo.IRs {
			w.elem("    ", "ppIRInfo"+strconv.Itoa(ir.Slot), []kv{
				{"ppIRNum", ir.IRNum},
				{"ppIRName", ir.Name},
				{"ppIRCRC", ir.CRC},
			}, true)
		}
		w.close("  ", "ppIRInfo")
	}

	for i := range b.Presets {
		p := &b.Presets[i]
		w.open("  ", "presets")
		w.wrapAttrs("presets", []kv{
			{"ppBank", strconv.Itoa(p.Bank)},
			{"ppName", p.Name},
			{"ppVolume", strconv.Itoa(p.Volume)},
			{"ppID", strconv.Itoa(p.ID)},
			{"ppBPM", strconv.Itoa(p.BPM)},
			{"ppIRNum", p.IRNum},
			{"ppType", strconv.Itoa(p.Type)},
			{"ppAuthor", p.Author},
			{"ppNotes", p.Notes},
			{"ppTypeName", p.TypeName},
		})
		w.endTag()
		for _, f := range p.Effects {
			attrs := []kv{
				{"effectModuleName", f.Module},
				{"effectName", f.Name},
				{"effectState", strconv.Itoa(f.State)},
				{"effectCode", strconv.FormatUint(uint64(f.Code), 10)},
				{"params_0", f.P[0]},
				{"x", strconv.Itoa(f.X)},
				{"y", strconv.Itoa(f.Y)},
			}
			for n := 1; n < 15; n++ {
				attrs = append(attrs, kv{"params_" + strconv.Itoa(n), f.P[n]})
			}
			w.elem("    ", "Effect", attrs, true)
		}
		if p.Ctrl != nil {
			w.elem("    ", "ppCtrl", []kv{
				{"c11", strconv.Itoa(p.Ctrl.C11)},
				{"c12", strconv.Itoa(p.Ctrl.C12)},
				{"c13", strconv.Itoa(p.Ctrl.C13)},
				{"c21", strconv.Itoa(p.Ctrl.C21)},
				{"c22", strconv.Itoa(p.Ctrl.C22)},
				{"c23", strconv.Itoa(p.Ctrl.C23)},
			}, true)
		}
		if p.EXP != nil {
			if len(p.EXP.Kids) == 0 {
				w.elem("    ", "ppEXP1", []kv{
					{"expTarget", strconv.Itoa(p.EXP.Target)},
					{"expVolume", strconv.Itoa(p.EXP.Volume)},
					{"expVolumeMin", strconv.Itoa(p.EXP.VolumeMin)},
					{"expVolumeMax", strconv.Itoa(p.EXP.VolumeMax)},
				}, true)
			} else {
				w.open("    ", "ppEXP1")
				w.wrapAttrs("ppEXP1", []kv{
					{"expTarget", strconv.Itoa(p.EXP.Target)},
					{"expVolume", strconv.Itoa(p.EXP.Volume)},
					{"expVolumeMin", strconv.Itoa(p.EXP.VolumeMin)},
					{"expVolumeMax", strconv.Itoa(p.EXP.VolumeMax)},
				})
				w.endTag()
				for _, k := range p.EXP.Kids {
					w.elem("      ", "ppEXP1_"+strconv.Itoa(k.Slot), []kv{
						{"expMId", k.MId},
						{"expCode", strconv.FormatUint(k.Code, 10)},
						{"expIndex", strconv.Itoa(k.Index)},
						{"expMin", strconv.Itoa(k.Min)},
						{"expMax", strconv.Itoa(k.Max)},
					}, true)
				}
				w.close("    ", "ppEXP1")
			}
		}
		w.close("  ", "presets")
	}
	w.raw("</Matribox>\r\n")

	return os.WriteFile(path, []byte(w.buf.String()), 0o644)
}

type kv struct{ k, v string }

const maxCols = 93 // factory exporter never exceeds 94 columns

// bundleWriter assembles factory-style XML: element attributes sit on as few
// physical lines as possible without exceeding maxCols, continuation lines
// aligned one space past the tag name.
type bundleWriter struct {
	buf strings.Builder
	// cur tracks the current line length and the wrap column of the element
	// being written.
	cur  int
	wrap int
}

func (w *bundleWriter) raw(s string) {
	w.buf.WriteString(s)
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		w.cur = len(s) - i - 1
	} else {
		w.cur += len(s)
	}
}

func (w *bundleWriter) nl() { w.raw("\r\n") }

// open writes an element's start; follow with wrapAttrs and endTag.
func (w *bundleWriter) open(indent, name string) {
	w.raw(indent + "<" + name)
	w.wrap = w.cur + 1
}

func (w *bundleWriter) endTag() { w.raw(">" + "\r\n") }

func (w *bundleWriter) close(indent, name string) {
	w.raw(indent + "</" + name + ">\r\n")
}

// wrapAttrs appends attributes to the element opened by open, wrapping at
// maxCols with continuation lines aligned to the element's wrap column.
func (w *bundleWriter) wrapAttrs(_ string, attrs []kv) {
	for _, a := range attrs {
		text := " " + a.k + "=\"" + escapeAttr(a.v) + "\""
		if w.cur+len(text) > maxCols {
			w.nl()
			w.raw(strings.Repeat(" ", w.wrap))
		}
		w.raw(text)
	}
}

// elem writes a complete element, self-closing or not.
func (w *bundleWriter) elem(indent, name string, attrs []kv, selfClose bool) {
	w.open(indent, name)
	w.wrapAttrs(name, attrs)
	if selfClose {
		w.raw("/>")
	} else {
		w.raw(">")
	}
	w.nl()
}

func escapeAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// FindIR validates a ppIRNum against the bundle's user IR table.
func (b *Bundle) FindIR(irNum string) (*UserIR, error) {
	for i := range b.IRInfo.IRs {
		if b.IRInfo.IRs[i].IRNum == irNum {
			return &b.IRInfo.IRs[i], nil
		}
	}
	return nil, fmt.Errorf("ppIRNum %s not in bundle IR table (%d slots)", irNum, len(b.IRInfo.IRs))
}

// PresetByID returns the preset with the given ppID.
func (b *Bundle) PresetByID(id int) (*Preset, error) {
	for i := range b.Presets {
		if b.Presets[i].ID == id {
			return &b.Presets[i], nil
		}
	}
	return nil, fmt.Errorf("no preset with ppID=%d", id)
}

// NextID returns one past the highest used ppID.
func (b *Bundle) NextID() int {
	max := -1
	for _, p := range b.Presets {
		if p.ID > max {
			max = p.ID
		}
	}
	return max + 1
}

// CloneAs deep-copies a preset into a new entry with the given ppID, name
// (space-padded to 12 chars like factory presets), and user IR number.
func (p *Preset) CloneAs(id int, name, irNum string) *Preset {
	c := *p
	c.ID = id
	c.Name = padName(name)
	if irNum != "" {
		c.IRNum = irNum
	}
	c.Effects = make([]Effect, len(p.Effects))
	copy(c.Effects, p.Effects)
	for i := range c.Effects {
		c.Effects[i].P = p.Effects[i].P
	}
	if p.EXP != nil {
		e := *p.EXP
		c.EXP = &e
	}
	if p.Ctrl != nil {
		ct := *p.Ctrl
		c.Ctrl = &ct
	}
	return &c
}

func padName(name string) string {
	if len(name) > 12 {
		return name[:12]
	}
	return name + strings.Repeat(" ", 12-len(name))
}
