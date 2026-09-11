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

func (u UserIR) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	name := "ppIRInfo" + strconv.Itoa(u.Slot)
	return e.EncodeElement(struct {
		IRNum string `xml:"ppIRNum,attr"`
		Name  string `xml:"ppIRName,attr"`
		CRC   string `xml:"ppIRCRC,attr"`
	}{u.IRNum, u.Name, u.CRC}, xml.StartElement{Name: xml.Name{Local: name}})
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

// MarshalXML emits module/name/state/code/x/y then params_0..params_14.
func (f Effect) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	start := xml.StartElement{Name: xml.Name{Local: "Effect"}}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "effectModuleName"}, Value: f.Module},
		{Name: xml.Name{Local: "effectName"}, Value: f.Name},
		{Name: xml.Name{Local: "effectState"}, Value: strconv.Itoa(f.State)},
		{Name: xml.Name{Local: "effectCode"}, Value: strconv.FormatUint(uint64(f.Code), 10)},
		{Name: xml.Name{Local: "params_0"}, Value: f.P[0]},
		{Name: xml.Name{Local: "x"}, Value: strconv.Itoa(f.X)},
		{Name: xml.Name{Local: "y"}, Value: strconv.Itoa(f.Y)},
	}
	for i := 1; i < 15; i++ {
		start.Attr = append(start.Attr, xml.Attr{
			Name: xml.Name{Local: "params_" + strconv.Itoa(i)}, Value: f.P[i],
		})
	}
	return e.EncodeElement(struct{}{}, start)
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

// MarshalXML keeps the original child order: Effects, ppEXP1, ppCtrl.
func (p Preset) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "presets"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "ppBank"}, Value: strconv.Itoa(p.Bank)},
		{Name: xml.Name{Local: "ppName"}, Value: p.Name},
		{Name: xml.Name{Local: "ppVolume"}, Value: strconv.Itoa(p.Volume)},
		{Name: xml.Name{Local: "ppID"}, Value: strconv.Itoa(p.ID)},
		{Name: xml.Name{Local: "ppBPM"}, Value: strconv.Itoa(p.BPM)},
		{Name: xml.Name{Local: "ppIRNum"}, Value: p.IRNum},
		{Name: xml.Name{Local: "ppType"}, Value: strconv.Itoa(p.Type)},
		{Name: xml.Name{Local: "ppAuthor"}, Value: p.Author},
		{Name: xml.Name{Local: "ppNotes"}, Value: p.Notes},
		{Name: xml.Name{Local: "ppTypeName"}, Value: p.TypeName},
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, f := range p.Effects {
		if err := e.Encode(f); err != nil {
			return err
		}
	}
	if p.EXP != nil {
		if err := e.Encode(*p.EXP); err != nil {
			return err
		}
	}
	if p.Ctrl != nil {
		if err := e.Encode(*p.Ctrl); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
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

func (c EXPChild) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	name := "ppEXP1_" + strconv.Itoa(c.Slot)
	type attrs struct {
		MId   string `xml:"expMId,attr"`
		Code  uint64 `xml:"expCode,attr"`
		Index int    `xml:"expIndex,attr"`
		Min   int    `xml:"expMin,attr"`
		Max   int    `xml:"expMax,attr"`
	}
	return e.EncodeElement(attrs{c.MId, c.Code, c.Index, c.Min, c.Max},
		xml.StartElement{Name: xml.Name{Local: name}})
}

type EXP struct {
	XMLName   xml.Name   `xml:"ppEXP1"`
	Target    int        `xml:"expTarget,attr"`
	Volume    int        `xml:"expVolume,attr"`
	VolumeMin int        `xml:"expVolumeMin,attr"`
	VolumeMax int        `xml:"expVolumeMax,attr"`
	Kids      []EXPChild `xml:"-"`
}

// MarshalXML emits ppEXP1 with its numbered ppEXP1_N children.
func (x EXP) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "ppEXP1"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "expTarget"}, Value: strconv.Itoa(x.Target)},
		{Name: xml.Name{Local: "expVolume"}, Value: strconv.Itoa(x.Volume)},
		{Name: xml.Name{Local: "expVolumeMin"}, Value: strconv.Itoa(x.VolumeMin)},
		{Name: xml.Name{Local: "expVolumeMax"}, Value: strconv.Itoa(x.VolumeMax)},
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, k := range x.Kids {
		if err := e.Encode(k); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
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

// MarshalXML emits the <ppIRInfo> wrapper with numbered child elements.
func (s IRInfoSet) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "ppIRInfo"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, ir := range s.IRs {
		if err := e.Encode(ir); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
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

// Save writes the bundle as XML (UTF-8 header, LF newlines).
func (b *Bundle) Save(path string) error {
	out, err := xml.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(xml.Header); err != nil {
		return err
	}
	_, err = f.Write(out)
	return err
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
