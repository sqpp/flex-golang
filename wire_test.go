package flex

import (
	"testing"
)

func TestReferenceWireRoundtrip(t *testing.T) {
	refBIW := uint32(0x19400807)
	refAddr := uint32(0xC9808779)

	for name, cws := range map[string][]uint32{
		"ref_wire": {refBIW, refAddr, 0, 0, 0, 0, 0, 0},
		"our_wire": {flexEncodeWord(0x807), flexEncodeWord(0x8779), 0, 0, 0, 0, 0, 0},
	} {
		codewords := make([]uint32, flexCodewords)
		for i := range codewords {
			codewords[i] = idleCodeword(i)
		}
		copy(codewords, cws)

		bits, err := bitstreamFromCodewords(codewords, encodeModes[Mode1600_2], 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		wav := modulateBits(bits, 1600)
		frames, err := DemodulateRawFrames(wav)
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
	infoMSB := reverse21(logical21 & 0x1FFFFF)
	poc := BCHEncode31_21(infoMSB) & 0x7FFFFFFF
	rev := (poc << 1) | uint32(popCount32(poc)&1)
	return reverse32(rev)
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

func flexWireLogical(w uint32) uint32 { return (w & 0x1FFFFF) ^ 0x1FFFFF }

func oldLayoutEncode(logical21 uint32) uint32 {
	poc := BCHEncode31_21(logical21 & 0x1FFFFF)
	cw := (poc >> 10) | ((poc & 0x3FF) << 21)
	if popCount32(cw)&1 != 0 {
		cw |= 1 << 31
	}
	return cw
}
