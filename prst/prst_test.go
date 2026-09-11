package prst

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixture = `<?xml version="1.0" encoding="UTF-8"?>
<Matribox>
  <preset_info software="1.0.3" firmware="1.1" product="Matribox" count="2" platform="MAC" time="1789136041945"/>
  <ppIRInfo>
    <ppIRInfo0 ppIRNum="1000" ppIRName="User IR 1" ppIRCRC="123"/>
    <ppIRInfo1 ppIRNum="1001" ppIRName="User IR 2" ppIRCRC="456"/>
  </ppIRInfo>
  <presets ppBank="0" ppName="MatriBox   " ppVolume="50" ppID="0" ppBPM="85"
           ppIRNum="0" ppType="4" ppAuthor="" ppNotes="" ppTypeName="Rock">
    <Effect effectModuleName="RVB" effectName="Hall" effectState="1" effectCode="201326593"
            params_0="30" x="8" y="0" params_1="36" params_2="42" params_3="0"
            params_4="65535" params_5="2303" params_6="7936" params_7="21760"
            params_8="1" params_9="0" params_10="0" params_11="0" params_12="0"
            params_13="0" params_14="0"/>
    <Effect effectModuleName="AMP" effectName="Calif DualV" effectState="1" effectCode="117440616"
            params_0="50" x="2" y="0" params_1="60" params_2="50" params_3="52" params_4="37"
            params_5="54" params_6="50" params_7="12800" params_8="65280" params_9="10495"
            params_10="20736" params_11="12800" params_12="0" params_13="0" params_14="0"/>
    <ppEXP1 expTarget="0" expVolume="0" expVolumeMin="0" expVolumeMax="99">
      <ppEXP1_0 expMId="0" expCode="83886088" expIndex="0" expMin="0" expMax="99"/>
      <ppEXP1_1 expMId="65535" expCode="524295" expIndex="0" expMin="0" expMax="99"/>
    </ppEXP1>
    <ppCtrl c11="1" c12="65535" c13="65535" c21="7" c22="8" c23="65535"/>
  </presets>
  <presets ppBank="0" ppName="60's OD  " ppVolume="70" ppID="1" ppBPM="120"
           ppIRNum="0" ppType="2" ppAuthor="" ppNotes="" ppTypeName="Blues">
    <Effect effectModuleName="AMP" effectName="B-Man N" effectState="1" effectCode="117440515"
            params_0="50" x="2" y="0" params_1="60" params_2="50" params_3="52" params_4="37"
            params_5="54" params_6="50" params_7="12800" params_8="65280" params_9="10495"
            params_10="20736" params_11="12800" params_12="0" params_13="0" params_14="0"/>
    <ppEXP1 expTarget="0" expVolume="0" expVolumeMin="0" expVolumeMax="99"/>
    <ppCtrl c11="1" c12="65535" c13="65535" c21="7" c22="8" c23="65535"/>
  </presets>
</Matribox>
`

func loadFixture(t *testing.T) *Bundle {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.prst")
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return b
}

func TestLoad(t *testing.T) {
	b := loadFixture(t)
	if len(b.Presets) != 2 {
		t.Fatalf("presets = %d, want 2", len(b.Presets))
	}
	if len(b.IRInfo.IRs) != 2 || b.IRInfo.IRs[0].Slot != 0 || b.IRInfo.IRs[1].Slot != 1 {
		t.Fatalf("IR slots parsed wrong: %+v", b.IRInfo.IRs)
	}
	p := b.Presets[0]
	if len(p.Effects) != 2 {
		t.Fatalf("effects = %d, want 2", len(p.Effects))
	}
	// Dynamic attributes must survive the round trip through the parser.
	if got := p.Effects[0].P[10]; got != "0" {
		t.Errorf("params_10 = %q", got)
	}
	if p.Effects[1].Code != 117440616 {
		t.Errorf("AMP effectCode = %d", p.Effects[1].Code)
	}
	if p.EXP == nil || len(p.EXP.Kids) != 2 || p.EXP.Kids[0].Slot != 0 {
		t.Fatalf("EXP children parsed wrong: %+v", p.EXP)
	}
	if p.Ctrl == nil || p.Ctrl.C11 != 1 {
		t.Fatalf("Ctrl parsed wrong: %+v", p.Ctrl)
	}
}

func TestRoundTripPreservesStructure(t *testing.T) {
	b := loadFixture(t)
	path := filepath.Join(t.TempDir(), "out.prst")
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
	b2, err := Load(path)
	if err != nil {
		t.Fatalf("re-load: %v", err)
	}
	if len(b2.Presets) != 2 || len(b2.IRInfo.IRs) != 2 {
		t.Fatalf("structure lost on round trip: %+v", b2)
	}
	p := b2.Presets[0]
	if len(p.Effects) != 2 || p.Effects[0].P[4] != "65535" {
		t.Fatalf("effect params lost: %+v", p.Effects[0])
	}
	if p.EXP == nil || len(p.EXP.Kids) != 2 || p.EXP.Kids[1].Code != 524295 {
		t.Fatalf("EXP children lost: %+v", p.EXP)
	}
	// Numbered element names must be regenerated, not Go type names.
	data, _ := os.ReadFile(path)
	s := string(data)
	for _, want := range []string{"<ppIRInfo0 ", "<ppEXP1_0 ", "<ppEXP1 ", "<ppCtrl "} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s", want)
		}
	}
	if strings.Contains(s, "<EXP") || strings.Contains(s, "<Ctrl") {
		t.Error("output leaked Go type names as XML elements")
	}
}

func TestFindIR(t *testing.T) {
	b := loadFixture(t)
	ir, err := b.FindIR("1001")
	if err != nil {
		t.Fatal(err)
	}
	if ir.Name != "User IR 2" {
		t.Errorf("FindIR name = %q", ir.Name)
	}
	if _, err := b.FindIR("9999"); err == nil {
		t.Error("expected error for unknown ppIRNum")
	}
}

func TestCloneAs(t *testing.T) {
	b := loadFixture(t)
	src, err := b.PresetByID(0)
	if err != nil {
		t.Fatal(err)
	}
	clone := src.CloneAs(b.NextID(), "My Long Tone Name", "1000")
	if clone.ID != 2 {
		t.Errorf("clone ID = %d, want 2", clone.ID)
	}
	if clone.Name != "My Long Tone" {
		t.Errorf("clone name = %q, want 12-char truncation", clone.Name)
	}
	if clone.IRNum != "1000" {
		t.Errorf("clone IRNum = %q", clone.IRNum)
	}
	// Deep copy: mutating the clone must not touch the source.
	clone.Effects[0].P[0] = "999"
	if src.Effects[0].P[0] == "999" {
		t.Error("clone shares effect params with source")
	}
}

func TestPadName(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"abc", "abc         "},
		{"abcdefghijkl", "abcdefghijkl"},
		{"abcdefghijklmno", "abcdefghijkl"},
	} {
		if got := padName(tc.in); got != tc.want {
			t.Errorf("padName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
