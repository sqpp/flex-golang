package flex_test

import (
	"testing"

	flex "github.com/sqpp/flex-golang"
)

func TestReferenceWireRoundtrip(t *testing.T) {
	refBIW := uint32(0x19400807)
	refAddr := uint32(0xC9808779)

	for name, cws := range map[string][]uint32{
		"ref_wire": {refBIW, refAddr, 0, 0, 0, 0, 0, 0},
		"our_wire": {flex.ExportFlexEncodeWord(0x807), flex.ExportFlexEncodeWord(0x8779), 0, 0, 0, 0, 0, 0},
	} {
		codewords := make([]uint32, flex.ExportFlexCodewordCount())
		for i := range codewords {
			codewords[i] = flex.ExportIdleCodeword(i)
		}
		copy(codewords, cws)

		wav, err := flex.ExportBitstreamFromCodewords(codewords, flex.Mode1600_2, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		frames, err := flex.DemodulateRawFrames(wav)
		if err != nil {
			t.Fatal(err)
		}
		if len(frames) == 0 {
			t.Fatalf("%s: no frames", name)
		}
		t.Logf("%s roundtrip cw[0]=0x%08X cw[1]=0x%08X", name, frames[0].Words[0], frames[0].Words[1])
	}
}

func refLayoutEncode(logical21 uint32) uint32 {
	infoMSB := flex.ExportReverse21(logical21 & 0x1FFFFF)
	poc := flex.ExportBCHEncode31_21(infoMSB) & 0x7FFFFFFF
	rev := (poc << 1) | uint32(flex.ExportPopCount32(poc)&1)
	return flex.ExportReverse32(rev)
}

func TestFindEncodeForRef(t *testing.T) {
	targets := map[string]uint32{
		"BIW":  0x19400807,
		"ADDR": 0xC9808779,
	}
	logicals := map[string]uint32{
		"BIW":  0x807,
		"ADDR": 0x8779,
	}

	for name, logical := range logicals {
		target := targets[name]
		ref := refLayoutEncode(logical)
		t.Logf("%s logical=0x%05X refLayout=0x%08X target=0x%08X match=%v",
			name, logical, ref, target, ref == target)
	}
}
